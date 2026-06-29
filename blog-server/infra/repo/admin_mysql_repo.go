/*
 * @Author: Yia 1372594487@gmail.com
 * @Date: 2026-04-30 01:46:01
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-10 01:12:41
 * @FilePath: \yia-blog\blog-server\infra\repo\admin_mysql_repo.go
 * @Description: 管理员 MySQL 仓储实现
 */
package repo

import (
	"blog-server/common/utils/logger"
	"blog-server/domain/repo"
	"blog-server/infra/model"
	"context"
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
		if err != gorm.ErrRecordNotFound {
			logger.Error("[MYSQL] 根据管理员名称查询失败", zap.String("admin_name", adminName), zap.Error(err))
		}
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
		if err != gorm.ErrRecordNotFound {
			logger.Error("[MYSQL] 根据管理员ID查询失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		}
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
func (r *adminMysqlRepo) UpdateAdminInfo(ctx context.Context, adminModel *model.AdminModel) error {
	if adminModel == nil || adminModel.ID == 0 {
		logger.Error("[MYSQL] 更新管理员信息失败，管理员模型非法")
		return gorm.ErrInvalidData
	}

	adminModel.UpdateAt = time.Now()

	result := r.adminBaseQuery(ctx).
		Where("id = ?", adminModel.ID).
		Select("admin_name", "avatar", "role", "email", "status", "remark", "update_at").
		Updates(adminModel)

	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 更新管理员信息失败", zap.Uint64("admin_id", adminModel.ID), zap.Error(err))
		return err
	}

	if result.RowsAffected == 0 {
		logger.Warn("[MYSQL] 更新管理员信息无数据变更", zap.Uint64("admin_id", adminModel.ID))
	}

	return nil
}

// --------------------- 6. 更新管理员状态 ---------------------
func (r *adminMysqlRepo) UpdateAdminStatus(ctx context.Context, adminID uint64, status int) error {
	updateData := map[string]interface{}{
		"status":    status,
		"update_at": time.Now(),
	}

	result := r.adminBaseQuery(ctx).
		Where("id = ?", adminID).
		Updates(updateData)

	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 更新管理员状态失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}

	if result.RowsAffected == 0 {
		logger.Warn("[MYSQL] 更新管理员状态无数据变更", zap.Uint64("admin_id", adminID))
	}

	return nil
}

// --------------------- 7. 更新管理员密码 ---------------------
func (r *adminMysqlRepo) UpdatePWD(ctx context.Context, adminID uint64, password string) error {
	result := r.adminBaseQuery(ctx).
		Where("id = ?", adminID).
		Updates(map[string]interface{}{
			"password":  password,
			"update_at": time.Now(),
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
