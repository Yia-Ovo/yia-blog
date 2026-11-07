package model

import "time"

// AIImportedThreadModel 从同步服务写入的受保护副本读取，账号归属由目录决定。
type AIImportedThreadModel struct {
	Version   int              `json:"version"`
	ID        string           `json:"id"`
	Title     string           `json:"title"`
	Workspace string           `json:"workspace"`
	UpdatedAt time.Time        `json:"updated_at"`
	Messages  []AIMessageModel `json:"messages"`
}
