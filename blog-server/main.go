/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-21 11:32:32
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-09 01:46:24
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
	"blog-server/common/db"
	"blog-server/common/utils/logger"
	"blog-server/config"
)

func main() {
	// 初始化配置
	conf := config.InitConfig()

	// 初始化日志
	logger.InitLogger(conf)
	// 确保日志文件被写入
	defer logger.Sync()
	// 初始化数据库
	gormDB := db.InitGorm(conf)
	// 初始化用户仓库
	userAppSrv := service.NewUserAppService(gormDB)
	// 初始化路由
	r := router.NewRouter(conf, userAppSrv, func() error { return nil })
	// 运行服务器
	app := server.NewApp(r)
	app.Run()
}
