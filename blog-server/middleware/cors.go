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
4、预检请求在来源、方法和请求头校验后返回完整允许头，不进入业务鉴权
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
		// 同一个 URL 的无 Origin 响应也要声明 Vary，避免共享缓存混用两类响应。
		c.Writer.Header().Add("Vary", "Origin")
		// 无 Origin 的请求无需 CORS 头，避免与凭据标记组合成无效的通配来源。
		if origin == "" {
			c.Next()
			return
		}
		if !slices.Contains(allowOrigins, origin) {
			logger.Warn("[MIDDLEWARE] CORS violation",
				zap.String("origin", origin),
				zap.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		requestedMethod := c.GetHeader("Access-Control-Request-Method")
		preflight := c.Request.Method == http.MethodOptions && requestedMethod != ""
		if preflight {
			c.Writer.Header().Add("Vary", "Access-Control-Request-Method")
			c.Writer.Header().Add("Vary", "Access-Control-Request-Headers")
			if !slices.Contains(conf.Gin.CORS.AllowMethods, requestedMethod) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			for _, header := range strings.Split(c.GetHeader("Access-Control-Request-Headers"), ",") {
				header = strings.TrimSpace(header)
				if header != "" && !slices.ContainsFunc(conf.Gin.CORS.AllowHeaders, func(allowed string) bool {
					return strings.EqualFold(header, strings.TrimSpace(allowed))
				}) {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			}
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", allowMethods)
		c.Header("Access-Control-Allow-Headers", allowHeaders)

		// 允许携带Cookie/Token（如JWT）
		if conf.Gin.CORS.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if preflight {
			c.Header("Access-Control-Max-Age", strconv.Itoa(conf.Gin.CORS.MaxAge))
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// 继续执行后续中间件/处理函数
		c.Next()
	}
}
