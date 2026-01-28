/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-27 11:00:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-01-28 08:35:15
 * @FilePath: \yia-blog\blog-server\common\gin\gin_init.go
 * @Description:
 集成所有自定义中间件：Recovery → CORS → AccessLog →  → Gzip
 遵循中间件注册优先级，适配生产级性能配置
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
*/

package gin_util

import (
	"blog-server/common/utils/logger" // 替换为你实际的日志工具包路径
	"blog-server/config"
	"blog-server/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewGinEngine() *gin.Engine {
	// 1. 获取全局配置（确保InitConfig已提前执行）
	globalConf := config.GetGlobalConfig()
	if globalConf == nil {
		logger.Panic("Gin引擎初始化失败：全局配置未加载")
	}
	ginConf := globalConf.Gin
	// 1. 基础配置：设置运行模式
	gin.SetMode(ginConf.Mode)
	logger.Info("Gin运行模式已设置", zap.String("mode", ginConf.Mode))

	// 2. 禁用控制台颜色（生产环境建议开启）
	if ginConf.DisableConsoleColor {
		gin.DisableConsoleColor()
		logger.Debug("Gin控制台颜色已禁用")
	}

	// 3. 创建Gin引擎实例（生产环境使用New()，避免默认中间件开销）
	engine := gin.New()

	// 4. 核心基础中间件
	// 4.1 Recovery：捕获panic，防止程序崩溃，返回500错误
	engine.Use(gin.RecoveryWithWriter(
		logger.GetWriter(), // 替换为日志工具的Writer，无则用gin.DefaultWriter
	))
	logger.Debug("Gin Recovery中间件已加载")
	// 4.2 CORS 中间件：处理跨域请求
	if ginConf.CORS.Enabled {
		engine.Use(middleware.Cors(globalConf))
		logger.Info("CORS 中间件已启用", zap.Any("allow_origins", ginConf.CORS.AllowOrigins))
	}
	// 4.3 AccessLog 中间件：记录请求日志
	engine.Use(middleware.AccessLogMiddleware(globalConf))
	logger.Info("AccessLog 中间件已启用")
	// 4.4 Gzip 中间件：压缩响应数据
	engine.Use(middleware.CustomGzipMiddleware(globalConf))
	logger.Info("Gzip 中间件已启用")

	logger.Info("Gin引擎初始化完成", zap.String("env", config.CurrentEnv))
	return engine
}

// GetEngineWithMiddlewares 扩展函数：支持动态添加自定义中间件
// 参数：middlewares - 自定义中间件列表
// 返回值：包含自定义中间件的Gin引擎
func GetEngineWithMiddlewares(middlewares ...gin.HandlerFunc) *gin.Engine {
	engine := NewGinEngine()
	if len(middlewares) > 0 {
		engine.Use(middlewares...)
		logger.Info("Gin自定义中间件已加载", zap.Int("count", len(middlewares)))
	}
	return engine
}

// ---------------- 扩展函数：用于业务路由中注册认证中间件 ----------------
// RegisterAuthMiddleware 注册用户认证中间件（按需注册，不全局注册）
// 参数：
//   - group: 需要认证的路由组（如 /api/v1/user）
//   - filter: 过滤函数（跳过某些接口的认证）
//   - tokenFun: 解析 Token 的业务函数
func RegisterAuthMiddleware(group *gin.RouterGroup, filter func(*gin.Context) bool, tokenFun middleware.TokenFun) {
	group.Use(middleware.AuthMiddleware(filter, tokenFun))
	logger.Info("用户认证中间件已注册到路由组", zap.String("group_path", group.BasePath()))
}
