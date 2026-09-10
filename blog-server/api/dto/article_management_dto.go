package dto

import "time"

// ArticleIDQueryReq 私有文章详情参数；当前操作者只来自鉴权 Context。
type ArticleIDQueryReq struct {
	ID uint64 `form:"id" binding:"required"`
}

// ManagedArticle 管理接口的输出白名单。列表不填 Content，详情供编辑器加载完整可编辑字段。
type ManagedArticle struct {
	ID            uint64     `json:"id"`
	UserID        uint64     `json:"user_id"`
	AdminID       uint64     `json:"admin_id"`
	Title         string     `json:"title"`
	Cover         string     `json:"cover"`
	Summary       string     `json:"summary"`
	Content       string     `json:"content,omitempty"`
	CategoryID    uint64     `json:"category_id"`
	TagIDs        string     `json:"tag_ids"`
	Status        int        `json:"status"`
	Sort          int        `json:"sort"`
	IsTop         int        `json:"is_top"`
	CommentStatus int        `json:"comment_status"`
	PublishAt     *time.Time `json:"publish_at"`
	CreateAt      time.Time  `json:"create_at"`
	UpdateAt      time.Time  `json:"update_at"`
}
