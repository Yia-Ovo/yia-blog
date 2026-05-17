/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-21 11:32:32
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-17 08:11:01
 * @FilePath: \yia-blog\blog-server\main.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package main

import (
	"blog-server/api/router"
	"blog-server/app/cmd/server"
	"blog-server/app/service"
	"blog-server/common/db/gorm"
	"blog-server/common/db/redis"
	"blog-server/common/utils/jwt"
	"blog-server/common/utils/logger"
	"blog-server/config"
	"blog-server/ioc"
)

func main() {
	// 初始化配置
	conf := config.InitConfig()
	// 初始化JWT
	jwt.Init(conf)
	// 初始化日志
	logger.InitLogger(conf)
	// 确保日志文件被写入
	defer logger.Sync()
	// 初始化数据库
	gormDB := gorm.InitGorm(conf)
	//初始化redis
	resDB := redis.InitRedis(conf)
	// 初始化ioc容器
	ioc.Init(gormDB, resDB, conf)
	// 初始化用户仓库
	userAppSrv := service.NewUserAppService(
		ioc.GetUserMysqlRepo(), // 传入领域接口
		ioc.GetUserRedisRepo(), // 传入领域接口
		conf,
	)
	// 初始化分页仓库
	categoryAppSrv := service.NewCategoryAppService(
		ioc.GetCategoryMysqlRepo(), // 传入领域接口
		ioc.GetCategoryRedisRepo(), // 传入领域接口
	)
	articleAppSrv := service.NewArticleAppService(
		ioc.GetArticleMysqlRepo(), // 传入领域接口
		ioc.GetArticleRedisRepo(), // 传入领域接口
	)

	// 初始化管理员应用服务
	adminAppService := service.NewAdminAppService(
		ioc.GetAdminMysqlRepo(),
		ioc.GetAdminRedisRepo(),
		ioc.Conf,
	)
	// 初始化路由
	r := router.NewRouter(conf, userAppSrv, categoryAppSrv, articleAppSrv, adminAppService, func() error { return nil })
	// 运行服务器
	app := server.NewApp(r)
	app.Run()
}
