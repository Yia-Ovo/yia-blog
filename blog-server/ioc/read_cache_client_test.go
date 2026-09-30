package ioc

import (
	"blog-server/common/consts"
	"crypto/tls"
	"reflect"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewReadCacheClientNil(t *testing.T) {
	if newReadCacheClient(nil) != nil {
		t.Fatal("nil source must keep caching disabled")
	}
}

func TestNewReadCacheClientUsesBoundedIndependentOptions(t *testing.T) {
	source := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:1", Username: "test-user", Password: "test-only-placeholder", DB: 8,
		Protocol: 3, DialTimeout: 3 * time.Second, ReadTimeout: 4 * time.Second,
		WriteTimeout: 5 * time.Second, PoolTimeout: 6 * time.Second,
		MaxRetries: 2, PoolSize: 20, MaxIdleConns: 10, MaxActiveConns: 30,
		ConnMaxIdleTime: time.Minute,
		TLSConfig:       &tls.Config{MinVersion: tls.VersionTLS12, ServerName: "test.invalid"},
	})
	t.Cleanup(func() { _ = source.Close() })
	before := *source.Options()
	client := newReadCacheClient(source)
	t.Cleanup(func() { _ = client.Close() })
	if client == source || client.Options() == source.Options() {
		t.Fatal("content cache must not alias the authentication client")
	}
	got := client.Options()
	if got.PushNotificationProcessor == before.PushNotificationProcessor || source.Options().PushNotificationProcessor != before.PushNotificationProcessor {
		t.Fatal("RESP3 notification processor must be independent from the source client")
	}
	if got.Addr != before.Addr || got.Network != before.Network || got.Username != before.Username || got.Password != before.Password || got.DB != before.DB || got.Protocol != before.Protocol || got.TLSConfig != before.TLSConfig || got.ConnMaxIdleTime != before.ConnMaxIdleTime {
		t.Fatal("connection identity or unrelated connection options changed")
	}
	if !got.ContextTimeoutEnabled || got.DialTimeout != consts.ReadCacheOperationTimeout || got.ReadTimeout != consts.ReadCacheOperationTimeout || got.WriteTimeout != consts.ReadCacheOperationTimeout || got.PoolTimeout != consts.ReadCacheOperationTimeout {
		t.Fatal("read cache does not enforce its independent timeout budget")
	}
	// go-redis 会将构造参数 MaxRetries=-1 归一化为 0，即不重试。
	if got.MaxRetries != 0 || got.PoolSize != consts.ReadCachePoolSize || got.MinIdleConns != 0 || got.MaxIdleConns != consts.ReadCachePoolSize || got.MaxActiveConns != consts.ReadCachePoolSize {
		t.Fatal("read cache retry or connection bounds differ from the configured limits")
	}
	after := source.Options()
	if after.ContextTimeoutEnabled != before.ContextTimeoutEnabled || after.DialTimeout != before.DialTimeout || after.ReadTimeout != before.ReadTimeout || after.WriteTimeout != before.WriteTimeout || after.PoolTimeout != before.PoolTimeout || after.MaxRetries != before.MaxRetries || after.PoolSize != before.PoolSize || after.MinIdleConns != before.MinIdleConns || after.MaxIdleConns != before.MaxIdleConns || after.MaxActiveConns != before.MaxActiveConns || reflect.ValueOf(after.Dialer).Pointer() != reflect.ValueOf(before.Dialer).Pointer() {
		t.Fatal("constructing content cache changed source options")
	}
}

func TestClosingReadCacheClientLeavesSourcePoolOpen(t *testing.T) {
	// MinIdleConns 为 0，不执行任何命令，因此无需真实 Redis 或网络连接。
	source := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", Protocol: 2})
	t.Cleanup(func() { _ = source.Close() })
	client := newReadCacheClient(source)
	if err := client.Close(); err != nil {
		t.Fatal("closing read cache failed")
	}
	// 若两个 Client 共享连接池，第二次关闭会返回 redis.ErrClosed。
	if err := source.Close(); err != nil {
		t.Fatal("closing read cache also closed the source connection pool")
	}
}
