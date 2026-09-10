/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-07 02:04:38
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-08 14:00:29
 * @FilePath: \yia-blog\blog-server\infra\repo\article_mysql_repo.go
 * @Description: 文章 MySQL 仓储层
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package repo

import (
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/infra/model"
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type articleMysqlRepo struct {
	db *gorm.DB
}

func NewArticleMysqlRepo(db *gorm.DB) *articleMysqlRepo {
	return &articleMysqlRepo{db: db}
}

// --------------------- 1. 创建文章 ---------------------
func (a *articleMysqlRepo) CreateArticle(ctx context.Context, article *model.ArticleModel) error {
	if err := a.db.WithContext(ctx).Create(article).Error; err != nil {
		logger.Error("[MYSQL] 创建文章失败", zap.String("title", article.Title), zap.Error(err))
		return err
	}
	return nil
}

// --------------------- 2. 文章分页查询 ---------------------
func (a *articleMysqlRepo) ListArticleByPage(ctx context.Context, query *do.ArticleQuery) ([]*model.ArticleModel, int64, error) {
	var (
		list  []*model.ArticleModel
		total int64
		db    = a.db.WithContext(ctx).Model(&model.ArticleModel{})
	)

	// 1. 拼接查询条件
	if query.Title != "" {
		db = db.Where("title LIKE ?", "%"+query.Title+"%")
	}
	if query.CategoryID != nil {
		db = db.Where("category_id = ?", *query.CategoryID)
	}
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}
	if query.IsTop != nil {
		db = db.Where("is_top = ?", *query.IsTop)
	}
	if query.UserID != nil {
		db = db.Where("user_id = ? AND admin_id = ?", *query.UserID, 0)
	}

	// 2. 查询总条数
	if err := db.Count(&total).Error; err != nil {
		logger.Error("[MYSQL] 文章分页查询统计失败", zap.Any("query", query), zap.Error(err))
		return nil, 0, err
	}
	if total == 0 {
		return []*model.ArticleModel{}, 0, nil
	}

	// 3. 分页查询
	offset := (query.PageNum - 1) * query.PageSize
	err := db.
		Order("sort ASC, publish_at DESC, id DESC").
		Limit(query.PageSize).
		Offset(offset).
		Find(&list).Error

	if err != nil {
		logger.Error("[MYSQL] 文章分页查询失败", zap.Any("query", query), zap.Error(err))
		return nil, 0, err
	}

	return list, total, nil
}

// --------------------- 3. 根据ID查询文章 ---------------------
func (a *articleMysqlRepo) GetArticleById(ctx context.Context, id uint64) (*model.ArticleModel, error) {
	var article model.ArticleModel
	err := a.db.WithContext(ctx).First(&article, id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Warn("[MYSQL] 文章不存在", zap.Uint64("article_id", id))
		return nil, nil
	}
	if err != nil {
		logger.Error("[MYSQL] 查询文章失败", zap.Uint64("article_id", id), zap.Error(err))
		return nil, err
	}

	return &article, nil
}

// --------------------- 4. 删除文章（软删除） ---------------------
func (a *articleMysqlRepo) DeleteArticle(ctx context.Context, id uint64) error {
	result := a.db.WithContext(ctx).Delete(&model.ArticleModel{}, id)

	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 删除文章失败", zap.Uint64("article_id", id), zap.Error(err))
		return err
	}
	if result.RowsAffected == 0 {
		logger.Warn("[MYSQL] 删除文章无数据影响", zap.Uint64("article_id", id))
	}
	return nil
}

// --------------------- 5. 更新文章 ---------------------
// UpdateArticle 只写可编辑列，作者、计数、创建时间与软删除标记由各自流程维护。
// Updates 不会像 Save 那样在目标已删除时回退为 INSERT，也不回写读取时的旧计数。
func (a *articleMysqlRepo) UpdateArticle(ctx context.Context, article *model.ArticleModel) error {
	if article == nil || article.ID == 0 {
		return gorm.ErrInvalidData
	}
	result := a.db.WithContext(ctx).Model(&model.ArticleModel{}).
		Where("id = ?", article.ID).
		Updates(map[string]interface{}{
			"title":          article.Title,
			"cover":          article.Cover,
			"summary":        article.Summary,
			"content":        article.Content,
			"category_id":    article.CategoryID,
			"tag_ids":        article.TagIDs,
			"status":         article.Status,
			"sort":           article.Sort,
			"is_top":         article.IsTop,
			"comment_status": article.CommentStatus,
			"publish_at":     article.PublishAt,
			"update_at":      time.Now(),
		})

	if err := result.Error; err != nil {
		logger.Error("[MYSQL] 更新文章失败", zap.Uint64("article_id", article.ID), zap.String("title", article.Title), zap.Error(err))
		return err
	}
	if result.RowsAffected == 0 {
		logger.Warn("[MYSQL] 更新文章无数据影响", zap.Uint64("article_id", article.ID))
	}
	return nil
}
