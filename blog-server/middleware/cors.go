/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-23 06:19:18
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-01-27 03:19:11
 * @FilePath: \yia-blog\blog-server\middleware\cors.go
 * @Description:
 集成方式：传入全局配置，仅当Enabled=true时加载，自动读取白名单、允许方法/头、缓存时间等配置
功能实现：
1、白名单校验：校验请求Origin是否在配置白名单中，非白名单返回403禁止访问
2、响应头设置：设置允许的请求方法、请求头、是否允许携带Credentials等
3、预检请求处理：OPTIONS请求直接返回204，设置预检缓存时间
4、跨域规范：严格遵循CORS协议规范，保证跨域请求合法性
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

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Cors(conf *config.GlobalConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查来源是否在白名单中
		if slices.Contains(conf.Gin.CORS.AllowOrigins, origin) {

			// 设置具体的Origin，不使用*
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers",
				"Content-Type, Authorization, X-Requested-With, Accept-Encoding, X-CSRF-Token, Accept, Origin")

			// c.Header("Access-Control-Allow-Headers", strings.Join(conf.Server.CorsAllowHeaders, ", "))
			// c.Header("Access-Control-Allow-Methods", strings.Join(conf.Server.CorsAllowMethods, ", "))
		} else {
			logger.Warn("CORS violation",
				zap.String("origin", origin),
				zap.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if conf.Gin.CORS.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			// 根据业务需求设置合理的缓存时间
			c.Header("Access-Control-Max-Age", strconv.Itoa(conf.Gin.CORS.MaxAge))
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
