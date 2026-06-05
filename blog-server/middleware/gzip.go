/*
  - @Author: Yia 1372594487@qq.com
  - @Date: 2026-01-23 05:26:56
  - @LastEditors: Yia 1372594487@qq.com
  - @LastEditTime: 2026-06-03 13:24:13
  - @FilePath: \yia-blog\blog-server\middleware\gzip.go
  - @Description:

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

func CustomGzipMiddleware(conf *config.GlobalConfig) gin.HandlerFunc {
	// 关闭gzip直接原生放行
	if !conf.Gin.Gzip.EnableGzip {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// 新版API：WithCustomShouldCompressFn 返回true=要压缩；false=跳过压缩，和之前逻辑相反！！
	return gzip.Gzip(gzip.DefaultCompression,
		gzip.WithMinLength(conf.Gin.Gzip.MinSize),
		gzip.WithCustomShouldCompressFn(func(c *gin.Context) bool {
			ct := c.Writer.Header().Get("Content-Type")
			// 遍历排除列表，命中→不压缩（return false）
			for _, excl := range conf.Gin.Gzip.GzipExcludeTypes {
				prefix := strings.TrimSuffix(excl, "*")
				if strings.HasPrefix(ct, prefix) {
					return false
				}
			}
			// 没命中排除→开启压缩
			return true
		}),
	)
}
