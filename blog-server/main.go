package main

import (
	"blog-server/config"
	"blog-server/router"
	"blog-server/utils/logger"
)

func main() {
	// 初始化配置
	conf := config.InitConfig()

	// 初始化日志
	logger.InitLogger(conf)
	// 确保日志文件被写入
	defer logger.Sync()
	// 初始化路由
	r := router.NewRouter(conf, nil)
	// 初始化应用
	app := router.NewApp(conf.Server.HttpPort, r)
	// 运行服务器
	app.Run()
}
