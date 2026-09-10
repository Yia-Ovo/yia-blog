package consts

import "time"

// MySQL 连接池默认值用于未配置或非法的上限，避免退回无限连接。
const (
	DefaultMySQLMaxOpen         = 20
	DefaultMySQLConnMaxLifetime = 5 * time.Minute
)
