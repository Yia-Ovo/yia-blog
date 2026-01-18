/*
 * @Author: 1372594487 1372594487@qq.com
 * @Date: 2026-01-19 04:15:44
 * @LastEditors: '1372594487
 * @LastEditTime: 2026-01-19 05:22:46
 * @Description: File description
 */
package router

import (
	"blog-server/utils/logger"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type App struct {
	server *gin.Engine
	addr   string
}

func NewApp(port int, router IRouter) *App {
	// 设置为发布模式
	gin.SetMode(gin.ReleaseMode)
	// 创建一个没有任何中间件的纯净引擎
	engine := gin.New()
	// 添加 Recovery 中间件：捕获 panic，防止程序直接崩溃，并返回 500
	engine.Use(gin.Recovery())
	// 注册业务路由
	router.Register(engine)

	return &App{server: engine, addr: ":" + strconv.Itoa(port)}

}

// 服务器运行与优雅关闭
func (app *App) Run() {
	srv := &http.Server{
		Addr:    app.addr,
		Handler: app.server,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// log.Fatalf 会打印错误信息到标准日志输出，然后调用 os.Exit(1) 终止整个程序。
			log.Fatalf("listen err: %v", err)
		}
	}()

	logger.Debug(fmt.Sprintf("server started, endpoint: http://localhost%s", app.addr))
	// 等待中断信号以优雅地关闭服务器（设置 5 秒的超时时间）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	msg := <-quit
	logger.Warn("shutting down server...", zap.String("msg", msg.String()))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown:", zap.Error(err))
	} else {
		logger.Info("server exiting")
	}

}
