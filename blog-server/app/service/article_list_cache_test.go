package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
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

// articleListCacheDB 在锁内取得快照、锁外等待，用于重现更新先完成而旧列表后返回的竞态。
type articleListCacheDB struct {
	repo.ArticleMysqlRepo
	mu                    sync.Mutex
	items                 map[uint64]model.ArticleModel
	wideReads, stateReads int
	wideErr, stateErr     error
	queries               []do.ArticleQuery
	started               chan struct{}
	gate                  <-chan struct{}
}

func (r *articleListCacheDB) pageLocked(query *do.ArticleQuery) ([]*model.ArticleModel, int64) {
	items := make([]*model.ArticleModel, 0)
	for _, item := range r.items {
		if item.DeletedAt.Valid || !strings.Contains(item.Title, query.Title) ||
			(query.CategoryID != nil && item.CategoryID != *query.CategoryID) ||
			(query.Status != nil && item.Status != *query.Status) ||
			(query.IsTop != nil && item.IsTop != *query.IsTop) ||
			(query.UserID != nil && (item.UserID != *query.UserID || item.AdminID != 0)) {
			continue
		}
		copy := item
		items = append(items, &copy)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Sort != items[j].Sort {
			return items[i].Sort < items[j].Sort
		}
		var left, right time.Time
		if items[i].PublishAt != nil {
			left = *items[i].PublishAt
		}
		if items[j].PublishAt != nil {
			right = *items[j].PublishAt
		}
		if !left.Equal(right) {
			return left.After(right)
		}
		return items[i].ID > items[j].ID
	})
	total := int64(len(items))
	start := (query.PageNum - 1) * query.PageSize
	if start >= len(items) {
		return []*model.ArticleModel{}, total
	}
	end := start + query.PageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], total
}

func (r *articleListCacheDB) ListArticleByPage(ctx context.Context, query *do.ArticleQuery) ([]*model.ArticleModel, int64, error) {
	r.mu.Lock()
	r.wideReads++
	// 保存归一化后的状态值，避免历史记录依赖查询对象中状态指针的后续变化。
	copy := *query
	if query.Status != nil {
		status := *query.Status
		copy.Status = &status
	}
	r.queries = append(r.queries, copy)
	items, total := r.pageLocked(query)
	err, started, gate := r.wideErr, r.started, r.gate
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
			return nil, 0, ctx.Err()
		}
	}
	return items, total, err
}

func (r *articleListCacheDB) GetArticlePageState(_ context.Context, query *do.ArticleQuery) (*do.ArticlePageState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stateReads++
	if r.stateErr != nil {
		return nil, r.stateErr
	}
	items, total := r.pageLocked(query)
	ids := make([]uint64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return &do.ArticlePageState{IDs: ids, Total: total}, nil
}

func (r *articleListCacheDB) counts() (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.wideReads, r.stateReads
}

func newArticleListCacheFixture() (*ArticleAppService, *articleListCacheDB, *categoryMemoryReadCache) {
	db := &articleListCacheDB{items: map[uint64]model.ArticleModel{
		1: {ID: 1, Title: "Go cache", Summary: "first", Content: "private-body-one", Status: consts.ArticleStatusPublished, CategoryID: 0, Sort: 1},
		2: {ID: 2, Title: "Go API", Content: "private-body-two", Status: consts.ArticleStatusPublished, CategoryID: 1, IsTop: 1, Sort: 2},
		3: {ID: 3, Title: "React", Status: consts.ArticleStatusPublished, CategoryID: 2, Sort: 3},
		4: {ID: 4, Title: "Redis", Status: consts.ArticleStatusPublished, CategoryID: 1, IsTop: 1, Sort: 4},
		5: {ID: 5, Title: "MySQL", Status: consts.ArticleStatusPublished, CategoryID: 2, Sort: 5},
		6: {ID: 6, Title: "Go draft", Content: "unpublished-secret", Status: consts.ArticleStatusDraft, Sort: 0},
	}}
	cache := &categoryMemoryReadCache{}
	return NewArticleAppService(domainservice.NewArticleDomainService(db), cache, nil, nil), db, cache
}

func articleListRows(t *testing.T, result *dto.PageResult) []map[string]interface{} {
	t.Helper()
	if result == nil {
		t.Fatal("nil page result")
	}
	rows, ok := result.List.([]map[string]interface{})
	if !ok {
		t.Fatalf("unexpected page rows: %T", result.List)
	}
	return rows
}

func assertArticleListPage(t *testing.T, result *dto.PageResult, ids []uint64, total int64) {
	t.Helper()
	rows := articleListRows(t, result)
	actual := make([]uint64, 0, len(rows))
	for _, row := range rows {
		id, ok := row["id"].(uint64)
		if !ok {
			t.Fatalf("unexpected id type: %T", row["id"])
		}
		actual = append(actual, id)
		if _, exists := row["content"]; exists {
			t.Fatal("list returned article body")
		}
	}
	if !reflect.DeepEqual(actual, ids) || result.Total != total {
		t.Fatalf("ids=%v total=%d; want ids=%v total=%d", actual, result.Total, ids, total)
	}
}

func TestArticleListCacheWarmHitChecksPageAndDoesNotCacheBody(t *testing.T) {
	s, db, cache := newArticleListCacheFixture()
	query := &dto.ArticlePageQuery{PageNum: 1, PageSize: 2}
	for i := 0; i < 2; i++ {
		result, err := s.ListArticleByPage(context.Background(), query)
		if err != nil {
			t.Fatal(err)
		}
		assertArticleListPage(t, result, []uint64{1, 2}, 5)
		rows := articleListRows(t, result)
		if rows[0]["summary"] != "first" || result.Pages != 3 {
			t.Fatalf("unexpected page: %+v", result)
		}
		rows[0]["summary"] = "caller mutation"
	}
	if wide, state := db.counts(); wide != 1 || state != 2 {
		t.Fatalf("wide reads=%d state reads=%d; want 1 and 2", wide, state)
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if len(cache.data) != 1 {
		t.Fatalf("cached pages=%d; want 1", len(cache.data))
	}
	for _, raw := range cache.data {
		if strings.Contains(string(raw), "\"Content\":") || strings.Contains(string(raw), "private-body-") || strings.Contains(string(raw), "unpublished-secret") {
			t.Fatalf("cached page contains body or unpublished data: %s", raw)
		}
	}
}

func TestArticleListCacheKeysIncludeEveryNormalizedFilter(t *testing.T) {
	s, db, _ := newArticleListCacheFixture()
	zeroCategory, oneCategory := uint64(0), uint64(1)
	zeroTop, oneTop := 0, 1
	cases := []struct {
		query dto.ArticlePageQuery
		ids   []uint64
		total int64
	}{
		{dto.ArticlePageQuery{PageNum: 1, PageSize: 2}, []uint64{1, 2}, 5},
		{dto.ArticlePageQuery{PageNum: 2, PageSize: 2}, []uint64{3, 4}, 5},
		{dto.ArticlePageQuery{PageNum: 1, PageSize: 3}, []uint64{1, 2, 3}, 5},
		{dto.ArticlePageQuery{PageNum: 1, PageSize: 2, Title: "Go"}, []uint64{1, 2}, 2},
		{dto.ArticlePageQuery{PageNum: 1, PageSize: 2, CategoryID: &zeroCategory}, []uint64{1}, 1},
		{dto.ArticlePageQuery{PageNum: 1, PageSize: 2, CategoryID: &oneCategory}, []uint64{2, 4}, 2},
		{dto.ArticlePageQuery{PageNum: 1, PageSize: 2, IsTop: &zeroTop}, []uint64{1, 3}, 3},
		{dto.ArticlePageQuery{PageNum: 1, PageSize: 2, IsTop: &oneTop}, []uint64{2, 4}, 2},
		{dto.ArticlePageQuery{PageNum: 1, PageSize: 2, Title: "Go", CategoryID: &oneCategory, IsTop: &oneTop}, []uint64{2}, 1},
	}
	// 第二遍交错访问仍必须全部命中，不能靠键冲突后的校验回源掩盖串页。
	for sweep := 0; sweep < 2; sweep++ {
		for _, tc := range cases {
			result, err := s.ListArticleByPage(context.Background(), &tc.query)
			if err != nil {
				t.Fatal(err)
			}
			assertArticleListPage(t, result, tc.ids, tc.total)
		}
	}
	if wide, state := db.counts(); wide != len(cases) || state != 2*len(cases) {
		t.Fatalf("wide reads=%d state reads=%d; want %d and %d", wide, state, len(cases), 2*len(cases))
	}
}

func TestArticleListCacheCanonicalPaginationAndPublishedStatusShareKey(t *testing.T) {
	s, db, _ := newArticleListCacheFixture()
	draft, offline, published := consts.ArticleStatusDraft, consts.ArticleStatusOffline, consts.ArticleStatusPublished
	queries := []dto.ArticlePageQuery{
		{},
		{PageNum: -5, PageSize: -1, Status: &draft},
		{PageNum: 0, PageSize: 101, Status: &offline},
		{PageNum: 1, PageSize: 10, Status: &published},
	}
	for _, query := range queries {
		result, err := s.ListArticleByPage(context.Background(), &query)
		if err != nil {
			t.Fatal(err)
		}
		assertArticleListPage(t, result, []uint64{1, 2, 3, 4, 5}, 5)
		if result.PageNum != 1 || result.PageSize != 10 {
			t.Fatalf("pagination not canonical: %+v", result)
		}
	}
	if wide, state := db.counts(); wide != 1 || state != len(queries) {
		t.Fatalf("wide reads=%d state reads=%d; want 1 and %d", wide, state, len(queries))
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, query := range db.queries {
		if query.Status == nil || *query.Status != published {
			t.Fatalf("public list did not force published status: %+v", query)
		}
	}
}

func TestArticleListCacheRebuildsWholePageOnStateChange(t *testing.T) {
	cases := []struct {
		name   string
		query  dto.ArticlePageQuery
		change func(*articleListCacheDB)
		ids    []uint64
		total  int64
	}{
		{"offline fills vacancy", dto.ArticlePageQuery{PageNum: 1, PageSize: 2}, func(db *articleListCacheDB) {
			item := db.items[1]
			item.Status = consts.ArticleStatusOffline
			db.items[1] = item
		}, []uint64{2, 3}, 4},
		{"previous page deletion shifts offset", dto.ArticlePageQuery{PageNum: 2, PageSize: 2}, func(db *articleListCacheDB) {
			delete(db.items, 1)
		}, []uint64{4, 5}, 4},
		{"unchanged ids with changed total", dto.ArticlePageQuery{PageNum: 1, PageSize: 2}, func(db *articleListCacheDB) {
			delete(db.items, 5)
		}, []uint64{1, 2}, 4},
		{"same ids reordered", dto.ArticlePageQuery{PageNum: 1, PageSize: 2}, func(db *articleListCacheDB) {
			item := db.items[2]
			item.Sort = 0
			db.items[2] = item
		}, []uint64{2, 1}, 5},
		{"empty page becomes populated", dto.ArticlePageQuery{PageNum: 4, PageSize: 2}, func(db *articleListCacheDB) {
			db.items[7] = model.ArticleModel{ID: 7, Title: "new seven", Status: consts.ArticleStatusPublished, Sort: 7}
			db.items[8] = model.ArticleModel{ID: 8, Title: "new eight", Status: consts.ArticleStatusPublished, Sort: 8}
		}, []uint64{8}, 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, db, cache := newArticleListCacheFixture()
			if _, err := s.ListArticleByPage(context.Background(), &tc.query); err != nil {
				t.Fatal(err)
			}
			// 模拟其他进程写入且 Redis 失效消息丢失。
			db.mu.Lock()
			tc.change(db)
			db.mu.Unlock()
			result, err := s.ListArticleByPage(context.Background(), &tc.query)
			if err != nil {
				t.Fatal(err)
			}
			assertArticleListPage(t, result, tc.ids, tc.total)
			if wide, _ := db.counts(); wide != 2 {
				t.Fatalf("whole-page load count=%d; want 2", wide)
			}
			cache.mu.Lock()
			defer cache.mu.Unlock()
			if cache.invalidations != 1 || len(cache.data) != 0 {
				t.Fatalf("invalidations=%d cached pages=%d; changed page must invalidate without unversioned refill", cache.invalidations, len(cache.data))
			}
		})
	}
}

func TestArticleListCacheDatabaseValidationFailureDoesNotReturnCachedPage(t *testing.T) {
	s, db, _ := newArticleListCacheFixture()
	query := &dto.ArticlePageQuery{PageNum: 1, PageSize: 2}
	if _, err := s.ListArticleByPage(context.Background(), query); err != nil {
		t.Fatal(err)
	}
	db.mu.Lock()
	db.stateErr = errors.New("database unavailable")
	db.mu.Unlock()
	result, err := s.ListArticleByPage(context.Background(), query)
	if result != nil || !sameCacheError(err, common.DatabaseErr) {
		t.Fatalf("validation failure returned cached page: result=%v err=%v", result, err)
	}
	if wide, state := db.counts(); wide != 1 || state != 2 {
		t.Fatalf("wide reads=%d state reads=%d", wide, state)
	}
}

func TestArticleListCacheRedisFailureStillReturnsCurrentDatabasePage(t *testing.T) {
	for _, failure := range []string{"read", "write", "invalidate"} {
		t.Run(failure, func(t *testing.T) {
			s, db, cache := newArticleListCacheFixture()
			query := &dto.ArticlePageQuery{PageNum: 1, PageSize: 2}
			if failure == "invalidate" {
				if _, err := s.ListArticleByPage(context.Background(), query); err != nil {
					t.Fatal(err)
				}
				db.mu.Lock()
				delete(db.items, 1)
				db.mu.Unlock()
			}
			cache.mu.Lock()
			switch failure {
			case "read":
				cache.readErr = errors.New("redis read failure")
			case "write":
				cache.writeErr = errors.New("redis write failure")
			case "invalidate":
				cache.invalidateErr = errors.New("redis invalidation failure")
			}
			cache.mu.Unlock()
			for i := 0; i < 2; i++ {
				result, err := s.ListArticleByPage(context.Background(), query)
				if err != nil {
					t.Fatal(err)
				}
				if failure == "invalidate" {
					assertArticleListPage(t, result, []uint64{2, 3}, 4)
				} else {
					assertArticleListPage(t, result, []uint64{1, 2}, 5)
				}
			}
		})
	}
}

func TestArticleListCacheLateOldLoadCannotPoisonNextGeneration(t *testing.T) {
	s, db, _ := newArticleListCacheFixture()
	gate := make(chan struct{})
	started := make(chan struct{}, 1)
	db.started, db.gate = started, gate
	query := &dto.ArticlePageQuery{PageNum: 1, PageSize: 2}
	oldDone := make(chan error, 1)
	go func() {
		_, err := s.ListArticleByPage(context.Background(), query)
		oldDone <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("old page read did not start")
	}
	db.mu.Lock()
	updated := db.items[1]
	updated.Summary = "new generation"
	db.items[1] = updated
	db.started, db.gate = nil, nil
	db.mu.Unlock()
	s.cache.invalidate()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := s.ListArticleByPage(ctx, query)
	if err != nil || articleListRows(t, result)[0]["summary"] != "new generation" {
		close(gate)
		t.Fatalf("new request joined old load: result=%v err=%v", result, err)
	}
	close(gate)
	select {
	case err := <-oldDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("old load did not complete")
	}
	result, err = s.ListArticleByPage(context.Background(), query)
	if err != nil || articleListRows(t, result)[0]["summary"] != "new generation" {
		t.Fatalf("late old load poisoned cache: result=%v err=%v", result, err)
	}
	if wide, _ := db.counts(); wide != 2 {
		t.Fatalf("wide reads=%d; expected one per generation", wide)
	}
}

func TestArticleListCacheConcurrentColdRequestsShareWideRead(t *testing.T) {
	s, db, _ := newArticleListCacheFixture()
	gate := make(chan struct{})
	started := make(chan struct{}, 1)
	db.started, db.gate = started, gate
	const callers = 20
	done := make(chan error, callers)
	for i := 0; i < callers; i++ {
		go func() {
			result, err := s.ListArticleByPage(context.Background(), &dto.ArticlePageQuery{PageNum: 1, PageSize: 2})
			if err == nil && (result == nil || result.Total != 5) {
				err = fmt.Errorf("unexpected result: %+v", result)
			}
			done <- err
		}()
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("shared read did not start")
	}
	close(gate)
	for i := 0; i < callers; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("caller did not complete")
		}
	}
	if wide, state := db.counts(); wide != 1 || state != callers {
		t.Fatalf("wide reads=%d state reads=%d; want 1 and %d", wide, state, callers)
	}
}

func TestArticleListCacheAuthorRemainsDynamicAndDeduplicatedPerPage(t *testing.T) {
	s, db, cache := newArticleListCacheFixture()
	for _, id := range []uint64{1, 2} {
		item := db.items[id]
		item.UserID = 9
		db.items[id] = item
	}
	users := &fakeUserRepo{user: &model.UserModel{ID: 9, NickName: "before-profile-change", Avatar: "/before.png"}}
	s.userDomainService = domainservice.NewUserDomainService(users)
	query := &dto.ArticlePageQuery{PageNum: 1, PageSize: 2}
	if _, err := s.ListArticleByPage(context.Background(), query); err != nil {
		t.Fatal(err)
	}
	users.user.NickName, users.user.Avatar = "after-profile-change", "/after.png"
	result, err := s.ListArticleByPage(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range articleListRows(t, result) {
		author, ok := row["author"].(*dto.AuthorInfo)
		if !ok || author == nil || author.Name != "after-profile-change" || author.Avatar != "/after.png" {
			t.Fatalf("stale author profile: %+v", row["author"])
		}
	}
	if wide, _ := db.counts(); wide != 1 || users.calls != 2 {
		t.Fatalf("wide reads=%d author reads=%d; want 1 and 2", wide, users.calls)
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	for _, raw := range cache.data {
		if strings.Contains(string(raw), "before-profile-change") || strings.Contains(string(raw), "/before.png") {
			t.Fatal("page cache contains author profile")
		}
	}
}
