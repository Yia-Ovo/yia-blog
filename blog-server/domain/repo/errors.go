package repo

import "errors"

// ErrCacheMiss 表示缓存未命中。Repo 实现应在底层缓存（如 go-redis 的 redis.Nil）
// 未命中时返回它，避免把具体缓存驱动的错误类型泄漏到 App 层。
var ErrCacheMiss = errors.New("repo: cache miss")
