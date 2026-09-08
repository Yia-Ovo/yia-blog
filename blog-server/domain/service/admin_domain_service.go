package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AdminDomainService struct {
	adminMysqlRepo repo.AdminMysqlRepo
}

func NewAdminDomainService(adminMysqlRepo repo.AdminMysqlRepo) *AdminDomainService {
	return &AdminDomainService{adminMysqlRepo: adminMysqlRepo}
}

// Login 管理员登录
func (a *AdminDomainService) Login(ctx context.Context, adminName string, password string) (*do.AdminDO, error) {
	if adminName == "" || password == "" {
		logger.Warn("[DOMAIN] 管理员登录参数为空")
		return nil, common.LoginParamEmptyErr
	}

	adminModel, err := a.adminMysqlRepo.FindAdminByName(ctx, adminName)
	if err != nil {
		logger.Warn("[DOMAIN] 管理员查询失败", zap.Error(err), zap.String("admin_name", adminName))
		return nil, common.AdminQueryErr.WithErr(err)
	}
	if adminModel == nil {
		return nil, common.LoginPasswordErr
	}

	adminDO := convert.AdminModelToDO(adminModel)

	// 状态校验
	if adminDO.Status != consts.AdminStatusEnabled {
		return nil, common.AdminAccountDisabledErr
	}

	// 密码校验
	if err := bcrypt.CompareHashAndPassword([]byte(adminDO.Password), []byte(password)); err != nil {
		return nil, common.LoginPasswordErr
	}

	_ = a.adminMysqlRepo.UpdateLastLoginInfo(ctx, adminDO.ID, adminDO.LastLoginAt)
	return adminDO, nil
}

// UpdatePWD 修改密码
func (a *AdminDomainService) UpdatePWD(ctx context.Context, adminId uint64, newPassword, oldPassword string) error {
	adminDO, err := a.adminMysqlRepo.FindAdminByID(ctx, adminId)
	if err != nil {
		logger.Warn("[DOMAIN] 管理员查询失败", zap.Error(err), zap.Uint64("admin_id", adminId))
		return common.AdminQueryErr
	}
	if adminDO == nil {
		return common.AdminNotFoundErr
	}

	if err := bcrypt.CompareHashAndPassword([]byte(adminDO.Password), []byte(oldPassword)); err != nil {
		return common.UserOldPasswordErr
	}

	if newPassword == oldPassword {
		return common.UserPasswordSameError
	}

	hashPwd, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Warn("[DOMAIN] 密码加密失败", zap.Error(err))
		return common.PasswordEncryptError
	}

	if err := a.adminMysqlRepo.UpdatePWD(ctx, adminId, string(hashPwd)); err != nil {
		logger.Warn("[DOMAIN] 密码更新失败", zap.Error(err), zap.Uint64("admin_id", adminId))
		return common.UpdatePasswordError
	}
	return nil
}

// UpdateAdminInfo 校验管理员资料更新，nil 字段保持不变，空字符串可清空非账号字段。
// Repo 直接接收部分更新，不把读取时的旧值重新写回，也不改变角色、状态或登录态。
func (a *AdminDomainService) UpdateAdminInfo(ctx context.Context, adminId uint64, data *do.AdminInfoUpdate) error {
	if data == nil || adminId == 0 {
		logger.Warn("[DOMAIN] 更新管理员信息失败：更新数据为空", zap.Uint64("admin_id", adminId))
		return common.ParamErr
	}
	if data.AdminName != nil && *data.AdminName == "" {
		return common.ParamErr.WithMsg("管理员账号不能为空")
	}
	if _, err := a.FindAdminByID(ctx, adminId); err != nil {
		return err
	}

	if err := a.adminMysqlRepo.UpdateAdminInfo(ctx, adminId, data); err != nil {
		logger.Warn("[DOMAIN] 管理员信息更新失败", zap.Error(err), zap.Uint64("admin_id", adminId))
		return common.AdminUpdateErr
	}
	return nil
}

// UpdateAdminRole 变更管理员角色。
// operatorID 是操作者（来自鉴权态），targetID 是被操作对象。
// 权限不变量由 ensureCanManageAdmin 统一保证：操作者须为启用的超管、不能操作自己、目标须为普通管理员。
// 目标不能是超管，沿用仅允许提升普通管理员的规则；相同角色返回 changed=false，
// App 据此保留缓存和在线令牌，避免未发生变更也强制登出。
func (a *AdminDomainService) UpdateAdminRole(ctx context.Context, operatorID uint64, targetID uint64, role int) (changed bool, err error) {
	if operatorID == targetID {
		return false, common.AdminChangeSelfRoleErr
	}
	if err := a.ensureCanManageAdmin(ctx, operatorID, targetID); err != nil {
		return false, err
	}

	if role != consts.AdminRoleSuper && role != consts.AdminRoleNormal {
		return false, common.AdminRoleInvalidErr
	}
	// ensureCanManageAdmin 已保证目标当前为普通管理员。
	if role == consts.AdminRoleNormal {
		return false, nil
	}

	if err := a.adminMysqlRepo.ChangeAdminRole(ctx, targetID, role); err != nil {
		if errors.Is(err, repo.ErrNoRowsAffected) {
			return false, common.AdminConcurrentModifyErr
		}
		logger.Warn("[DOMAIN] 管理员角色修改失败", zap.Error(err), zap.Uint64("admin_id", targetID))
		return false, common.AdminUpdateErr.WithErr(err)
	}
	return true, nil
}

// CreateAdmin 创建管理员
func (a *AdminDomainService) CreateAdmin(ctx context.Context, name, pwd string, role int) (*do.AdminDO, error) {
	_, err := a.FindAdminByName(ctx, name)
	if err == nil {
		logger.Warn("[DOMAIN] 管理员已存在", zap.String("admin_name", name))
		return nil, common.AdminAlreadyExistErr
	}
	if !errors.Is(err, common.AdminNotFoundErr) {
		return nil, err
	}

	hashPwd, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		logger.Warn("[DOMAIN] 密码加密失败", zap.Error(err))
		return nil, common.PasswordEncryptError
	}

	adminDO := &do.AdminDO{
		AdminName: name,
		Password:  string(hashPwd),
		Role:      role,
		Status:    1,
		CreateAt:  time.Now(),
		UpdateAt:  time.Now(),
	}

	model := convert.AdminDOToModel(adminDO)
	if err := a.adminMysqlRepo.CreateAdmin(ctx, model); err != nil {
		logger.Warn("[DOMAIN] 创建管理员失败", zap.Error(err), zap.String("admin_name", name))
		return nil, common.AdminCreateErr
	}
	return adminDO, nil
}

// FindAdminByID ID 查询
func (a *AdminDomainService) FindAdminByID(ctx context.Context, id uint64) (*do.AdminDO, error) {
	model, err := a.adminMysqlRepo.FindAdminByID(ctx, id)
	if err != nil {
		logger.Warn("[DOMAIN] 管理员查询失败", zap.Error(err), zap.Uint64("admin_id", id))
		return nil, common.AdminQueryErr.WithErr(err)
	}
	if model == nil {
		return nil, common.AdminNotFoundErr
	}
	return convert.AdminModelToDO(model), nil
}

// FindAdminByName 用户名查询
func (a *AdminDomainService) FindAdminByName(ctx context.Context, name string) (*do.AdminDO, error) {
	model, err := a.adminMysqlRepo.FindAdminByName(ctx, name)
	if err != nil {
		logger.Warn("[DOMAIN] 管理员查询失败", zap.Error(err), zap.String("admin_name", name))
		return nil, common.AdminQueryErr.WithErr(err)
	}
	if model == nil {
		return nil, common.AdminNotFoundErr
	}
	return convert.AdminModelToDO(model), nil
}

// ListAdmin 列表
func (a *AdminDomainService) ListAdmin(ctx context.Context, page, size int) ([]*do.AdminDO, int64, error) {
	list, total, err := a.adminMysqlRepo.ListAdmin(ctx, page, size)
	if err != nil {
		logger.Warn("[DOMAIN] 管理员列表查询失败", zap.Error(err))
		return nil, 0, common.AdminQueryErr
	}

	var res []*do.AdminDO
	for _, v := range list {
		res = append(res, convert.AdminModelToDO(v))
	}
	return res, total, nil
}

// UpdateAdminStatus 超管启用/禁用指定管理员。
// operatorId 是操作者（来自鉴权态），adminID 是被操作对象；
// 权限不变量由 ensureCanManageAdmin 保证，Repo 侧 SQL 条件做并发兜底。
func (a *AdminDomainService) UpdateAdminStatus(ctx context.Context, operatorId uint64, adminID uint64, status int) error {
	if status != consts.AdminStatusEnabled && status != consts.AdminStatusDisabled {
		return common.ParamErr.WithMsg("管理员状态不合法")
	}
	if err := a.ensureCanManageAdmin(ctx, operatorId, adminID); err != nil {
		return err
	}

	if err := a.adminMysqlRepo.UpdateAdminStatus(ctx, adminID, status); err != nil {
		// 校验通过后目标被并发提升为超管或删除：如实报冲突，不能静默当成功
		if errors.Is(err, repo.ErrNoRowsAffected) {
			logger.Warn("[DOMAIN] 状态更新未生效，目标已变更", zap.Uint64("admin_id", adminID))
			return common.AdminConcurrentModifyErr.WithMsg("目标管理员状态已变更，请刷新后重试")
		}
		logger.Warn("[DOMAIN] 管理员状态更新失败", zap.Error(err), zap.Uint64("admin_id", adminID))
		return common.AdminUpdateErr
	}
	return nil
}

// ResetAdminPWD 超管重置指定管理员的密码。
// operatorID 是操作者（来自鉴权态），adminID 是被重置对象；
// 权限不变量由 ensureCanManageAdmin 保证，Repo 侧 SQL 条件做并发兜底。
func (a *AdminDomainService) ResetAdminPWD(ctx context.Context, operatorID uint64, adminID uint64, newPwd string) error {
	if newPwd == "" {
		return common.ParamErr.WithMsg("新密码不能为空")
	}
	if err := a.ensureCanManageAdmin(ctx, operatorID, adminID); err != nil {
		return err
	}
	hashPwd, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		logger.Warn("[DOMAIN] 密码加密失败", zap.Error(err))
		return common.PasswordEncryptError
	}

	if err := a.adminMysqlRepo.ResetAdminPWDByID(ctx, adminID, string(hashPwd)); err != nil {
		// 校验通过后目标被并发提升为超管或删除：拒绝，而不是假装成功
		if errors.Is(err, repo.ErrNoRowsAffected) {
			logger.Warn("[DOMAIN] 重置密码未生效，目标已变更", zap.Uint64("admin_id", adminID))
			return common.AdminConcurrentModifyErr.WithMsg("目标管理员状态已变更，请刷新后重试")
		}
		logger.Warn("[DOMAIN] 重置密码失败", zap.Error(err), zap.Uint64("admin_id", adminID))
		return common.UpdatePasswordError
	}
	return nil
}

// ensureCanManageAdmin 保证启用的超管只能管理其他普通管理员，任何入口都必须遵守。
func (a *AdminDomainService) ensureCanManageAdmin(ctx context.Context, operatorID, targetID uint64) error {
	if operatorID == targetID {
		return common.PermissionDeniedErr.WithMsg("不能对自己执行该操作")
	}
	// 1. 操作者：必须存在、启用、且为超级管理员
	operator, err := a.adminMysqlRepo.FindAdminByID(ctx, operatorID)
	if err != nil {
		return common.AdminQueryErr.WithErr(err)
	}
	if operator == nil {
		return common.PermissionDeniedErr.WithMsg("操作者不存在")
	}
	op := convert.AdminModelToDO(operator)
	if op.Status != consts.AdminStatusEnabled || op.Role != consts.AdminRoleSuper {
		return common.PermissionDeniedErr.WithMsg("仅启用的超级管理员可执行该操作")
	}
	// 2. 目标：必须存在、且为普通管理员
	target, err := a.adminMysqlRepo.FindAdminByID(ctx, targetID)
	if err != nil {
		return common.AdminQueryErr.WithErr(err)
	}
	if target == nil {
		return common.AdminNotFoundErr
	}

	if convert.AdminModelToDO(target).Role != consts.AdminRoleNormal {
		return common.PermissionDeniedErr.WithMsg("只能操作普通管理员")
	}

	return nil
}
