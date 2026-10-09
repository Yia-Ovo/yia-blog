package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
)

type commentCountMysqlRepo struct {
	repo.CommentMysqlRepo
	total   int64
	err     error
	queries []do.CommentQuery
}

func (r *commentCountMysqlRepo) ListCommentByPage(_ context.Context, query *do.CommentQuery) ([]*model.CommentModel, int64, error) {
	snapshot := *query
	if query.Status != nil {
		status := *query.Status
		snapshot.Status = &status
	}
	r.queries = append(r.queries, snapshot)
	return nil, r.total, r.err
}

type commentCountArticleRepo struct {
	repo.ArticleMysqlRepo
	article *model.ArticleModel
	err     error
	reads   int
}

func (r *commentCountArticleRepo) GetArticleById(context.Context, uint64) (*model.ArticleModel, error) {
	r.reads++
	return r.article, r.err
}

type commentCountRedisRepo struct {
	repo.CommentRedisRepo
	count        int
	readErr      error
	writeErr     error
	reads        int
	readID       uint64
	writes       int
	writeID      uint64
	writtenCount int
}

func (r *commentCountRedisRepo) GetArticleCommentCount(_ context.Context, articleID uint64) (int, error) {
	r.reads++
	r.readID = articleID
	return r.count, r.readErr
}

func (r *commentCountRedisRepo) SetArticleCommentCount(_ context.Context, articleID uint64, count int) error {
	r.writes++
	r.writeID = articleID
	r.writtenCount = count
	return r.writeErr
}

func TestCommentCountCacheHit(t *testing.T) {
	for _, cachedCount := range []int{0, 17} {
		t.Run(fmt.Sprintf("count_%d", cachedCount), func(t *testing.T) {
			source := &commentCountMysqlRepo{total: 99}
			articles := &commentCountArticleRepo{article: &model.ArticleModel{ID: 42, Status: consts.ArticleStatusPublished}}
			cache := &commentCountRedisRepo{count: cachedCount}
			app := NewCommentAppService(domainservice.NewCommentDomainService(source, articles), cache)

			count, err := app.GetArticleCommentCount(context.Background(), 42)
			if err != nil || count != cachedCount {
				t.Fatalf("cache hit = (%d, %v), want (%d, nil)", count, err, cachedCount)
			}
			if articles.reads == 0 || cache.reads != 1 || cache.readID != 42 || len(source.queries) != 0 || cache.writes != 0 {
				t.Fatalf("cache hit skipped visibility or queried/wrote a count: articleReads=%d cache=%+v queries=%+v", articles.reads, cache, source.queries)
			}
		})
	}
}

func TestCommentCountCacheFallback(t *testing.T) {
	readErr := errors.New("test Redis read failure")
	corruptErr := errors.New("test invalid cached count")
	writeErr := errors.New("test Redis write failure")
	databaseErr := errors.New("test MySQL failure")
	for _, tc := range []struct {
		name        string
		readErr     error
		writeErr    error
		databaseErr error
		total       int64
		wantCount   int
		wantErr     error
	}{
		{name: "miss", readErr: repo.ErrCacheMiss, total: 7, wantCount: 7},
		{name: "wrapped miss", readErr: fmt.Errorf("wrapped: %w", repo.ErrCacheMiss), total: 7, wantCount: 7},
		{name: "read failure", readErr: readErr, total: 7, wantCount: 7},
		{name: "corrupt cache", readErr: corruptErr, total: 7, wantCount: 7},
		{name: "database zero is cached", readErr: repo.ErrCacheMiss, total: 0, wantCount: 0},
		{name: "write failure preserves result", readErr: repo.ErrCacheMiss, writeErr: writeErr, total: 7, wantCount: 7},
		{name: "read and write failures preserve result", readErr: readErr, writeErr: writeErr, total: 7, wantCount: 7},
		{name: "database failure after miss", readErr: repo.ErrCacheMiss, databaseErr: databaseErr, wantErr: common.DatabaseErr},
		{name: "cache and database failures", readErr: readErr, databaseErr: databaseErr, wantErr: common.DatabaseErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &commentCountMysqlRepo{total: tc.total, err: tc.databaseErr}
			articles := &commentCountArticleRepo{article: &model.ArticleModel{ID: 42, Status: consts.ArticleStatusPublished}}
			// 错误伴随的值不能成为有效结果，回源结果才是本次公开计数。
			cache := &commentCountRedisRepo{count: 999, readErr: tc.readErr, writeErr: tc.writeErr}
			app := NewCommentAppService(domainservice.NewCommentDomainService(source, articles), cache)

			count, err := app.GetArticleCommentCount(context.Background(), 42)
			if count != tc.wantCount || !errors.Is(err, tc.wantErr) {
				t.Fatalf("fallback = (%d, %v), want (%d, %v)", count, err, tc.wantCount, tc.wantErr)
			}
			if cache.reads != 1 || cache.readID != 42 || len(source.queries) != 1 {
				t.Fatalf("fallback did not query once for the requested article: cache=%+v queries=%+v", cache, source.queries)
			}
			query := source.queries[0]
			if query.ArticleID != 42 || query.Status == nil || *query.Status != consts.CommentStatusPassed || query.WithDeleted || !query.PublishedArticleOnly || query.Page != 1 || query.Size != 1 {
				t.Fatalf("fallback lost the public approved-comment scope: %+v", query)
			}
			if tc.wantErr != nil {
				if cache.writes != 0 {
					t.Fatalf("database failure was cached: %+v", cache)
				}
				return
			}
			if cache.writes != 1 || cache.writeID != 42 || cache.writtenCount != tc.wantCount {
				t.Fatalf("successful database count was not written back: %+v", cache)
			}
		})
	}
}

func TestCommentCountRejectsUnavailableArticleBeforeCache(t *testing.T) {
	for _, tc := range []struct {
		name             string
		articleID        uint64
		article          *model.ArticleModel
		articleErr       error
		wantErr          error
		wantArticleReads int
	}{
		{name: "invalid id", wantErr: common.ParamErr},
		{name: "draft", articleID: 42, article: &model.ArticleModel{ID: 42, Status: consts.ArticleStatusDraft}, wantErr: common.ArticleNotFoundErr, wantArticleReads: 1},
		{name: "offline", articleID: 42, article: &model.ArticleModel{ID: 42, Status: consts.ArticleStatusOffline}, wantErr: common.ArticleNotFoundErr, wantArticleReads: 1},
		{name: "missing", articleID: 42, wantErr: common.ArticleNotFoundErr, wantArticleReads: 1},
		{name: "lookup failure", articleID: 42, articleErr: errors.New("test article lookup failure"), wantErr: common.DatabaseErr, wantArticleReads: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &commentCountMysqlRepo{total: 7}
			articles := &commentCountArticleRepo{article: tc.article, err: tc.articleErr}
			cache := &commentCountRedisRepo{count: 99}
			app := NewCommentAppService(domainservice.NewCommentDomainService(source, articles), cache)

			count, err := app.GetArticleCommentCount(context.Background(), tc.articleID)
			if count != 0 || !errors.Is(err, tc.wantErr) {
				t.Fatalf("unavailable article returned (%d, %v), want (0, %v)", count, err, tc.wantErr)
			}
			if articles.reads != tc.wantArticleReads || cache.reads != 0 || cache.writes != 0 || len(source.queries) != 0 {
				t.Fatalf("unavailable article reached the count cache or query: articleReads=%d cache=%+v queries=%+v", articles.reads, cache, source.queries)
			}
		})
	}
}
