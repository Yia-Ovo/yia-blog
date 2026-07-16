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
		if errors.Is(err, common.AdminNotFoundErr) {
			return nil, common.LoginPasswordErr
		}
		logger.Warn("[DOMAIN] 管理员查询失败", zap.Error(err), zap.String("admin_name", adminName))
		return nil, common.AdminQueryErr
	}

	adminDO := convert.AdminModelToDO(adminModel)

	// 状态校验
	if adminDO.Status == 0 {
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

// UpdateAdminInfo 更新信息，roleProvided 表示请求体是否显式传了 role 字段。
// 这里不能只用 data.Role != 0 判断，因为 0 本身就是超管角色。
func (a *AdminDomainService) UpdateAdminInfo(ctx context.Context, adminId uint64, data *do.AdminDO, roleProvided bool) (bool, error) {
	adminDO, err := a.FindAdminByID(ctx, adminId)
	if err != nil {
		return false, err
	}

	if data.AdminName != "" {
		adminDO.AdminName = data.AdminName
	}
	if data.Avatar != "" {
		adminDO.Avatar = data.Avatar
	}
	// 只有“请求传了 role”且“新旧角色不同”才算真正修改角色。
	// 这样普通资料更新误带相同 role 不会被当成越权，也不会无意义踢登录态。
	roleChanged := roleProvided && data.Role != adminDO.Role
	if roleChanged {
		if adminDO.Role != consts.AdminRoleSuper {
			return false, common.PermissionDeniedErr.WithMsg("无权修改管理员角色")
		}
		adminDO.Role = data.Role
	}
	if data.Email != "" {
		adminDO.Email = data.Email
	}
	if data.Remark != "" {
		adminDO.Remark = data.Remark
	}

	adminDO.UpdateAt = time.Now()
	model := convert.AdminDOToModel(adminDO)

	if err := a.adminMysqlRepo.UpdateAdminInfo(ctx, model, roleChanged); err != nil {
		logger.Warn("[DOMAIN] 管理员信息更新失败", zap.Error(err), zap.Uint64("admin_id", adminId))
		return false, common.AdminUpdateErr
	}
	return roleChanged, nil
}

// CreateAdmin 创建管理员
func (a *AdminDomainService) CreateAdmin(ctx context.Context, name, pwd string, role int) (*do.AdminDO, error) {
	_, err := a.FindAdminByName(ctx, name)
	if err == nil {
		logger.Warn("[DOMAIN] 管理员已存在", zap.String("admin_name", name))
		return nil, common.AdminAlreadyExistErr
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
		return nil, common.AdminQueryErr
	}
	return convert.AdminModelToDO(model), nil
}

// FindAdminByName 用户名查询
func (a *AdminDomainService) FindAdminByName(ctx context.Context, name string) (*do.AdminDO, error) {
	model, err := a.adminMysqlRepo.FindAdminByName(ctx, name)
	if err != nil {
		logger.Warn("[DOMAIN] 管理员查询失败", zap.Error(err), zap.String("admin_name", name))
		return nil, common.AdminQueryErr
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

// UpdateAdminStatus 更新状态
func (a *AdminDomainService) UpdateAdminStatus(ctx context.Context, operatorId uint64, adminID uint64, status int) error {
	if err := a.ensureCanManageAdmin(ctx, operatorId, adminID); err != nil {
		return err
	}

	if err := a.adminMysqlRepo.UpdateAdminStatus(ctx, adminID, status); err != nil {
		logger.Warn("[DOMAIN] 管理员状态更新失败", zap.Error(err), zap.Uint64("admin_id", adminID))
		return common.AdminUpdateErr
	}
	return nil
}

// ResetAdminPWD 重置密码
func (a *AdminDomainService) ResetAdminPWD(ctx context.Context, operatorID uint64, adminID uint64, newPwd string) error {
	if err := a.ensureCanManageAdmin(ctx, operatorID, adminID); err != nil {
		return err
	}
	hashPwd, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		logger.Warn("[DOMAIN] 密码加密失败", zap.Error(err))
		return common.PasswordEncryptError
	}

	if err := a.adminMysqlRepo.UpdatePWD(ctx, adminID, string(hashPwd)); err != nil {
		logger.Warn("[DOMAIN] 重置密码失败", zap.Error(err), zap.Uint64("admin_id", adminID))
		return common.UpdatePasswordError
	}
	return nil
}

// 工具函数
func (s *AdminDomainService) ensureCanManageAdmin(ctx context.Context, operatorID, targetID uint64) error {
	if operatorID == targetID {
		return common.PermissionDeniedErr.WithMsg("不能对自己执行该操作")
	}
	target, err := s.FindAdminByID(ctx, targetID)
	if err != nil {
		return err.(common.Errno).WithMsg("目标管理员不存在")
	}
	if target.Role == consts.AdminRoleSuper {
		return common.PermissionDeniedErr.WithMsg("不能操作其他超级管理员")
	}
	return nil
}
