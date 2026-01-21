/*
 * @Author: 1372594487 1372594487@qq.com
 * @Date: 2026-01-21 04:36:18
 * @LastEditors: '1372594487
 * @LastEditTime: 2026-01-21 09:31:11
 * @Description: File description
 */
package middleware

import (
	"blog-server/utils/logger"
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AccessLogMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 读取请求体
		body, newBody := GetRequestBody(ctx)
		ctx.Request.Body = newBody

		begin := time.Now()

		// 准备日志字段
		fields := []zap.Field{
			zap.String("ip", ctx.ClientIP()),
			zap.String("method", ctx.Request.Method),
			zap.String("path", ctx.Request.URL.Path),
			zap.String("params", ctx.Request.URL.RawQuery),
			zap.String("body", body),
			zap.String("user-agent", ctx.Request.UserAgent()),
			zap.String("referer", ctx.Request.Referer()),
		}

		// 定义一个缓冲区，用于存放响应数据的副本
		var responseBody bytes.Buffer
		// 同时写入原始响应和缓冲区
		multiWriter := io.MultiWriter(ctx.Writer, &responseBody)
		// 创建一个包装了原始响应写入器的自定义响应写入器
		ctx.Writer = &responseWriterWrapper{
			ResponseWriter: ctx.Writer,  // 保留原始 Writer 的所有能力（如写入 Header、获取状态码等）
			targetWriter:   multiWriter, // 将实际的数据写入操作重定向到 multiWriter
		}

		ctx.Next()
		respBody := responseBody.String()
		if len(respBody) > 1024 {
			respBody = respBody[:1024]
		}
		fields = append(fields, zap.Int64("dur_ms", time.Since(begin).Milliseconds()))
		fields = append(fields, zap.Int("status", ctx.Writer.Status()))
		fields = append(fields, zap.String("resp", respBody))
		logger.Info("access_log", fields...)

	}
}

type responseWriterWrapper struct {
	gin.ResponseWriter //类型嵌入
	targetWriter       io.Writer
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	return w.targetWriter.Write(b)
}

// 使用 TeeReader 避免完全复制
func GetRequestBody(ctx *gin.Context) (string, io.ReadCloser) {
	var buf bytes.Buffer
	tee := io.TeeReader(ctx.Request.Body, &buf)
	data, _ := io.ReadAll(tee)
	return string(data), io.NopCloser(&buf)
}
