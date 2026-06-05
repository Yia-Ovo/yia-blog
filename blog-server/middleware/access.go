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
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type responseWriterWrapper struct {
	gin.ResponseWriter
	targetWriter io.Writer
	compressed   bool
	size         int
	minSize      int
}

func AccessLogMiddleware(conf *config.GlobalConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 只读取不打印
		body, newBody := GetRequestBody(ctx)
		_ = body // 不再打印请求体，避免乱码
		ctx.Request.Body = newBody

		begin := time.Now()

		// 包装响应
		var responseBody bytes.Buffer
		multiWriter := io.MultiWriter(ctx.Writer, &responseBody)
		ctx.Writer = &responseWriterWrapper{
			ResponseWriter: ctx.Writer,
			targetWriter:   multiWriter,
			minSize:        conf.Gin.Gzip.MinSize,
		}

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

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	w.size += len(b)
	if w.size >= w.minSize && w.Header().Get("Content-Encoding") != "" {
		w.compressed = true
	}
	return w.targetWriter.Write(b)
}

func GetRequestBody(ctx *gin.Context) (string, io.ReadCloser) {
	var buf bytes.Buffer
	tee := io.TeeReader(ctx.Request.Body, &buf)
	_, _ = io.ReadAll(tee)
	return "", io.NopCloser(&buf)
}
