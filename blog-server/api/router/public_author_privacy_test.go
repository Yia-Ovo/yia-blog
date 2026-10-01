package router_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-server/api/dto"
	appservice "blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
	"blog-server/ioc"
)

type publicAuthorAdminRepo struct {
	repo.AdminMysqlRepo
	admin *model.AdminModel
}

func (r *publicAuthorAdminRepo) FindAdminByID(context.Context, uint64) (*model.AdminModel, error) {
	return r.admin, nil
}

type publicAuthorUserRepo struct {
	repo.UserMysqlRepo
	user *model.UserModel
}

func (r *publicAuthorUserRepo) FindUserByID(context.Context, uint64) (*model.UserModel, error) {
	return r.user, nil
}

// TestPublicAuthorPrivacy 通过匿名 HTTP 同时验证公开详情与列表的输出边界。
func TestPublicAuthorPrivacy(t *testing.T) {
	const internalRemark = "internal-admin-note-must-not-be-public"
	adminRepo := &publicAuthorAdminRepo{admin: &model.AdminModel{
		ID: 3, AdminName: "站长", Avatar: "/uploads/admin.png", Remark: internalRemark,
	}}
	userRepo := &publicAuthorUserRepo{user: &model.UserModel{
		ID: 5, NickName: "作者", Avatar: "/uploads/user.png", Signature: "公开签名",
	}}
	wantAdmin := dto.AuthorInfo{Type: consts.AuthorTypeAdmin, Name: "站长", Avatar: "/uploads/admin.png"}
	wantUser := dto.AuthorInfo{Type: consts.AuthorTypeUser, Name: "作者", Avatar: "/uploads/user.png", Signature: "公开签名"}
	r := &publicArticleRepo{articles: []*model.ArticleModel{
		{ID: 31, AdminID: 3, Status: consts.ArticleStatusPublished, Title: "管理员文章"},
		{ID: 32, UserID: 5, Status: consts.ArticleStatusPublished, Title: "用户文章"},
	}}
	app := appservice.NewArticleAppService(domainservice.NewArticleDomainService(r), nil,
		domainservice.NewUserDomainService(userRepo), domainservice.NewAdminDomainService(adminRepo))
	engine := publicTestEngine(&ioc.Container{ArticleApp: app})
	for _, tt := range []struct {
		path string
		list bool
		want []dto.AuthorInfo
	}{
		{"/api/blog/v1/article/getArticleById?id=31", false, []dto.AuthorInfo{wantAdmin}},
		{"/api/blog/v1/article/getArticleById?id=32", false, []dto.AuthorInfo{wantUser}},
		{"/api/blog/v1/article/listArticle?page_num=1&page_size=10", true, []dto.AuthorInfo{wantAdmin, wantUser}},
	} {
		t.Run(tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
			var body struct {
				Code int
				Data json.RawMessage
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != http.StatusOK || body.Code != common.OK.Code {
				t.Fatalf("匿名读取失败: HTTP %d %s", w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), internalRemark) {
				t.Fatal("匿名响应暴露管理员内部备注")
			}
			type article struct {
				Author *dto.AuthorInfo `json:"author"`
			}
			var articles []article
			if tt.list {
				var page struct{ List []article }
				if err := json.Unmarshal(body.Data, &page); err != nil {
					t.Fatal(err)
				}
				articles = page.List
			} else {
				var detail article
				if err := json.Unmarshal(body.Data, &detail); err != nil {
					t.Fatal(err)
				}
				articles = []article{detail}
			}
			if len(articles) != len(tt.want) {
				t.Fatalf("文章数量错误: %s", w.Body.String())
			}
			for i, a := range articles {
				if a.Author == nil || *a.Author != tt.want[i] {
					t.Fatalf("公开作者字段错误: got=%+v want=%+v", a.Author, tt.want[i])
				}
			}
		})
	}
	if adminRepo.admin.Remark != internalRemark {
		t.Fatal("读取公开作者不应修改后台备注")
	}
}
