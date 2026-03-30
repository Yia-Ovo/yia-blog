/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:25:51
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-03-31 03:46:13
 * @FilePath: \yia-blog\blog-server\domain\service\user_domain_service.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserService 用户领域服务：封装核心业务逻辑
type UserDomainService struct {
	userRepo repo.UserRepo // 注入仓库接口
}

// 对外提供创建实例的方法，给app层调用
func NewUserDomainService(userRepo repo.UserRepo) *UserDomainService {
	return &UserDomainService{
		userRepo: userRepo,
	}
}

func (u *UserDomainService) UpdatePWD(ctx context.Context, userId int64, newPassword string, oldPassword string) error {
	userDO, err := u.userRepo.FindUserByID(ctx, userId)
	if err != nil {
		logger.Warn("通过用户ID查询用户失败", zap.Error(err), zap.Int64("userId", userId))
		return err
	}
	// 1. 校验用户是否存在（兜底判断）
	if userDO == nil {
		logger.Warn("用户不存在", zap.Int64("userId", userId))
		return common.UserNotFoundErr
	}

	// 2. 验证旧密码是否正确（对比数据库加密密码）
	// userDO.Password 是数据库存储的 bcrypt 加密密文
	if err := bcrypt.CompareHashAndPassword([]byte(userDO.Password), []byte(oldPassword)); err != nil {
		logger.Warn("用户旧密码验证失败", zap.Int64("userId", userId))
		return common.UserOldPasswordErr // 旧密码错误
	}

	// 3. 业务校验：新密码不能与旧密码相同
	if newPassword == oldPassword {
		logger.Warn("新密码与旧密码一致", zap.Int64("userId", userId))
		return common.UserPasswordSameError // 新旧密码重复
	}

	// 4. 密码加密：使用 bcrypt 加密新密码（自动加盐，安全不可逆）
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Warn("新密码加密失败", zap.Error(err), zap.Int64("userId", userId))
		return common.PasswordEncryptError // 加密异常
	}

	// 5. 数据库更新：调用仓储层更新密码
	if err := u.userRepo.UpdatePWD(ctx, userId, string(hashedPassword)); err != nil {
		logger.Warn("更新用户密码失败", zap.Error(err), zap.Int64("userId", userId))
		return common.UpdatePasswordError // 更新失败
	}

	// 密码修改成功
	logger.Info("用户密码修改成功", zap.Int64("userId", userId))

	return nil
}

func (u *UserDomainService) Login(nickName string, password string) (*common.User, error) {
	// 1. 调用仓储层查询用户（领域层不直接操作DB，通过仓储层抽象）
	// 传递context，用于trace、超时控制
	ctx := context.Background()
	userDO, err := u.userRepo.FindUserByNickname(ctx, nickName)
	if err != nil {
		// 仓储层错误：区分“查询失败”和“用户不存在”
		if errors.Is(err, common.UserNotFoundErr) {
			// 安全考量：不暴露“用户不存在”，统一提示“用户名或密码错误”
			return nil, errors.New("用户名或密码错误")
		}
		// 其他仓储层异常（如DB连接失败）
		return nil, errors.New("用户查询失败：" + err.Error())
	}

	// 2. 校验账号状态（核心业务规则）
	if userDO.Status == consts.UserStatusDisabled { // 假设UserStatusDisabled=2（禁用）
		return nil, errors.New("账号已被禁用，请联系管理员")
	}
	if userDO.Status == consts.UserStatusLocked { // 假设UserStatusLocked=3（锁定）
		return nil, errors.New("账号已被锁定，请稍后重试")
	}

	// 3. 密码校验（核心：哈希比对，禁止明文存储/比对）
	// userDO.Password 是数据库中存储的bcrypt哈希值，password是前端传入的明文
	err = bcrypt.CompareHashAndPassword([]byte(userDO.Password), []byte(password))
	if err != nil {
		// 密码比对失败（包括哈希格式错误、密码不匹配）
		return nil, errors.New("用户名或密码错误")
	}

	// 4. 登录成功后的核心业务操作
	// 例：更新最后登录时间、登录IP
	updateErr := u.userRepo.UpdateLastLoginInfo(ctx, userDO.ID, userDO.LastLoginAt)
	if updateErr != nil {
		// 此处可记录日志，但不阻断登录（非核心失败）
		logger.Warn("更新登录信息失败", zap.Error(err), zap.Int64("user_id", userDO.ID))
	}

	// 5. 返回用户信息（注意：脱敏，不返回密码等敏感字段）
	commonUser := common.UserDOToUser(userDO)
	return commonUser, nil
}

// Register 注册核心业务逻辑：密码bcrypt加密 → 封装DO → 调用仓库创建
func (u *UserDomainService) Register(nickName string, password string) error {
	// 1. 核心：明文密码加密（bcrypt，不可逆，生产级安全）
	hashPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Warn("bcrypt加密失败", zap.Error(err), zap.String("nickName", nickName))
		return err // 加密失败，返回错误
	}

	// 2. 封装领域对象DO（默认状态1-正常，无头像，未登录）
	userDO := &do.UserDO{
		NickName:  nickName,
		Password:  string(hashPwd), // 存储加密后的密码
		Gender:    0,               // 0=保密，1=男，2=女
		Status:    1,               // 1=正常，2=禁用
		Avatar:    "",              // 默认无头像
		Signature: "",              // 个性签名
		Email:     "",              // 邮箱
	}
	ctx := context.Background()
	// 3. 调用仓库接口，写入数据库（业务层不用管数据库细节）
	if err := u.userRepo.CreateUser(ctx, userDO); err != nil {
		logger.Warn("仓库创建用户失败", zap.Error(err), zap.String("nickName", nickName))
		return err
	}

	return nil
}

func (u *UserDomainService) FindUserByNickname(nickName string) (*do.UserDO, error) {
	ctx := context.Background()
	// 调用注入的仓库接口查询（仓库实现由基础设施层提供）
	userDO, err := u.userRepo.FindUserByNickname(ctx, nickName)
	if err != nil {
		logger.Warn("通过用户昵称查询用户失败", zap.Error(err), zap.String("nickName", nickName))
		return nil, err
	}
	return userDO, nil
}

func (u *UserDomainService) FindUserByID(ctx context.Context, userId int64) (*do.UserDO, error) {
	userDO, err := u.userRepo.FindUserByID(ctx, userId)
	if err != nil {
		logger.Warn("通过用户ID查询用户失败", zap.Error(err), zap.Int64("userId", userId))
		return nil, err
	}
	return userDO, nil
}

func (u *UserDomainService) UpdateUserInfo(userId int64, updateData *do.UserDO) error {
	ctx := context.Background()
	userDO, err := u.FindUserByID(ctx, userId)
	if err != nil {
		logger.Warn("通过用户ID查询用户失败", zap.Error(err), zap.Int64("userId", userId))
		return err
	}
	if updateData.NickName != "" { // 字符串字段：非空则更新
		userDO.NickName = updateData.NickName
	}

	if updateData.Avatar != "" {
		userDO.Avatar = updateData.Avatar
	}

	if updateData.Signature != "" {
		userDO.Signature = updateData.Signature
	}

	// 3. 自动填充更新时间（通用字段，强制更新）
	userDO.UpdateAt = time.Now()

	// 4. 执行更新（Repo层需支持动态更新仅修改的字段）
	err = u.userRepo.UpdateUserInfo(ctx, userDO)
	if err != nil {
		logger.Warn("更新用户信息失败", zap.Error(err), zap.Int64("userId", userId))
		return err
	}

	return nil

}
