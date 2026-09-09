/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-16 08:26:43
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-05 17:41:14
 * @FilePath: \yia-blog\blog-server\infra\repo\user_mysql_repo.go
 * @Description: 用户 MySQL 仓储实现
 */
package repo

import (
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

type userMysqlRepo struct {
	db *gorm.DB
}

func NewUserMysqlRepo(db *gorm.DB) repo.UserMysqlRepo {
	return &userMysqlRepo{db: db}
}

// userBaseQuery 基础查询构造器（统一上下文与模型）
func (r *userMysqlRepo) userBaseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.UserModel{})
}

// ====================== 用户基础操作 ======================

// --------------------- 1. 创建用户 ---------------------
func (r *userMysqlRepo) CreateUser(ctx context.Context, userModel *model.UserModel) error {
	if userModel == nil {
		logger.Error("[MYSQL] 创建用户失败，用户模型为空")
		return gorm.ErrInvalidData
	}

	if err := r.db.WithContext(ctx).Create(userModel).Error; err != nil {
		logger.Error("[MYSQL] 创建用户失败", zap.String("nick_name", userModel.NickName), zap.Error(err))
		return err
	}

	return nil
}

// --------------------- 2. 根据昵称查询用户 ---------------------
func (r *userMysqlRepo) FindUserByNickname(ctx context.Context, nickName string) (*model.UserModel, error) {
	var userModel model.UserModel

	err := r.userBaseQuery(ctx).
		Where("nick_name = ?", nickName).
		First(&userModel).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		logger.Error("[MYSQL] 根据昵称查询用户失败", zap.String("nick_name", nickName), zap.Error(err))
		return nil, err
	}

	return &userModel, nil
}

// --------------------- 3. 根据用户ID查询用户 ---------------------
func (r *userMysqlRepo) FindUserByID(ctx context.Context, userID uint64) (*model.UserModel, error) {
	var userModel model.UserModel

	err := r.userBaseQuery(ctx).
		Where("id = ?", userID).
		First(&userModel).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		logger.Error("[MYSQL] 根据用户ID查询用户失败", zap.Uint64("user_id", userID), zap.Error(err))
		return nil, err
	}

	return &userModel, nil
}

// --------------------- 4. 更新用户信息 ---------------------
// UpdateUserInfo 只写本次提供的资料列，不回写查询时的账号状态或其他旧资料。
func (r *userMysqlRepo) UpdateUserInfo(ctx context.Context, userID uint64, update *do.UserInfoUpdate) error {
	if update == nil || userID == 0 {
		logger.Error("[MYSQL] 更新用户信息失败，用户模型非法")
		return gorm.ErrInvalidData
	}

	updateData := map[string]interface{}{}
	if update.NickName != nil {
		updateData["nick_name"] = *update.NickName
	}
	if update.Avatar != nil {
		updateData["avatar"] = *update.Avatar
	}
	if update.Gender != nil {
		updateData["gender"] = *update.Gender
	}
	if update.Signature != nil {
		updateData["signature"] = *update.Signature
	}
	if update.Email != nil {
		updateData["email"] = *update.Email
	}
	if len(updateData) == 0 {
		return nil
	}
	updateData["update_at"] = time.Now()

	result := r.userBaseQuery(ctx).
		Where("id = ?", userID).
		Updates(updateData)

	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 更新用户信息失败", zap.Uint64("user_id", userID), zap.Error(err))
		return err
	}

	if result.RowsAffected == 0 {
		// 相同值更新可能返回 0，复核存在性以区分幂等重试与并发删除。
		var count int64
		if err := r.userBaseQuery(ctx).Where("id = ?", userID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return repo.ErrNoRowsAffected
		}
	}

	return nil
}

// --------------------- 5. 更新用户最后登录时间 ---------------------
func (r *userMysqlRepo) UpdateLastLoginInfo(ctx context.Context, userID uint64, loginAt *time.Time) error {
	updateData := map[string]interface{}{
		"last_login_at": loginAt,
		"update_at":     time.Now(),
	}

	result := r.userBaseQuery(ctx).
		Where("id = ?", userID).
		Updates(updateData)

	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 更新用户最后登录时间失败", zap.Uint64("user_id", userID), zap.Error(err))
		return err
	}

	if result.RowsAffected == 0 {
		logger.Warn("[MYSQL] 更新用户最后登录时间无数据变更", zap.Uint64("user_id", userID))
	}

	return nil
}

// --------------------- 6. 更新用户密码 ---------------------

func (r *userMysqlRepo) UpdatePWD(ctx context.Context, userID uint64, password string) error {
	// 必须更新 update_at！！
	result := r.userBaseQuery(ctx).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"password":      password,
			"token_version": gorm.Expr("token_version + ?", 1),
			"update_at":     time.Now(),
		})

	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 更新用户密码失败", zap.Uint64("user_id", userID), zap.Error(err))
		return err
	}

	if result.RowsAffected == 0 {
		logger.Warn("[MYSQL] 更新用户密码无数据变更", zap.Uint64("user_id", userID))
	}

	return nil
}
