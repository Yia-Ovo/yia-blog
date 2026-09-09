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
	for _, query := range []string{"PageNum=1&PageSize=10", "PageNum=1&PageSize=10&Status=0", "PageNum=1&PageSize=10&Status=2"} {
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
