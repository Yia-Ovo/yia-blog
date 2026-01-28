/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-26 19:58:36
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-01-28 10:03:23
 * @FilePath: \yia-blog\blog-server\app\cmd\server\server.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
// app/cmd/server/server.go  服务器生命周期管理（核心修改）
package server

import (
	"blog-server/api/router"
	gin_util "blog-server/common/gin"
	"blog-server/common/utils/logger"
	"blog-server/config"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

type App struct {
	// 仅保留核心的httpSrv，完全满足服务器生命周期管理
	httpSrv *http.Server
}

// NewApp 创建服务器实例（依赖注入路由实现）
func NewApp(router router.IRouter) *App {

	// 1. 从配置读取gin模式
	globalConf := config.GetGlobalConfig()
	if globalConf == nil {
		logger.Panic("服务器初始化失败：全局配置未加载")
	}

	// 2. 创建gin引擎并添加基础中间件
	engine := gin_util.NewGinEngine()

	// 3. 注册业务路由（路由职责完全归属于router包）
	router.Register(engine)

	// 4. 构建http服务器
	addr := fmt.Sprintf(":%d", globalConf.Server.HttpPort)
	httpSrv := &http.Server{
		Addr:    addr,
		Handler: engine,
	}

	return &App{
		httpSrv: httpSrv,
	}
}

// Run 启动服务器并实现优雅关闭
func (app *App) Run() {
	// 1. 异步启动http服务
	go func() {
		if err := app.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// 统一使用自定义logger，替代标准log库
			logger.Fatal("HTTP服务启动失败", zap.Error(err))
		}
	}()

	// 2. 打印启动日志
	endpoint := fmt.Sprintf("http://localhost:%d", config.GetGlobalConfig().Server.HttpPort)
	logger.Info("服务器启动成功", zap.String("endpoint", endpoint))

	// 3. 监听退出信号（优雅关闭）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	sig := <-quit
	logger.Warn("开始关闭服务器", zap.String("收到信号", sig.String()))

	// 4. 带超时的优雅关闭
	timeout := time.Duration(config.GetGlobalConfig().Server.ShutdownTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := app.httpSrv.Shutdown(ctx); err != nil {
		logger.Error("服务器强制关闭", zap.Error(err))
	} else {
		logger.Info("服务器已优雅退出")
	}
}
