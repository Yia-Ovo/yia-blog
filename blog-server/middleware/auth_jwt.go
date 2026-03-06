// middleware/auth_jwt.go
package middleware

import (
	"blog-server/common"
	"blog-server/common/utils/jwt"
	"context"
	"errors"

	jwtv4 "github.com/golang-jwt/jwt/v4"
)

// GetUserFromJWT 解析Token后，构建完整的common.User对象返回
func GetUserFromJWT(ctx context.Context, tokenStr string) (*common.User, error) {
	// 1. 解析Token，获取扩展后的Claims
	claims, err := jwt.ParseToken(tokenStr)
	if err != nil {
		switch {
		case errors.Is(err, jwtv4.ErrSignatureInvalid):
			return nil, errors.New("Token签名无效")
		case errors.Is(err, jwtv4.ErrTokenExpired):
			return nil, errors.New("Token已过期")
		default:
			return nil, errors.New("Token解析失败：" + err.Error())
		}
	}

	// 2. 构建完整的common.User对象（映射Claims字段）
	user := &common.User{
		UserID: claims.UserID,
	}

	return user, nil
}
