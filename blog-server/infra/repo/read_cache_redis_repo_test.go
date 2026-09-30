package repo

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"

	"github.com/redis/go-redis/v9"
)

type readCacheCommandHook struct {
	process func(context.Context, redis.Cmder) error
}

func (h readCacheCommandHook) DialHook(next redis.DialHook) redis.DialHook     { return next }
func (h readCacheCommandHook) ProcessHook(redis.ProcessHook) redis.ProcessHook { return h.process }
func (h readCacheCommandHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func readCacheMockClient(t *testing.T, handler func(context.Context, redis.Cmder) error) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Dialer: func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("unexpected network access in unit test")
	}})
	rdb.AddHook(readCacheCommandHook{process: handler})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func TestReadCacheReadResponseAndKeys(t *testing.T) {
	generation := strings.Repeat("a", 32)
	for _, tc := range []struct {
		name      string
		response  interface{}
		wantMiss  bool
		wantError bool
		wantData  string
	}{
		{name: "hit", response: []interface{}{generation, "article snapshot"}, wantData: "article snapshot"},
		{name: "empty value is a hit", response: []interface{}{generation, ""}},
		{name: "miss retains generation", response: []interface{}{generation, nil}, wantMiss: true},
		{name: "invalid shape", response: []interface{}{generation}, wantError: true},
		{name: "invalid generation", response: []interface{}{"invalid", "data"}, wantError: true},
		{name: "invalid payload type", response: []interface{}{generation, int64(1)}, wantError: true},
		{name: "oversized payload", response: []interface{}{generation, strings.Repeat("x", consts.ReadCacheMaxBytes+1)}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rdb := readCacheMockClient(t, func(_ context.Context, cmd redis.Cmder) error {
				args := cmd.Args()
				if args[0] != "eval" || args[1] != readCacheReadScript || args[2] != 1 || args[3] != consts.ReadCacheKeyPrefix+"article:generation" {
					t.Fatalf("unexpected read command: %v", args[:4])
				}
				if !validReadCacheGeneration(args[4].(string)) || args[5] != consts.ReadCacheKeyPrefix+"article:data:" || args[6] != readCacheQueryHash("public:article:42") || args[7] != consts.ReadCacheMaxBytes {
					t.Fatal("read command omitted random generation, isolated prefix, query hash, or size bound")
				}
				cmd.(*redis.Cmd).SetVal(tc.response)
				return nil
			})
			data, gotGeneration, err := NewReadCacheRedisRepo(rdb, consts.ReadCacheArticleNamespace).Read(context.Background(), "public:article:42")
			if tc.wantError {
				if err == nil || gotGeneration != "" || data != nil {
					t.Fatalf("unsafe error response: %d bytes, %q, %v", len(data), gotGeneration, err)
				}
				return
			}
			if tc.wantMiss {
				if !errors.Is(err, domainrepo.ErrCacheMiss) || data != nil || gotGeneration != generation {
					t.Fatalf("miss = %q, %q, %v", data, gotGeneration, err)
				}
				return
			}
			if err != nil || string(data) != tc.wantData || gotGeneration != generation {
				t.Fatalf("hit = %q, %q, %v", data, gotGeneration, err)
			}
		})
	}
}

func TestReadCacheWriteTTLAndGenerationGuard(t *testing.T) {
	for _, negative := range []bool{false, true} {
		for _, accepted := range []int64{0, 1} {
			generation := strings.Repeat("b", 32)
			rdb := readCacheMockClient(t, func(_ context.Context, cmd redis.Cmder) error {
				args := cmd.Args()
				prefix := consts.ReadCacheKeyPrefix + "category:"
				if args[0] != "eval" || args[1] != readCacheWriteScript || args[2] != 2 || args[3] != prefix+"generation" || args[4] != prefix+"data:"+generation+":"+readCacheQueryHash("all") || args[5] != generation || string(args[6].([]byte)) != "[]" {
					t.Fatal("write did not include both generation and content keys")
				}
				ttl := time.Duration(args[7].(int64)) * time.Millisecond
				if negative {
					if ttl != consts.ReadCacheNegativeTTL {
						t.Fatalf("negative TTL: %v", ttl)
					}
				} else if ttl < 4*time.Minute || ttl > 6*time.Minute {
					t.Fatalf("normal TTL: %v", ttl)
				}
				cmd.(*redis.Cmd).SetVal(accepted)
				return nil
			})
			ok, err := NewReadCacheRedisRepo(rdb, consts.ReadCacheCategoryNamespace).Write(context.Background(), "all", generation, []byte("[]"), negative)
			if err != nil || ok != (accepted == 1) {
				t.Fatalf("write = %v, %v", ok, err)
			}
		}
	}
}

func TestReadCacheRejectsInvalidInputBeforeRedis(t *testing.T) {
	rdb := readCacheMockClient(t, func(context.Context, redis.Cmder) error { t.Fatal("invalid input reached Redis"); return nil })
	for _, repo := range []*ReadCacheRedisRepo{nil, {}, NewReadCacheRedisRepo(nil, "article"), NewReadCacheRedisRepo(rdb, ""), NewReadCacheRedisRepo(rdb, "arbitrary")} {
		if _, generation, err := repo.Read(context.Background(), "key"); err == nil || generation != "" {
			t.Fatal("invalid repo read succeeded")
		}
		if ok, err := repo.Write(context.Background(), "key", strings.Repeat("a", 32), nil, false); ok || err == nil {
			t.Fatal("invalid repo write succeeded")
		}
		if err := repo.Invalidate(context.Background()); err == nil {
			t.Fatal("invalid repo invalidate succeeded")
		}
	}
	r := NewReadCacheRedisRepo(rdb, "article")
	for _, generation := range []string{"", "0", strings.Repeat("A", 32), strings.Repeat("g", 32), strings.Repeat("a", 31)} {
		if ok, err := r.Write(context.Background(), "key", generation, nil, false); ok || err == nil {
			t.Fatal("invalid generation accepted")
		}
	}
	if ok, err := r.Write(context.Background(), "key", strings.Repeat("a", 32), make([]byte, consts.ReadCacheMaxBytes+1), false); ok || err == nil {
		t.Fatal("oversized payload accepted")
	}
}

func TestReadCacheFailureDoesNotPermitWriteback(t *testing.T) {
	failure := errors.New("test Redis failure")
	rdb := readCacheMockClient(t, func(context.Context, redis.Cmder) error { return failure })
	r := NewReadCacheRedisRepo(rdb, "article")
	if data, generation, err := r.Read(context.Background(), "key"); data != nil || generation != "" || !errors.Is(err, failure) {
		t.Fatalf("failure = %v, %q, %v", data, generation, err)
	}
	if ok, err := r.Write(context.Background(), "key", strings.Repeat("a", 32), nil, false); ok || !errors.Is(err, failure) {
		t.Fatalf("write = %v, %v", ok, err)
	}
	if err := r.Invalidate(context.Background()); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}

func TestReadCacheInvalidationRandomizesPermanentGeneration(t *testing.T) {
	previous := ""
	rdb := readCacheMockClient(t, func(_ context.Context, cmd redis.Cmder) error {
		args := cmd.Args()
		if len(args) != 3 || args[0] != "set" || args[1] != consts.ReadCacheKeyPrefix+"article:generation" {
			t.Fatal("generation must not have a TTL")
		}
		generation := args[2].(string)
		if !validReadCacheGeneration(generation) || generation == previous {
			t.Fatal("generation reused or invalid")
		}
		previous = generation
		cmd.(*redis.StatusCmd).SetVal("OK")
		return nil
	})
	r := NewReadCacheRedisRepo(rdb, "article")
	for i := 0; i < 100; i++ {
		if err := r.Invalidate(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReadCacheJitterIsBoundedAndVaried(t *testing.T) {
	seen := make(map[time.Duration]bool)
	for i := 0; i < 100; i++ {
		ttl := readCacheTTL(false)
		if ttl < 4*time.Minute || ttl > 6*time.Minute {
			t.Fatalf("out of range: %v", ttl)
		}
		seen[ttl] = true
	}
	if len(seen) < 2 {
		t.Fatal("TTL jitter never varied")
	}
	if readCacheTTL(true) != 30*time.Second {
		t.Fatal("negative TTL changed")
	}
}
