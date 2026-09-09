package router_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appservice "blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
	"blog-server/ioc"
)

type publicUserRepo struct {
	repo.UserMysqlRepo
	user *model.UserModel
}

func (r *publicUserRepo) FindUserByNickname(context.Context, string) (*model.UserModel, error) {
	return r.user, nil
}

type privateUserCache struct {
	repo.UserRedisRepo
	user *model.UserModel
}

func (r *privateUserCache) GetUserFromRedis(context.Context, uint64) (*model.UserModel, error) {
	return r.user, nil
}

func TestPublicUserProfileHasExactAllowlist(t *testing.T) {
	lastLogin := time.Now().Add(-time.Hour)
	user := &model.UserModel{
		ID: 7, NickName: "author", Avatar: "avatar.png", Signature: "hello",
		Password: "private-hash", Email: "private@example.com", LastLoginAt: &lastLogin,
		Status: consts.UserStatusEnabled, TokenVersion: 9, Gender: 2,
	}
	app := appservice.NewUserAppService(domainservice.NewUserDomainService(&publicUserRepo{user: user}), &privateUserCache{user: user})
	engine := publicTestEngine(&ioc.Container{UserApp: app})
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/blog/v1/user/findUserByNickname?nick_name=author", nil))
	var body struct {
		Code int
		Data map[string]interface{}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || body.Code != common.OK.Code || len(body.Data) != 4 {
		t.Fatalf("unexpected public response: HTTP %d %s", w.Code, w.Body.String())
	}
	for _, key := range []string{"id", "nick_name", "avatar", "signature"} {
		if _, ok := body.Data[key]; !ok {
			t.Fatalf("missing public field %q: %s", key, w.Body.String())
		}
	}
	// 本人资料仍保留邮箱、状态和登录时间；公开投影不能删掉本人用例的数据。
	private, err := app.GetUserInfo(context.Background(), user.ID)
	if err != nil || private.Email != user.Email || private.Status != user.Status || private.LastLoginAt == nil || !private.LastLoginAt.Equal(lastLogin) {
		t.Fatalf("private profile changed: %+v err=%v", private, err)
	}
}

func TestPublicUserMissingReturnsNotFound(t *testing.T) {
	app := appservice.NewUserAppService(domainservice.NewUserDomainService(&publicUserRepo{}), nil)
	w := httptest.NewRecorder()
	publicTestEngine(&ioc.Container{UserApp: app}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/blog/v1/user/findUserByNickname?nick_name=missing", nil))
	var body struct{ Code int }
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != http.StatusOK || body.Code != common.UserNotFoundErr.Code {
		t.Fatalf("missing nickname reported as query failure: HTTP %d %s", w.Code, w.Body.String())
	}
}
