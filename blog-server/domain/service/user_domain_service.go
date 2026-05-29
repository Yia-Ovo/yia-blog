/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:25:51
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 13:37:11
 * @FilePath: \yia-blog\blog-server\domain\service\user_domain_service.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserDomainService 纯净领域服务：仅依赖 MySQL 仓储，仅处理业务规则
type UserDomainService struct {
	userMysqlRepo repo.UserMysqlRepo // 🔥 只保留MySQL，Redis完全移除
}

// 构造函数：只注入MySQL
func NewUserDomainService(userMysqlRepo repo.UserMysqlRepo) *UserDomainService {
	return &UserDomainService{
		userMysqlRepo: userMysqlRepo,
	}
}

// UpdatePWD 领域层：只做密码修改业务逻辑，无缓存
func (u *UserDomainService) UpdatePWD(ctx context.Context, userId uint64, newPassword string, oldPassword string) error {
	userDO, err := u.userMysqlRepo.FindUserByID(ctx, userId)
	if err != nil {
		logger.Warn("通过用户ID查询用户失败", zap.Error(err), zap.Uint64("userId", userId))
		return err
	}
	if userDO == nil {
		return common.UserNotFoundErr
	}

	// 旧密码校验
	if err := bcrypt.CompareHashAndPassword([]byte(userDO.Password), []byte(oldPassword)); err != nil {
		return common.UserOldPasswordErr
	}

	// 新旧密码不能相同
	if newPassword == oldPassword {
		return common.UserPasswordSameError
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return common.PasswordEncryptError
	}

	// 更新数据库
	if err := u.userMysqlRepo.UpdatePWD(ctx, userId, string(hashedPassword)); err != nil {
		return common.UpdatePasswordError
	}

	logger.Info("用户密码修改成功", zap.Uint64("userId", userId))
	return nil
}

// Login 领域层：只做登录校验、状态判断、DB操作
func (u *UserDomainService) Login(ctx context.Context, nickName string, password string) (*do.UserDO, error) {
	userModel, err := u.userMysqlRepo.FindUserByNickname(ctx, nickName)
	if err != nil {
		if errors.Is(err, common.UserNotFoundErr) {
			return nil, common.LoginPasswordErr
		}
		logger.Error("用户查询数据库失败", zap.Error(err), zap.String("nickname", nickName))
		return nil, common.DatabaseErr
	}

	userDO := convert.UserModelToDO(userModel)

	// 账号状态校验（业务规则）
	switch userDO.Status {
	case consts.UserStatusDisabled:
		return nil, common.UserAccountDisabledErr
	case consts.UserStatusLocked:
		return nil, common.LoginAccountLockedErr
	}

	// 密码校验
	if err = bcrypt.CompareHashAndPassword([]byte(userDO.Password), []byte(password)); err != nil {
		logger.Warn("用户密码校验失败", zap.Uint64("user_id", userDO.ID))
		return nil, common.LoginPasswordErr
	}

	// 更新登录信息
	_ = u.userMysqlRepo.UpdateLastLoginInfo(ctx, userDO.ID, userDO.LastLoginAt)
	return userDO, nil
}

// Register 领域层：只做注册业务逻辑
func (u *UserDomainService) Register(ctx context.Context, nickName string, password string) error {
	hashPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Warn("bcrypt加密失败", zap.Error(err), zap.String("nickName", nickName))
		return err
	}

	userModel := &model.UserModel{
		NickName:  nickName,
		Password:  string(hashPwd),
		Gender:    0,
		Status:    1,
		Avatar:    "",
		Signature: "",
		Email:     "",
	}

	if err := u.userMysqlRepo.CreateUser(ctx, userModel); err != nil {
		logger.Warn("仓库创建用户失败", zap.Error(err), zap.String("nickName", nickName))
		return err
	}
	return nil
}

// FindUserByNickname 领域层：只查DB
func (u *UserDomainService) FindUserByNickname(ctx context.Context, nickName string) (*do.UserDO, error) {
	userModel, err := u.userMysqlRepo.FindUserByNickname(ctx, nickName)
	if err != nil {
		return nil, err
	}
	return convert.UserModelToDO(userModel), nil
}

// FindUserByID 领域层：只查DB
func (u *UserDomainService) FindUserByID(ctx context.Context, userId uint64) (*do.UserDO, error) {
	userModel, err := u.userMysqlRepo.FindUserByID(ctx, userId)
	if err != nil {
		return nil, err
	}
	return convert.UserModelToDO(userModel), nil
}

// UpdateUserInfo 领域层：只做更新业务逻辑
func (u *UserDomainService) UpdateUserInfo(ctx context.Context, userId uint64, updateData *do.UserDO) error {
	userDO, err := u.FindUserByID(ctx, userId)
	if err != nil {
		return err
	}

	// 业务赋值规则
	if updateData.NickName != "" {
		userDO.NickName = updateData.NickName
	}
	if updateData.Avatar != "" {
		userDO.Avatar = updateData.Avatar
	}
	if updateData.Signature != "" {
		userDO.Signature = updateData.Signature
	}
	if updateData.Email != "" {
		userDO.Email = updateData.Email
	}
	if updateData.Gender >= 0 {
		userDO.Gender = updateData.Gender
	}

	userDO.UpdateAt = time.Now()
	userModel := convert.UserDOToModel(userDO)

	return u.userMysqlRepo.UpdateUserInfo(ctx, userModel)
}
