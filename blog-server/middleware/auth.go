/*
* @Author: Yia 1372594487@qq.com
* @Date: 2026-01-25 23:13:50
  - @LastEditors: Yia 1372594487@qq.com
  - @LastEditTime: 2026-02-17 14:39:42
  - @FilePath: \yia-blog\blog-server\middleware\auth.go

* @Description:
集成方式：支持用户/管理员双认证体系，传入过滤函数和Token解析函数，灵活跳过无需认证接口
功能实现：
1、用户认证：从请求头获取用户Token，校验通过后将用户信息存入上下文，失败返回401未授权
2、管理员认证：从请求头获取管理员Token，校验通过后将管理员信息存入上下文，失败返回401未授权
3、过滤机制：支持自定义过滤函数，跳过指定接口的认证校验
*
* Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
*/
package middleware

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/gin/response"
	"context"

	"github.com/gin-gonic/gin"
)

type TokenFun func(ctx context.Context, token string) (*common.User, error)
type TokenAdminFun func(ctx context.Context, token string) (*common.AdminUser, error)

func AuthMiddleware(filter func(*gin.Context) bool, getTokenFun TokenFun) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if filter != nil && !filter(ctx) {
			ctx.Next()
			return
		}

		token := ctx.GetHeader(consts.UserTokenKey)
		if len(token) == 0 {
			response.WriteResp(ctx, nil, common.AuthErr.WithMsg("未携带用户Token，请先登录"))
			ctx.Abort()
			return
		}

		// 解析Token，获取完整的User对象
		user, err := getTokenFun(ctx, token)
		if err != nil {
			response.WriteResp(ctx, nil, common.AuthErr.WithMsg(err.Error()))
			ctx.Abort()
			return
		}

		// 将完整User对象存入上下文（后续可直接取整个对象）
		ctx.Set(consts.CustomerUserKey, user)
		ctx.Next()
	}
}
