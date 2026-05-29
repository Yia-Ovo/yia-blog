/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-07 02:10:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-07 03:18:34
 * @FilePath: \yia-blog\blog-server\domain\repo\article_repo.go
 * @Description: 文章仓储层接口
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import (
	"blog-server/domain/do"
	"blog-server/infra/model"
	"context"
)

// ArticleMysqlRepo 文章MySQL仓储接口
type ArticleMysqlRepo interface {
	// CreateArticle 创建文章
	CreateArticle(ctx context.Context, articleModel *model.ArticleModel) error
	// ListArticleByPage 文章分页查询
	ListArticleByPage(ctx context.Context, query *do.ArticleQuery) ([]*model.ArticleModel, int64, error)
	// GetArticleById 根据ID查询文章
	GetArticleById(ctx context.Context, id uint64) (*model.ArticleModel, error)
	// DeleteArticle 删除文章（软删除）
	DeleteArticle(ctx context.Context, id uint64) error
	// UpdateArticle 更新文章
	UpdateArticle(ctx context.Context, articleModel *model.ArticleModel) error
}

// ArticleRedisRepo 文章Redis仓储接口
type ArticleRedisRepo interface {
	// GetArticleFromRedis 从Redis获取单个文章信息
	GetArticleFromRedis(ctx context.Context, articleID uint64) (*model.ArticleModel, error)
	// SetArticleToRedis 将单个文章写入Redis
	SetArticleToRedis(ctx context.Context, articleID uint64, article *model.ArticleModel) error
	// DelArticleFromRedis 删除单个文章缓存
	DelArticleFromRedis(ctx context.Context, articleID uint64) error

	// GetArticleListFromRedis 从Redis获取文章列表
	GetArticleListFromRedis(ctx context.Context) ([]model.ArticleModel, error)
	// SetArticleListToRedis 将文章列表写入Redis
	SetArticleListToRedis(ctx context.Context, articleList []model.ArticleModel) error
	// DelArticleListFromRedis 删除文章列表缓存
	DelArticleListFromRedis(ctx context.Context) error
}
