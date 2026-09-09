package service

import (
	"context"
	"errors"
	"testing"

	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
)

type commentAppMysqlRepo struct {
	repo.CommentMysqlRepo
	comment *model.CommentModel
	created *model.CommentModel
}

func (f *commentAppMysqlRepo) CreateComment(_ context.Context, comment *model.CommentModel) error {
	f.created = comment
	return nil
}

func (f *commentAppMysqlRepo) GetCommentByID(_ context.Context, _ uint64) (*model.CommentModel, error) {
	return f.comment, nil
}

func (f *commentAppMysqlRepo) UpdateCommentByUser(_ context.Context, _ uint64, _ uint64, _ string, _ int) (int64, error) {
	return 1, nil
}

type commentAppArticleRepo struct {
	repo.ArticleMysqlRepo
	article *model.ArticleModel
}

func (f *commentAppArticleRepo) GetArticleById(context.Context, uint64) (*model.ArticleModel, error) {
	return f.article, nil
}

type commentAppRedisRepo struct {
	repo.CommentRedisRepo
	deletedArticleID uint64
	countReads       int
}

func (f *commentAppRedisRepo) GetArticleCommentCount(context.Context, uint64) (int, error) {
	f.countReads++
	return 99, nil
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

func TestCreateCommentRespectsPublicationAndSwitch(t *testing.T) {
	for _, tt := range []struct {
		name    string
		article *model.ArticleModel
		allowed bool
	}{
		{"enabled", &model.ArticleModel{ID: 11, Status: consts.ArticleStatusPublished, CommentStatus: consts.ArticleCommentsEnabled}, true},
		{"closed", &model.ArticleModel{ID: 11, Status: consts.ArticleStatusPublished, CommentStatus: consts.ArticleCommentsDisabled}, false},
		{"draft", &model.ArticleModel{ID: 11, Status: consts.ArticleStatusDraft, CommentStatus: consts.ArticleCommentsEnabled}, false},
		{"offline", &model.ArticleModel{ID: 11, Status: consts.ArticleStatusOffline, CommentStatus: consts.ArticleCommentsEnabled}, false},
		{"missing", nil, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, cache := &commentAppMysqlRepo{}, &commentAppRedisRepo{}
			app := NewCommentAppService(domainservice.NewCommentDomainService(r, &commentAppArticleRepo{article: tt.article}), cache)
			err := app.CreateComment(context.Background(), &CommentCreateCommand{UserID: 7, ArticleID: 11, Nickname: "reader", Content: "hello"})
			if tt.allowed {
				if err != nil || r.created == nil || r.created.Status != consts.CommentStatusPending || cache.deletedArticleID != 11 {
					t.Fatalf("enabled article rejected: created=%+v err=%v", r.created, err)
				}
			} else if err == nil || r.created != nil || cache.deletedArticleID != 0 {
				t.Fatalf("blocked article mutated: created=%+v cache=%+v err=%v", r.created, cache, err)
			}
		})
	}
}

func TestPublicCommentCountChecksArticleBeforeCache(t *testing.T) {
	for _, status := range []int{consts.ArticleStatusDraft, consts.ArticleStatusOffline, consts.ArticleStatusPublished} {
		cache := &commentAppRedisRepo{}
		domain := domainservice.NewCommentDomainService(nil, &commentAppArticleRepo{article: &model.ArticleModel{ID: 11, Status: status}})
		count, err := NewCommentAppService(domain, cache).GetArticleCommentCount(context.Background(), 11)
		if status == consts.ArticleStatusPublished {
			if err != nil || count != 99 || cache.countReads != 1 {
				t.Fatalf("published cache missed: count=%d err=%v", count, err)
			}
		} else if !errors.Is(err, common.ArticleNotFoundErr) || count != 0 || cache.countReads != 0 {
			t.Fatalf("private article leaked cache: count=%d reads=%d err=%v", count, cache.countReads, err)
		}
	}
}
