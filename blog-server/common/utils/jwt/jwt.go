/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-17 07:43:04
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-10 16:10:08
 * @FilePath: \yia-blog\blog-server\common\utils\jwt\jwt.go
 * @Description: JWT 工具包（用户/管理员双体系、与DTO对齐、安全加固）
 */
package jwt

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/config"
	"context"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// --------------------------- 声明结构（最小化，只存ID） ---------------------------
type CustomClaims struct {
	UserID       uint64  `json:"user_id"`
	TokenVersion *uint64 `json:"token_version"`
	jwt.RegisteredClaims
}

type AdminClaims struct {
	AdminID      uint64  `json:"admin_id"`
	TokenVersion *uint64 `json:"token_version"`
	jwt.RegisteredClaims
}

// --------------------------- 全局配置 ---------------------------
var (
	userSecret  []byte
	adminSecret []byte
	tokenExpire time.Duration
	issuer      string
)

// Init 初始化JWT（项目启动调用一次）
func Init(cfg *config.GlobalConfig) {
	jwtConf := cfg.JWT
	userSecret = []byte(jwtConf.CustomSecret)
	adminSecret = []byte(jwtConf.AdminSecret)
	tokenExpire = jwtConf.ExpireHour
	issuer = jwtConf.Issuer
}

// --------------------------- 生成Token（用户/管理员） ---------------------------
// GenerateUserToken 生成用户Token（传入 dto.UserBaseDTO）
func GenerateUserToken(user *dto.UserBaseDTO) (string, error) {
	exp := time.Now().Add(tokenExpire * time.Hour)
	tokenVersion := user.TokenVersion
	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
		},
		UserID:       user.ID,
		TokenVersion: &tokenVersion,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(userSecret)
}

// GenerateAdminToken 生成管理员Token 传入 AdminBaseDTO
func GenerateAdminToken(admin *dto.AdminBaseDTO) (string, error) {
	exp := time.Now().Add(tokenExpire * time.Hour)
	tokenVersion := admin.TokenVersion
	claims := AdminClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
		},
		AdminID:      admin.ID,
		TokenVersion: &tokenVersion,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(adminSecret)
}

// --------------------------- 解析Token（返回错误码，适配中间件） ---------------------------
// ParseUserToken 解析用户Token
func ParseUserToken(tokenStr string) (*CustomClaims, common.Errno) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(t *jwt.Token) (any, error) {
		return userSecret, nil
	})
	if err != nil || token == nil {
		return nil, common.AuthErr.WithMsg("Token解析失败")
	}
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, common.AuthErr.WithMsg("Token无效")
	}
	return claims, common.OK
}

// ParseAdminToken 解析管理员Token
func ParseAdminToken(tokenStr string) (*AdminClaims, common.Errno) {
	token, err := jwt.ParseWithClaims(tokenStr, &AdminClaims{}, func(t *jwt.Token) (any, error) {
		return adminSecret, nil
	})
	if err != nil || token == nil {
		return nil, common.AuthErr.WithMsg("管理员Token解析失败")
	}
	claims, ok := token.Claims.(*AdminClaims)
	if !ok || !token.Valid {
		return nil, common.AuthErr.WithMsg("管理员Token无效")
	}
	return claims, common.OK
}

// --------------------------- 中间件专用 ---------------------------
// ParseTokenToUserWithCtx 适配 auth.go
func ParseTokenToUserWithCtx(ctx context.Context, tokenStr string) (*dto.UserBaseDTO, common.Errno) {
	_ = ctx

	if tokenStr == "" {
		return nil, common.LoginNotLoginErr
	}
	claims, errNo := ParseUserToken(tokenStr)
	if !errNo.IsOk() {
		return nil, errNo
	}
	if claims.TokenVersion == nil {
		return nil, common.LoginNotLoginErr.WithMsg("登录已失效，请重新登录")
	}
	return &dto.UserBaseDTO{
		ID:           claims.UserID,
		TokenVersion: *claims.TokenVersion,
	}, common.OK
}

// ParseTokenToAdminWithCtx 适配 auth.go
func ParseTokenToAdminWithCtx(ctx context.Context, tokenStr string) (*dto.AdminBaseDTO, common.Errno) {
	_ = ctx

	if tokenStr == "" {
		return nil, common.LoginNotLoginErr
	}
	claims, errNo := ParseAdminToken(tokenStr)
	if !errNo.IsOk() {
		return nil, errNo
	}
	if claims.TokenVersion == nil {
		return nil, common.LoginNotLoginErr.WithMsg("登录已失效，请重新登录")
	}
	return &dto.AdminBaseDTO{
		ID:           claims.AdminID,
		TokenVersion: *claims.TokenVersion,
	}, common.OK
}

// --------------------------- 工具函数 ---------------------------
// ExtractToken 提取Bearer Token
func ExtractToken(header string) string {
	header = strings.TrimSpace(header)
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		header = header[7:]
	}
	return strings.TrimSpace(header)
}

// MaskToken 脱敏Token
func MaskToken(token string) string {
	if len(token) <= 10 {
		return "***"
	}
	return token[:10] + "***"
}

// GetTokenRemainingTime 获取Token剩余有效期
func GetTokenRemainingTime(tokenStr string) (time.Duration, common.Errno) {
	var c CustomClaims
	if _, _, err := new(jwt.Parser).ParseUnverified(tokenStr, &c); err == nil {
		rem := time.Until(c.ExpiresAt.Time)
		if rem < 0 {
			return 0, common.LoginTokenExpiredErr
		}
		return rem, common.OK
	}

	var ac AdminClaims
	if _, _, err := new(jwt.Parser).ParseUnverified(tokenStr, &ac); err == nil {
		rem := time.Until(ac.ExpiresAt.Time)
		if rem < 0 {
			return 0, common.LoginTokenExpiredErr
		}
		return rem, common.OK
	}
	return 0, common.LoginTokenInvalidErr
}

// GetTokenExpireAt 直接返回过期时间字符串
func GetTokenExpireAt(tokenStr string) (string, common.Errno) {
	var userClaims CustomClaims
	if _, _, err := new(jwt.Parser).ParseUnverified(tokenStr, &userClaims); err == nil {
		if userClaims.ExpiresAt == nil {
			return "", common.LoginTokenInvalidErr
		}
		return userClaims.ExpiresAt.Time.Format("2006-01-02 15:04:05"), common.OK
	}

	var adminClaims AdminClaims
	if _, _, err := new(jwt.Parser).ParseUnverified(tokenStr, &adminClaims); err == nil {
		if adminClaims.ExpiresAt == nil {
			return "", common.LoginTokenInvalidErr
		}
		return adminClaims.ExpiresAt.Time.Format("2006-01-02 15:04:05"), common.OK
	}

	return "", common.LoginTokenInvalidErr
}
