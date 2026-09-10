package repo

import (
	"blog-server/domain/do"
	"blog-server/infra/model"
	"context"
	"time"

	"gorm.io/gorm"
)

// ReviewComment 用单条条件更新保护审核期间的并发编辑；只写审核状态与更新时间。
func (r *commentMysqlRepo) ReviewComment(ctx context.Context, review *do.CommentReview) (int64, error) {
	if review == nil || review.CommentID == 0 {
		return 0, gorm.ErrInvalidData
	}
	// 按字节比较，不能让 MySQL 的大小写不敏感排序规则把不同正文视为相同。
	result := r.db.WithContext(ctx).Model(&model.CommentModel{}).
		Where("id = ? AND status = ? AND BINARY content = BINARY ?", review.CommentID, review.ExpectedStatus, review.ExpectedContent).
		Updates(map[string]interface{}{"status": review.Status, "update_at": time.Now()})
	return result.RowsAffected, result.Error
}
