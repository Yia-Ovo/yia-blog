/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:25:51
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-15 13:13:08
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
	"blog-server/common/utils/jwt"
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
	userMysqlRepo repo.UserMysqlRepo // 注入仓库接口
	userRedisRepo repo.UserRedisRepo
}

// 对外提供创建实例的方法，给app层调用
func NewUserDomainService(userMysqlRepo repo.UserMysqlRepo, userRedisRepo repo.UserRedisRepo) *UserDomainService {
	return &UserDomainService{
		userMysqlRepo: userMysqlRepo,
		userRedisRepo: userRedisRepo,
	}
}

func (u *UserDomainService) UpdatePWD(ctx context.Context, userId uint64, newPassword string, oldPassword string) error {
	userDO, err := u.userMysqlRepo.FindUserByID(ctx, userId)
	if err != nil {
		logger.Warn("通过用户ID查询用户失败", zap.Error(err), zap.Uint64("userId", userId))
		return err
	}
	// 1. 校验用户是否存在（兜底判断）
	if userDO == nil {
		logger.Warn("用户不存在", zap.Uint64("userId", userId))
		return common.UserNotFoundErr
	}

	// 2. 验证旧密码是否正确（对比数据库加密密码）
	// userDO.Password 是数据库存储的 bcrypt 加密密文
	if err := bcrypt.CompareHashAndPassword([]byte(userDO.Password), []byte(oldPassword)); err != nil {
		logger.Warn("用户旧密码验证失败", zap.Uint64("userId", userId))
		return common.UserOldPasswordErr // 旧密码错误
	}

	// 3. 业务校验：新密码不能与旧密码相同
	if newPassword == oldPassword {
		logger.Warn("新密码与旧密码一致", zap.Uint64("userId", userId))
		return common.UserPasswordSameError // 新旧密码重复
	}

	// 4. 密码加密：使用 bcrypt 加密新密码（自动加盐，安全不可逆）
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Warn("新密码加密失败", zap.Error(err), zap.Uint64("userId", userId))
		return common.PasswordEncryptError // 加密异常
	}

	// 5. 数据库更新：调用仓储层更新密码
	if err := u.userMysqlRepo.UpdatePWD(ctx, userId, string(hashedPassword)); err != nil {
		logger.Warn("更新用户密码失败", zap.Error(err), zap.Uint64("userId", userId))
		return common.UpdatePasswordError // 更新失败
	}

	// 密码修改成功
	logger.Info("用户密码修改成功", zap.Uint64("userId", userId))

	return nil
}

func (u *UserDomainService) Login(ctx context.Context, nickName string, password string) (*convert.User, string, error) {
	// 1. 调用仓储层查询用户（领域层不直接操作DB，通过仓储层抽象）
	userDO, err := u.userMysqlRepo.FindUserByNickname(ctx, nickName)
	if err != nil {
		// 仓储层错误：区分“查询失败”和“用户不存在”
		if errors.Is(err, common.UserNotFoundErr) {
			// 安全考量：不暴露“用户不存在”，统一提示“用户名或密码错误”
			// 查不到：缓存空值（防穿透）
			_ = u.userRedisRepo.SetUserToRedis(ctx, 0, nil) // 0表示用户名不存在
			return nil, "", common.LoginPasswordErr
		}
		// 数据库异常：返回标准数据库错误码
		logger.Error("用户查询数据库失败", zap.Error(err), zap.String("nickname", nickName))
		return nil, "", common.DatabaseErr.WithErr(err)
	}

	// 2. 校验账号状态（核心业务规则）
	switch userDO.Status {
	case consts.UserStatusDisabled:
		return nil, "", common.UserAccountDisabledErr
	case consts.UserStatusLocked:
		return nil, "", common.LoginAccountLockedErr
	}

	// 3. bcrypt 密码校验(哈希比对，禁止明文存储/比对)
	if err = bcrypt.CompareHashAndPassword([]byte(userDO.Password), []byte(password)); err != nil {
		logger.Warn("用户密码校验失败", zap.Uint64("user_id", userDO.ID))
		return nil, "", common.LoginPasswordErr
	}

	// 4. 登录成功后的核心业务操作
	// 例：更新最后登录时间、登录IP
	updateErr := u.userMysqlRepo.UpdateLastLoginInfo(ctx, userDO.ID, userDO.LastLoginAt)
	if updateErr != nil {
		logger.Warn("更新用户登录信息失败", zap.Uint64("user_id", userDO.ID), zap.Error(updateErr))
	}

	// 5. DO 转换为前端通用对象（脱敏）
	commonUser := convert.UserDOToUser(userDO)
	// 6. 生成token并返回用户信息（注意：脱敏，不返回密码等敏感字段）
	token, err := jwt.GenerateToken(commonUser)
	if err != nil {
		logger.Error("生成用户Token失败", zap.Uint64("user_id", userDO.ID), zap.Error(err))
		return nil, "", common.ServerErr.WithErr(err)
	}
	return commonUser, token, nil
}
func (u *UserDomainService) Logout(ctx context.Context, userId uint64, token string) error {
	// 1. 【核心】将当前Token加入Redis黑名单（防止复用）
	// 获取JWT剩余过期时间（和登录时的过期时间保持一致）
	expireDuration, err := jwt.GetTokenRemainingTime(token)
	if err != nil {
		logger.Error("获取Token剩余时间失败", zap.Error(err))
		return err
	}
	// 2. 将当前Token加入Redis黑名单
	err = u.userRedisRepo.AddTokenToBlackList(ctx, token, expireDuration)
	if err != nil {
		logger.Warn("将Token加入黑名单失败", zap.Uint64("userId", userId), zap.Error(err))
	}
	// 3. 【全端下线】清空该用户所有在线Token
	err = u.userRedisRepo.ClearUserAllOnlineToken(ctx, userId)
	if err != nil {
		logger.Error("清空用户在线Token失败", zap.Uint64("userId", userId), zap.Error(err))
		return err
	}

	logger.Info("用户登出成功", zap.Uint64("userId", userId))
	return nil
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
	if err := u.userMysqlRepo.CreateUser(ctx, userDO); err != nil {
		logger.Warn("仓库创建用户失败", zap.Error(err), zap.String("nickName", nickName))
		return err
	}

	return nil
}

func (u *UserDomainService) FindUserByNickname(nickName string) (*do.UserDO, error) {
	ctx := context.Background()
	// 调用注入的仓库接口查询（仓库实现由基础设施层提供）
	userDO, err := u.userMysqlRepo.FindUserByNickname(ctx, nickName)
	if err != nil {
		logger.Warn("通过用户昵称查询用户失败", zap.Error(err), zap.String("nickName", nickName))
		return nil, err
	}
	return userDO, nil
}

func (u *UserDomainService) FindUserByID(ctx context.Context, userId uint64) (*do.UserDO, error) {
	// 1. 先查 Redis 缓存
	user, err := u.userRedisRepo.GetUserFromRedis(ctx, userId)
	if err == nil {
		// 缓存命中：
		// - 有数据 → 直接返回
		// - 空数据（缓存空值）→ 直接返回用户不存在
		if user == nil {
			return nil, common.UserNotFoundErr.WithErr(err)
		}
		return convert.UserModelToDO(user), nil
	}

	// 2. 缓存未命中/缓存错误 → 查数据库
	if !errors.Is(err, common.CacheNotExist) {
		// 非缓存不存在错误（如Redis故障），可以降级直接查库
		logger.Warn("redis error, fallback to db", zap.Error(err))
	}

	// 3. 查询数据库
	userDO, err := u.userMysqlRepo.FindUserByID(ctx, userId)
	if err != nil {
		// 4. 数据库也不存在 → 写入【空值缓存】防穿透
		_ = u.userRedisRepo.SetUserToRedis(ctx, userId, nil)
		logger.Warn("通过用户ID查询用户失败", zap.Error(err), zap.Uint64("userId", userId))
		return nil, err
	}

	// 5. 数据库存在 → 写入【正常缓存】
	_ = u.userRedisRepo.SetUserToRedis(ctx, userId, user)

	return userDO, nil
}

func (u *UserDomainService) UpdateUserInfo(userId uint64, updateData *do.UserDO) error {
	ctx := context.Background()
	userDO, err := u.FindUserByID(ctx, userId)
	if err != nil {
		logger.Warn("通过用户ID查询用户失败", zap.Error(err), zap.Uint64("userId", userId))
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
	err = u.userMysqlRepo.UpdateUserInfo(ctx, userDO)
	if err != nil {
		logger.Warn("更新用户信息失败", zap.Error(err), zap.Uint64("userId", userId))
		return err
	}

	return nil

}
