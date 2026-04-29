/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-01 04:59:06
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-20 19:45:26
 * @FilePath: \yia-blog\blog-server\infra\repo\user_mysql_repo.go
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

// 私有结构体，实现domain/repo中的UserRepo接口
type userMysqlRepo struct {
	db *gorm.DB
}

func NewUserMysqlRepo(db *gorm.DB) repo.UserMysqlRepo {
	return &userMysqlRepo{db: db}
}

// --------------------- 1. 更新密码 ---------------------
func (r *userMysqlRepo) UpdatePWD(ctx context.Context, userID uint64, password string) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	err := r.db.WithContext(ctx).
		Model(&model.UserModel{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Update("password", password).Error

	if err != nil {
		logger.Error("UpdatePWD failed", zap.Uint64("user_id", userID), zap.Error(err))
		return err
	}
	// 无数据更新：打Warn日志
	if r.db.RowsAffected == 0 {
		logger.Warn("UpdatePWD no rows affected", zap.Uint64("user_id", userID))
	}
	return nil
}

// --------------------- 2. 创建用户 ---------------------
// 实现UserRepo接口的Create方法：创建用户（DO转Model+写入数据库）
func (r *userMysqlRepo) CreateUser(ctx context.Context, userModel *model.UserModel) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	if err := r.db.WithContext(ctx).Create(userModel).Error; err != nil {
		logger.Error("CreateUser failed", zap.String("nickname", userModel.NickName), zap.Error(err))
		return err
	}
	return nil
}

// --------------------- 3. 根据昵称查询用户 ---------------------
func (r *userMysqlRepo) FindUserByNickname(ctx context.Context, nickName string) (*model.UserModel, error) {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	var userModel model.UserModel
	err := r.db.WithContext(ctx).
		Where("nick_name = ? AND deleted_at IS NULL", nickName).
		First(&userModel).Error

	if err != nil {
		// 未找到数据：只打日志，返回原生err
		if err == gorm.ErrRecordNotFound {
			logger.Warn("FindUserByNickname: user not found", zap.String("nickname", nickName))
		} else {
			logger.Error("FindUserByNickname failed", zap.String("nickname", nickName), zap.Error(err))
		}
		return nil, err
	}
	return &userModel, nil
}

// --------------------- 4. 根据ID查询用户 ---------------------
func (r *userMysqlRepo) FindUserByID(ctx context.Context, userID uint64) (*model.UserModel, error) {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	var userModel model.UserModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", userID).
		First(&userModel).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("FindUserByID: user not found", zap.Uint64("user_id", userID))
		} else {
			logger.Error("FindUserByID failed", zap.Uint64("user_id", userID), zap.Error(err))
		}
		return nil, err
	}
	return &userModel, nil

}

// --------------------- 5. 更新用户信息 ---------------------
func (r *userMysqlRepo) UpdateUserInfo(ctx context.Context, userModel *model.UserModel) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	err := r.db.WithContext(ctx).
		Model(&model.UserModel{}).
		Where("id = ? AND deleted_at IS NULL", userModel.ID).
		Updates(userModel).Error

	if err != nil {
		logger.Error("UpdateUserInfo failed", zap.Uint64("user_id", userModel.ID), zap.Error(err))
		return err
	}
	if r.db.RowsAffected == 0 {
		logger.Warn("UpdateUserInfo no rows affected", zap.Uint64("user_id", userModel.ID))
	}
	return nil
}

// --------------------- 6. 更新最后登录信息 ---------------------
func (r *userMysqlRepo) UpdateLastLoginInfo(ctx context.Context, userID uint64, loginAt *time.Time) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	// 构造更新参数（匹配Model层字段名）
	updateData := map[string]interface{}{
		"last_login_at": loginAt,
		"update_at":     time.Now(),
	}

	err := r.db.WithContext(ctx).
		Model(&model.UserModel{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Updates(updateData).Error

	if err != nil {
		logger.Error("UpdateLastLoginInfo failed", zap.Uint64("user_id", userID), zap.Error(err))
		return err
	}
	if r.db.RowsAffected == 0 {
		logger.Warn("UpdateLastLoginInfo no rows affected", zap.Uint64("user_id", userID))
	}
	return nil
}
