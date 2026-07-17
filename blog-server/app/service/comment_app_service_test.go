package service

import (
	"context"
	"testing"

	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
)

type commentAppMysqlRepo struct {
	repo.CommentMysqlRepo
	comment *model.CommentModel
}

func (f *commentAppMysqlRepo) GetCommentByID(_ context.Context, _ uint64) (*model.CommentModel, error) {
	return f.comment, nil
}

func (f *commentAppMysqlRepo) UpdateCommentByUser(_ context.Context, _ uint64, _ uint64, _ string, _ int) (int64, error) {
	return 1, nil
}

type commentAppArticleRepo struct {
	repo.ArticleMysqlRepo
}

type commentAppRedisRepo struct {
	repo.CommentRedisRepo
	deletedArticleID uint64
}

func (f *commentAppRedisRepo) DelArticleCommentCache(_ context.Context, articleID uint64) error {
	f.deletedArticleID = articleID
	return nil
}

func TestUpdateCommentClearsDatabaseArticleCache(t *testing.T) {
	mysqlRepo := &commentAppMysqlRepo{
		comment: &model.CommentModel{ID: 3, UserID: 7, ArticleID: 42, Content: "old"},
	}
	redisRepo := &commentAppRedisRepo{}
	domainService := domainservice.NewCommentDomainService(mysqlRepo, &commentAppArticleRepo{})
	appService := NewCommentAppService(domainService, redisRepo)

	if err := appService.UpdateComment(context.Background(), 7, 3, "new"); err != nil {
		t.Fatalf("更新评论失败: %v", err)
	}
	if redisRepo.deletedArticleID != 42 {
		t.Fatalf("应清理数据库真实文章ID对应缓存，got=%d want=42", redisRepo.deletedArticleID)
	}
}
