package repo

import "context"

// ReadCacheRepo 保存应用层的查询快照；generation 用于阻止失效前的慢查询重新写回旧数据。
// 数据编码、负缓存含义及缓存回源流程由 App 维护，领域服务不调用此接口。
type ReadCacheRepo interface {
	// Read 原子读取命名空间代次和内容。未命中返回 nil、有效代次及 ErrCacheMiss；
	// 其他错误返回空代次，调用者不能在一次未知状态的读取之后回填缓存。
	Read(ctx context.Context, key string) (data []byte, generation string, err error)
	// Write 仅在代次仍然相同时写入；false,nil 表示已有更新使本次回填过时。
	// negative 只控制短 TTL，空值占位由应用层编码在 data 中。
	Write(ctx context.Context, key, generation string, data []byte, negative bool) (bool, error)
	// Invalidate 切换整个命名空间的代次；旧内容靠原有 TTL 回收，不遍历或清空 Redis。
	Invalidate(ctx context.Context) error
}
