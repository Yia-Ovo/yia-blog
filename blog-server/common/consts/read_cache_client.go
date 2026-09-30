package consts

// ReadCachePoolSize 限制可降级内容缓存占用的连接数，避免拖累鉴权使用的独立连接池。
const ReadCachePoolSize = 4
