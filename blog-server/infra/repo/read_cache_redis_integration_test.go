package repo

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"

	"github.com/goccy/go-yaml"
	"github.com/redis/go-redis/v9"
)

// TestReadCacheRedisIntegration 仅在显式提供测试地址时使用真实 Redis 执行 Lua。
// 每次运行都生成独立随机前缀，清理仅针对自己创建并登记的键，不 FLUSHDB 或扫描业务键。
func TestReadCacheRedisIntegration(t *testing.T) {
	address := os.Getenv("READ_CACHE_TEST_REDIS_ADDR")
	password := os.Getenv("READ_CACHE_TEST_REDIS_PASSWORD")
	database := 0
	configPath := os.Getenv("READ_CACHE_TEST_REDIS_CONFIG")
	if address == "" && configPath == "" {
		t.Skip("set READ_CACHE_TEST_REDIS_ADDR or READ_CACHE_TEST_REDIS_CONFIG to run isolated Redis integration")
	}
	if configPath != "" {
		// 只提取 Redis 连接信息，避免把密码放进命令行参数或测试输出。
		var connection struct {
			Redis struct {
				Addr     string `yaml:"addr"`
				Password string `yaml:"password"`
				DB       int    `yaml:"db_index"`
			} `yaml:"redis"`
		}
		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal("cannot read Redis integration configuration")
		}
		if err := yaml.Unmarshal(content, &connection); err != nil {
			t.Fatal("cannot parse Redis integration configuration")
		}
		address, password, database = connection.Redis.Addr, connection.Redis.Password, connection.Redis.DB
		if address == "" {
			t.Fatal("Redis integration configuration has no address")
		}
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: address, Password: password, DB: database,
		DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second,
		ContextTimeoutEnabled: true, MaxRetries: -1,
	})
	t.Cleanup(func() { _ = rdb.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("test Redis unavailable: %v", err)
	}
	newIsolatedRepo := func(t *testing.T) (*ReadCacheRedisRepo, func(string, string)) {
		t.Helper()
		id, err := readCacheGeneration()
		if err != nil {
			t.Fatal(err)
		}
		r := NewReadCacheRedisRepo(rdb, consts.ReadCacheArticleNamespace)
		r.prefix += "test:" + id + ":"
		keys := []string{r.prefix + "generation"}
		t.Cleanup(func() {
			cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := rdb.Del(cleanup, keys...).Err(); err != nil {
				t.Errorf("cleanup own cache test keys: %v", err)
			}
		})
		return r, func(key, generation string) {
			keys = append(keys, r.prefix+"data:"+generation+":"+readCacheQueryHash(key))
		}
	}

	t.Run("initialization write hit TTL and invalidation", func(t *testing.T) {
		r, track := newIsolatedRepo(t)
		data, first, err := r.Read(ctx, "one")
		if data != nil || !validReadCacheGeneration(first) || !errors.Is(err, domainrepo.ErrCacheMiss) {
			t.Fatalf("initial read = %v, %q, %v", data, first, err)
		}
		if ttl := rdb.PTTL(ctx, r.prefix+"generation").Val(); ttl != -1 {
			t.Fatalf("generation expires: %v", ttl)
		}
		track("one", first)
		if ok, err := r.Write(ctx, "one", first, []byte(`{"title":"one"}`), false); err != nil || !ok {
			t.Fatalf("write = %v, %v", ok, err)
		}
		if ttl := rdb.PTTL(ctx, r.prefix+"data:"+first+":"+readCacheQueryHash("one")).Val(); ttl < 4*time.Minute-time.Second || ttl > 6*time.Minute {
			t.Fatalf("normal TTL = %v", ttl)
		}
		data, actual, err := r.Read(ctx, "one")
		if string(data) != `{"title":"one"}` || actual != first || err != nil {
			t.Fatalf("hit = %q, %q, %v", data, actual, err)
		}
		if err := r.Invalidate(ctx); err != nil {
			t.Fatal(err)
		}
		if ok, err := r.Write(ctx, "one", first, []byte("stale result"), false); ok || err != nil {
			t.Fatalf("stale write = %v, %v", ok, err)
		}
		data, second, err := r.Read(ctx, "one")
		if data != nil || second == first || !errors.Is(err, domainrepo.ErrCacheMiss) {
			t.Fatalf("invalidated read = %q, %q, %v", data, second, err)
		}
		track("one", second)
		if ok, err := r.Write(ctx, "one", second, []byte(`{"missing":true}`), true); err != nil || !ok {
			t.Fatalf("negative write = %v, %v", ok, err)
		}
		if ttl := rdb.PTTL(ctx, r.prefix+"data:"+second+":"+readCacheQueryHash("one")).Val(); ttl <= 28*time.Second || ttl > 30*time.Second {
			t.Fatalf("negative TTL = %v", ttl)
		}
		// generation 丢失后新建随机代次，不会重新接受先前 DB 查询的回填。
		if err := rdb.Del(ctx, r.prefix+"generation").Err(); err != nil {
			t.Fatal(err)
		}
		_, third, err := r.Read(ctx, "one")
		if !errors.Is(err, domainrepo.ErrCacheMiss) || third == first || third == second {
			t.Fatalf("reinitialized generation = %q, %v", third, err)
		}
		if ok, err := r.Write(ctx, "one", second, []byte("ABA stale"), false); ok || err != nil {
			t.Fatalf("ABA write = %v, %v", ok, err)
		}
	})

	t.Run("concurrent first reads share one generation", func(t *testing.T) {
		r, _ := newIsolatedRepo(t)
		const count = 20
		generations := make([]string, count)
		errorsFound := make([]error, count)
		var wg sync.WaitGroup
		for i := range count {
			wg.Add(1)
			go func() { defer wg.Done(); _, generations[i], errorsFound[i] = r.Read(ctx, "concurrent") }()
		}
		wg.Wait()
		for i := range count {
			if !errors.Is(errorsFound[i], domainrepo.ErrCacheMiss) || generations[i] == "" || generations[i] != generations[0] {
				t.Fatalf("concurrent read %d = %q, %v", i, generations[i], errorsFound[i])
			}
		}
	})

	t.Run("corrupt generation and excessive payload fail closed", func(t *testing.T) {
		r, track := newIsolatedRepo(t)
		if err := rdb.Set(ctx, r.prefix+"generation", "corrupt", 0).Err(); err != nil {
			t.Fatal(err)
		}
		if data, generation, err := r.Read(ctx, "large"); err == nil || generation != "" || data != nil {
			t.Fatalf("corrupt read = %d bytes, %q, %v", len(data), generation, err)
		}
		if err := r.Invalidate(ctx); err != nil {
			t.Fatal(err)
		}
		_, generation, err := r.Read(ctx, "large")
		if !errors.Is(err, domainrepo.ErrCacheMiss) || generation == "" {
			t.Fatalf("repair = %q, %v", generation, err)
		}
		track("large", generation)
		dataKey := r.prefix + "data:" + generation + ":" + readCacheQueryHash("large")
		if err := rdb.Set(ctx, dataKey, strings.Repeat("x", consts.ReadCacheMaxBytes+1), time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
		if data, generation, err := r.Read(ctx, "large"); err == nil || generation != "" || data != nil {
			t.Fatalf("oversize read = %d bytes, %q, %v", len(data), generation, err)
		}
	})
}
