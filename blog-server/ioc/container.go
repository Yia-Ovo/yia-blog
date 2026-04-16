/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-16 09:13:28
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-16 09:40:45
 * @FilePath: \yia-blog\blog-server\ioc\container.go
 * @Description:IOC容器
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package ioc

import (
	"blog-server/config"
	domainRepo "blog-server/domain/repo"
	infraRepo "blog-server/infra/repo"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// 全局单例：托管所有仓储实现
var (
	// 基础全局依赖
	Conf  *config.GlobalConfig
	DB    *gorm.DB
	Redis *redis.Client

	// ===================== 用户端仓储 =====================
	UserMysqlRepo domainRepo.UserMysqlRepo // MySQL接口
	UserRedisRepo domainRepo.UserRedisRepo // Redis接口

	// ===================== 管理员端仓储 =====================
	// AdminRedisRepo domainRepo.AdminRedisRepo // 管理员Redis接口
)

// Init 初始化IOC：绑定接口与实现
func Init(db *gorm.DB, rdb *redis.Client, conf *config.GlobalConfig) {
	// 1. 赋值基础依赖
	DB = db
	Redis = rdb
	Conf = conf

	// 2. 初始化基础设施实现
	UserMysqlRepo = infraRepo.NewUserMysqlRepo(db)
	UserRedisRepo = infraRepo.NewUserRedisRepo(rdb, conf)
	// AdminRedisRepo = infraRepo.NewAdminRedisRepo(rdb, conf)
}

// ===================== 对外获取方法 =====================
func GetUserMysqlRepo() domainRepo.UserMysqlRepo { return UserMysqlRepo }
func GetUserRedisRepo() domainRepo.UserRedisRepo { return UserRedisRepo }

// func GetAdminRedisRepo() domainRepo.AdminRedisRepo { return AdminRedisRepo }
