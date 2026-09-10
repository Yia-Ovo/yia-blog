package service

import (
	"context"
	"errors"
	"testing"

	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
)

type contentTestUsers struct {
	repo.UserMysqlRepo
	user *model.UserModel
}

func (r *contentTestUsers) FindUserByID(context.Context, uint64) (*model.UserModel, error) {
	return r.user, nil
}

type contentTestAdmins struct {
	repo.AdminMysqlRepo
	admin *model.AdminModel
	err   error
}

func (r *contentTestAdmins) FindAdminByID(context.Context, uint64) (*model.AdminModel, error) {
	return r.admin, r.err
}

type managedArticleTestRepo struct {
	repo.ArticleMysqlRepo
	article *model.ArticleModel
	query   *do.ArticleQuery
	calls   int
}

func (r *managedArticleTestRepo) GetArticleById(context.Context, uint64) (*model.ArticleModel, error) {
	r.calls++
	return r.article, nil
}

func (r *managedArticleTestRepo) ListArticleByPage(_ context.Context, q *do.ArticleQuery) ([]*model.ArticleModel, int64, error) {
	r.calls++
	copy := *q
	r.query = &copy
	return nil, 0, nil
}

func TestManagedArticleOwnerScopeAndVisibility(t *testing.T) {
	ctx := context.Background()
	r := &managedArticleTestRepo{article: &model.ArticleModel{ID: 3, UserID: 7, Status: consts.ArticleStatusDraft, Content: "draft"}}
	users := &contentTestUsers{user: &model.UserModel{ID: 7, Status: consts.UserStatusEnabled}}
	s := NewArticleManagementDomainService(NewArticleDomainService(r), NewUserDomainService(users), nil)
	other := uint64(8)
	q := &do.ArticleQuery{UserID: &other}
	if _, _, err := s.ListMyArticles(ctx, 7, q); err != nil || r.query.UserID == nil || *r.query.UserID != 7 || r.query.Status != nil || q.PageNum != 1 || q.PageSize != 10 {
		t.Fatalf("owner scope/defaults lost: query=%+v err=%v", r.query, err)
	}
	for _, state := range []int{consts.ArticleStatusDraft, consts.ArticleStatusPublished, consts.ArticleStatusOffline} {
		r.article.Status = state
		if article, err := s.GetMyArticle(ctx, 7, 3); err != nil || article == nil || article.Content != "draft" {
			t.Fatalf("owner cannot read state %d: %+v %v", state, article, err)
		}
	}
	for _, article := range []*model.ArticleModel{{ID: 3, UserID: 8}, {ID: 3, AdminID: 7}, {ID: 3, UserID: 7, AdminID: 9}, nil} {
		r.article = article
		if result, err := s.GetMyArticle(ctx, 7, 3); result != nil || !errors.Is(err, common.ArticleNotFoundErr) {
			t.Fatalf("foreign/missing article leaked: %+v %v", result, err)
		}
	}
	users.user.Status = consts.UserStatusDisabled
	before := r.calls
	if _, err := s.GetMyArticle(ctx, 7, 3); !errors.Is(err, common.UserAccountDisabledErr) || r.calls != before {
		t.Fatalf("disabled user reached article storage: %v", err)
	}
	if _, _, err := s.ListMyArticles(ctx, 0, &do.ArticleQuery{}); !errors.Is(err, common.LoginNotLoginErr) {
		t.Fatalf("anonymous domain call returned %v", err)
	}
}

func TestManagedArticleChecksAdminInDomain(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       uint64
		admin    *model.AdminModel
		queryErr error
		want     error
	}{
		{"normal", 9, &model.AdminModel{ID: 9, Status: 1, Role: 1}, nil, nil},
		{"super", 9, &model.AdminModel{ID: 9, Status: 1, Role: 0}, nil, nil},
		{"anonymous", 0, nil, nil, common.LoginNotLoginErr},
		{"missing", 9, nil, nil, common.AdminNotFoundErr},
		{"disabled", 9, &model.AdminModel{ID: 9, Status: 0, Role: 1}, nil, common.AdminAccountDisabledErr},
		{"invalid role", 9, &model.AdminModel{ID: 9, Status: 1, Role: 3}, nil, common.PermissionDeniedErr},
		{"database failure", 9, nil, errors.New("offline"), common.AdminQueryErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &managedArticleTestRepo{article: &model.ArticleModel{ID: 3, Status: consts.ArticleStatusDraft}}
			s := NewArticleManagementDomainService(NewArticleDomainService(r), nil, NewAdminDomainService(&contentTestAdmins{admin: tc.admin, err: tc.queryErr}))
			_, err := s.GetAdminArticle(context.Background(), tc.id, 3)
			if !contentErrorMatches(err, tc.want) || (tc.want != nil && r.calls != 0) {
				t.Fatalf("detail err=%v calls=%d want=%v", err, r.calls, tc.want)
			}
			before := r.calls
			_, _, err = s.ListAdminArticles(context.Background(), tc.id, &do.ArticleQuery{})
			if !contentErrorMatches(err, tc.want) || (tc.want != nil && r.calls != before) {
				t.Fatalf("list err=%v calls=%d want=%v", err, r.calls, tc.want)
			}
		})
	}
	s := NewArticleManagementDomainService(nil, nil, nil)
	badStatus := 99
	for _, q := range []*do.ArticleQuery{nil, {Status: &badStatus}, {IsTop: &badStatus}} {
		if _, _, err := s.ListMyArticles(context.Background(), 7, q); !errors.Is(err, common.ParamErr) {
			t.Fatalf("invalid query reached dependencies: %v", err)
		}
	}
}

// HTTP 契约以业务码区分错误；WithErr 附带内部详情后，Errno 的结构值不再相等。
func contentErrorMatches(err, want error) bool {
	if err == nil || want == nil {
		return err == nil && want == nil
	}
	var gotCode, wantCode common.Errno
	return errors.As(err, &gotCode) && errors.As(want, &wantCode) && gotCode.Code == wantCode.Code
}
