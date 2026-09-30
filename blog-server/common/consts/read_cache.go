package consts

import "time"

const (
	// ReadCacheKeyPrefix 与未接入业务的旧缓存隔离，避免读取旧结构。
	ReadCacheKeyPrefix         = "blog:read-cache:v1:"
	ReadCacheArticleNamespace  = "article"
	ReadCacheCategoryNamespace = "category"

	ReadCacheTTL              = 5 * time.Minute
	ReadCacheNegativeTTL      = 30 * time.Second
	ReadCacheMaxBytes         = 2 << 20
	ReadCacheOperationTimeout = 200 * time.Millisecond
	ReadCacheLoadTimeout      = 2 * time.Second
	// ReadCacheJitterDivisor 将正常 TTL 抖动限制为基准值的正负 20%。
	ReadCacheJitterDivisor = 5
)
