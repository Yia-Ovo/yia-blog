/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-21 11:32:32
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-05 16:46:35
 * @FilePath: \yia-blog\blog-server\middleware\access.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package middleware

import (
	"blog-server/common/utils/logger"
	"blog-server/config"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AccessLogMiddleware 仅记录请求元数据，不预读请求体或复制响应体，保留流式读写能力。
func AccessLogMiddleware(_ *config.GlobalConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		begin := time.Now()

		ctx.Next()

		// 直观日志
		dur := time.Since(begin).Milliseconds()
		status := ctx.Writer.Status()
		method := ctx.Request.Method
		path := ctx.Request.URL.Path

		// 极简格式：[ACCESS] 200 POST /api/xxx | 耗时:xxms
		logger.Info("[ACCESS]",
			zap.Int("status", status),
			zap.String("method", method),
			zap.String("path", path),
			zap.Int64("dur_ms", dur),
		)
	}
}
