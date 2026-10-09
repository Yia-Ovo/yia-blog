package router_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/gin/response"
	"blog-server/domain/repo"
	"blog-server/infra/model"
	"blog-server/ioc"
	"blog-server/middleware"

	"github.com/gin-gonic/gin"
)

const (
	authRedisTestToken        = "auth-redis-test-token-do-not-expose"
	authRedisTestVersion      = uint64(7)
	authRedisProtectedResult  = "protected-handler-result"
	authRedisPrivateErrorText = "private-redis-diagnostic: dial 127.0.0.1:6379 failed"
)

type authRedisFailureState struct {
	t                           *testing.T
	id                          uint64
	online, blacklisted         bool
	onlineErr, blacklistErr     error
	parseCalls, onlineCalls     int
	blacklistCalls, lookupCalls int
	handlerCalls                int
}

func (s *authRedisFailureState) checkToken(token string) {
	s.t.Helper()
	if token != authRedisTestToken {
		s.t.Fatal("auth check did not receive the extracted request token")
	}
}

func (s *authRedisFailureState) checkID(id uint64) {
	s.t.Helper()
	if id != s.id {
		s.t.Fatalf("auth check received identity %d, want %d", id, s.id)
	}
}

func (s *authRedisFailureState) onlineResult(id uint64, token string) (bool, error) {
	s.checkID(id)
	s.checkToken(token)
	s.onlineCalls++
	return s.online, s.onlineErr
}

func (s *authRedisFailureState) blacklistResult(token string) (bool, error) {
	s.checkToken(token)
	s.blacklistCalls++
	return s.blacklisted, s.blacklistErr
}

type authRedisFailureUsers struct {
	repo.UserMysqlRepo
	state *authRedisFailureState
}

func (r *authRedisFailureUsers) FindUserByID(_ context.Context, id uint64) (*model.UserModel, error) {
	r.state.checkID(id)
	r.state.lookupCalls++
	return &model.UserModel{ID: id, Status: consts.UserStatusEnabled, TokenVersion: authRedisTestVersion}, nil
}

type authRedisFailureAdmins struct {
	repo.AdminMysqlRepo
	state *authRedisFailureState
}

func (r *authRedisFailureAdmins) FindAdminByID(_ context.Context, id uint64) (*model.AdminModel, error) {
	r.state.checkID(id)
	r.state.lookupCalls++
	return &model.AdminModel{ID: id, Status: consts.AdminStatusEnabled, Role: consts.AdminRoleNormal, TokenVersion: authRedisTestVersion}, nil
}

type authRedisFailureUserRedis struct {
	repo.UserRedisRepo
	state *authRedisFailureState
}

func (r *authRedisFailureUserRedis) IsUserOnlineToken(_ context.Context, id uint64, token string) (bool, error) {
	return r.state.onlineResult(id, token)
}

func (r *authRedisFailureUserRedis) IsTokenBlacklisted(_ context.Context, token string) (bool, error) {
	return r.state.blacklistResult(token)
}

type authRedisFailureAdminRedis struct {
	repo.AdminRedisRepo
	state *authRedisFailureState
}

func (r *authRedisFailureAdminRedis) IsAdminOnlineToken(_ context.Context, id uint64, token string) (bool, error) {
	return r.state.onlineResult(id, token)
}

func (r *authRedisFailureAdminRedis) IsAdminTokenBlacklisted(_ context.Context, token string) (bool, error) {
	return r.state.blacklistResult(token)
}

func authRedisFailureEngine(t *testing.T, admin bool, state *authRedisFailureState) *gin.Engine {
	t.Helper()
	container := &ioc.Container{}
	identityKey := consts.UserIDKey
	var auth gin.HandlerFunc
	// 注入已通过解析的身份，只验证 Redis 鉴权边界，避免修改 JWT 包的全局配置。
	if admin {
		container.AdminRepo = &authRedisFailureAdmins{state: state}
		container.AdminRedis = &authRedisFailureAdminRedis{state: state}
		identityKey = consts.AdminIDKey
		auth = middleware.AdminAuthMiddleware(nil, func(_ context.Context, token string) (*dto.AdminBaseDTO, common.Errno) {
			state.checkToken(token)
			state.parseCalls++
			return &dto.AdminBaseDTO{ID: state.id, TokenVersion: authRedisTestVersion}, common.OK
		})
	} else {
		container.UserRepo = &authRedisFailureUsers{state: state}
		container.UserRedis = &authRedisFailureUserRedis{state: state}
		auth = middleware.AuthMiddleware(nil, func(_ context.Context, token string) (*dto.UserBaseDTO, common.Errno) {
			state.checkToken(token)
			state.parseCalls++
			return &dto.UserBaseDTO{ID: state.id, TokenVersion: authRedisTestVersion}, common.OK
		})
	}
	previousContainer := ioc.AppContainer
	ioc.AppContainer = container
	t.Cleanup(func() { ioc.AppContainer = previousContainer })

	engine := gin.New()
	engine.GET("/api/blog/v1/auth-redis-test", auth, func(ctx *gin.Context) {
		state.handlerCalls++
		id, exists := ctx.Get(identityKey)
		if !exists || id != state.id || ctx.Request.Context().Value(identityKey) != state.id {
			t.Error("authorized identity is missing from Gin or request context")
		}
		response.WriteResp(ctx, authRedisProtectedResult, common.OK)
	})
	return engine
}

func TestAuthRedisFailureDeniesProtectedRequest(t *testing.T) {
	// IOC 和 Gin mode 是包级状态，本测试及子测试不得并行执行。
	previousContainer, previousMode := ioc.AppContainer, gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(previousMode) })
	redisErr := errors.New(authRedisPrivateErrorText)
	for _, identity := range []struct {
		name, header string
		admin        bool
		id           uint64
	}{
		{name: "user", header: consts.UserTokenKey, id: 7},
		{name: "admin", header: consts.AdminTokenKey, admin: true, id: 9},
	} {
		t.Run(identity.name, func(t *testing.T) {
			for _, tc := range []struct {
				name                         string
				online, blacklisted          bool
				onlineErr, blacklistErr      error
				wantCode                     int
				wantMsg                      string
				wantBlacklist, wantProtected int
			}{
				{name: "healthy", online: true, wantCode: common.OK.Code, wantMsg: common.OK.Msg, wantBlacklist: 1, wantProtected: 1},
				{name: "offline", wantCode: common.LoginNotLoginErr.Code, wantMsg: "登录已失效，请重新登录"},
				{name: "blacklisted", online: true, blacklisted: true, wantCode: common.LoginNotLoginErr.Code, wantMsg: "登录已失效，请重新登录", wantBlacklist: 1},
				{name: "online_lookup_error", onlineErr: redisErr, wantCode: common.ServerErr.Code, wantMsg: "系统异常"},
				{name: "blacklist_lookup_error", online: true, blacklistErr: redisErr, wantCode: common.ServerErr.Code, wantMsg: "系统异常", wantBlacklist: 1},
			} {
				t.Run(tc.name, func(t *testing.T) {
					state := &authRedisFailureState{
						t: t, id: identity.id, online: tc.online, blacklisted: tc.blacklisted,
						onlineErr: tc.onlineErr, blacklistErr: tc.blacklistErr,
					}
					engine := authRedisFailureEngine(t, identity.admin, state)
					w, code := managementRequest(t, engine, http.MethodGet, "/auth-redis-test", identity.header, authRedisTestToken, "")
					if code != tc.wantCode {
						t.Fatalf("business code=%d want=%d: %s", code, tc.wantCode, w.Body.String())
					}
					var envelope struct {
						Msg    string          `json:"msg"`
						ErrMsg *string         `json:"err_msg"`
						Data   json.RawMessage `json:"data"`
					}
					if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
						t.Fatal(err)
					}
					if envelope.Msg != tc.wantMsg || envelope.ErrMsg == nil || *envelope.ErrMsg != "" {
						t.Fatalf("unexpected public error fields: %s", w.Body.String())
					}
					for _, private := range []string{authRedisPrivateErrorText, authRedisTestToken} {
						if strings.Contains(w.Body.String(), private) {
							t.Fatalf("internal Redis diagnostic or token exposed: %s", w.Body.String())
						}
					}
					if state.parseCalls != 1 || state.onlineCalls != 1 || state.blacklistCalls != tc.wantBlacklist || state.lookupCalls != tc.wantProtected || state.handlerCalls != tc.wantProtected {
						t.Fatalf("auth did not stop at the expected boundary: parse=%d online=%d blacklist=%d lookup=%d handler=%d",
							state.parseCalls, state.onlineCalls, state.blacklistCalls, state.lookupCalls, state.handlerCalls)
					}
					if tc.wantProtected == 0 {
						if string(envelope.Data) != "null" || strings.Contains(w.Body.String(), authRedisProtectedResult) {
							t.Fatalf("protected data exposed after auth rejection: %s", w.Body.String())
						}
					} else {
						var data string
						if err := json.Unmarshal(envelope.Data, &data); err != nil || data != authRedisProtectedResult {
							t.Fatalf("healthy control did not return protected data: %s", w.Body.String())
						}
					}
				})
				if ioc.AppContainer != previousContainer {
					t.Fatal("auth fixture did not restore the global IOC container")
				}
			}
		})
	}
}
