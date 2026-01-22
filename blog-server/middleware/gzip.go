/*
 * @Author: 1372594487 1372594487@qq.com
 * @Date: 2026-01-23 05:26:56
 * @LastEditors: '1372594487
 * @LastEditTime: 2026-01-23 05:42:18
 * @Description: File description
 */
package middleware

import (
	"blog-server/config"
	"strings"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func CustonGzipMiddleware(conf *config.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !conf.Server.EnableGzip {
			ctx.Next()
			return
		}

		// 先执行后续处理
		ctx.Next()

		// 在响应时检查内容类型
		contentType := ctx.Writer.Header().Get("Content-Type")
		for _, excludeType := range conf.Server.GzipExcludeTypes {
			if strings.HasPrefix(contentType, strings.TrimSuffix(excludeType, "*")) {
				return
			}
		}

		// 应用压缩
		gzip.Gzip(gzip.DefaultCompression)(ctx)
	}
}
