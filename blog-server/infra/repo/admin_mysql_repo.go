/*
 * @Author: Yia 1372594487@gmail.com
 * @Date: 2026-04-30 01:46:01
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-08-12 12:43:17
 * @FilePath: \yia-blog\blog-server\infra\repo\admin_mysql_repo.go
 * @Description: 管理员 MySQL 仓储实现
 */
package repo

import (
	"blog-server/common/consts"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type adminMysqlRepo struct {
	db *gorm.DB
}

func NewAdminMysqlRepo(db *gorm.DB) repo.AdminMysqlRepo {
	return &adminMysqlRepo{db: db}
}

// adminBaseQuery 基础查询构造器（统一上下文与模型）
func (r *adminMysqlRepo) adminBaseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.AdminModel{}).Where("deleted_at IS NULL")
}

// ====================== 管理员基础操作 ======================

// --------------------- 1. 创建管理员 ---------------------
func (r *adminMysqlRepo) CreateAdmin(ctx context.Context, adminModel *model.AdminModel) error {
	if adminModel == nil {
		logger.Error("[MYSQL] 创建管理员失败，管理员模型为空")
		return gorm.ErrInvalidData
	}

	if err := r.db.WithContext(ctx).Create(adminModel).Error; err != nil {
		logger.Error("[MYSQL] 创建管理员失败", zap.String("admin_name", adminModel.AdminName), zap.Error(err))
		return err
	}

	return nil
}

// --------------------- 2. 根据管理员名称查询 ---------------------
func (r *adminMysqlRepo) FindAdminByName(ctx context.Context, adminName string) (*model.AdminModel, error) {
	var adminModel model.AdminModel

	err := r.adminBaseQuery(ctx).
		Where("admin_name = ?", adminName).
		First(&adminModel).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.Error("[MYSQL] 根据管理员名称查询失败", zap.String("admin_name", adminName), zap.Error(err))
		return nil, err
	}

	return &adminModel, nil
}

// --------------------- 3. 根据管理员ID查询 ---------------------
func (r *adminMysqlRepo) FindAdminByID(ctx context.Context, adminID uint64) (*model.AdminModel, error) {
	var adminModel model.AdminModel

	err := r.adminBaseQuery(ctx).
		Where("id = ?", adminID).
		First(&adminModel).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		logger.Error("[MYSQL] 根据管理员ID查询失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return nil, err
	}

	return &adminModel, nil
}

// --------------------- 4. 分页查询管理员列表 ---------------------
func (r *adminMysqlRepo) ListAdmin(ctx context.Context, page, pageSize int) ([]*model.AdminModel, int64, error) {
	var list []*model.AdminModel
	var total int64

	err := r.adminBaseQuery(ctx).Count(&total).Error
	if err != nil {
		logger.Error("[MYSQL] 统计管理员总数失败", zap.Error(err))
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = r.adminBaseQuery(ctx).
		Order("id DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&list).Error

	if err != nil {
		logger.Error("[MYSQL] 分页查询管理员列表失败", zap.Error(err))
		return nil, 0, err
	}

	return list, total, nil
}

// --------------------- 5. 更新管理员信息 ---------------------
// UpdateAdminInfo 只更新本次提交的资料列，避免用读出的旧资料覆盖并发修改。
// nil 不写入，非 nil 的空字符串用于清空；权限列与 token_version 不属于本用例。
func (r *adminMysqlRepo) UpdateAdminInfo(ctx context.Context, adminID uint64, update *do.AdminInfoUpdate) error {
	if update == nil || adminID == 0 {
		logger.Error("[MYSQL] 更新管理员信息失败，更新入参非法")
		return gorm.ErrInvalidData
	}

	updateData := map[string]interface{}{}
	if update.AdminName != nil {
		updateData["admin_name"] = *update.AdminName
	}
	if update.Avatar != nil {
		updateData["avatar"] = *update.Avatar
	}
	if update.Email != nil {
		updateData["email"] = *update.Email
	}
	if update.Remark != nil {
		updateData["remark"] = *update.Remark
	}
	if len(updateData) == 0 {
		return nil
	}
	updateData["update_at"] = time.Now()

	result := r.adminBaseQuery(ctx).
		Where("id = ?", adminID).
		Updates(updateData)

	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 更新管理员信息失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}

	// 这里不返回 ErrNoRowsAffected：本方法无角色等业务条件，0 行只说明提交内容与库中一致，
	// 属正常情况，不是并发冲突。
	if result.RowsAffected == 0 {
		logger.Warn("[MYSQL] 更新管理员信息无数据变更", zap.Uint64("admin_id", adminID))
	}

	return nil
}

// ChangeAdminRole 变更普通管理员的角色，并递增 token_version 使旧 token 立即失效。
// 不接收操作者 id：操作者身份是领域规则，由 AdminDomainService.ensureCanManageAdmin 校验。
// 目标在校验后已被提升或删除时返回 ErrNoRowsAffected，不能静默报告成功。
func (r *adminMysqlRepo) ChangeAdminRole(ctx context.Context, targetID uint64, role int) error {
	updateData := map[string]interface{}{
		"role":          role,
		"token_version": gorm.Expr("token_version + ?", 1),
		"update_at":     time.Now(),
	}
	result := r.adminBaseQuery(ctx).
		Where("id = ? AND role = ?", targetID, consts.AdminRoleNormal).
		Updates(updateData)

	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 更新管理员信息失败", zap.Uint64("admin_id", targetID), zap.Error(err))
		return err
	}
	if result.RowsAffected == 0 {
		return repo.ErrNoRowsAffected
	}
	return nil
}

// --------------------- 6. 更新管理员状态 ---------------------
// UpdateAdminStatus 启用/禁用管理员。WHERE 固定叠加 role = AdminRoleNormal，
// 与 Domain 的 ensureCanManageAdmin 形成双保险：校验通过后目标若被并发提升为超管，
// 这里影响 0 行并返回 repo.ErrNoRowsAffected，避免静默"假成功"。
// 更新列含每次必变的 token_version / update_at，因此 0 行只可能是 WHERE 未匹配。
func (r *adminMysqlRepo) UpdateAdminStatus(ctx context.Context, adminID uint64, status int) error {
	updateData := map[string]interface{}{
		"status":        status,
		"token_version": gorm.Expr("token_version + ?", 1), // 禁用后旧 token 立即失效
		"update_at":     time.Now(),
	}

	result := r.adminBaseQuery(ctx).
		Where("id = ? AND role = ?", adminID, consts.AdminRoleNormal).
		Updates(updateData)

	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 更新管理员状态失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}

	// 条件未命中：目标不存在、已软删除，或已不是普通管理员
	if result.RowsAffected == 0 {
		logger.Warn("[MYSQL] 更新管理员状态未命中目标", zap.Uint64("admin_id", adminID))
		return repo.ErrNoRowsAffected
	}

	return nil
}

// --------------------- 7. 更新管理员密码 ---------------------
// UpdatePWD 管理员修改自己的密码。故意不加 role 条件——超管改自己密码同样要走这里，
// 加上 role = AdminRoleNormal 会把超管误伤成"改不了自己的密码"。
// 超管重置他人密码请走 ResetAdminPWDByID（带角色兜底）。
func (r *adminMysqlRepo) UpdatePWD(ctx context.Context, adminID uint64, password string) error {
	result := r.adminBaseQuery(ctx).
		Where("id = ?", adminID).
		Updates(map[string]interface{}{
			"password":      password,
			"token_version": gorm.Expr("token_version + ?", 1),
			"update_at":     time.Now(),
		})

	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 更新管理员密码失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}

	if result.RowsAffected == 0 {
		logger.Warn("[MYSQL] 更新管理员密码无数据变更", zap.Uint64("admin_id", adminID))
	}

	return nil
}

// --------------------- 7.1 重置管理员密码（仅普通管理员） ---------------------
// ResetAdminPWDByID 供超管重置他人密码。WHERE 固定叠加 role = AdminRoleNormal，
// 与 Domain 的 ensureCanManageAdmin 形成双保险：校验通过后目标若被并发提升为超管，
// 这里影响 0 行并返回 repo.ErrNoRowsAffected，不会静默"假成功"。
// 管理员改自己密码走 UpdatePWD（不带 role 条件，否则超管无法改自己的密码）。
func (r *adminMysqlRepo) ResetAdminPWDByID(ctx context.Context, adminID uint64, password string) error {
	result := r.adminBaseQuery(ctx).
		Where("id = ? AND role = ?", adminID, consts.AdminRoleNormal).
		Updates(map[string]interface{}{
			"password":      password,
			"token_version": gorm.Expr("token_version + ?", 1),
			"update_at":     time.Now(),
		})
	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 重置管理员密码失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}
	// 条件未命中：目标不存在、已软删除，或已不是普通管理员
	if result.RowsAffected == 0 {
		logger.Warn("[MYSQL] 重置管理员密码未命中目标", zap.Uint64("admin_id", adminID))
		return repo.ErrNoRowsAffected
	}
	return nil
}

// --------------------- 8. 更新最后登录时间 ---------------------
func (r *adminMysqlRepo) UpdateLastLoginInfo(ctx context.Context, adminID uint64, loginAt *time.Time) error {
	updateData := map[string]interface{}{
		"last_login_at": loginAt,
		"update_at":     time.Now(),
	}

	result := r.adminBaseQuery(ctx).
		Where("id = ?", adminID).
		Updates(updateData)

	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 更新管理员最后登录时间失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}

	if result.RowsAffected == 0 {
		logger.Warn("[MYSQL] 更新管理员最后登录时间无数据变更", zap.Uint64("admin_id", adminID))
	}

	return nil
}
