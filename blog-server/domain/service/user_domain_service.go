/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:25:51
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-05 16:57:45
 * @FilePath: \yia-blog\blog-server\domain\service\user_domain_service.go
 * @Description: 领域服务：纯业务逻辑，无外部依赖，无缓存
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

// UserDomainService 纯净领域服务：仅依赖 MySQL 仓储
type UserDomainService struct {
	userMysqlRepo repo.UserMysqlRepo
}

// NewUserDomainService 构造函数
func NewUserDomainService(userMysqlRepo repo.UserMysqlRepo) *UserDomainService {
	return &UserDomainService{
		userMysqlRepo: userMysqlRepo,
	}
}

// UpdatePWD 领域层：修改密码业务逻辑
func (u *UserDomainService) UpdatePWD(ctx context.Context, userId uint64, newPassword string, oldPassword string) error {
	userDO, err := u.userMysqlRepo.FindUserByID(ctx, userId)
	if err != nil {
		logger.Warn("[DOMAIN] 用户查询失败", zap.Error(err), zap.Uint64("userId", userId))
		return common.UserQueryErr
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
		logger.Warn("[DOMAIN] 密码加密失败", zap.Error(err))
		return common.PasswordEncryptError
	}

	// 更新数据库
	if err := u.userMysqlRepo.UpdatePWD(ctx, userId, string(hashedPassword)); err != nil {
		logger.Warn("[DOMAIN] 密码更新失败", zap.Error(err), zap.Uint64("userId", userId))
		return common.UpdatePasswordError
	}

	return nil
}

// Login 领域层：登录校验逻辑
func (u *UserDomainService) Login(ctx context.Context, nickName string, password string) (*do.UserDO, error) {
	userModel, err := u.userMysqlRepo.FindUserByNickname(ctx, nickName)
	if err != nil {
		if errors.Is(err, common.UserNotFoundErr) {
			return nil, common.LoginPasswordErr
		}
		logger.Warn("[DOMAIN] 用户查询失败", zap.Error(err), zap.String("nickname", nickName))
		return nil, common.UserQueryErr
	}

	userDO := convert.UserModelToDO(userModel)

	// 账号状态校验
	switch userDO.Status {
	case consts.UserStatusDisabled:
		return nil, common.UserAccountDisabledErr
	case consts.UserStatusLocked:
		return nil, common.LoginAccountLockedErr
	}

	// 密码校验
	if err = bcrypt.CompareHashAndPassword([]byte(userDO.Password), []byte(password)); err != nil {
		return nil, common.LoginPasswordErr
	}

	// 更新登录信息（允许失败）
	_ = u.userMysqlRepo.UpdateLastLoginInfo(ctx, userDO.ID, userDO.LastLoginAt)
	return userDO, nil
}

// Register 领域层：用户注册逻辑
func (u *UserDomainService) Register(ctx context.Context, nickName string, password string) error {
	hashPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Warn("[DOMAIN] 密码加密失败", zap.Error(err), zap.String("nickName", nickName))
		return common.PasswordEncryptError
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
		logger.Warn("[DOMAIN] 创建用户失败", zap.Error(err), zap.String("nickName", nickName))
		return common.UserCreateErr
	}
	return nil
}

// FindUserByNickname 领域层：按昵称查询用户
func (u *UserDomainService) FindUserByNickname(ctx context.Context, nickName string) (*do.UserDO, error) {
	userModel, err := u.userMysqlRepo.FindUserByNickname(ctx, nickName)
	if err != nil {
		logger.Warn("[DOMAIN] 用户查询失败", zap.Error(err), zap.String("nickname", nickName))
		return nil, common.UserQueryErr
	}
	return convert.UserModelToDO(userModel), nil
}

// FindUserByID 领域层：按ID查询用户
func (u *UserDomainService) FindUserByID(ctx context.Context, userId uint64) (*do.UserDO, error) {
	userModel, err := u.userMysqlRepo.FindUserByID(ctx, userId)
	if err != nil {
		logger.Warn("[DOMAIN] 用户查询失败", zap.Error(err), zap.Uint64("userId", userId))
		return nil, common.UserQueryErr
	}
	return convert.UserModelToDO(userModel), nil
}

// UpdateUserInfo 领域层：更新用户信息
func (u *UserDomainService) UpdateUserInfo(ctx context.Context, userId uint64, updateData *do.UserDO) error {
	// 判空必须在第一次解引用之前：下方按字段覆盖会直接读 updateData
	if updateData == nil {
		logger.Warn("[DOMAIN] 更新用户信息失败：更新数据为空", zap.Uint64("userId", userId))
		return common.ParamErr
	}

	userDO, err := u.FindUserByID(ctx, userId)
	if err != nil {
		return common.UserQueryErr
	}
	// 记录不存在时由上面的 err 拦截；此处兜底 FindUserByID 理论上的 (nil, nil)，
	// 避免下方读-改-写解引用空指针
	if userDO == nil {
		logger.Warn("[DOMAIN] 更新用户信息失败：用户不存在", zap.Uint64("userId", userId))
		return common.UserNotFoundErr
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

	if err := u.userMysqlRepo.UpdateUserInfo(ctx, userModel); err != nil {
		logger.Warn("[DOMAIN] 用户信息更新失败", zap.Error(err), zap.Uint64("userId", userId))
		return common.UserUpdateErr
	}
	return nil
}
