/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-03-06 13:24:11
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-05 15:41:50
 * @FilePath: \yia-blog\blog-server\common\gin\response\response.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package response

import (
	"blog-server/common"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 统一响应结构体
type Resp struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	ErrMsg string `json:"err_msg"`
	Data   any    `json:"data"`
}

// 统一响应写入
func WriteResp(ctx *gin.Context, data any, errno common.Errno) {
	ctx.JSON(http.StatusOK, Resp{
		Code:   errno.Code,
		Msg:    errno.Msg, // 对外提示
		ErrMsg: "",        // 保留兼容字段，内部诊断信息不对外返回。
		Data:   data,
	})
}

func WithDBTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 5*time.Second)
}
