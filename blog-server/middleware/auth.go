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
		// 1. 获取Token
		rawToken := ctx.GetHeader(consts.UserTokenKey)
		token := jwt.ExtractToken(rawToken)
		// 2. 空Token拦截
		if token == "" {
			logger.Warn("[MIDDLEWARE] 用户认证失败：未携带Token",
				zap.String("path", path),
				zap.String("header_key", consts.UserTokenKey),
			)
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("请先登录"))
			ctx.Abort()
			return
		}
		// 3. 解析 token，得到 userBase.ID 校验用户在线token
		userBase, errNo := getTokenFun(ctx.Request.Context(), token)
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

		// 4. userBase 判空、ID 判 0
		if userBase == nil || userBase.ID == 0 {
			logger.Error("[MIDDLEWARE] 用户信息异常",
				zap.String("token", jwt.MaskToken(token)),
				zap.String("err", errNo.Error()),
			)
			response.WriteResp(ctx, nil, common.AuthErr.WithMsg("用户信息异常"))
			ctx.Abort()
			return
		}

		userRedisRepo := ioc.AppContainer.UserRedis
		online, err := userRedisRepo.IsUserOnlineToken(ctx.Request.Context(), userBase.ID, token)
		if err != nil {
			response.WriteResp(ctx, nil, common.ServerErr.WithMsg("系统异常"))
			ctx.Abort()
			return
		}
		if !online {
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("登录已失效，请重新登录"))
			ctx.Abort()
			return
		}

		// 5. 黑名单校验，失败直接返回系统异常
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

		// 6. 查数据库拿当前 userModel
		userModel, err := ioc.AppContainer.UserRepo.FindUserByID(ctx.Request.Context(), userBase.ID)
		if err != nil || userModel == nil {
			response.WriteResp(ctx, nil, common.AuthErr.WithMsg("用户不存在或已失效"))
			ctx.Abort()
			return
		}
		// 7. 校验 status
		if userModel.Status != consts.UserStatusEnabled {
			response.WriteResp(ctx, nil, common.PermissionDeniedErr.WithMsg("用户账号已禁用"))
			ctx.Abort()
			return
		}
		if userBase.TokenVersion != userModel.TokenVersion {
			logger.Warn("[MIDDLEWARE] 用户Token版本已失效",
				zap.Uint64("user_id", userBase.ID),
				zap.Uint64("token_version", userBase.TokenVersion),
				zap.Uint64("current_version", userModel.TokenVersion),
			)
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("登录已失效，请重新登录"))
			ctx.Abort()
			return
		}
		// 8. 组装 dto.UserBaseDTO
		user := &dto.UserBaseDTO{
			ID:           userModel.ID,
			NickName:     userModel.NickName,
			Avatar:       userModel.Avatar,
			Status:       userModel.Status,
			TokenVersion: userModel.TokenVersion,
			Gender:       userModel.Gender,
			Signature:    userModel.Signature,
			Email:        userModel.Email,
		}
		// 9. 写入 Gin context 和 request context
		ctx.Set(consts.CustomerUserKey, user)
		ctx.Set(consts.UserIDKey, user.ID)
		wrapRequestContext(ctx, consts.CustomerUserKey, user)
		wrapRequestContext(ctx, consts.UserIDKey, user.ID)
		// 10. ctx.Next()
		logger.Info("[MIDDLEWARE] 用户认证成功",
			zap.Uint64("user_id", user.ID),
			zap.String("path", path),
		)
		ctx.Next()
	}
}

// -------------------------- 管理员认证中间件 --------------------------
func AdminAuthMiddleware(shouldAuth func(*gin.Context) bool, getTokenFun TokenAdminFun) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.FullPath()
		method := ctx.Request.Method

		logger.Debug("[MIDDLEWARE] 进入管理员认证中间件",
			zap.String("path", path),
			zap.String("method", method),
		)

		if shouldAuth != nil && !shouldAuth(ctx) {
			logger.Debug("[MIDDLEWARE] 跳过管理员认证", zap.String("path", path))
			ctx.Next()
			return
		}
		// 1. 取 token
		rawToken := ctx.GetHeader(consts.AdminTokenKey)
		token := jwt.ExtractToken(rawToken)
		// 2. 空 token 拦截
		if token == "" {
			logger.Warn("[MIDDLEWARE] 管理员认证失败：未携带Token", zap.String("path", path))
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("管理员请先登录"))
			ctx.Abort()
			return
		}

		// 3. 解析 token，得到 adminBase.ID 校验管理员在线token
		adminBase, errNo := getTokenFun(ctx.Request.Context(), token)
		if !errNo.IsOk() {
			logger.Error("[MIDDLEWARE] 管理员Token解析失败",
				zap.String("token", jwt.MaskToken(token)),
				zap.String("err", errNo.Error()),
			)
			response.WriteResp(ctx, nil, errNo)
			ctx.Abort()
			return
		}

		// 4. adminBase 判空、ID 判 0
		if adminBase == nil || adminBase.ID == 0 {
			logger.Error("[MIDDLEWARE] 管理员身份异常",
				zap.String("token", jwt.MaskToken(token)),
				zap.String("err", errNo.Error()),
			)
			response.WriteResp(ctx, nil, common.AuthErr.WithMsg("管理员身份异常"))
			ctx.Abort()
			return
		}
		adminRedisRepo := ioc.AppContainer.AdminRedis
		online, err := adminRedisRepo.IsAdminOnlineToken(ctx.Request.Context(), adminBase.ID, token)
		if err != nil {
			logger.Error("[MIDDLEWARE] 查询管理员在线Token失败", zap.Error(err))
			response.WriteResp(ctx, nil, common.ServerErr.WithMsg("系统异常"))
			ctx.Abort()
			return
		}
		if !online {
			logger.Warn("[MIDDLEWARE] 管理员Token已失效", zap.String("path", path))
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("登录已失效，请重新登录"))
			ctx.Abort()
			return
		}

		// 5. 黑名单校验，失败直接返回系统异常
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

		// 6. 查数据库拿当前 adminModel
		adminModel, err := ioc.AppContainer.AdminRepo.FindAdminByID(ctx.Request.Context(), adminBase.ID)
		if err != nil || adminModel == nil {
			response.WriteResp(ctx, nil, common.AuthErr.WithMsg("管理员不存在或已失效"))
			ctx.Abort()
			return
		}
		// 7. 校验 status
		if adminModel.Status != consts.AdminStatusEnabled {
			response.WriteResp(ctx, nil, common.PermissionDeniedErr.WithMsg("管理员账号已禁用"))
			ctx.Abort()
			return
		}
		if adminBase.TokenVersion != adminModel.TokenVersion {
			logger.Warn("[MIDDLEWARE] 管理员Token版本已失效",
				zap.Uint64("admin_id", adminBase.ID),
				zap.Uint64("token_version", adminBase.TokenVersion),
				zap.Uint64("current_version", adminModel.TokenVersion),
			)
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("登录已失效，请重新登录"))
			ctx.Abort()
			return
		}
		// 8. 组装 dto.AdminBaseDTO
		admin := &dto.AdminBaseDTO{
			ID:           adminModel.ID,
			AdminName:    adminModel.AdminName,
			Avatar:       adminModel.Avatar,
			Role:         adminModel.Role,
			Status:       adminModel.Status,
			TokenVersion: adminModel.TokenVersion,
			Email:        adminModel.Email,
			Remark:       adminModel.Remark,
		}
		// 9. 写入 Gin context 和 request context
		ctx.Set(consts.CustomerAdminKey, admin)
		ctx.Set(consts.AdminIDKey, admin.ID)
		ctx.Set(consts.AdminRoleKey, admin.Role)
		wrapRequestContext(ctx, consts.CustomerAdminKey, admin)
		wrapRequestContext(ctx, consts.AdminIDKey, admin.ID)
		wrapRequestContext(ctx, consts.AdminRoleKey, admin.Role)

		// 10. ctx.Next()
		logger.Info("[MIDDLEWARE] 管理员认证成功", zap.Uint64("admin_id", admin.ID), zap.String("path", path))
		ctx.Next()
	}
}

// -------------------------- 超级管理员权限中间件  --------------------------
func AdminSuperAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminVal, exists := ctx.Get(consts.CustomerAdminKey)
		admin, ok := adminVal.(*dto.AdminBaseDTO)
		if !exists || !ok || admin == nil || admin.ID == 0 || admin.Role != consts.AdminRoleSuper {
			logger.Warn("[MIDDLEWARE] 未获取到管理员角色 | 权限不足，非超级管理员")
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
