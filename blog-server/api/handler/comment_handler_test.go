package handler

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-server/api/dto"
	appservice "blog-server/app/service"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"

	"github.com/gin-gonic/gin"
)

type commentHandlerMysqlRepo struct {
	repo.CommentMysqlRepo
	created *model.CommentModel

	// listResult 模拟库里已有的评论，用于验证公开响应的字段白名单
	listResult []*model.CommentModel
	listTotal  int64

	// comment 模拟按ID查到的评论，用于验证公开详情的可见性
	comment *model.CommentModel
}

func (f *commentHandlerMysqlRepo) GetCommentByID(_ context.Context, _ uint64) (*model.CommentModel, error) {
	return f.comment, nil
}

func (f *commentHandlerMysqlRepo) CreateComment(_ context.Context, comment *model.CommentModel) error {
	copy := *comment
	f.created = &copy
	return nil
}

func (f *commentHandlerMysqlRepo) ListCommentByPage(
	_ context.Context,
	_ *do.CommentQuery,
) ([]*model.CommentModel, int64, error) {
	return f.listResult, f.listTotal, nil
}

type commentHandlerArticleRepo struct {
	repo.ArticleMysqlRepo
}

func (f *commentHandlerArticleRepo) GetArticleById(_ context.Context, articleID uint64) (*model.ArticleModel, error) {
	return &model.ArticleModel{ID: articleID, Status: consts.ArticleStatusPublished, CommentStatus: consts.ArticleCommentsEnabled}, nil
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

// TestListCommentByPageHidesPrivateFields 公开评论列表是输出边界：
// 响应体只能包含展示字段，评论者邮箱、IP、User-Agent、审核状态和作者ID 都不得出现。
// 与 TestCreateCommentIgnoresServerManagedJSONFields（输入边界）互为镜像。
func TestListCommentByPageHidesPrivateFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mysqlRepo := &commentHandlerMysqlRepo{
		listResult: []*model.CommentModel{
			{
				ID:        1,
				UserID:    7,
				AdminID:   88,
				ArticleID: 10,
				Nickname:  "user",
				Email:     "victim@example.com",
				Avatar:    "avatar.png",
				Content:   "hello",
				IP:        "1.2.3.4",
				UserAgent: "secret-agent",
				Status:    consts.CommentStatusPassed,
				Likes:     3,
			},
		},
		listTotal: 1,
	}
	domainService := domainservice.NewCommentDomainService(mysqlRepo, &commentHandlerArticleRepo{})
	appService := appservice.NewCommentAppService(domainService, &commentHandlerRedisRepo{})
	h := NewCommentHandler(appService)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/comment/listComment?articleID=10&status=0", nil)

	h.ListCommentByPage(c)

	if recorder.Code != 200 {
		t.Fatalf("HTTP 状态码错误: got=%d want=200", recorder.Code)
	}

	body := recorder.Body.String()
	// CommentDO 没有 json tag，直接序列化会输出大写驼峰字段名；
	// VO 用小写下划线。两种命名都要断言，才能挡住"改回返回 DO"的回归。
	for _, leaked := range []string{
		"victim@example.com", // 邮箱
		"1.2.3.4",            // IP
		"secret-agent",       // User-Agent
		`"email"`, `"Email"`,
		`"ip"`, `"IP"`,
		`"user_agent"`, `"UserAgent"`,
		`"status"`, `"Status"`,
		`"user_id"`, `"UserID"`,
		`"admin_id"`, `"AdminID"`,
		`"deleted_at"`, `"DeletedAt"`,
	} {
		if strings.Contains(body, leaked) {
			t.Fatalf("公开评论响应泄露了 %s: %s", leaked, body)
		}
	}

	// 展示字段仍需正常返回
	for _, want := range []string{`"nickname"`, `"content"`, `"likes"`, "hello"} {
		if !strings.Contains(body, want) {
			t.Fatalf("公开评论响应缺少展示字段 %s: %s", want, body)
		}
	}
}

// TestGetCommentByIDHidesUnapprovedComment 未通过审核的评论不能通过公开详情接口读取。
func TestGetCommentByIDHidesUnapprovedComment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mysqlRepo := &commentHandlerMysqlRepo{
		comment: &model.CommentModel{
			ID:        5,
			ArticleID: 10,
			Nickname:  "user",
			Email:     "pending@example.com",
			Content:   "waiting for audit",
			IP:        "9.9.9.9",
			Status:    consts.CommentStatusPending,
		},
	}
	domainService := domainservice.NewCommentDomainService(mysqlRepo, &commentHandlerArticleRepo{})
	appService := appservice.NewCommentAppService(domainService, &commentHandlerRedisRepo{})
	h := NewCommentHandler(appService)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/comment/getCommentById?id=5", nil)

	h.GetCommentByID(c)

	body := recorder.Body.String()
	if strings.Contains(body, "pending@example.com") || strings.Contains(body, "9.9.9.9") {
		t.Fatalf("未通过审核的评论泄露到公开详情接口: %s", body)
	}
}
