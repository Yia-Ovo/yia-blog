package router_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"blog-server/api/dto"
	appservice "blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/jwt"
	"blog-server/config"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
	"blog-server/ioc"

	"github.com/gin-gonic/gin"
)

type managementHTTPUsers struct{ repo.UserMysqlRepo }

func (*managementHTTPUsers) FindUserByID(_ context.Context, id uint64) (*model.UserModel, error) {
	if id != 7 && id != 8 {
		return nil, nil
	}
	return &model.UserModel{ID: id, Status: consts.UserStatusEnabled}, nil
}

type managementHTTPAdmins struct {
	repo.AdminMysqlRepo
	disabled bool
}

func (r *managementHTTPAdmins) FindAdminByID(_ context.Context, id uint64) (*model.AdminModel, error) {
	status := consts.AdminStatusEnabled
	if r.disabled {
		status = consts.AdminStatusDisabled
	}
	return &model.AdminModel{ID: id, Status: status, Role: consts.AdminRoleNormal}, nil
}

type managementHTTPUserRedis struct{ repo.UserRedisRepo }

func (*managementHTTPUserRedis) IsUserOnlineToken(context.Context, uint64, string) (bool, error) {
	return true, nil
}
func (*managementHTTPUserRedis) IsTokenBlacklisted(context.Context, string) (bool, error) {
	return false, nil
}

type managementHTTPAdminRedis struct{ repo.AdminRedisRepo }

func (*managementHTTPAdminRedis) IsAdminOnlineToken(context.Context, uint64, string) (bool, error) {
	return true, nil
}
func (*managementHTTPAdminRedis) IsAdminTokenBlacklisted(context.Context, string) (bool, error) {
	return false, nil
}

type managementHTTPArticles struct {
	repo.ArticleMysqlRepo
	articles []*model.ArticleModel
	query    *do.ArticleQuery
}

func (r *managementHTTPArticles) GetArticleById(_ context.Context, id uint64) (*model.ArticleModel, error) {
	for _, a := range r.articles {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, nil
}
func (r *managementHTTPArticles) ListArticleByPage(_ context.Context, q *do.ArticleQuery) ([]*model.ArticleModel, int64, error) {
	copy := *q
	r.query = &copy
	list := []*model.ArticleModel{}
	for _, a := range r.articles {
		if q.UserID != nil && (a.UserID != *q.UserID || a.AdminID != 0) {
			continue
		}
		if q.Status != nil && a.Status != *q.Status {
			continue
		}
		list = append(list, a)
	}
	return list, int64(len(list)), nil
}

func managementHTTPContainer(t *testing.T) (*ioc.Container, *managementHTTPArticles) {
	t.Helper()
	jwt.Init(&config.GlobalConfig{JWT: config.JWTConfig{CustomSecret: "test-user-only", AdminSecret: "test-admin-only", ExpireHour: time.Hour}})
	r := &managementHTTPArticles{articles: []*model.ArticleModel{
		{ID: 1, UserID: 7, Status: consts.ArticleStatusDraft, Title: "own draft", Content: "private draft"},
		{ID: 2, UserID: 8, Status: consts.ArticleStatusOffline, Content: "foreign private"},
		{ID: 3, AdminID: 9, Status: consts.ArticleStatusDraft, Content: "admin private"},
	}}
	c := &ioc.Container{UserRepo: &managementHTTPUsers{}, AdminRepo: &managementHTTPAdmins{}, UserRedis: &managementHTTPUserRedis{}, AdminRedis: &managementHTTPAdminRedis{}, ArticleRepo: r}
	c.UserDomain = domainservice.NewUserDomainService(c.UserRepo)
	c.AdminDomain = domainservice.NewAdminDomainService(c.AdminRepo)
	c.ArticleDomain = domainservice.NewArticleDomainService(r)
	c.ArticleApp = appservice.NewArticleAppService(c.ArticleDomain, nil, c.UserDomain, c.AdminDomain)
	c.ArticleManagementDomain = domainservice.NewArticleManagementDomainService(c.ArticleDomain, c.UserDomain, c.AdminDomain)
	c.ArticleManagementApp = appservice.NewArticleManagementAppService(c.ArticleManagementDomain)
	old := ioc.AppContainer
	ioc.AppContainer = c
	t.Cleanup(func() { ioc.AppContainer = old })
	return c, r
}

func managementRequest(t *testing.T, engine *gin.Engine, method, path, header, token, body string) (*httptest.ResponseRecorder, int) {
	t.Helper()
	req := httptest.NewRequest(method, "/api/blog/v1"+path, strings.NewReader(body))
	if header != "" {
		req.Header.Set(header, "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	var envelope struct{ Code int }
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid HTTP response %d: %s", w.Code, w.Body.String())
	}
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected HTTP status: %d %s", w.Code, w.Body.String())
	}
	return w, envelope.Code
}

func TestManagedArticleHTTPAuthAndOwnerBoundary(t *testing.T) {
	c, r := managementHTTPContainer(t)
	engine := publicTestEngine(c)
	userToken, err := jwt.GenerateUserToken(&dto.UserBaseDTO{ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	adminToken, err := jwt.GenerateAdminToken(&dto.AdminBaseDTO{ID: 9})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, header, token string
		want                int
	}{
		{"/article/getMyArticle?id=1", "", "", common.LoginNotLoginErr.Code},
		{"/article/getMyArticle?id=1", consts.AdminTokenKey, adminToken, common.LoginNotLoginErr.Code},
		{"/article/getMyArticle?id=1", consts.UserTokenKey, userToken, common.OK.Code},
		{"/article/getMyArticle?id=2", consts.UserTokenKey, userToken, common.ArticleNotFoundErr.Code},
		{"/article/getMyArticle?id=3", consts.UserTokenKey, userToken, common.ArticleNotFoundErr.Code},
		{"/article/getMyArticle?id=0", consts.UserTokenKey, userToken, common.ParamErr.Code},
		{"/article/getMyArticle?id=no", consts.UserTokenKey, userToken, common.ParamErr.Code},
		{"/article/getAdminArticle?id=2", consts.UserTokenKey, userToken, common.LoginNotLoginErr.Code},
		{"/article/getAdminArticle?id=2", consts.AdminTokenKey, userToken, common.AuthErr.Code},
		{"/article/getAdminArticle?id=2", consts.AdminTokenKey, adminToken, common.OK.Code},
	} {
		w, code := managementRequest(t, engine, http.MethodGet, tc.path, tc.header, tc.token, "")
		if code != tc.want {
			t.Fatalf("%s code=%d want=%d: %s", tc.path, code, tc.want, w.Body.String())
		}
		if code != common.OK.Code && strings.Contains(w.Body.String(), "private") {
			t.Fatal("private content leaked on failure")
		}
	}
	w, code := managementRequest(t, engine, http.MethodGet, "/article/listMyArticles?user_id=8&status=0", consts.UserTokenKey, userToken, "")
	var page struct{ Data dto.PageResult }
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if code != common.OK.Code || page.Data.Total != 1 || page.Data.PageNum != 1 || page.Data.PageSize != 10 || r.query.UserID == nil || *r.query.UserID != 7 || strings.Contains(w.Body.String(), "content") {
		t.Fatalf("HTTP owner scope or response mapping incorrect: %s", w.Body.String())
	}
	w, code = managementRequest(t, engine, http.MethodGet, "/article/listAdminArticles", consts.AdminTokenKey, adminToken, "")
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if code != common.OK.Code || page.Data.Total != 3 {
		t.Fatalf("admin list failed: %s", w.Body.String())
	}
	c.AdminRepo.(*managementHTTPAdmins).disabled = true
	_, code = managementRequest(t, engine, http.MethodGet, "/article/listAdminArticles", consts.AdminTokenKey, adminToken, "")
	if code != common.PermissionDeniedErr.Code {
		t.Fatalf("disabled admin accepted: %d", code)
	}
}
