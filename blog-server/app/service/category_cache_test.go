package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
)

type categoryMemoryReadCache struct {
	mu                                      sync.Mutex
	data                                    map[string][]byte
	generation                              int
	reads, writes, negatives, invalidations int
	readErr, writeErr, invalidateErr        error
}

func (r *categoryMemoryReadCache) Read(_ context.Context, key string) ([]byte, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reads++
	if r.readErr != nil {
		return nil, "", r.readErr
	}
	generation := fmt.Sprint(r.generation)
	data, ok := r.data[key]
	if !ok {
		return nil, generation, repo.ErrCacheMiss
	}
	return append([]byte(nil), data...), generation, nil
}

func (r *categoryMemoryReadCache) Write(_ context.Context, key, generation string, data []byte, negative bool) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writes++
	if r.writeErr != nil {
		return false, r.writeErr
	}
	if generation != fmt.Sprint(r.generation) {
		return false, nil
	}
	if r.data == nil {
		r.data = make(map[string][]byte)
	}
	r.data[key] = append([]byte(nil), data...)
	if negative {
		r.negatives++
	}
	return true, nil
}

func (r *categoryMemoryReadCache) Invalidate(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invalidations++
	if r.invalidateErr != nil {
		return r.invalidateErr
	}
	r.generation++
	r.data = make(map[string][]byte)
	return nil
}

type categoryCacheDB struct {
	repo.CategoryMysqlRepo
	mu                                       sync.Mutex
	items                                    map[uint64]model.CategoryModel
	gets, lists                              int
	readErr, createErr, updateErr, deleteErr error
	getStarted                               chan struct{}
	getGate                                  <-chan struct{}
}

func (r *categoryCacheDB) GetCategoryById(ctx context.Context, id uint64) (*model.CategoryModel, error) {
	r.mu.Lock()
	r.gets++
	item, ok := r.items[id]
	err, started, gate := r.readErr, r.getStarted, r.getGate
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
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (r *categoryCacheDB) ListCategoryByPage(_ context.Context, query *do.CategoryQuery) ([]*model.CategoryModel, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lists++
	if r.readErr != nil {
		return nil, 0, r.readErr
	}
	items := make([]*model.CategoryModel, 0)
	for _, item := range r.items {
		if !strings.Contains(item.Name, query.Name) || (query.Status != nil && item.Status != *query.Status) {
			continue
		}
		copy := item
		items = append(items, &copy)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	total := len(items)
	start := (query.PageNum - 1) * query.PageSize
	if start >= total {
		return nil, int64(total), nil
	}
	end := start + query.PageSize
	if end > total {
		end = total
	}
	return items[start:end], int64(total), nil
}

func (r *categoryCacheDB) CreateCategory(_ context.Context, item *model.CategoryModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createErr != nil {
		return r.createErr
	}
	var next uint64 = 1
	for id := range r.items {
		if id >= next {
			next = id + 1
		}
	}
	copy := *item
	copy.ID = next
	r.items[next] = copy
	return nil
}

func (r *categoryCacheDB) UpdateCategory(_ context.Context, update *do.CategoryUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateErr != nil {
		return r.updateErr
	}
	item := r.items[update.ID]
	item.Name = update.Name
	if update.Sort != nil {
		item.Sort = *update.Sort
	}
	if update.Status != nil {
		item.Status = *update.Status
	}
	r.items[update.ID] = item
	return nil
}

func (r *categoryCacheDB) DeleteCategory(_ context.Context, id uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.items, id)
	return nil
}

func newCategoryCacheFixture() (*CategoryAppService, *categoryCacheDB, *categoryMemoryReadCache) {
	db := &categoryCacheDB{items: map[uint64]model.CategoryModel{
		1: {ID: 1, Name: "Go", Status: 1},
		2: {ID: 2, Name: "Redis", Status: 0},
	}}
	cache := &categoryMemoryReadCache{}
	return NewCategoryAppService(domainservice.NewCategoryDomainService(db), cache), db, cache
}

func categoryPageIDs(t *testing.T, page *dto.PageResult) []uint64 {
	t.Helper()
	items, ok := page.List.([]map[string]interface{})
	if !ok {
		t.Fatalf("unexpected list shape: %T", page.List)
	}
	ids := make([]uint64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item["id"].(uint64))
	}
	return ids
}

func TestCategoryReadCacheNormalizesAndSeparatesQueries(t *testing.T) {
	s, db, _ := newCategoryCacheFixture()
	ctx := context.Background()
	disabled, enabled := 0, 1
	cases := []struct {
		query dto.CategoryPageQuery
		ids   string
		total int64
	}{
		{dto.CategoryPageQuery{}, "[1 2]", 2},
		{dto.CategoryPageQuery{PageNum: 1, PageSize: 51}, "[1 2]", 2},
		{dto.CategoryPageQuery{PageNum: 1, PageSize: 1}, "[1]", 2},
		{dto.CategoryPageQuery{PageNum: 2, PageSize: 1}, "[2]", 2},
		{dto.CategoryPageQuery{Name: "Go"}, "[1]", 1},
		{dto.CategoryPageQuery{Status: &disabled}, "[2]", 1},
		{dto.CategoryPageQuery{Status: &enabled}, "[1]", 1},
	}
	for _, tc := range cases {
		for i := 0; i < 2; i++ {
			page, err := s.ListCategoryByPage(ctx, &tc.query)
			if err != nil {
				t.Fatal(err)
			}
			if ids := fmt.Sprint(categoryPageIDs(t, page)); ids != tc.ids || page.Total != tc.total {
				t.Fatalf("query=%+v page=%+v ids=%s", tc.query, page, ids)
			}
			if tc.query.PageSize > 50 || tc.query.PageSize == 0 {
				if page.PageSize != 10 {
					t.Fatalf("effective page size=%d", page.PageSize)
				}
			}
		}
	}
	if db.lists != 6 {
		t.Fatalf("database list queries=%d, want 6", db.lists)
	}
	if _, err := s.ListCategoryByPage(ctx, nil); !errors.Is(err, common.ParamErr) {
		t.Fatalf("nil query err=%v", err)
	}
	if _, _, err := domainservice.NewCategoryDomainService(db).ListCategoryByPage(ctx, nil); !errors.Is(err, common.ParamErr) {
		t.Fatalf("domain nil query err=%v", err)
	}
}

func TestCategoryReadCacheDetailAndNegative(t *testing.T) {
	s, db, cache := newCategoryCacheFixture()
	for i := 0; i < 3; i++ {
		detail, err := s.GetCategoryById(context.Background(), 1)
		if err != nil || detail["name"] != "Go" {
			t.Fatalf("detail=%v err=%v", detail, err)
		}
		// 调用方修改响应，不应污染缓存中的值。
		detail["name"] = "mutated"
		_, err = s.GetCategoryById(context.Background(), 99)
		if !errors.Is(err, common.CategoryNotFoundErr) {
			t.Fatalf("missing category err=%v", err)
		}
	}
	if db.gets != 2 || cache.negatives != 1 {
		t.Fatalf("gets=%d negatives=%d", db.gets, cache.negatives)
	}
	if _, err := s.GetCategoryById(context.Background(), 0); !errors.Is(err, common.ParamErr) {
		t.Fatalf("invalid id err=%v", err)
	}
}

func TestCategoryReadCacheWritesInvalidateListsDetailsAndNegatives(t *testing.T) {
	s, _, cache := newCategoryCacheFixture()
	ctx := context.Background()
	_, _ = s.ListCategoryByPage(ctx, &dto.CategoryPageQuery{})
	_, _ = s.GetCategoryById(ctx, 1)
	_, _ = s.GetCategoryById(ctx, 3)
	if err := s.CreateCategory(ctx, &dto.CreateCategoryReq{Name: "New"}); err != nil {
		t.Fatal(err)
	}
	if created, err := s.GetCategoryById(ctx, 3); err != nil || created["name"] != "New" {
		t.Fatalf("negative not invalidated: %v %v", created, err)
	}
	if page, err := s.ListCategoryByPage(ctx, &dto.CategoryPageQuery{}); err != nil || page.Total != 3 {
		t.Fatalf("created list=%v err=%v", page, err)
	}
	disabled := 0
	if err := s.UpdateCategory(ctx, 1, "Renamed", nil, &disabled); err != nil {
		t.Fatal(err)
	}
	if detail, err := s.GetCategoryById(ctx, 1); err != nil || detail["name"] != "Renamed" || detail["status"] != 0 {
		t.Fatalf("updated detail=%v err=%v", detail, err)
	}
	if page, err := s.ListCategoryByPage(ctx, &dto.CategoryPageQuery{Name: "Renamed"}); err != nil || page.Total != 1 {
		t.Fatalf("updated list=%v err=%v", page, err)
	}
	if err := s.DeleteCategory(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetCategoryById(ctx, 1); !errors.Is(err, common.CategoryNotFoundErr) {
		t.Fatalf("deleted detail err=%v", err)
	}
	if page, err := s.ListCategoryByPage(ctx, &dto.CategoryPageQuery{}); err != nil || page.Total != 2 {
		t.Fatalf("deleted list=%v err=%v", page, err)
	}
	if cache.invalidations != 3 {
		t.Fatalf("invalidations=%d", cache.invalidations)
	}
}

func TestCategoryReadCacheFailedWritesDoNotInvalidate(t *testing.T) {
	s, db, cache := newCategoryCacheFixture()
	ctx := context.Background()
	_, _ = s.GetCategoryById(ctx, 1)
	db.createErr, db.updateErr, db.deleteErr = errors.New("create failed"), errors.New("update failed"), errors.New("delete failed")
	if err := s.CreateCategory(ctx, &dto.CreateCategoryReq{Name: "New"}); err == nil {
		t.Fatal("create should fail")
	}
	if err := s.UpdateCategory(ctx, 1, "Changed", nil, nil); err == nil {
		t.Fatal("update should fail")
	}
	if err := s.DeleteCategory(ctx, 1); err == nil {
		t.Fatal("delete should fail")
	}
	if cache.invalidations != 0 {
		t.Fatalf("failed writes invalidated %d times", cache.invalidations)
	}
	if detail, err := s.GetCategoryById(ctx, 1); err != nil || detail["name"] != "Go" {
		t.Fatalf("detail=%v err=%v", detail, err)
	}
}

func TestCategoryReadCacheFailureFallback(t *testing.T) {
	for _, failure := range []string{"read", "write", "invalid"} {
		t.Run(failure, func(t *testing.T) {
			s, db, cache := newCategoryCacheFixture()
			switch failure {
			case "read":
				cache.readErr = errors.New("redis unavailable")
			case "write":
				cache.writeErr = errors.New("redis unavailable")
			case "invalid":
				cache.data = map[string][]byte{"detail:1": []byte("broken JSON")}
			}
			for i := 0; i < 2; i++ {
				detail, err := s.GetCategoryById(context.Background(), 1)
				if err != nil || detail["name"] != "Go" {
					t.Fatalf("detail=%v err=%v", detail, err)
				}
			}
			if db.gets < 1 {
				t.Fatal("expected database fallback")
			}
		})
	}
}

func TestCategoryReadCacheDoesNotCacheDatabaseFailure(t *testing.T) {
	s, db, cache := newCategoryCacheFixture()
	db.readErr = errors.New("mysql unavailable")
	for i := 0; i < 2; i++ {
		if _, err := s.GetCategoryById(context.Background(), 1); !errors.Is(err, common.DatabaseErr) {
			t.Fatalf("err=%v", err)
		}
	}
	if cache.writes != 0 || db.gets != 2 {
		t.Fatalf("writes=%d gets=%d", cache.writes, db.gets)
	}
	db.readErr = nil
	if _, err := s.GetCategoryById(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
}

func TestCategoryReadCacheCoalescesConcurrentColdDetail(t *testing.T) {
	s, db, _ := newCategoryCacheFixture()
	gate := make(chan struct{})
	db.getStarted = make(chan struct{}, 20)
	db.getGate = gate
	start := make(chan struct{})
	results := make(chan error, 20)
	for i := 0; i < 20; i++ {
		go func() {
			<-start
			item, err := s.GetCategoryById(context.Background(), 1)
			if err == nil && item["name"] != "Go" {
				err = fmt.Errorf("unexpected detail: %v", item)
			}
			results <- err
		}()
	}
	close(start)
	select {
	case <-db.getStarted:
	case <-time.After(time.Second):
		t.Fatal("database query did not start")
	}
	close(gate)
	for i := 0; i < 20; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("read did not finish")
		}
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.gets != 1 {
		t.Fatalf("database detail queries=%d, want 1", db.gets)
	}
}

func TestCategoryReadCacheInvalidationFailureDoesNotReturnStale(t *testing.T) {
	s, _, cache := newCategoryCacheFixture()
	ctx := context.Background()
	_, _ = s.GetCategoryById(ctx, 1)
	cache.invalidateErr = errors.New("redis unavailable")
	if err := s.UpdateCategory(ctx, 1, "Changed", nil, nil); err != nil {
		t.Fatalf("successful database write must stay successful: %v", err)
	}
	if detail, err := s.GetCategoryById(ctx, 1); err != nil || detail["name"] != "Changed" {
		t.Fatalf("dirty cache returned stale data: %v err=%v", detail, err)
	}
	cache.invalidateErr = nil
	if detail, err := s.GetCategoryById(ctx, 1); err != nil || detail["name"] != "Changed" {
		t.Fatalf("recovered cache: %v err=%v", detail, err)
	}
}

func TestCategoryReadCacheCreateDoesNotAcceptOlderNegativeFill(t *testing.T) {
	s, db, _ := newCategoryCacheFixture()
	gate := make(chan struct{})
	db.getStarted = make(chan struct{}, 1)
	db.getGate = gate
	oldResult := make(chan error, 1)
	go func() {
		_, err := s.GetCategoryById(context.Background(), 3)
		oldResult <- err
	}()
	select {
	case <-db.getStarted:
	case <-time.After(time.Second):
		t.Fatal("old database read did not start")
	}
	if err := s.CreateCategory(context.Background(), &dto.CreateCategoryReq{Name: "Created"}); err != nil {
		t.Fatal(err)
	}
	db.mu.Lock()
	db.getGate = nil
	db.mu.Unlock()
	// 写后新请求必须独立查询，不能等待/复用创建前的“不存在”结果。
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	detail, err := s.GetCategoryById(ctx, 3)
	close(gate)
	if err != nil || detail["name"] != "Created" {
		t.Fatalf("post-create detail=%v err=%v", detail, err)
	}
	select {
	case err := <-oldResult:
		if !errors.Is(err, common.CategoryNotFoundErr) {
			t.Fatalf("old snapshot err=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("old database read did not finish")
	}
	detail, err = s.GetCategoryById(context.Background(), 3)
	if err != nil || detail["name"] != "Created" {
		t.Fatalf("old negative overwrote new cache: detail=%v err=%v", detail, err)
	}
}
