package repo

import (
	"blog-server/domain/do"
	"context"
)

// CommentReviewRepo 只负责审核条件更新，不暴露持久化模型或审核权限判断。
type CommentReviewRepo interface {
	// ReviewComment 在评论仍存在且正文/状态未变化时更新审核状态，未命中返回零行。
	ReviewComment(ctx context.Context, review *do.CommentReview) (int64, error)
}
