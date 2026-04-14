/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-17 07:43:04
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-15 00:40:28
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
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims JWT载荷，包含UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID uint64 `json:"user_id"`
}

// 自定义密钥
var jwtSecret = []byte("yia-blog-key-123456")

func GenerateToken(user *convert.User) (string, error) {
	expireTime := time.Now().Add(2 * time.Hour)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "yia_blog",
		},
		UserID: user.UserID, // 从User对象取ID

	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// 解析Token：返回扩展后的Claims
func ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}

// ParseTokenToUser 解析Token并返回common.User（通用工具函数，替代原GetUserFromJWT）
func ParseTokenToUser(tokenStr string) (*convert.User, error) {
	claims, err := ParseToken(tokenStr)
	if err != nil {
		// 统一转换错误类型（便于上层处理）
		switch {
		case errors.Is(err, jwt.ErrSignatureInvalid):
			return nil, common.LoginTokenInvalidErr
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, common.LoginTokenExpiredErr
		default:
			return nil, common.AuthErr
		}
	}

	return &convert.User{UserID: claims.UserID}, nil
}

// GetTokenRemainingTime 获取Token剩余有效期（秒）
func GetTokenRemainingTime(tokenStr string) (int64, error) {
	claims, err := ParseToken(tokenStr)
	if err != nil {
		return 0, err
	}
	return claims.ExpiresAt.Time.Unix() - time.Now().Unix(), nil
}
