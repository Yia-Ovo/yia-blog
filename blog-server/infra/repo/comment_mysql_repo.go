/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-06 11:40:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-22 00:52:14
 * @FilePath: \yia-blog\blog-server\infra\repo\comment_mysql_repo.go
 * @Description: 评论 MySQL 仓储实现
 */
package repo

import (
	"blog-server/domain/do"
	"blog-server/infra/model"
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// commentMysqlRepo CommentMysqlRepo 接口实现
type commentMysqlRepo struct {
	db *gorm.DB
}

// NewCommentMysqlRepo 创建评论MySQL仓储实例（依赖注入）
func NewCommentMysqlRepo(db *gorm.DB) *commentMysqlRepo {
	return &commentMysqlRepo{
		db: db,
	}
}

// CreateComment 创建评论
func (r *commentMysqlRepo) CreateComment(ctx context.Context, commentModel *model.CommentModel) error {
	return r.db.WithContext(ctx).Create(commentModel).Error
}

// GetCommentByID 根据ID查询评论
func (r *commentMysqlRepo) GetCommentByID(ctx context.Context, commentID uint64) (*model.CommentModel, error) {
	var comment model.CommentModel
	err := r.db.WithContext(ctx).First(&comment, commentID).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// ListCommentByPage 分页查询评论列表 + 总数
func (r *commentMysqlRepo) ListCommentByPage(ctx context.Context, query *do.CommentQuery) (list []*model.CommentModel, total int64, err error) {
	tx := r.db.WithContext(ctx).Model(&model.CommentModel{})

	// 1. 构建查询条件
	// 文章ID
	if query.ArticleID > 0 {
		tx = tx.Where("article_id = ?", query.ArticleID)
	}
	// 父评论ID
	if query.ParentID >= 0 {
		tx = tx.Where("parent_id = ?", query.ParentID)
	}
	// 评论状态
	if query.Status != 0 {
		tx = tx.Where("status = ?", query.Status)
	}
	// 管理员ID
	if query.AdminID > 0 {
		tx = tx.Where("admin_id = ?", query.AdminID)
	}
	// 昵称模糊搜索
	if query.Nickname != "" {
		tx = tx.Where("nickname LIKE ?", "%"+query.Nickname+"%")
	}
	// 内容模糊搜索
	if query.Content != "" {
		tx = tx.Where("content LIKE ?", "%"+query.Content+"%")
	}

	// 2. 查询总数
	if err = tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*model.CommentModel{}, 0, nil
	}

	// 3. 排序（默认按创建时间倒序）
	if query.OrderBy == "" {
		query.OrderBy = "create_at"
	}
	if query.OrderDir == "" {
		query.OrderDir = "desc"
	}
	tx = tx.Order(clause.OrderByColumn{
		Column: clause.Column{Name: query.OrderBy},
		Desc:   query.OrderDir == "desc",
	})

	// 4. 分页（默认值处理）
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Size <= 0 {
		query.Size = 10
	}
	tx = tx.Offset((query.Page - 1) * query.Size).Limit(query.Size)

	// 5. 查询列表
	err = tx.Find(&list).Error
	return list, total, err
}

// UpdateComment 更新评论
func (r *commentMysqlRepo) UpdateComment(ctx context.Context, commentModel *model.CommentModel) error {
	return r.db.WithContext(ctx).
		Model(commentModel).
		// 忽略不可更新字段
		Omit("id", "create_at", "deleted_at").
		Updates(commentModel).Error
}

// DeleteComment 软删除评论
func (r *commentMysqlRepo) DeleteComment(ctx context.Context, commentID uint64) error {
	return r.db.WithContext(ctx).Delete(&model.CommentModel{}, commentID).Error
}

// LikeComment 评论点赞（原子操作，并发安全）
func (r *commentMysqlRepo) LikeComment(ctx context.Context, commentID uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.CommentModel{}).
		Where("id = ?", commentID).
		UpdateColumn("likes", gorm.Expr("likes + 1")).Error
}

// DisLikeComment 取消评论点赞（原子操作，防止负数）
func (r *commentMysqlRepo) DisLikeComment(ctx context.Context, commentID uint64) error {
	return r.db.WithContext(ctx).
		Model(&model.CommentModel{}).
		Where("id = ? AND likes > 0", commentID).
		UpdateColumn("likes", gorm.Expr("likes - 1")).Error
}
