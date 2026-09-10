package router_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"blog-server/api/dto"
	appservice "blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/jwt"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
)

type moderationHTTPRepo struct {
	repo.CommentMysqlRepo
	comment *model.CommentModel
	query   *do.CommentQuery
}

func (r *moderationHTTPRepo) GetCommentByID(context.Context, uint64) (*model.CommentModel, error) {
	copy := *r.comment
	return &copy, nil
}
func (r *moderationHTTPRepo) ListCommentByPage(_ context.Context, q *do.CommentQuery) ([]*model.CommentModel, int64, error) {
	copy := *q
	r.query = &copy
	if q.Status != nil && r.comment.Status != *q.Status {
		return nil, 0, nil
	}
	return []*model.CommentModel{r.comment}, 1, nil
}
func (r *moderationHTTPRepo) ReviewComment(_ context.Context, q *do.CommentReview) (int64, error) {
	if r.comment.Content != q.ExpectedContent || r.comment.Status != q.ExpectedStatus {
		return 0, nil
	}
	r.comment.Status = q.Status
	return 1, nil
}
func (r *moderationHTTPRepo) UpdateCommentByUser(_ context.Context, id, userID uint64, content string, status int) (int64, error) {
	if r.comment.ID != id || r.comment.UserID != userID {
		return 0, nil
	}
	r.comment.Content, r.comment.Status = content, status
	return 1, nil
}

type moderationHTTPCache struct {
	repo.CommentRedisRepo
	cached  *int
	deleted int
	err     error
}

func (r *moderationHTTPCache) DelArticleCommentCache(context.Context, uint64) error {
	r.deleted++
	if r.err == nil {
		r.cached = nil
	}
	return r.err
}
func (r *moderationHTTPCache) GetArticleCommentCount(context.Context, uint64) (int, error) {
	if r.cached == nil {
		return 0, repo.ErrCacheMiss
	}
	return *r.cached, nil
}
func (r *moderationHTTPCache) SetArticleCommentCount(_ context.Context, _ uint64, count int) error {
	r.cached = &count
	return nil
}

func TestCommentModerationHTTPWorkflowAndPrivacy(t *testing.T) {
	c, articles := managementHTTPContainer(t)
	articles.articles[0].Status = consts.ArticleStatusPublished
	r := &moderationHTTPRepo{comment: &model.CommentModel{ID: 3, ArticleID: 1, UserID: 7, Content: "original", Status: 0, Email: "private-email-marker", IP: "private-ip-marker", UserAgent: "private-agent-marker"}}
	cache := &moderationHTTPCache{}
	c.CommentRepo = r
	c.CommentRedis = cache
	c.CommentDomain = domainservice.NewCommentDomainService(r, articles)
	c.CommentApp = appservice.NewCommentAppService(c.CommentDomain, cache)
	c.CommentModerationDomain = domainservice.NewCommentModerationDomainService(c.CommentDomain, c.AdminDomain, r)
	c.CommentModerationApp = appservice.NewCommentModerationAppService(c.CommentModerationDomain, cache)
	engine := publicTestEngine(c)
	adminToken, err := jwt.GenerateAdminToken(&dto.AdminBaseDTO{ID: 9})
	if err != nil {
		t.Fatal(err)
	}
	userToken, err := jwt.GenerateUserToken(&dto.UserBaseDTO{ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	for _, header := range []string{"", consts.UserTokenKey} {
		_, code := managementRequest(t, engine, http.MethodGet, "/comment/listAdminComments?status=0", header, userToken, "")
		if code != common.LoginNotLoginErr.Code {
			t.Fatalf("user/anonymous can query moderation: %d", code)
		}
		_, code = managementRequest(t, engine, http.MethodPost, "/comment/reviewComment", header, userToken, `{"id":3,"expected_content":"original","expected_status":0,"status":1}`)
		if code != common.LoginNotLoginErr.Code {
			t.Fatalf("user/anonymous can review: %d", code)
		}
	}
	w, code := managementRequest(t, engine, http.MethodGet, "/comment/listAdminComments?status=0", consts.AdminTokenKey, adminToken, "")
	if code != common.OK.Code || r.query.Status == nil || *r.query.Status != 0 || !strings.Contains(w.Body.String(), "original") {
		t.Fatalf("pending comments unavailable: %s", w.Body.String())
	}
	for _, secret := range []string{"private-email-marker", "private-ip-marker", "private-agent-marker"} {
		if strings.Contains(w.Body.String(), secret) {
			t.Fatalf("private field leaked: %s", secret)
		}
	}
	for _, payload := range []string{`{"id":3,"status":1}`, `{"id":3,"expected_content":"original","expected_status":0,"status":0}`} {
		_, code = managementRequest(t, engine, http.MethodPost, "/comment/reviewComment", consts.AdminTokenKey, adminToken, payload)
		if code != common.ParamErr.Code {
			t.Fatalf("invalid review payload accepted: %d", code)
		}
	}
	assertCount := func(want int) {
		t.Helper()
		w, code := managementRequest(t, engine, http.MethodGet, "/comment/getArticleCommentCount?article_id=1", "", "", "")
		var body struct{ Data struct{ Count int } }
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if code != common.OK.Code || body.Data.Count != want {
			t.Fatalf("public count=%d want=%d: %s", body.Data.Count, want, w.Body.String())
		}
	}
	assertCount(0)
	approve := `{"id":3,"expected_content":"original","expected_status":0,"status":1}`
	for attempt := 0; attempt < 2; attempt++ {
		_, code = managementRequest(t, engine, http.MethodPost, "/comment/reviewComment", consts.AdminTokenKey, adminToken, approve)
		if code != common.OK.Code {
			t.Fatalf("approval/retry failed: %d", code)
		}
	}
	assertCount(1)
	_, code = managementRequest(t, engine, http.MethodPost, "/comment/updateComment", consts.UserTokenKey, userToken, `{"id":3,"content":"edited"}`)
	if code != common.OK.Code || r.comment.Status != consts.CommentStatusPending {
		t.Fatalf("edit did not require review again: code=%d comment=%+v", code, r.comment)
	}
	assertCount(0)
	before := cache.deleted
	_, code = managementRequest(t, engine, http.MethodPost, "/comment/reviewComment", consts.AdminTokenKey, adminToken, approve)
	if code != common.ParamErr.Code || r.comment.Status != 0 || cache.deleted != before {
		t.Fatalf("stale review changed data/cache: code=%d", code)
	}
	cache.err = errors.New("redis unavailable")
	_, code = managementRequest(t, engine, http.MethodPost, "/comment/reviewComment", consts.AdminTokenKey, adminToken, `{"id":3,"expected_content":"edited","expected_status":0,"status":1}`)
	if code != common.OK.Code || r.comment.Status != 1 || cache.deleted != before+1 {
		t.Fatalf("cache failure hid committed review: code=%d", code)
	}
	cache.err = nil
	_, code = managementRequest(t, engine, http.MethodPost, "/comment/reviewComment", consts.AdminTokenKey, adminToken, `{"id":3,"expected_content":"edited","expected_status":1,"status":2}`)
	if code != common.OK.Code {
		t.Fatalf("withdrawal failed: %d", code)
	}
	assertCount(0)
	_, code = managementRequest(t, engine, http.MethodGet, "/comment/getCommentById?id=3", "", "", "")
	if code != common.CommentNotFoundErr.Code {
		t.Fatalf("rejected comment remains public: %d", code)
	}
}
