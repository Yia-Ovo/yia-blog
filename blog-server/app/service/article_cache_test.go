package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
)

type articleCacheDB struct {
	repo.ArticleMysqlRepo
	mu                    sync.Mutex
	article               *model.ArticleModel
	bodyReads, stateReads int
	fail                  error
	started               chan struct{}
	gate                  <-chan struct{}
}

func (r *articleCacheDB) GetArticleById(ctx context.Context, id uint64) (*model.ArticleModel, error) {
	r.mu.Lock()
	r.bodyReads++
	var value *model.ArticleModel
	if r.article != nil && r.article.ID == id && !r.article.DeletedAt.Valid {
		copy := *r.article
		value = &copy
	}
	started, gate, err := r.started, r.gate, r.fail
	r.mu.Unlock()
	if started != nil {
		select {
		case started <- struct{}{}:
		default:
		}
	}
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return value, err
}
func (r *articleCacheDB) GetArticlePublicState(_ context.Context, id uint64) (*do.ArticlePublicState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stateReads++
	if r.fail != nil {
		return nil, r.fail
	}
	if r.article == nil || r.article.ID != id || r.article.DeletedAt.Valid {
		return nil, nil
	}
	return &do.ArticlePublicState{ID: id, Status: r.article.Status}, nil
}
func (r *articleCacheDB) CreateArticle(_ context.Context, article *model.ArticleModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail != nil {
		return r.fail
	}
	copy := *article
	copy.ID = 31
	r.article = &copy
	return nil
}
func (r *articleCacheDB) UpdateArticle(_ context.Context, article *model.ArticleModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail != nil {
		return r.fail
	}
	copy := *article
	r.article = &copy
	return nil
}
func (r *articleCacheDB) DeleteArticle(_ context.Context, id uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail != nil {
		return r.fail
	}
	if r.article != nil && r.article.ID == id {
		r.article = nil
	}
	return nil
}
func newArticleCacheTestApp() (*ArticleAppService, *articleCacheDB, *categoryMemoryReadCache) {
	db := &articleCacheDB{article: &model.ArticleModel{ID: 31, Status: consts.ArticleStatusPublished, Title: "published", Content: "original"}}
	cache := &categoryMemoryReadCache{}
	return NewArticleAppService(domainservice.NewArticleDomainService(db), cache, nil, nil), db, cache
}
func TestArticleReadCacheWarmHitSkipsBodyButChecksPublicState(t *testing.T) {
	s, db, _ := newArticleCacheTestApp()
	for i := 0; i < 2; i++ {
		result, err := s.GetArticleById(context.Background(), 31)
		if err != nil || result["content"] != "original" {
			t.Fatalf("result=%v err=%v", result, err)
		}
		result["content"] = "caller edit"
	}
	if db.bodyReads != 1 || db.stateReads != 2 {
		t.Fatalf("body=%d state=%d", db.bodyReads, db.stateReads)
	}
}
func TestArticleReadCachePublicGuardRejectsStaleRedisAfterOfflineDeleteOrDBFailure(t *testing.T) {
	for _, change := range []string{"offline", "delete", "db-failure"} {
		t.Run(change, func(t *testing.T) {
			s, db, _ := newArticleCacheTestApp()
			if _, err := s.GetArticleById(context.Background(), 31); err != nil {
				t.Fatal(err)
			}
			db.mu.Lock()
			switch change {
			case "offline":
				db.article.Status = consts.ArticleStatusOffline
			case "delete":
				db.article = nil
			case "db-failure":
				db.fail = errors.New("db unavailable")
			}
			db.mu.Unlock()
			result, err := s.GetArticleById(context.Background(), 31)
			if result != nil || err == nil {
				t.Fatalf("stale public content leaked: %v %v", result, err)
			}
			if db.bodyReads != 1 {
				t.Fatal("expected stale cache hit plus independent guard")
			}
		})
	}
}
func TestArticleReadCacheNegativeAndWriteInvalidation(t *testing.T) {
	s, db, cache := newArticleCacheTestApp()
	db.article = nil
	for i := 0; i < 2; i++ {
		if _, err := s.GetArticleById(context.Background(), 31); !sameCacheError(err, common.ArticleNotFoundErr) {
			t.Fatalf("not found: %v", err)
		}
	}
	if db.bodyReads != 1 || cache.negatives != 1 {
		t.Fatalf("negative not cached: %d %d", db.bodyReads, cache.negatives)
	}
	if err := s.AdminCreateArticle(context.Background(), 0, &dto.CreateArticleReq{Title: "new", Content: "new body", Status: 1}); err != nil {
		t.Fatal(err)
	}
	result, err := s.GetArticleById(context.Background(), 31)
	if err != nil || result["content"] != "new body" {
		t.Fatalf("created body missing: %v %v", result, err)
	}
	if err := s.AdminUpdateArticle(context.Background(), 0, &dto.UpdateArticleReq{ID: 31, Title: "changed", Content: "changed body", Status: 1}); err != nil {
		t.Fatal(err)
	}
	result, err = s.GetArticleById(context.Background(), 31)
	if err != nil || result["content"] != "changed body" {
		t.Fatalf("old content after update: %v %v", result, err)
	}
	if err := s.AdminDeleteArticle(context.Background(), 0, 31); err != nil {
		t.Fatal(err)
	}
	if _, err = s.GetArticleById(context.Background(), 31); !sameCacheError(err, common.ArticleNotFoundErr) {
		t.Fatalf("deleted body visible: %v", err)
	}
	if cache.invalidations != 3 {
		t.Fatalf("invalidations=%d", cache.invalidations)
	}
}
func TestArticleReadCacheUserWritesAlsoInvalidateAndFailedWritesDoNot(t *testing.T) {
	s, db, cache := newArticleCacheTestApp()
	if err := s.CreateArticle(context.Background(), 7, &dto.CreateArticleReq{Title: "user", Content: "body", Status: 0}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateArticle(context.Background(), 7, &dto.UpdateArticleReq{ID: 31, Title: "edit", Content: "new", Status: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteArticle(context.Background(), 7, 31); err != nil {
		t.Fatal(err)
	}
	if cache.invalidations != 3 {
		t.Fatalf("invalidations=%d", cache.invalidations)
	}
	db.fail = errors.New("db error")
	if err := s.AdminCreateArticle(context.Background(), 0, &dto.CreateArticleReq{Title: "failed", Content: "body"}); err == nil {
		t.Fatal("write unexpectedly succeeded")
	}
	if cache.invalidations != 3 {
		t.Fatal("failed DB write changed generation")
	}
}
func TestArticleReadCacheUnavailableAndMalformedCacheFallBack(t *testing.T) {
	for _, mode := range []string{"read", "write", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			s, db, cache := newArticleCacheTestApp()
			switch mode {
			case "read":
				cache.readErr = errors.New("Redis down")
			case "write":
				cache.writeErr = errors.New("Redis down")
			case "malformed":
				cache.data = map[string][]byte{"detail:31": []byte("{broken")}
			}
			result, err := s.GetArticleById(context.Background(), 31)
			if err != nil || result["content"] != "original" || db.bodyReads != 1 {
				t.Fatalf("DB fallback failed: %v %v", result, err)
			}
		})
	}
}
func TestArticleReadCacheOldFillCannotReplaceNewGeneration(t *testing.T) {
	s, db, _ := newArticleCacheTestApp()
	gate := make(chan struct{})
	started := make(chan struct{}, 1)
	db.gate = gate
	db.started = started
	oldDone := make(chan error, 1)
	go func() { _, err := s.GetArticleById(context.Background(), 31); oldDone <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("load not started")
	}
	db.mu.Lock()
	db.gate = nil
	db.started = nil
	db.mu.Unlock()
	if err := s.AdminUpdateArticle(context.Background(), 0, &dto.UpdateArticleReq{ID: 31, Title: "new", Content: "new generation", Status: 1}); err != nil {
		t.Fatal(err)
	}
	result, err := s.GetArticleById(context.Background(), 31)
	if err != nil || result["content"] != "new generation" {
		t.Fatalf("new read joined old flight: %v %v", result, err)
	}
	close(gate)
	if err := <-oldDone; err != nil {
		t.Fatal(err)
	}
	result, err = s.GetArticleById(context.Background(), 31)
	if err != nil || result["content"] != "new generation" {
		t.Fatalf("late read poisoned cache: %v %v", result, err)
	}
}
func TestArticleReadCacheColdRequestsShareBodyRead(t *testing.T) {
	s, db, _ := newArticleCacheTestApp()
	gate := make(chan struct{})
	started := make(chan struct{}, 1)
	db.gate = gate
	db.started = started
	const count = 20
	done := make(chan error, count)
	for i := 0; i < count; i++ {
		go func() {
			result, err := s.GetArticleById(context.Background(), 31)
			if err == nil && result["content"] != "original" {
				err = fmt.Errorf("wrong body")
			}
			done <- err
		}()
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("load not started")
	}
	close(gate)
	for i := 0; i < count; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if db.bodyReads != 1 || db.stateReads != count {
		t.Fatalf("body=%d state=%d", db.bodyReads, db.stateReads)
	}
}
