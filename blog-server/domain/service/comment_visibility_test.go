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

type commentVisibilityArticleRepo struct {
	repo.ArticleMysqlRepo
	article *model.ArticleModel
}

func (r *commentVisibilityArticleRepo) GetArticleById(context.Context, uint64) (*model.ArticleModel, error) {
	return r.article, nil
}

func TestPublicCommentsHideUnpublishedArticles(t *testing.T) {
	for _, status := range []int{consts.ArticleStatusDraft, consts.ArticleStatusOffline} {
		r := &fakeCommentMysqlRepo{comment: &model.CommentModel{ID: 1, ArticleID: 11, Status: consts.CommentStatusPassed, Content: "private"}}
		s := NewCommentDomainService(r, &commentVisibilityArticleRepo{article: &model.ArticleModel{ID: 11, Status: status}})
		if comment, err := s.GetPublicComment(context.Background(), 1); comment != nil || !errors.Is(err, common.CommentNotFoundErr) {
			t.Fatalf("private comment exposed: %+v err=%v", comment, err)
		}
		if _, _, err := s.ListPublicComments(context.Background(), &do.CommentQuery{ArticleID: 11}); !errors.Is(err, common.ArticleNotFoundErr) || r.listQuery != nil {
			t.Fatalf("private article list allowed: query=%+v err=%v", r.listQuery, err)
		}
		if comment, err := s.GetCommentByID(context.Background(), 1); err != nil || comment == nil {
			t.Fatalf("internal read broken: %v", err)
		}
	}
}

func TestPublicCommentsAllArticlesStillConstrainVisibility(t *testing.T) {
	r := &fakeCommentMysqlRepo{}
	s := NewCommentDomainService(r, nil)
	query := &do.CommentQuery{WithDeleted: true, PublishedArticleOnly: false}
	if _, _, err := s.ListPublicComments(context.Background(), query); err != nil {
		t.Fatal(err)
	}
	if r.listQuery == nil || !r.listQuery.PublishedArticleOnly || r.listQuery.WithDeleted || r.listQuery.Status == nil || *r.listQuery.Status != consts.CommentStatusPassed {
		t.Fatalf("cross-article scope not constrained: %+v", r.listQuery)
	}
}
