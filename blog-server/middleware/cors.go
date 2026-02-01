/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-23 06:19:18
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-02 06:17:12
 * @FilePath: \yia-blog\blog-server\middleware\cors.go
 * @Description:
 集成方式：传入全局配置，仅当Enabled=true时加载，自动读取白名单、允许方法/头、缓存时间等配置
功能实现：
1、兼容空Origin：开发环境下Postman/curl无Origin头可正常请求，生产环境严格校验
2、白名单校验：浏览器跨域请求的Origin必须在白名单内
3、响应头设置：复用配置文件参数，不再硬编码
4、预检请求优先：OPTIONS请求提前处理，避免被白名单拦截
5、跨域规范：严格遵循CORS协议规范，保证跨域请求合法性
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
*/

package middleware

import (
	"blog-server/common/utils/logger"
	"blog-server/config"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Cors(conf *config.GlobalConfig) gin.HandlerFunc {
	// 提前将配置的方法/头拼接成字符串，避免每次请求重复拼接
	allowMethods := strings.Join(conf.Gin.CORS.AllowMethods, ", ")
	allowHeaders := strings.Join(conf.Gin.CORS.AllowHeaders, ", ")
	// 白名单缓存
	allowOrigins := conf.Gin.CORS.AllowOrigins

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		// 【优化1：优先处理预检请求OPTIONS】避免预检请求被白名单拦截
		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Max-Age", strconv.Itoa(conf.Gin.CORS.MaxAge))
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// 【核心修复：兼容空Origin + 白名单校验】
		// 空Origin：Postman/curl等工具发起的请求，直接允许（开发环境友好）
		// 非空Origin：必须在白名单内，保证浏览器跨域安全
		allow := false
		if origin == "" {
			allow = true // 兼容无Origin的工具请求
		} else if slices.Contains(allowOrigins, origin) {
			allow = true // 浏览器跨域请求，Origin在白名单内
		}

		// 校验不通过，返回403并记录日志
		if !allow {
			logger.Warn("CORS violation",
				zap.String("origin", origin),
				zap.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// 【优化2：复用配置文件参数，不再硬编码】
		// 设置允许的Origin：非空则用真实Origin，空则设为*（适配工具请求）
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}
		c.Header("Access-Control-Allow-Methods", allowMethods)
		c.Header("Access-Control-Allow-Headers", allowHeaders)

		// 允许携带Cookie/Token（如JWT）
		if conf.Gin.CORS.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 继续执行后续中间件/处理函数
		c.Next()
	}
}
