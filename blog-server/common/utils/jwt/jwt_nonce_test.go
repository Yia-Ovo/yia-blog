package jwt

import (
	"blog-server/api/dto"
	"context"
	"encoding/base64"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v4"
)

func TestGeneratedTokensHaveIndependentIDs(t *testing.T) {
	initTestJWT(time.Hour)
	seen := make(map[string]bool)
	for _, identity := range []string{"user", "admin"} {
		t.Run(identity, func(t *testing.T) {
			for index := 0; index < 32; index++ {
				var token, id string
				var err error
				if identity == "user" {
					token, err = GenerateUserToken(&dto.UserBaseDTO{ID: 17, TokenVersion: 4})
					if err != nil {
						t.Fatal(err)
					}
					claims, errno := ParseUserToken(token)
					if !errno.IsOk() || claims.UserID != 17 || claims.TokenVersion == nil || *claims.TokenVersion != 4 {
						t.Fatal("用户身份与版本契约不能变化")
					}
					id = claims.ID
				} else {
					token, err = GenerateAdminToken(&dto.AdminBaseDTO{ID: 17, TokenVersion: 4})
					if err != nil {
						t.Fatal(err)
					}
					claims, errno := ParseAdminToken(token)
					if !errno.IsOk() || claims.AdminID != 17 || claims.TokenVersion == nil || *claims.TokenVersion != 4 {
						t.Fatal("管理员身份与版本契约不能变化")
					}
					id = claims.ID
				}
				bytes, err := base64.RawURLEncoding.DecodeString(id)
				if err != nil || len(bytes) != 16 {
					t.Fatal("每次签发必须有 128 位随机 jti")
				}
				if seen[id] {
					t.Fatal("不同签发不能复用同一 jti")
				}
				seen[id] = true
				assertTokenRemainingTime(t, token, time.Hour)
			}
		})
	}
}

func TestLegacyTokensWithoutIDRemainValid(t *testing.T) {
	initTestJWT(time.Hour)
	version := uint64(4)
	registered := jwtlib.RegisteredClaims{ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour)), IssuedAt: jwtlib.NewNumericDate(time.Now()), Issuer: issuer}
	userToken, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, CustomClaims{UserID: 17, TokenVersion: &version, RegisteredClaims: registered}).SignedString(userSecret)
	if err != nil {
		t.Fatal(err)
	}
	user, errno := ParseTokenToUserWithCtx(context.Background(), userToken)
	if !errno.IsOk() || user == nil || user.ID != 17 || user.TokenVersion != 4 {
		t.Fatal("旧用户令牌仍应可解析")
	}
	adminToken, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, AdminClaims{AdminID: 17, TokenVersion: &version, RegisteredClaims: registered}).SignedString(adminSecret)
	if err != nil {
		t.Fatal(err)
	}
	admin, errno := ParseTokenToAdminWithCtx(context.Background(), adminToken)
	if !errno.IsOk() || admin == nil || admin.ID != 17 || admin.TokenVersion != 4 {
		t.Fatal("旧管理员令牌仍应可解析")
	}
}
