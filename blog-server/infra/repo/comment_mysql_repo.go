/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-06 11:40:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-16 19:20:58
 * @FilePath: \yia-blog\blog-server\infra\repo\comment_mysql_repo.go
 * @Description: 评论 MySQL 仓储实现
 */
package repo

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/infra/model"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// commentOrderColumns 评论列表允许的排序字段白名单，防止非法列名导致 SQL 报错或列名泄露
var commentOrderColumns = map[string]struct{}{
	"create_at": {},
	"likes":     {},
	"id":        {},
}

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
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// ListCommentByPage 分页查询评论列表 + 总数
func (r *commentMysqlRepo) ListCommentByPage(ctx context.Context, query *do.CommentQuery) (list []*model.CommentModel, total int64, err error) {
	tx := r.db.WithContext(ctx).Model(&model.CommentModel{})
	if query.PublishedArticleOnly {
		tx = tx.Where("EXISTS (SELECT 1 FROM `article` WHERE `article`.`id` = `comment`.`article_id` AND `article`.`status` = ? AND `article`.`deleted_at` IS NULL)", consts.ArticleStatusPublished)
	}

	// 1. 构建查询条件
	// 文章ID
	if query.ArticleID > 0 {
		tx = tx.Where("article_id = ?", query.ArticleID)
	}
	// 父评论ID：nil 表示不筛选；指向 0 查顶级，指向 N 查某父评论的回复
	if query.ParentID != nil {
		tx = tx.Where("parent_id = ?", *query.ParentID)
	}
	// 评论状态：nil 表示不筛选；可显式筛选待审核(0)/已通过(1)/已拒绝(2)
	if query.Status != nil {
		tx = tx.Where("status = ?", *query.Status)
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
	// 排序字段白名单校验：非白名单字段一律回退默认列，避免 SQL 报错或列名泄露
	orderBy := query.OrderBy
	if _, ok := commentOrderColumns[orderBy]; !ok {
		orderBy = "create_at"
	}
	tx = tx.Order(clause.OrderByColumn{
		Column: clause.Column{Name: orderBy},
		Desc:   query.OrderDir != "asc", // 仅 asc 升序，其余（含空）一律降序
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

// UpdateCommentByUser 按归属关系更新评论正文和审核状态
func (r *commentMysqlRepo) UpdateCommentByUser(
	ctx context.Context,
	commentID uint64,
	userID uint64,
	content string,
	status int,
) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&model.CommentModel{}).
		Where("id = ? AND user_id = ?", commentID, userID).
		Updates(map[string]interface{}{
			"content":   content,
			"status":    status,
			"update_at": time.Now(),
		})

	return result.RowsAffected, result.Error
}

// DeleteCommentByUser 按归属关系软删除评论
func (r *commentMysqlRepo) DeleteCommentByUser(
	ctx context.Context,
	commentID uint64,
	userID uint64,
) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", commentID, userID).
		Delete(&model.CommentModel{})
	return result.RowsAffected, result.Error
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
