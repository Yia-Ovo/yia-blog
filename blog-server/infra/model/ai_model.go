package model

import "time"

// AIStateModel 保存私有运行数据，不属于公开博客内容，不参与 GORM 建表。
type AIStateModel struct {
	Version          int                   `json:"version"`
	AdminID          uint64                `json:"admin_id"`
	PrivateKnowledge bool                  `json:"private_knowledge"`
	Memory           string                `json:"memory"`
	GeneralMemory    string                `json:"general_memory"`
	Day              string                `json:"day"`
	Requests         int                   `json:"requests"`
	RecentRequests   []time.Time           `json:"recent_requests"`
	Conversations    []AIConversationModel `json:"conversations"`
}

type AIConversationModel struct {
	OriginImportID string           `json:"origin_import_id,omitempty"`
	ContextMode    string           `json:"context_mode"`
	ID             string           `json:"id"`
	Title          string           `json:"title"`
	UpdateAt       time.Time        `json:"update_at"`
	Messages       []AIMessageModel `json:"messages"`
}

type AIMessageModel struct {
	ID        string          `json:"id"`
	RequestID string          `json:"request_id"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	Status    string          `json:"status"`
	CreateAt  time.Time       `json:"create_at"`
	Sources   []AISourceModel `json:"sources"`
}

type AISourceModel struct {
	Path      string    `json:"path"`
	Line      int       `json:"line"`
	UpdatedAt time.Time `json:"updated_at"`
	Excerpt   string    `json:"excerpt"`
}
