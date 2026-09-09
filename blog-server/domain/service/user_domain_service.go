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
	"regexp"
	"strings"
	"unicode/utf8"

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

var userEmailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// UpdateUserInfo 校验本人资料的部分更新，保留未提交字段与显式零值的区别。
func (u *UserDomainService) UpdateUserInfo(ctx context.Context, userId uint64, updateData *do.UserInfoUpdate) error {
	if userId == 0 || updateData == nil {
		return common.ParamErr
	}
	if updateData.NickName != nil && (strings.TrimSpace(*updateData.NickName) == "" || utf8.RuneCountInString(*updateData.NickName) > consts.UserNicknameMaxLength) {
		return common.ParamErr.WithMsg("昵称不能为空且最多16个字符")
	}
	if updateData.Gender != nil && (*updateData.Gender < consts.UserGenderUnknown || *updateData.Gender > consts.UserGenderFemale) {
		return common.ParamErr.WithMsg("性别取值不合法，仅支持0/1/2")
	}
	if updateData.Email != nil && *updateData.Email != "" && !userEmailPattern.MatchString(*updateData.Email) {
		return common.ParamErr.WithMsg("邮箱格式不正确")
	}

	userDO, err := u.FindUserByID(ctx, userId)
	if err != nil {
		return err
	}
	if userDO == nil {
		logger.Warn("[DOMAIN] 更新用户信息失败：用户不存在", zap.Uint64("userId", userId))
		return common.UserNotFoundErr
	}

	if err := u.userMysqlRepo.UpdateUserInfo(ctx, userId, updateData); err != nil {
		if errors.Is(err, repo.ErrNoRowsAffected) {
			return common.UserNotFoundErr
		}
		logger.Warn("[DOMAIN] 用户信息更新失败", zap.Error(err), zap.Uint64("userId", userId))
		return common.UserUpdateErr
	}
	return nil
}
