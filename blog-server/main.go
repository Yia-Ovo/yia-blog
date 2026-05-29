/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-21 11:32:32
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 14:23:14
 * @FilePath: \yia-blog\blog-server\main.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-21 11:32:32
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-27 14:47:41
 * @FilePath: \yia-blog\blog-server\main.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package main

import (
	"blog-server/api/router"
	"blog-server/app/cmd/server"
	"blog-server/common/db/gorm"
	"blog-server/common/db/redis"
	"blog-server/common/utils/jwt"
	"blog-server/common/utils/logger"
	"blog-server/config"
	"blog-server/ioc"
)

func main() {
	// 1. 基础初始化
	conf := config.InitConfig()
	jwt.Init(conf)
	logger.InitLogger(conf)
	defer logger.Sync()

	// 2. 数据存储
	db := gorm.InitGorm(conf)
	rdb := redis.InitRedis(conf)

	// 3. 初始化 IOC 容器（自动注入所有依赖）
	c := ioc.NewContainer(db, rdb, conf)
	ioc.AppContainer = c // 赋值给全局访问点

	// 4. 初始化路由（只传容器）
	r := router.NewRouter(conf, c, func() error { return nil })

	// 5. 启动服务
	app := server.NewApp(r)
	app.Run()
}
