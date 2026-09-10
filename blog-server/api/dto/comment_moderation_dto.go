package dto

import "time"

// CommentModerationPageQueryReq 在公开查询条件上增加审核状态，仅用于管理员入口。
type CommentModerationPageQueryReq struct {
	CommentPageQueryReq
	Status *int `form:"status" binding:"omitempty,oneof=0 1 2"`
}

// CommentReviewReq 只提交审核决定和查看时的正文/状态；操作者从管理员鉴权态取得。
// 指针区分缺失与合法零值，预期正文可为空以便处理历史异常评论。
type CommentReviewReq struct {
	ID              uint64  `json:"id" binding:"required"`
	ExpectedContent *string `json:"expected_content" binding:"required"`
	ExpectedStatus  *int    `json:"expected_status" binding:"required,oneof=0 1 2"`
	Status          *int    `json:"status" binding:"required,oneof=1 2"`
}

// ManagedComment 提供审核所需字段，不输出邮箱、IP、User-Agent 等请求来源信息。
type ManagedComment struct {
	ID        uint64    `json:"id"`
	ArticleID uint64    `json:"article_id"`
	ParentID  uint64    `json:"parent_id"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Content   string    `json:"content"`
	Status    int       `json:"status"`
	Likes     int       `json:"likes"`
	CreateAt  time.Time `json:"create_at"`
	UpdateAt  time.Time `json:"update_at"`
}
