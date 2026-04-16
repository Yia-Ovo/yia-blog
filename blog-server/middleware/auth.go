/*
* @Author: Yia 1372594487@qq.com
* @Date: 2026-01-25 23:13:50
  - @LastEditors: Yia 1372594487@qq.com
  - @LastEditTime: 2026-04-16 10:12:19
  - @FilePath: \yia-blog\blog-server\middleware\auth.go

* @Description:
集成方式：支持用户/管理员双认证体系，传入过滤函数和Token解析函数，灵活跳过无需认证接口
功能实现：
1、用户认证：从请求头获取用户Token，校验通过后将用户信息存入上下文，失败返回401未授权
2、管理员认证：从请求头获取管理员Token，校验通过后将管理员信息存入上下文，失败返回401未授权
3、过滤机制：支持自定义过滤函数，跳过指定接口的认证校验
4、Redis黑名单：校验Token是否被拉黑，强制作废Token
*
* Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
*/
package middleware

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/gin/response"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/jwt"
	"blog-server/common/utils/logger"
	"blog-server/ioc"
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TokenFun 用户Token解析函数签名
type TokenFun func(ctx context.Context, token string) (*convert.User, common.Errno)

// TokenAdminFun 管理员Token解析函数签名
type TokenAdminFun func(ctx context.Context, token string) (*convert.AdminUser, common.Errno)

// -------------------------- 用户认证中间件 --------------------------
// AuthMiddleware 用户认证中间件（支持自定义过滤函数，跳过无需认证的接口）
func AuthMiddleware(filter func(*gin.Context) bool, getTokenFun TokenFun) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		path := ctx.FullPath()
		method := ctx.Request.Method

		logger.Debug("进入用户认证中间件",
			zap.String("path", path),
			zap.String("method", method),
		)
		// 1. 过滤逻辑：返回false 直接跳过认证
		if filter != nil && !filter(ctx) {
			logger.Debug("跳过用户认证", zap.String("path", path))
			ctx.Next()
			return
		}
		// 2. 获取Token
		rawToken := ctx.GetHeader(consts.UserTokenKey)
		token := jwt.ExtractToken(rawToken)

		if token == "" {
			logger.Warn("用户认证失败：未携带Token",
				zap.String("path", path),
				zap.String("header_key", consts.UserTokenKey),
			)
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("请先登录"))
			ctx.Abort()
			return
		}

		// 3. 解析Token
		user, errNo := getTokenFun(ctx.Request.Context(), token)
		if !errNo.IsOk() {
			logger.Error("用户Token解析失败",
				zap.String("token", jwt.MaskToken(token)),
				zap.String("path", path),
				zap.String("err", errNo.Error()),
			)
			response.WriteResp(ctx, nil, errNo)
			ctx.Abort()
			return
		}

		// 4. 空值防护（避免空指针崩溃）
		if user == nil {
			logger.Error("用户信息为空", zap.String("path", path))
			response.WriteResp(ctx, nil, common.AuthErr.WithMsg("用户信息异常"))
			ctx.Abort()
			return
		}

		// 5. 黑名单校验（调用领域接口，不关心底层实现）
		userRedisRepo := ioc.GetUserRedisRepo() // 从IOC拿取用户Redis仓储
		blackListed, err := userRedisRepo.IsTokenBlacklisted(ctx.Request.Context(), token)
		if err != nil {
			logger.Error("查询用户Token黑名单失败", zap.Error(err), zap.String("token", jwt.MaskToken(token)))
		}
		if blackListed {
			logger.Warn("用户Token已被拉黑", zap.String("token", jwt.MaskToken(token)), zap.String("path", path))
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("登录已失效，请重新登录"))
			ctx.Abort()
			return
		}

		// 6. 存入上下文
		ctx.Set(consts.CustomerUserKey, user)
		ctx.Set(consts.UserIDKey, user.UserID)

		reqCtx := context.WithValue(ctx.Request.Context(), consts.CustomerUserKey, user)
		ctx.Request = ctx.Request.WithContext(reqCtx)

		logger.Info("用户认证成功",
			zap.Uint64("user_id", user.UserID),
			zap.String("path", path),
		)

		ctx.Next()

	}
}

// -------------------------- 管理员认证中间件 --------------------------
// AdminAuthMiddleware 管理员认证中间件（双认证体系核心）
func AdminAuthMiddleware(filter func(*gin.Context) bool, getTokenFun TokenAdminFun) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.FullPath()
		method := ctx.Request.Method

		logger.Debug("进入管理员认证中间件",
			zap.String("path", path),
			zap.String("method", method),
		)

		// 1. 过滤逻辑
		if filter != nil && !filter(ctx) {
			logger.Debug("跳过管理员认证", zap.String("path", path))
			ctx.Next()
			return
		}

		// 2. 获取Token
		rawToken := ctx.GetHeader(consts.AdminTokenKey)
		token := jwt.ExtractToken(rawToken)

		if token == "" {
			logger.Warn("管理员认证失败：未携带Token", zap.String("path", path))
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("管理员请先登录"))
			ctx.Abort()
			return
		}

		// 3. 解析管理员Token
		admin, errNo := getTokenFun(ctx.Request.Context(), token)
		if !errNo.IsOk() {
			logger.Error("管理员Token解析失败",
				zap.String("token", jwt.MaskToken(token)),
				zap.String("err", errNo.Error()),
			)
			response.WriteResp(ctx, nil, errNo)
			ctx.Abort()
			return
		}

		// 4. 空值防护（避免空指针崩溃）
		if admin == nil {
			logger.Error("用户信息为空", zap.String("path", path))
			response.WriteResp(ctx, nil, common.AuthErr.WithMsg("用户信息异常"))
			ctx.Abort()
			return
		}

		// 5. 黑名单校验（调用领域接口，不关心底层实现）
		// adminRedisRepo := ioc.GetAdminRedisRepo() // 从IOC拿取用户Redis仓储
		// blackListed, err := adminRedisRepo.IsTokenBlacklisted(ctx.Request.Context(), token)
		// if err != nil {
		// 	logger.Error("查询用户Token黑名单失败", zap.Error(err), zap.String("token", jwt.MaskToken(token)))
		// }
		// if blackListed {
		// 	logger.Warn("用户Token已被拉黑", zap.String("token", jwt.MaskToken(token)), zap.String("path", path))
		// 	response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("登录已失效，请重新登录"))
		// 	ctx.Abort()
		// 	return
		// }

		// 6. 存入上下文
		ctx.Set(consts.AdminUserKey, admin)
		ctx.Set(consts.AdminIDKey, admin.AdminID)

		reqCtx := context.WithValue(ctx.Request.Context(), consts.AdminUserKey, admin)
		ctx.Request = ctx.Request.WithContext(reqCtx)

		logger.Info("管理员认证成功", zap.Uint64("admin_id", admin.AdminID), zap.String("path", path))
		ctx.Next()
	}
}

// -------------------------- 快捷函数 --------------------------
// DefaultUserAuth 默认用户认证中间件
func DefaultUserAuth() gin.HandlerFunc {
	return AuthMiddleware(nil, jwt.ParseTokenToUserWithCtx)
}

func DefaultAdminAuth() gin.HandlerFunc {
	return AdminAuthMiddleware(nil, jwt.ParseTokenToAdminWithCtx)
}
