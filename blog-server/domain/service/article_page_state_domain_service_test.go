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

func TestNormalizeArticleQueriesPreserveScopeAndPublicVisibility(t *testing.T) {
	for _, normalize := range []func(*do.ArticleQuery) error{NormalizeArticleQuery, NormalizePublicArticleQuery} {
		if err := normalize(nil); !errors.Is(err, common.ParamErr) {
			t.Fatalf("nil query returned %v", err)
		}
	}
	for _, tc := range []struct {
		page, size, wantPage, wantSize int
	}{
		{0, 0, 1, 10}, {-1, -2, 1, 10}, {2, 101, 2, 10}, {3, 100, 3, 100}, {1, 1, 1, 1},
	} {
		status, top, category, user := consts.ArticleStatusDraft, 0, uint64(0), uint64(7)
		query := &do.ArticleQuery{PageNum: tc.page, PageSize: tc.size, Title: " title ", Status: &status, IsTop: &top, CategoryID: &category, UserID: &user}
		if err := NormalizeArticleQuery(query); err != nil {
			t.Fatal(err)
		}
		if query.PageNum != tc.wantPage || query.PageSize != tc.wantSize || query.Status != &status || query.IsTop != &top || query.CategoryID != &category || query.UserID != &user || query.Title != " title " {
			t.Fatalf("normalization changed existing scope: %+v", query)
		}
		if err := NormalizePublicArticleQuery(query); err != nil || *query.Status != consts.ArticleStatusPublished || status != consts.ArticleStatusDraft {
			t.Fatalf("public scope must override without mutating the caller's status value: query=%+v err=%v", query, err)
		}
		if query.IsTop != &top || query.CategoryID != &category || query.UserID != &user || query.Title != " title " {
			t.Fatalf("public normalization changed unrelated filters: %+v", query)
		}
	}
	query := &do.ArticleQuery{}
	if err := NormalizePublicArticleQuery(query); err != nil || query.CategoryID != nil || query.IsTop != nil || query.UserID != nil {
		t.Fatalf("nil optional filters must stay nil: %+v err=%v", query, err)
	}
}

type pageStateTestRepo struct {
	repo.ArticleMysqlRepo
	state *do.ArticlePageState
	err   error
	query do.ArticleQuery
	calls int
	ctx   context.Context
}

func (r *pageStateTestRepo) GetArticlePageState(ctx context.Context, query *do.ArticleQuery) (*do.ArticlePageState, error) {
	r.calls++
	r.ctx = ctx
	r.query = *query
	return r.state, r.err
}

func TestValidatePublicArticlePageChecksOrderedMembershipAndTotal(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state *do.ArticlePageState
		ids   []uint64
		total int64
		match bool
	}{
		{"unchanged", &do.ArticlePageState{IDs: []uint64{10, 9}, Total: 3}, []uint64{10, 9}, 3, true},
		{"offline page item with refill", &do.ArticlePageState{IDs: []uint64{9, 8}, Total: 2}, []uint64{10, 9}, 3, false},
		{"earlier page shift", &do.ArticlePageState{IDs: []uint64{6, 5}, Total: 7}, []uint64{7, 6}, 8, false},
		{"another page changes total", &do.ArticlePageState{IDs: []uint64{10, 9}, Total: 4}, []uint64{10, 9}, 3, false},
		{"same count different membership", &do.ArticlePageState{IDs: []uint64{10, 8}, Total: 3}, []uint64{10, 9}, 3, false},
		{"same membership different order", &do.ArticlePageState{IDs: []uint64{9, 10}, Total: 3}, []uint64{10, 9}, 3, false},
		{"empty page", &do.ArticlePageState{IDs: []uint64{}, Total: 0}, nil, 0, true},
		{"out of range empty page", &do.ArticlePageState{IDs: []uint64{}, Total: 3}, nil, 3, true},
		{"empty page becomes populated", &do.ArticlePageState{IDs: []uint64{10}, Total: 1}, nil, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &pageStateTestRepo{state: tc.state}
			draft, top, category := consts.ArticleStatusDraft, 0, uint64(0)
			query := &do.ArticleQuery{PageSize: 101, Status: &draft, Title: "needle", IsTop: &top, CategoryID: &category}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			match, err := NewArticleDomainService(r).ValidatePublicArticlePage(ctx, query, tc.ids, tc.total)
			if err != nil || match != tc.match || r.calls != 1 || r.ctx != ctx {
				t.Fatalf("match=%v err=%v calls=%d; want match=%v", match, err, r.calls, tc.match)
			}
			if *r.query.Status != consts.ArticleStatusPublished || r.query.PageNum != 1 || r.query.PageSize != 10 || r.query.Title != "needle" || r.query.IsTop != &top || r.query.CategoryID != &category {
				t.Fatalf("validation bypassed public query scope: %+v", r.query)
			}
		})
	}
}

func TestValidatePublicArticlePageDoesNotTrustUnavailableDatabase(t *testing.T) {
	for _, queryErr := range []error{errors.New("database offline"), context.Canceled, nil} {
		r := &pageStateTestRepo{err: queryErr}
		if match, err := NewArticleDomainService(r).ValidatePublicArticlePage(context.Background(), &do.ArticleQuery{}, nil, 0); match || !errors.Is(err, common.DatabaseErr) {
			t.Fatalf("database failure or invalid nil state accepted: match=%v err=%v", match, err)
		}
	}
	r := &pageStateTestRepo{}
	if match, err := NewArticleDomainService(r).ValidatePublicArticlePage(context.Background(), nil, nil, 0); match || !errors.Is(err, common.ParamErr) || r.calls != 0 {
		t.Fatalf("nil query reached repository: match=%v err=%v calls=%d", match, err, r.calls)
	}
}

type pageStateLegacyTestRepo struct {
	repo.ArticleMysqlRepo
	items []*model.ArticleModel
	total int64
	err   error
	query do.ArticleQuery
}

func (r *pageStateLegacyTestRepo) ListArticleByPage(_ context.Context, query *do.ArticleQuery) ([]*model.ArticleModel, int64, error) {
	r.query = *query
	return r.items, r.total, r.err
}

func TestValidatePublicArticlePageFallsBackForLegacyRepository(t *testing.T) {
	r := &pageStateLegacyTestRepo{items: []*model.ArticleModel{{ID: 10}, {ID: 9}}, total: 3}
	domain := NewArticleDomainService(r)
	draft := consts.ArticleStatusDraft
	query := &do.ArticleQuery{Status: &draft}
	if match, err := domain.ValidatePublicArticlePage(context.Background(), query, []uint64{10, 9}, 3); !match || err != nil || r.query.PageNum != 1 || r.query.PageSize != 10 || *r.query.Status != consts.ArticleStatusPublished {
		t.Fatalf("legacy fallback lost identity or scope: match=%v err=%v query=%+v", match, err, r.query)
	}
	if match, err := domain.ValidatePublicArticlePage(context.Background(), query, []uint64{9, 10}, 3); match || err != nil {
		t.Fatalf("legacy fallback ignored ordering: match=%v err=%v", match, err)
	}
	r.err = errors.New("database offline")
	if match, err := domain.ValidatePublicArticlePage(context.Background(), query, []uint64{10, 9}, 3); match || !errors.Is(err, common.DatabaseErr) {
		t.Fatalf("legacy fallback ignored database failure: match=%v err=%v", match, err)
	}
}
