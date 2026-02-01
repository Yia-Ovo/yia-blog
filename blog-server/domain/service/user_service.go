/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:25:51
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-02 02:45:08
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
type UserService struct{}

// 对外提供创建实例的方法，给app层调用
func NewUserService() *UserService {
	return &UserService{}
}

// Register 注册核心业务逻辑：密码bcrypt加密 → 封装DO → 调用仓库创建
func (u *UserService) Register(nickName string, password string) error {
	// 1. 核心：明文密码加密（bcrypt，不可逆，生产级安全）
	hashPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Warn("bcrypt加密失败", zap.Error(err))
		return err // 加密失败，返回错误
	}
	// 2. 封装领域对象DO（默认状态1-正常，无头像，未登录）
	userDO := &do.UserDO{
		NickName: nickName,
		Password: string(hashPwd), // 加密后的密码字符串
		Status:   1,
		Avatar:   "",
	}
	// 3. 调用仓库接口，写入数据库（业务层不用管数据库细节）
	return repo.UserRepoIns.CreateUser(userDO)
}
