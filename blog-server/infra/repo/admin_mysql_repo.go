/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-30 01:46:01
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-30 01:46:28
 * @FilePath: \yia-blog\blog-server\infra\repo\admin_mysql_repo.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package repo

import (
	"blog-server/common/gin/response"
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

func (r *adminMysqlRepo) adminBaseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.AdminModel{}).Where("deleted_at IS NULL")
}

// --------------------- 1. 更新密码 ---------------------
func (r *adminMysqlRepo) UpdatePWD(ctx context.Context, adminID uint64, password string) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	result := r.adminBaseQuery(ctx).
		Where("id = ?", adminID).
		Update("password", password)

	if err := result.Error; err != nil {
		logger.Error("UpdatePWD failed", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}
	if result.RowsAffected == 0 {
		logger.Warn("UpdatePWD no rows affected", zap.Uint64("admin_id", adminID))
	}
	return nil
}

// --------------------- 2. 创建管理员用户 ---------------------
func (r *adminMysqlRepo) CreateAdmin(ctx context.Context, adminModel *model.AdminModel) error {
	// Nil pointer check to prevent panic
	if adminModel == nil {
		logger.Error("CreateAdmin: adminModel is nil")
		return gorm.ErrInvalidData
	}

	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	if err := r.db.WithContext(ctx).Create(adminModel).Error; err != nil {
		logger.Error("CreateAdmin failed", zap.String("admin_name", adminModel.AdminName), zap.Error(err))
		return err
	}
	return nil
}

// --------------------- 3. 通过名称获取管理员信息 ---------------------
func (r *adminMysqlRepo) FindAdminByName(ctx context.Context, adminName string) (*model.AdminModel, error) {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	var adminModel model.AdminModel
	err := r.adminBaseQuery(ctx).
		Where("admin_name = ?", adminName).
		First(&adminModel).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("FindAdminByName: admin not found", zap.String("admin_name", adminName))
		} else {
			logger.Error("FindAdminByName failed", zap.String("admin_name", adminName), zap.Error(err))
		}
		return nil, err
	}
	return &adminModel, nil
}

// --------------------- 4. 通过ID获取管理员信息 ---------------------
func (r *adminMysqlRepo) FindAdminByID(ctx context.Context, adminID uint64) (*model.AdminModel, error) {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	var adminModel model.AdminModel
	err := r.adminBaseQuery(ctx).
		Where("id = ?", adminID).
		First(&adminModel).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("FindAdminByID: admin not found", zap.Uint64("admin_id", adminID))
		} else {
			logger.Error("FindAdminByID failed", zap.Uint64("admin_id", adminID), zap.Error(err))
		}
		return nil, err
	}
	return &adminModel, nil
}

// --------------------- 5. 更新管理员信息 ---------------------
func (r *adminMysqlRepo) UpdateAdminInfo(ctx context.Context, adminModel *model.AdminModel) error {
	// Validate parameter to prevent invalid update
	if adminModel == nil || adminModel.ID == 0 {
		logger.Error("UpdateAdminInfo: invalid adminModel")
		return gorm.ErrInvalidData
	}

	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	// Restrict update fields to avoid accidental password modification
	result := r.adminBaseQuery(ctx).
		Where("id = ?", adminModel.ID).
		Select("admin_name", "avatar", "role", "email", "status", "remark", "update_at").
		Updates(adminModel)

	if err := result.Error; err != nil {
		logger.Error("UpdateAdminInfo failed", zap.Uint64("admin_id", adminModel.ID), zap.Error(err))
		return err
	}
	if result.RowsAffected == 0 {
		logger.Warn("UpdateAdminInfo no rows affected", zap.Uint64("admin_id", adminModel.ID))
	}
	return nil
}

// --------------------- 6. 更新管理员上次登录时间 ---------------------
func (r *adminMysqlRepo) UpdateLastLoginInfo(ctx context.Context, adminID uint64, loginAt *time.Time) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	updateData := map[string]interface{}{
		"last_login_at": loginAt,
		"update_at":     time.Now(),
	}

	result := r.adminBaseQuery(ctx).
		Where("id = ?", adminID).
		Updates(updateData)

	if err := result.Error; err != nil {
		logger.Error("UpdateLastLoginInfo failed", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}
	if result.RowsAffected == 0 {
		logger.Warn("UpdateLastLoginInfo no rows affected", zap.Uint64("admin_id", adminID))
	}
	return nil
}
