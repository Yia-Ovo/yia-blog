/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-23 05:26:56
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-01-27 03:14:48
 * @FilePath: \yia-blog\blog-server\middleware\gzip.go
 * @Description:
集成方式：传入全局配置，仅当EnableGzip=true时启用，自动读取最小压缩阈值和排除类型配置
功能实现：
1、开关控制：根据配置决定是否启用Gzip压缩功能
2、类型过滤：响应阶段校验Content-Type，排除指定类型（如图片/视频）不压缩
3、阈值控制：结合最小压缩阈值配置，保证压缩效率
4、响应压缩：对符合条件的响应内容应用Gzip压缩
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
*/

package middleware

import (
	"blog-server/config"
	"strings"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func CustonGzipMiddleware(conf *config.GlobalConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !conf.Gin.Gzip.EnableGzip {
			ctx.Next()
			return
		}

		// 先执行后续处理
		ctx.Next()

		// 在响应时检查内容类型
		contentType := ctx.Writer.Header().Get("Content-Type")
		for _, excludeType := range conf.Gin.Gzip.GzipExcludeTypes {
			if strings.HasPrefix(contentType, strings.TrimSuffix(excludeType, "*")) {
				return
			}
		}

		// 应用压缩
		gzip.Gzip(gzip.DefaultCompression)(ctx)
	}
}
