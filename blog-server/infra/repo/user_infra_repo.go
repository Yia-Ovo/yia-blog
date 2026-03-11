/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-01 04:59:06
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-03-11 16:57:19
 * @FilePath: \yia-blog\blog-server\infra\repo\user_infra_repo.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import (
	"blog-server/common"
	"blog-server/common/gin/response"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type userInfraRepo struct {
	db *gorm.DB
}

// 私有结构体，实现domain/repo中的UserRepo接口
func NewUserInfraRepo(db *gorm.DB) repo.UserRepo {
	return &userInfraRepo{db: db}
}

// 实现UserRepo接口的Create方法：创建用户（DO转Model+写入数据库）
func (r *userInfraRepo) CreateUser(ctx context.Context, userDO *do.UserDO) error {

	// 数据层校验：字段长度兜底（比如数据库nick_name字段是varchar(20)）
	if len(userDO.NickName) > 20 {
		return common.UserNickNameTooLongErr.WithMsg("昵称长度不能超过20个字符")
	}

	// 枚举值校验：status只能是1（正常）或-1（禁用）
	if userDO.Status != 1 && userDO.Status != -1 {
		return common.UserStatusInvalidErr.WithMsg("用户状态只能为1（正常）或-1（禁用）")
	}

	// 领域对象DO转换成数据库实体Model
	userModel := &model.UserModel{
		NickName: userDO.NickName,
		Password: userDO.Password,
		Avatar:   userDO.Avatar,
		Status:   userDO.Status,
	}

	// 1. 给 ctx 设置超时（比如5秒），传递给数据库操作
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	// 2. 绑定超时ctx到Create操作
	// 效果：如果Create操作超过5秒，会返回context.DeadlineExceeded错误，而非一直阻塞
	return r.db.WithContext(ctx).Debug().Create(userModel).Error
	// Create inserts value, returning the inserted data's primary key in value's id
	// 把userModel这个结构体实例的字段值，插入到对应的数据库表中（GORM 会根据model.UserModel的结构体标签 / 名称映射到对应的表，
	// GORM 会在插入成功后，自动把数据库生成的主键 ID、创建时间（gorm.Model 的 CreatedAt） 等字段回写到这个指针对象中
}

func (r *userInfraRepo) FindUserByNickname(ctx context.Context, nickName string) (*do.UserDO, error) {
	var userModel model.UserModel
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	// 查询一条记录，匹配第一个找到的字段
	if err := r.db.WithContext(ctx).Debug().Where("nick_name = ? AND deleted_at IS NULL", nickName).First(&userModel).Error; err != nil {
		logger.Error("func FindByNickName error", zap.Error(err))
		return nil, err
	}
	// 把数据库实体Model转换成领域对象DO
	userDO := &do.UserDO{
		ID:        int64(userModel.ID),
		NickName:  userModel.NickName,
		Password:  userModel.Password,
		Avatar:    userModel.Avatar,
		Gender:    userModel.Gender,
		Status:    userModel.Status,
		CreateAt:  userModel.CreateAt,
		UpdateAt:  userModel.UpdateAt,
		IsDeleted: userModel.DeletedAt.Valid,
	}

	// 最后登录时间：Model中是指针，需判断是否为nil
	if userModel.LastLoginAt != nil {
		loginAt := time.Time(*userModel.LastLoginAt)
		userDO.LastLoginAt = &loginAt
	}
	return userDO, nil
}

func (r *userInfraRepo) FindUserByID(ctx context.Context, id int64) (*do.UserDO, error) {
	var userModel model.UserModel
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	if err := r.db.WithContext(ctx).Debug().Where("id = ? AND deleted_at IS NULL", id).First(&userModel).Error; err != nil {
		logger.Error("func FindByID error", zap.Error(err))
		return nil, err
	}
	userDO := &do.UserDO{
		ID:       int64(userModel.ID),
		NickName: userModel.NickName,
		Gender:   userModel.Gender,
		Avatar:   userModel.Avatar,
		Status:   userModel.Status,
	}

	return userDO, nil

}

func (r *userInfraRepo) UpdateUserInfo(ctx context.Context, userDO *do.UserDO) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	// GORM的Updates方法会自动忽略零值字段，仅更新非零字段
	err := r.db.WithContext(ctx).Model(&model.UserModel{}).Where("id = ? AND deleted_at IS NULL", userDO.ID).Updates(userDO).Error
	if err != nil {
		logger.Error("UpdateUserInfo failed", zap.Int64("user_id", userDO.ID), zap.Error(err))
		return err
	}
	if r.db.RowsAffected == 0 {
		logger.Warn("UpdateUserInfo no rows affected", zap.Int64("user_id", userDO.ID))
		return common.UserNotFoundErr
	}
	return nil
}

func (r *userInfraRepo) UpdateLastLoginInfo(ctx context.Context, userID int64, loginAt *time.Time) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	// 1. 构造更新参数（匹配Model层字段名）
	updateData := map[string]interface{}{
		// "last_login_ip": loginIP,
		"last_login_at": loginAt,    // GORM会自动处理time.Time到数据库时间类型的转换
		"update_at":     time.Now(), // 同步更新修改时间（可选，根据你的业务规则）
	}

	// 2. 执行更新操作（使用WithContext传递上下文，保证链路追踪）
	// Where条件：匹配用户ID + 未被软删除（根据你的软删除字段调整）
	err := r.db.WithContext(ctx).
		Model(&model.UserModel{}).
		Where("id = ? AND deleted_at IS NULL", userID). // 排除软删除用户
		Updates(updateData).
		Error

	// 3. 错误处理（区分“无记录更新”和“数据库异常”）
	if err != nil {
		logger.Error("func UpdateLastLoginInfo error",
			zap.Int64("user_id", userID),
			// zap.String("login_ip", loginIP),
			zap.Time("login_at", *loginAt),
			zap.Error(err),
		)
		return err
	}

	// 4. 可选：检查是否真的更新了记录（防止用户ID不存在但无报错的情况）
	// 如果你的业务需要严格校验，可添加以下逻辑
	if r.db.RowsAffected == 0 {
		logger.Warn("func UpdateLastLoginInfo no rows affected",
			zap.Int64("user_id", userID),
			// zap.String("login_ip", loginIP),
		)
		// 可选：返回自定义错误（比如repository.ErrUserNotFound）
		// return repository.ErrUserNotFound
	}

	return nil
}
