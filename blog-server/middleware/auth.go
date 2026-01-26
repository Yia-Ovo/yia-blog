/*
* @Author: Yia 1372594487@qq.com
* @Date: 2026-01-25 23:13:50
  - @LastEditors: Yia 1372594487@qq.com
  - @LastEditTime: 2026-01-27 03:17:39
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
	"context"
	"net/http"

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
			ctx.JSON(http.StatusUnauthorized, common.AuthErr)
			ctx.Abort()
			return
		}
		user, err := getTokenFun(ctx, token)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, common.AuthErr.WithErr(err))
			ctx.Abort()
			return
		}
		ctx.Set(consts.CustomerUserKey, user)
		ctx.Next()
	}
}

func AuthAdminMiddleware(filter func(*gin.Context) bool, getTokenFun TokenAdminFun) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if filter != nil && !filter(ctx) {
			ctx.Next()
			return
		}
		token := ctx.GetHeader(consts.AdminTokenKey)
		if len(token) == 0 {
			ctx.JSON(http.StatusUnauthorized, common.AuthErr)
			ctx.Abort()
			return
		}
		adminUser, err := getTokenFun(ctx, token)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, common.AuthErr.WithErr(err))
			ctx.Abort()
			return
		}
		ctx.Set(consts.AdminTokenKey, adminUser)
		ctx.Next()
	}
}
