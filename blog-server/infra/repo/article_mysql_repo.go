/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-07 02:04:38
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 16:52:21
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
		logger.Error("CreateArticle failed", zap.String("title", article.Title), zap.Error(err))
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

	// 2. 查询总条数
	if err := db.Count(&total).Error; err != nil {
		logger.Error("ListArticleByPage count failed", zap.Any("query", query), zap.Error(err))
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
		logger.Error("ListArticleByPage find failed", zap.Any("query", query), zap.Error(err))
		return nil, 0, err
	}

	return list, total, nil
}

// --------------------- 3. 根据ID查询文章 ---------------------
func (a *articleMysqlRepo) GetArticleById(ctx context.Context, id uint64) (*model.ArticleModel, error) {
	var article model.ArticleModel
	err := a.db.WithContext(ctx).First(&article, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("GetArticleById: article not found", zap.Uint64("article_id", id))
		} else {
			logger.Error("GetArticleById failed", zap.Uint64("article_id", id), zap.Error(err))
		}
		return nil, err
	}

	return &article, nil
}

// --------------------- 4. 删除文章（软删除） ---------------------
func (a *articleMysqlRepo) DeleteArticle(ctx context.Context, id uint64) error {
	err := a.db.WithContext(ctx).Delete(&model.ArticleModel{}, id).Error

	if err != nil {
		logger.Error("DeleteArticle failed", zap.Uint64("article_id", id), zap.Error(err))
		return err
	}
	if a.db.RowsAffected == 0 {
		logger.Warn("DeleteArticle no rows affected", zap.Uint64("article_id", id))
	}
	return nil
}

// --------------------- 5. 更新文章 ---------------------
func (a *articleMysqlRepo) UpdateArticle(ctx context.Context, article *model.ArticleModel) error {
	err := a.db.WithContext(ctx).Save(article).Error

	if err != nil {
		logger.Error("UpdateArticle failed", zap.Uint64("article_id", article.ID), zap.String("title", article.Title), zap.Error(err))
		return err
	}
	if a.db.RowsAffected == 0 {
		logger.Warn("UpdateArticle no rows affected", zap.Uint64("article_id", article.ID))
	}
	return nil
}
