/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:25:51
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-09 00:41:38
 * @FilePath: \yia-blog\blog-server\domain\service\user_service.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"

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
		NickName: nickName,
		Password: string(hashPwd), // 存储加密后的密码
		Status:   1,               // 1=正常，2=禁用
		Avatar:   "",              // 默认无头像
	}
	// 3. 调用仓库接口，写入数据库（业务层不用管数据库细节）
	if err := u.userRepo.CreateUser(userDO); err != nil {
		logger.Error("仓库创建用户失败", zap.Error(err), zap.String("nickName", nickName))
		return err
	}

	return nil
}

func (u *UserDomainService) FindUserByNickname(nickName string) (*do.UserDO, error) {
	// 调用注入的仓库接口查询（仓库实现由基础设施层提供）
	userDO, err := u.userRepo.FindUserByNickname(nickName)
	if err != nil {
		logger.Warn("查询用户失败", zap.Error(err), zap.String("nickName", nickName))
		return nil, err
	}
	return userDO, nil
}
