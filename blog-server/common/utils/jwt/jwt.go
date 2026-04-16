/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-17 07:43:04
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-15 19:43:31
 * @FilePath: \yia-blog\blog-server\common\utils\jwt\jwt.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
// util/jwt/jwt.go
package jwt

import (
	"blog-server/common"
	"blog-server/common/utils/convert"
	"blog-server/config"
	"context"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims JWT载荷，包含UserID
type CustomClaims struct {
	UserID uint64 `json:"user_id"`
	jwt.RegisteredClaims
}

type AdminClaims struct {
	ID uint64 `json:"admin_id"`
	jwt.RegisteredClaims
}

var (
	CustomSecret []byte        // 用户Token密钥
	AdminSecret  []byte        // 管理员Token密钥
	tokenExpire  time.Duration // 从配置读取
	issuer       string
)

// Init 初始化 JWT 配置（项目启动时调用一次）
func Init(cfg *config.GlobalConfig) {
	jwtConf := cfg.JWT
	CustomSecret = []byte(jwtConf.CustomSecret)
	AdminSecret = []byte(jwtConf.AdminSecret)
	tokenExpire = jwtConf.ExpireHour
	issuer = jwtConf.Issuer

}

// ===================== Token生成（用户+管理员） =====================
// GenerateToken 生成用户Token
func GenerateToken(user *convert.User) (string, error) {
	expireTime := time.Now().Add(tokenExpire * time.Hour)
	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
		},
		UserID: user.UserID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(CustomSecret)
}

// GenerateAdminToken 生成管理员Token
func GenerateAdminToken(admin *convert.AdminUser) (string, error) {
	expireTime := time.Now().Add(tokenExpire * time.Hour)
	claims := AdminClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
		},
		ID: admin.AdminID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(AdminSecret)
}

// =====================  Token解析（用户+管理员） =====================
// ParseToken 解析用户Token
func ParseToken(tokenStr string, secret []byte) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}

// ParseAdminToken 解析管理员Token
func ParseAdminToken(token string, secret []byte) (*AdminClaims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &AdminClaims{}, func(token *jwt.Token) (any, error) {
		return secret, nil
	})

	if err != nil || tokenClaims == nil {
		return nil, common.AuthErr.WithMsg("管理员Token无效")
	}

	if claims, ok := tokenClaims.Claims.(*AdminClaims); ok && tokenClaims.Valid {
		return claims, nil
	}

	return nil, common.AuthErr.WithMsg("管理员Token解析失败")
}

// ===================== 中间件专用适配函数 =====================
func ParseTokenToUserWithCtx(ctx context.Context, tokenStr string) (*convert.User, common.Errno) {
	if tokenStr == "" {
		return nil, common.LoginNotLoginErr
	}
	claims, err := ParseToken(tokenStr, CustomSecret)
	if err != nil {
		if e, ok := err.(common.Errno); ok {
			return nil, e
		}
		return nil, common.AuthErr.WithMsg("Token无效")
	}
	return &convert.User{UserID: claims.UserID}, common.OK
}

func ParseTokenToAdminWithCtx(ctx context.Context, tokenStr string) (*convert.AdminUser, common.Errno) {
	if tokenStr == "" {
		return nil, common.LoginNotLoginErr
	}
	claims, err := ParseAdminToken(tokenStr, AdminSecret)
	if err != nil {
		if e, ok := err.(common.Errno); ok {
			return nil, e
		}
		return nil, common.AuthErr.WithMsg("管理员Token无效")
	}
	return &convert.AdminUser{AdminID: claims.ID}, common.OK
}

// ===================== 通用：获取Token剩余时间（支持用户+管理员） =====================
// GetTokenRemainingTime 获取Token剩余有效期，返回time.Duration（适配登出黑名单逻辑）
func GetTokenRemainingTime(tokenString string) (time.Duration, error) {
	// 尝试用户Token
	claims := &CustomClaims{}
	if _, _, err := new(jwt.Parser).ParseUnverified(tokenString, claims); err == nil {
		rem := time.Until(claims.ExpiresAt.Time)
		if rem < 0 {
			return 0, common.LoginTokenExpiredErr
		}
		return rem, nil
	}

	// 尝试管理员Token
	adminClaims := &AdminClaims{}
	if _, _, err := new(jwt.Parser).ParseUnverified(tokenString, adminClaims); err == nil {
		rem := time.Until(adminClaims.ExpiresAt.Time)
		if rem < 0 {
			return 0, common.LoginTokenExpiredErr
		}
		return rem, nil
	}

	return 0, common.LoginTokenInvalidErr
}

// ===================== 通用工具函数 =====================
// extractToken 从请求头提取Token，自动处理Bearer前缀、脱敏
// 兼容大小写、多余空格：Bearer xxx / bearer xxx / Bearer   xxx
func ExtractToken(headerValue string) string {
	if headerValue == "" {
		return ""
	}
	// 先去除首尾空格
	headerValue = strings.TrimSpace(headerValue)
	// 不区分大小写匹配Bearer前缀
	if strings.HasPrefix(strings.ToLower(headerValue), "bearer ") {
		headerValue = headerValue[7:]
	}
	// 再次去空格，避免Bearer后有多余空格
	return strings.TrimSpace(headerValue)
}

// maskToken Token脱敏（生产环境保护隐私，只显示前10位+***）
func MaskToken(token string) string {
	if len(token) <= 10 {
		return "***"
	}
	return token[:10] + "***"
}
