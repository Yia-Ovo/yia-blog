/*
 * @Author: 1372594487 1372594487@qq.com
 * @Date: 2026-01-23 06:19:18
 * @LastEditors: '1372594487
 * @LastEditTime: 2026-01-24 00:37:17
 * @Description: File description
 */
package middleware

import (
	"blog-server/config"
	"blog-server/utils/logger"
	"net/http"
	"slices"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Cors(conf *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查来源是否在白名单中
		if slices.Contains(conf.Server.CorsOrigins, origin) {

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

		if conf.Server.CorsCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			// 根据业务需求设置合理的缓存时间
			c.Header("Access-Control-Max-Age", strconv.Itoa(conf.Server.CorsMaxAge))
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
