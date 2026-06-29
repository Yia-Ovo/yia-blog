package middleware

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/gin/response"
	"blog-server/common/utils/jwt"
	"blog-server/common/utils/logger"
	"blog-server/ioc"
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TokenFun 用户Token解析函数签名
type TokenFun func(ctx context.Context, token string) (*dto.UserBaseDTO, common.Errno)

// TokenAdminFun 管理员Token解析函数签名
type TokenAdminFun func(ctx context.Context, token string) (*dto.AdminBaseDTO, common.Errno)

// -------------------------- 用户认证中间件 --------------------------
func AuthMiddleware(filter func(*gin.Context) bool, getTokenFun TokenFun) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.FullPath()
		method := ctx.Request.Method

		logger.Debug("[MIDDLEWARE] 进入用户认证中间件",
			zap.String("path", path),
			zap.String("method", method),
		)

		if filter != nil && !filter(ctx) {
			logger.Debug("跳过用户认证", zap.String("path", path))
			ctx.Next()
			return
		}

		rawToken := ctx.GetHeader(consts.UserTokenKey)
		token := jwt.ExtractToken(rawToken)

		if token == "" {
			logger.Warn("[MIDDLEWARE] 用户认证失败：未携带Token",
				zap.String("path", path),
				zap.String("header_key", consts.UserTokenKey),
			)
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("请先登录"))
			ctx.Abort()
			return
		}

		user, errNo := getTokenFun(ctx.Request.Context(), token)
		if !errNo.IsOk() {
			logger.Error("[MIDDLEWARE] 用户Token解析失败",
				zap.String("token", jwt.MaskToken(token)),
				zap.String("path", path),
				zap.String("err", errNo.Error()),
			)
			response.WriteResp(ctx, nil, errNo)
			ctx.Abort()
			return
		}

		if user == nil {
			logger.Error("[MIDDLEWARE] 用户信息为空", zap.String("path", path))
			response.WriteResp(ctx, nil, common.AuthErr.WithMsg("用户信息异常"))
			ctx.Abort()
			return
		}

		// 黑名单校验
		userRedisRepo := ioc.AppContainer.UserRedis
		blackListed, err := userRedisRepo.IsTokenBlacklisted(ctx.Request.Context(), token)
		if err != nil {
			logger.Error("[MIDDLEWARE] 查询用户Token黑名单失败", zap.Error(err), zap.String("token", jwt.MaskToken(token)))
			response.WriteResp(ctx, nil, common.ServerErr.WithMsg("系统异常"))
			ctx.Abort()
			return
		}
		if blackListed {
			logger.Warn("[MIDDLEWARE] 用户Token已被拉黑", zap.String("token", jwt.MaskToken(token)), zap.String("path", path))
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("登录已失效，请重新登录"))
			ctx.Abort()
			return
		}

		ctx.Set(consts.CustomerUserKey, user)
		ctx.Set(consts.UserIDKey, user.ID)
		wrapRequestContext(ctx, consts.CustomerUserKey, user)

		logger.Info("[MIDDLEWARE] 用户认证成功",
			zap.Uint64("user_id", user.ID),
			zap.String("path", path),
		)

		ctx.Next()
	}
}

// -------------------------- 管理员认证中间件 --------------------------
func AdminAuthMiddleware(filter func(*gin.Context) bool, getTokenFun TokenAdminFun) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.FullPath()
		method := ctx.Request.Method

		logger.Debug("[MIDDLEWARE] 进入管理员认证中间件",
			zap.String("path", path),
			zap.String("method", method),
		)

		if filter != nil && !filter(ctx) {
			logger.Debug("[MIDDLEWARE] 跳过管理员认证", zap.String("path", path))
			ctx.Next()
			return
		}

		rawToken := ctx.GetHeader(consts.AdminTokenKey)
		token := jwt.ExtractToken(rawToken)

		if token == "" {
			logger.Warn("[MIDDLEWARE] 管理员认证失败：未携带Token", zap.String("path", path))
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("管理员请先登录"))
			ctx.Abort()
			return
		}

		admin, errNo := getTokenFun(ctx.Request.Context(), token)
		if !errNo.IsOk() {
			logger.Error("[MIDDLEWARE] 管理员Token解析失败",
				zap.String("token", jwt.MaskToken(token)),
				zap.String("err", errNo.Error()),
			)
			response.WriteResp(ctx, nil, errNo)
			ctx.Abort()
			return
		}

		if admin == nil {
			logger.Error("[MIDDLEWARE] 管理员信息为空", zap.String("path", path))
			response.WriteResp(ctx, nil, common.AuthErr.WithMsg("管理员信息异常"))
			ctx.Abort()
			return
		}

		// 黑名单校验
		adminRedisRepo := ioc.AppContainer.AdminRedis
		blackListed, err := adminRedisRepo.IsAdminTokenBlacklisted(ctx.Request.Context(), token)
		if err != nil {
			logger.Error("[MIDDLEWARE] 查询管理员Token黑名单失败", zap.Error(err))
			response.WriteResp(ctx, nil, common.ServerErr.WithMsg("系统异常"))
			ctx.Abort()
			return
		}
		if blackListed {
			logger.Warn("[MIDDLEWARE] 管理员Token已拉黑", zap.String("path", path))
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("登录已失效，请重新登录"))
			ctx.Abort()
			return
		}

		// 存入上下文（统一用 const key）
		ctx.Set(consts.CustomerAdminKey, admin)
		ctx.Set(consts.AdminIDKey, admin.ID)
		ctx.Set(consts.AdminRoleKey, admin.Role)
		wrapRequestContext(ctx, consts.CustomerAdminKey, admin)

		logger.Info("[MIDDLEWARE] 管理员认证成功", zap.Uint64("admin_id", admin.ID), zap.String("path", path))
		ctx.Next()
	}
}

// -------------------------- 超级管理员权限中间件  --------------------------
func AdminSuperAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		roleVal, exists := ctx.Get(consts.AdminRoleKey)
		if !exists {
			logger.Warn("[MIDDLEWARE] 未获取到管理员角色")
			response.WriteResp(ctx, nil, common.LoginNotLoginErr)
			ctx.Abort()
			return
		}

		role, ok := roleVal.(int)
		if !ok || role != consts.AdminRoleSuper {
			logger.Warn("[MIDDLEWARE] 权限不足，非超级管理员", zap.Any("role", roleVal))
			response.WriteResp(ctx, nil, common.PermissionDeniedErr)
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

// -------------------------- 快捷函数 --------------------------
func DefaultUserAuth() gin.HandlerFunc {
	return AuthMiddleware(nil, jwt.ParseTokenToUserWithCtx)
}

func DefaultAdminAuth() gin.HandlerFunc {
	return AdminAuthMiddleware(nil, jwt.ParseTokenToAdminWithCtx)
}

// wrapRequestContext 包装请求上下文（抽公共逻辑）
func wrapRequestContext(ctx *gin.Context, key string, val any) {
	reqCtx := context.WithValue(ctx.Request.Context(), key, val)
	ctx.Request = ctx.Request.WithContext(reqCtx)
}
