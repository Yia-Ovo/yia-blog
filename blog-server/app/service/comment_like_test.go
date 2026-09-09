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

type reactionRepo struct {
	repo.CommentMysqlRepo
	liked  bool
	userID uint64
	likes  int
}

func (r *reactionRepo) GetCommentByID(context.Context, uint64) (*model.CommentModel, error) {
	return &model.CommentModel{ID: 11, ArticleID: 21, Status: consts.CommentStatusPassed}, nil
}
func (r *reactionRepo) LikeComment(_ context.Context, userID, _ uint64) (bool, error) {
	r.userID = userID
	if r.liked {
		return false, nil
	}
	r.liked = true
	r.likes++
	return true, nil
}
func (r *reactionRepo) DisLikeComment(_ context.Context, userID, _ uint64) (bool, error) {
	r.userID = userID
	if !r.liked {
		return false, nil
	}
	r.liked = false
	r.likes--
	return true, nil
}

// 只提供失效接口，调用任何旧 Bitmap/增量缓存方法都会失败，防止重新依赖缓存判重。
type failedReactionCache struct {
	repo.CommentRedisRepo
	invalidations int
}

func (r *failedReactionCache) DelCommentLikeCount(context.Context, uint64) error {
	r.invalidations++
	return errors.New("redis unavailable")
}

func TestCommentLikesDoNotDependOnRedisAvailabilityOrExpiry(t *testing.T) {
	r, cache := &reactionRepo{}, &failedReactionCache{}
	domain := domainservice.NewCommentDomainService(r, &commentAppArticleRepo{article: &model.ArticleModel{ID: 21, Status: consts.ArticleStatusPublished}})
	app := NewCommentAppService(domain, cache)
	if err := app.LikeComment(context.Background(), 7, 11); err != nil {
		t.Fatal(err)
	}
	if err := app.LikeComment(context.Background(), 7, 11); !errors.Is(err, common.ErrCommentAlreadyLiked) || r.likes != 1 || r.userID != 7 {
		t.Fatalf("duplicate like changed count: err=%v likes=%d user=%d", err, r.likes, r.userID)
	}
	if err := app.DisLikeComment(context.Background(), 7, 11); err != nil {
		t.Fatal(err)
	}
	if err := app.DisLikeComment(context.Background(), 7, 11); !errors.Is(err, common.ErrCommentNotLiked) || r.likes != 0 {
		t.Fatalf("duplicate unlike changed count: err=%v likes=%d", err, r.likes)
	}
	if cache.invalidations != 4 {
		t.Fatalf("retries did not attempt cache repair: %d", cache.invalidations)
	}
}

func TestCommentLikeDomainRequiresIdentity(t *testing.T) {
	domain := domainservice.NewCommentDomainService(nil, nil)
	if err := domain.LikeComment(context.Background(), 0, 11); !errors.Is(err, common.ParamErr) {
		t.Fatalf("missing identity: %v", err)
	}
	if err := domain.DisLikeComment(context.Background(), 0, 11); !errors.Is(err, common.ParamErr) {
		t.Fatalf("missing identity: %v", err)
	}
}
