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
