package consts

import "time"

// DefaultShutdownTimeout 是未配置或非正关闭时长时的保底期限。
const DefaultShutdownTimeout = 10 * time.Second

// MySQL 连接池默认值用于未配置或非法的上限，避免退回无限连接。
const (
	DefaultMySQLMaxOpen         = 20
	DefaultMySQLConnMaxLifetime = 5 * time.Minute
)
