/*
* @Author: Yia 1372594487@qq.com
* @Date: 2026-01-25 23:13:50
  - @LastEditors: Yia 1372594487@qq.com
  - @LastEditTime: 2026-03-14 00:37:02
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
	"blog-server/common/utils/jwt"
	"blog-server/common/utils/logger"
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TokenFun 用户Token解析函数签名
type TokenFun func(ctx context.Context, token string) (*common.User, common.Errno)

// TokenAdminFun 管理员Token解析函数签名
type TokenAdminFun func(ctx context.Context, token string) (*common.AdminUser, common.Errno)

// -------------------------- 用户认证中间件 --------------------------
// AuthMiddleware 用户认证中间件（支持自定义过滤函数，跳过无需认证的接口）
func AuthMiddleware(filter func(*gin.Context) bool, getTokenFun TokenFun) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		logger.Debug("进入用户认证中间件",
			zap.String("request_path", ctx.FullPath()), // 请求完整路径
			zap.String("method", ctx.Request.Method),   // 请求方法
		)
		// 1. 过滤逻辑：如果过滤函数返回false，跳过认证
		if filter != nil && !filter(ctx) {
			ctx.Next()
			return
		}

		// 2. 从请求头获取Token（兼容Bearer格式）

		tokenStr := ctx.GetHeader(consts.UserTokenKey)

		logger.Debug("从Header获取原始Token",
			zap.String("header_key", consts.UserTokenKey), // Header键名
			zap.String("raw_token", tokenStr),             // 原始Token（dev环境可看，生产可屏蔽）
			zap.Int("token_length", len(tokenStr)),        // Token长度（判断是否为空）
		)
		if len(tokenStr) == 0 {
			logger.Warn("用户认证失败：未获取到Token",
				zap.String("header_key", consts.UserTokenKey),
				zap.String("request_path", ctx.FullPath()),
			)
			response.WriteResp(ctx, nil, common.LoginNotLoginErr.WithMsg("未携带用户Token，请先登录"))
			ctx.Abort()
			return
		}

		// 3. 处理Bearer前缀
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
		logger.Debug("处理Bearer前缀后的Token",
			zap.String("processed_token", tokenStr), // 处理后的Token
			zap.Int("token_length", len(tokenStr)),  // 处理后长度
		)
		// 4. 调用自定义Token解析函数（GetUserFromJWT）
		user, err := getTokenFun(ctx.Request.Context(), tokenStr)
		if !err.IsOk() {
			logger.Error("Token解析失败",
				zap.String("processed_token", tokenStr),
				zap.String("error_msg", err.Error()),
				zap.String("request_path", ctx.FullPath()),
			)
			response.WriteResp(ctx, nil, common.AuthErr.WithMsg(err.Error()))
			ctx.Abort()
			return
		}

		// 5. 存入上下文（兼容两种方式：完整User对象 + 单独user_id）
		ctx.Set(consts.CustomerUserKey, user) // 存入完整User对象
		ctx.Set("user_id", user.UserID)       // 兼容GetInfo接口的common.GetUserID(c)
		logger.Debug("用户信息已存入Gin上下文",
			zap.Int64("user_id", user.UserID),
			zap.String("context_key", consts.CustomerUserKey),
		)
		// 6. 继续执行后续逻辑
		ctx.Next()
	}
}

// -------------------------- 快捷函数：默认用户认证（无需过滤） --------------------------
// DefaultUserAuth 默认用户认证中间件（直接使用你的GetUserFromJWT，无过滤）
func DefaultUserAuth() gin.HandlerFunc {
	return AuthMiddleware(nil, ParseTokenToUser)
}

// token解析函数
func ParseTokenToUser(ctx context.Context, tokenStr string) (*common.User, common.Errno) {
	logger.Debug("进入ParseTokenToUser函数",
		zap.String("token", tokenStr),
	)
	// 调用jwt.ParseTokenToUser，转换错误为common.Errno
	user, err := jwt.ParseTokenToUser(tokenStr)
	if err != nil {
		logger.Error("jwt.ParseTokenToUser执行失败",
			zap.String("token", tokenStr),
			zap.Error(err), // 记录原始error对象（zap会自动解析）
		)
		if errno, ok := err.(common.Errno); ok {
			logger.Debug("jwt.ParseTokenToUser解析成功",
				zap.Any("user_info", user), // 打印完整用户信息（dev环境用，生产可注释）
			)
			return nil, errno
		}
		return nil, common.AuthErr.WithMsg("Token解析失败：" + err.Error())
	}
	return user, common.OK
}
