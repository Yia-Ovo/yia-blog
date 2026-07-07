package jwt

import (
	"blog-server/api/dto"
	"blog-server/config"
	"testing"
	"time"
)

func TestGenerateUserTokenUsesConfiguredDuration(t *testing.T) {
	const expire = 2 * time.Hour
	initTestJWT(expire)

	token, err := GenerateUserToken(&dto.UserBaseDTO{
		ID:           1,
		TokenVersion: 1,
	})
	if err != nil {
		t.Fatalf("GenerateUserToken() error = %v", err)
	}

	assertTokenRemainingTime(t, token, expire)
}

func TestGenerateAdminTokenUsesConfiguredDuration(t *testing.T) {
	const expire = 2 * time.Hour
	initTestJWT(expire)

	token, err := GenerateAdminToken(&dto.AdminBaseDTO{
		ID:           1,
		TokenVersion: 1,
	})
	if err != nil {
		t.Fatalf("GenerateAdminToken() error = %v", err)
	}

	assertTokenRemainingTime(t, token, expire)
}

func initTestJWT(expire time.Duration) {
	Init(&config.GlobalConfig{
		JWT: config.JWTConfig{
			CustomSecret: "test-user-secret",
			AdminSecret:  "test-admin-secret",
			ExpireHour:   expire,
			Issuer:       "test",
		},
	})
}

func assertTokenRemainingTime(t *testing.T, token string, want time.Duration) {
	t.Helper()

	got, errNo := GetTokenRemainingTime(token)
	if !errNo.IsOk() {
		t.Fatalf("GetTokenRemainingTime() errNo = %+v", errNo)
	}
	if got <= want-time.Minute || got > want {
		t.Fatalf("remaining time = %v, want within (%v, %v]", got, want-time.Minute, want)
	}
}
