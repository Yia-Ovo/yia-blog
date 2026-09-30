package ioc

import (
	"blog-server/common/consts"

	"github.com/redis/go-redis/v9"
)

// newReadCacheClient 为可降级的内容读取缓存建立独立连接池。
// 继承地址、认证和数据库配置，但不修改鉴权客户端的超时或生命周期。
// 本项目使用默认直连；此客户端保留 TLS 配置，但不继承自定义 Dialer。
func newReadCacheClient(source *redis.Client) *redis.Client {
	if source == nil {
		return nil
	}
	opts := *source.Options()
	opts.ContextTimeoutEnabled = true
	opts.DialTimeout = consts.ReadCacheOperationTimeout
	opts.ReadTimeout = consts.ReadCacheOperationTimeout
	opts.WriteTimeout = consts.ReadCacheOperationTimeout
	opts.PoolTimeout = consts.ReadCacheOperationTimeout
	opts.MaxRetries = -1
	opts.PoolSize = consts.ReadCachePoolSize
	opts.MinIdleConns = 0
	opts.MaxIdleConns = consts.ReadCachePoolSize
	opts.MaxActiveConns = consts.ReadCachePoolSize
	// Options 含 source 初始化后的通知处理器，不能共享客户端维护通知的运行态。
	opts.PushNotificationProcessor = nil
	// source 的默认 Dialer 闭包持有原 Options，必须重建才能应用新的拨号超时。
	opts.Dialer = redis.NewDialer(&opts)
	return redis.NewClient(&opts)
}
