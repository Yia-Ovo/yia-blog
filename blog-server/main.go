package main

import (
	"fmt"

	"blog-server/config"
	"blog-server/utils/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 1. 初始化配置
	conf := config.InitConfig()

	defer logger.Sync()

	// 3. 启动Gin服务
	r := gin.New()
	// 测试接口
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// 监听端口
	if err := r.Run(fmt.Sprintf(":%d", conf.Server.HttpPort)); err != nil {
		zap.L().Fatal("服务启动失败", zap.Error(err))
	}
}
