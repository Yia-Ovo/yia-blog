package jwt

import (
	"blog-server/api/dto"
	"blog-server/common"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v4"
)

func TestParseTokensOnlyAcceptHS256(t *testing.T) {
	initTestJWT(time.Hour)
	identities := []struct {
		name     string
		idClaim  string
		secret   []byte
		opposite []byte
		parse    func(string) (bool, common.Errno)
	}{
		{
			name: "user", idClaim: "user_id", secret: userSecret, opposite: adminSecret,
			parse: func(token string) (bool, common.Errno) {
				claims, errno := ParseUserToken(token)
				return claims != nil, errno
			},
		},
		{
			name: "admin", idClaim: "admin_id", secret: adminSecret, opposite: userSecret,
			parse: func(token string) (bool, common.Errno) {
				claims, errno := ParseAdminToken(token)
				return claims != nil, errno
			},
		},
	}
	for _, identity := range identities {
		t.Run(identity.name, func(t *testing.T) {
			cases := []struct {
				name    string
				method  jwtlib.SigningMethod
				key     any
				expires time.Duration
				wantOK  bool
			}{
				{name: "HS256", method: jwtlib.SigningMethodHS256, key: identity.secret, expires: time.Hour, wantOK: true},
				{name: "HS384 with same secret", method: jwtlib.SigningMethodHS384, key: identity.secret, expires: time.Hour},
				{name: "HS512 with same secret", method: jwtlib.SigningMethodHS512, key: identity.secret, expires: time.Hour},
				{name: "none", method: jwtlib.SigningMethodNone, key: jwtlib.UnsafeAllowNoneSignatureType, expires: time.Hour},
				{name: "wrong secret", method: jwtlib.SigningMethodHS256, key: []byte("unrelated-test-secret"), expires: time.Hour},
				{name: "opposite identity secret", method: jwtlib.SigningMethodHS256, key: identity.opposite, expires: time.Hour},
				{name: "expired", method: jwtlib.SigningMethodHS256, key: identity.secret, expires: -time.Minute},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					claims := jwtlib.MapClaims{
						identity.idClaim: uint64(17),
						"token_version":  uint64(4),
						"jti":            "algorithm-test-id",
						"iss":            issuer,
						"iat":            time.Now().Add(-time.Hour).Unix(),
						"exp":            time.Now().Add(tc.expires).Unix(),
					}
					token, err := jwtlib.NewWithClaims(tc.method, claims).SignedString(tc.key)
					if err != nil {
						t.Fatal("failed to sign test token")
					}
					hasClaims, errno := identity.parse(token)
					if errno.IsOk() != tc.wantOK || hasClaims != tc.wantOK {
						t.Fatalf("accepted = %v, returned claims = %v, want %v", errno.IsOk(), hasClaims, tc.wantOK)
					}
					if !tc.wantOK && errno.Code != common.AuthErr.Code {
						t.Fatalf("error code = %d, want %d", errno.Code, common.AuthErr.Code)
					}
				})
			}
		})
	}
}

func TestGeneratedTokensKeepIdentityIsolation(t *testing.T) {
	initTestJWT(time.Hour)
	userToken, err := GenerateUserToken(&dto.UserBaseDTO{ID: 17, TokenVersion: 4})
	if err != nil {
		t.Fatal("failed to generate user test token")
	}
	adminToken, err := GenerateAdminToken(&dto.AdminBaseDTO{ID: 17, TokenVersion: 4})
	if err != nil {
		t.Fatal("failed to generate admin test token")
	}
	if claims, errno := ParseAdminToken(userToken); errno.IsOk() || claims != nil {
		t.Fatal("user token must not authenticate as admin")
	}
	if claims, errno := ParseUserToken(adminToken); errno.IsOk() || claims != nil {
		t.Fatal("admin token must not authenticate as user")
	}
}
