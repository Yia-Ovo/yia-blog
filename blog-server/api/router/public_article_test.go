package router_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-server/api/router"
	appservice "blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/config"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
	"blog-server/ioc"

	"github.com/gin-gonic/gin"
)

type publicArticleRepo struct {
	repo.ArticleMysqlRepo
	articles []*model.ArticleModel
	query    *do.ArticleQuery
}

func (r *publicArticleRepo) GetArticleById(_ context.Context, id uint64) (*model.ArticleModel, error) {
	for _, article := range r.articles {
		if article.ID == id {
			copied := *article
			return &copied, nil
		}
	}
	return nil, nil
}

func (r *publicArticleRepo) ListArticleByPage(_ context.Context, query *do.ArticleQuery) ([]*model.ArticleModel, int64, error) {
	copied := *query
	r.query = &copied
	var list []*model.ArticleModel
	for _, article := range r.articles {
		if query.Status == nil || article.Status == *query.Status {
			list = append(list, article)
		}
	}
	total := int64(len(list))
	offset := (query.PageNum - 1) * query.PageSize
	if offset >= len(list) {
		return nil, total, nil
	}
	end := offset + query.PageSize
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end], total, nil
}

func publicTestEngine(container *ioc.Container) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.RecoveryWithWriter(io.Discard))
	router.NewRouter(&config.GlobalConfig{}, container, func() error { return nil }).Register(engine)
	return engine
}

func articleTestApp(r repo.ArticleMysqlRepo) *appservice.ArticleAppService {
	return appservice.NewArticleAppService(domainservice.NewArticleDomainService(r), nil, nil, nil)
}

func TestPublicArticleDetailVisibility(t *testing.T) {
	for _, status := range []int{consts.ArticleStatusDraft, consts.ArticleStatusPublished, consts.ArticleStatusOffline, 99} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			r := &publicArticleRepo{articles: []*model.ArticleModel{{ID: 31, Status: status, Content: "private-body-marker"}}}
			engine := publicTestEngine(&ioc.Container{ArticleApp: articleTestApp(r)})
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/blog/v1/article/getArticleById?id=31", nil))
			var body struct{ Code int }
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			wantCode := common.ArticleNotFoundErr.Code
			if status == consts.ArticleStatusPublished {
				wantCode = common.OK.Code
			}
			if w.Code != http.StatusOK || body.Code != wantCode {
				t.Fatalf("unexpected response: HTTP %d %s", w.Code, w.Body.String())
			}
			if status != consts.ArticleStatusPublished && strings.Contains(w.Body.String(), "private-body-marker") {
				t.Fatal("unpublished body exposed")
			}
			// 内部读取仍需支持作者编辑草稿，不能把公开规则误加到所有查询。
			article, err := domainservice.NewArticleDomainService(r).GetArticleById(context.Background(), 31)
			if err != nil || article == nil || article.Status != status {
				t.Fatalf("internal read changed: article=%+v err=%v", article, err)
			}
		})
	}
}

func TestPublicArticleListForcesPublished(t *testing.T) {
	for _, query := range []string{"page_num=1&page_size=10", "page_num=1&page_size=10&status=0", "page_num=1&page_size=10&status=2"} {
		t.Run(query, func(t *testing.T) {
			r := &publicArticleRepo{articles: []*model.ArticleModel{
				{ID: 1, Status: consts.ArticleStatusDraft, Summary: "private-summary-marker"},
				{ID: 2, Status: consts.ArticleStatusPublished, Title: "public"},
				{ID: 3, Status: consts.ArticleStatusOffline, Summary: "private-summary-marker"},
			}}
			engine := publicTestEngine(&ioc.Container{ArticleApp: articleTestApp(r)})
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/blog/v1/article/listArticle?"+query, nil))
			var body struct {
				Code int
				Data struct {
					Total int64
					List  []map[string]interface{}
				}
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != http.StatusOK || body.Code != common.OK.Code || body.Data.Total != 1 || len(body.Data.List) != 1 {
				t.Fatalf("unpublished rows affect list/count: HTTP %d %s", w.Code, w.Body.String())
			}
			if r.query.Status == nil || *r.query.Status != consts.ArticleStatusPublished || strings.Contains(w.Body.String(), "private-summary-marker") {
				t.Fatalf("public scope bypassed: query=%+v response=%s", r.query, w.Body.String())
			}
		})
	}
}

func TestArticlePaginationUsesEffectiveQuery(t *testing.T) {
	for _, tt := range []struct {
		query                    string
		total                    int
		page, size, pages, count int
	}{
		{"", 0, 1, 10, 0, 0},
		{"page_num=0&page_size=0", 11, 1, 10, 2, 10},
		{"page_num=-1&page_size=-5", 11, 1, 10, 2, 10},
		{"page_num=1&page_size=101", 11, 1, 10, 2, 10},
		{"page_num=2&page_size=5", 11, 2, 5, 3, 5},
		{"page_num=2&page_size=10", 11, 2, 10, 2, 1},
		{"page_num=3&page_size=10", 11, 3, 10, 2, 0},
	} {
		t.Run(tt.query, func(t *testing.T) {
			r := &publicArticleRepo{}
			for i := 0; i < tt.total; i++ {
				r.articles = append(r.articles, &model.ArticleModel{ID: uint64(i + 1), Status: consts.ArticleStatusPublished})
			}
			w := httptest.NewRecorder()
			publicTestEngine(&ioc.Container{ArticleApp: articleTestApp(r)}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/blog/v1/article/listArticle?"+tt.query, nil))
			var body struct {
				Code int
				Data struct {
					Total, PageNum, PageSize, Pages int
					List                            []map[string]interface{}
				}
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid response HTTP %d: %s", w.Code, w.Body.String())
			}
			if w.Code != http.StatusOK || body.Code != common.OK.Code || body.Data.Total != tt.total || body.Data.PageNum != tt.page || body.Data.PageSize != tt.size || body.Data.Pages != tt.pages || len(body.Data.List) != tt.count || body.Data.List == nil {
				t.Fatalf("pagination mismatch: HTTP %d %s", w.Code, w.Body.String())
			}
			if r.query.PageNum != body.Data.PageNum || r.query.PageSize != body.Data.PageSize {
				t.Fatalf("query/response mismatch: query=%+v response=%+v", r.query, body.Data)
			}
		})
	}
}

func TestArticleQueryBindsSnakeCaseFilters(t *testing.T) {
	r := &publicArticleRepo{}
	w := httptest.NewRecorder()
	publicTestEngine(&ioc.Container{ArticleApp: articleTestApp(r)}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/blog/v1/article/listArticle?page_num=2&page_size=5&title=needle&category_id=9&is_top=0&status=0", nil))
	q := r.query
	if q == nil || q.PageNum != 2 || q.PageSize != 5 || q.Title != "needle" || q.CategoryID == nil || *q.CategoryID != 9 || q.IsTop == nil || *q.IsTop != 0 || q.Status == nil || *q.Status != consts.ArticleStatusPublished {
		t.Fatalf("filters lost or visibility bypassed: query=%+v response=%s", q, w.Body.String())
	}
	for _, bad := range []string{"page_num=no", "page_size=no", "category_id=no", "is_top=no", "status=no"} {
		r := &publicArticleRepo{}
		w := httptest.NewRecorder()
		publicTestEngine(&ioc.Container{ArticleApp: articleTestApp(r)}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/blog/v1/article/listArticle?"+bad, nil))
		var body struct{ Code int }
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Code != common.ParamErr.Code || r.query != nil {
			t.Fatalf("invalid query reached repo: %s response=%s", bad, w.Body.String())
		}
	}
}
