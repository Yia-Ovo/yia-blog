package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
	"context"
	"errors"
	"testing"
)

type publicStateTestRepo struct {
	repo.ArticleMysqlRepo
	state *do.ArticlePublicState
	err   error
	calls int
	ctx   context.Context
}

func (r *publicStateTestRepo) GetArticlePublicState(ctx context.Context, id uint64) (*do.ArticlePublicState, error) {
	r.calls++
	r.ctx = ctx
	return r.state, r.err
}

func TestEnsurePublicArticleUsesLightweightVisibility(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       uint64
		state    *do.ArticlePublicState
		queryErr error
		want     error
	}{
		{name: "published", id: 7, state: &do.ArticlePublicState{ID: 7, Status: consts.ArticleStatusPublished}},
		{name: "draft", id: 7, state: &do.ArticlePublicState{ID: 7, Status: consts.ArticleStatusDraft}, want: common.ArticleNotFoundErr},
		{name: "offline", id: 7, state: &do.ArticlePublicState{ID: 7, Status: consts.ArticleStatusOffline}, want: common.ArticleNotFoundErr},
		{name: "missing or soft deleted", id: 7, want: common.ArticleNotFoundErr},
		{name: "wrong identity", id: 7, state: &do.ArticlePublicState{ID: 8, Status: consts.ArticleStatusPublished}, want: common.ArticleNotFoundErr},
		{name: "invalid identity", want: common.ParamErr},
		{name: "database failure", id: 7, queryErr: errors.New("offline"), want: common.DatabaseErr},
		{name: "cancelled database query", id: 7, queryErr: context.Canceled, want: common.DatabaseErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &publicStateTestRepo{state: tc.state, err: tc.queryErr}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if err := NewArticleDomainService(r).EnsurePublicArticle(ctx, tc.id); !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
			if tc.id == 0 {
				if r.calls != 0 {
					t.Fatalf("invalid ID reached repository: %d", r.calls)
				}
			} else if r.calls != 1 || r.ctx != ctx {
				t.Fatalf("lightweight query lost request context: calls=%d", r.calls)
			}
		})
	}
}

type publicStateLegacyTestRepo struct {
	repo.ArticleMysqlRepo
	article *model.ArticleModel
	err     error
	calls   int
}

func (r *publicStateLegacyTestRepo) GetArticleById(context.Context, uint64) (*model.ArticleModel, error) {
	r.calls++
	return r.article, r.err
}

func TestEnsurePublicArticleFallsBackForLegacyRepository(t *testing.T) {
	for _, tc := range []struct {
		name     string
		article  *model.ArticleModel
		queryErr error
		want     error
	}{
		{name: "published", article: &model.ArticleModel{ID: 7, Status: consts.ArticleStatusPublished}},
		{name: "draft", article: &model.ArticleModel{ID: 7, Status: consts.ArticleStatusDraft}, want: common.ArticleNotFoundErr},
		{name: "missing", want: common.ArticleNotFoundErr},
		{name: "database failure", queryErr: errors.New("offline"), want: common.DatabaseErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &publicStateLegacyTestRepo{article: tc.article, err: tc.queryErr}
			if err := NewArticleDomainService(r).EnsurePublicArticle(context.Background(), 7); !errors.Is(err, tc.want) || r.calls != 1 {
				t.Fatalf("fallback must enforce public visibility: calls=%d err=%v want=%v", r.calls, err, tc.want)
			}
		})
	}
}
