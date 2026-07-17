package handler

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-server/api/dto"
	appservice "blog-server/app/service"
	"blog-server/common/consts"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"

	"github.com/gin-gonic/gin"
)

type commentHandlerMysqlRepo struct {
	repo.CommentMysqlRepo
	created *model.CommentModel
}

func (f *commentHandlerMysqlRepo) CreateComment(_ context.Context, comment *model.CommentModel) error {
	copy := *comment
	f.created = &copy
	return nil
}

type commentHandlerArticleRepo struct {
	repo.ArticleMysqlRepo
}

func (f *commentHandlerArticleRepo) GetArticleById(_ context.Context, articleID uint64) (*model.ArticleModel, error) {
	return &model.ArticleModel{ID: articleID}, nil
}

type commentHandlerRedisRepo struct {
	repo.CommentRedisRepo
}

func (f *commentHandlerRedisRepo) DelArticleCommentCache(_ context.Context, _ uint64) error {
	return nil
}

func TestCreateCommentIgnoresServerManagedJSONFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mysqlRepo := &commentHandlerMysqlRepo{}
	domainService := domainservice.NewCommentDomainService(mysqlRepo, &commentHandlerArticleRepo{})
	appService := appservice.NewCommentAppService(domainService, &commentHandlerRedisRepo{})
	h := NewCommentHandler(appService)

	body := `{
		"article_id": 10,
		"content": "hello",
		"user_id": 999,
		"admin_id": 88,
		"status": 1,
		"likes": 100,
		"ip": "1.1.1.1"
	}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/comment/createComment", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(consts.CustomerUserKey, &dto.UserBaseDTO{ID: 7, NickName: "user", Email: "user@example.com"})

	h.CreateComment(c)

	if recorder.Code != 200 {
		t.Fatalf("HTTP 状态码错误: got=%d want=200", recorder.Code)
	}
	if mysqlRepo.created == nil {
		t.Fatal("期望创建评论")
	}
	if mysqlRepo.created.UserID != 7 || mysqlRepo.created.AdminID != 0 {
		t.Fatalf("作者字段被客户端覆盖: user_id=%d admin_id=%d", mysqlRepo.created.UserID, mysqlRepo.created.AdminID)
	}
	if mysqlRepo.created.Status != consts.CommentStatusPending || mysqlRepo.created.Likes != 0 {
		t.Fatalf("服务端字段被客户端覆盖: status=%d likes=%d", mysqlRepo.created.Status, mysqlRepo.created.Likes)
	}
	if mysqlRepo.created.IP == "1.1.1.1" {
		t.Fatal("客户端提交的 IP 不应写入评论")
	}
}
