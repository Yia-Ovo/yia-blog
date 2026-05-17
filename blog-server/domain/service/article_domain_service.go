/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-07 03:20:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-07 05:23:14
 * @FilePath: \yia-blog\blog-server\domain\service\article_domain_service.go
 * @Description: 文章领域服务
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/common"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
	"context"
	"errors"

	"go.uber.org/zap"
)

type ArticleDomainService struct {
	articleMysqlRepo repo.ArticleMysqlRepo
	articleRedisRepo repo.ArticleRedisRepo
}

func NewArticleDomainService(
	articleMysqlRepo repo.ArticleMysqlRepo,
	articleRedisRepo repo.ArticleRedisRepo,
) *ArticleDomainService {
	return &ArticleDomainService{
		articleMysqlRepo: articleMysqlRepo,
		articleRedisRepo: articleRedisRepo,
	}
}

// CreateArticle 创建文章
func (a *ArticleDomainService) CreateArticle(ctx context.Context, articleDO *do.ArticleDO) error {
	// 业务规则校验
	if articleDO.Title == "" {
		logger.Warn("创建文章失败：标题不能为空", zap.String("title", articleDO.Title))
		return common.ErrArticleTitleEmpty
	}
	if len(articleDO.Title) > 128 {
		logger.Warn("创建文章失败：标题超长", zap.String("title", articleDO.Title))
		return common.ErrArticleTitleTooLong
	}
	if articleDO.Sort < 0 {
		logger.Warn("创建文章失败：排序值非法", zap.Int("sort", articleDO.Sort))
		return common.ErrArticleSortInvalid
	}
	if articleDO.Content == "" {
		logger.Warn("创建文章失败：内容不能为空", zap.String("title", articleDO.Title))
		return common.ErrArticleContentEmpty
	}

	// DO转Model
	articleModel := convert.ArticleDOToModel(articleDO)

	// 调用MySQL仓储
	err := a.articleMysqlRepo.CreateArticle(ctx, articleModel)
	if err != nil {
		logger.Error("创建文章数据库操作失败", zap.String("title", articleDO.Title), zap.Error(err))
		return err
	}

	// 新增成功，删除列表缓存（保证一致性）
	_ = a.articleRedisRepo.DelArticleListFromRedis(ctx)
	logger.Info("创建文章成功，缓存已刷新", zap.String("title", articleDO.Title))
	return nil
}

// ListArticleByPage 文章分页查询
func (a *ArticleDomainService) ListArticleByPage(ctx context.Context, query *do.ArticleQuery) ([]do.ArticleDO, int64, error) {
	// 分页参数合法性校验
	if query.PageNum < 1 {
		query.PageNum = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 10
	}

	// 优先从Redis获取列表缓存
	cacheList, err := a.articleRedisRepo.GetArticleListFromRedis(ctx)
	if err == nil {
		logger.Info("文章列表命中缓存", zap.Int("cache_count", len(cacheList)))
		return convert.ArticleModelListToDOList(&cacheList), int64(len(cacheList)), nil
	}
	// 缓存异常/不存在，降级查询数据库
	if !errors.Is(err, common.CacheNotExist) {
		logger.Warn("文章列表缓存异常，降级查询DB", zap.Error(err))
	}

	// 查询数据库
	list, total, err := a.articleMysqlRepo.ListArticleByPage(ctx, query)
	if err != nil {
		logger.Error("分页查询文章失败", zap.Any("query", query), zap.Error(err))
		return nil, 0, common.DatabaseErr
	}

	// 查询成功，写入Redis缓存
	modelList := convert.ArticleDOListToModelList(&list)
	_ = a.articleRedisRepo.SetArticleListToRedis(ctx, modelList)

	logger.Info("分页查询文章成功", zap.Int64("total", total))
	return list, total, nil
}

// GetArticleById 根据ID查询文章
func (a *ArticleDomainService) GetArticleById(ctx context.Context, id uint64) (*do.ArticleDO, error) {
	// 参数校验
	if id <= 0 {
		logger.Warn("根据ID查询文章失败：ID非法", zap.Uint64("article_id", id))
		return nil, common.ParamErr
	}

	// 优先从Redis获取缓存
	articleModel, err := a.articleRedisRepo.GetArticleFromRedis(ctx, id)
	if err == nil {
		logger.Info("文章详情命中缓存", zap.Uint64("article_id", id))
		return convert.ArticleModelToDO(articleModel), nil
	}
	// 缓存异常/不存在，降级查询数据库
	if !errors.Is(err, common.CacheNotExist) {
		logger.Warn("文章详情缓存异常，降级查询DB", zap.Uint64("article_id", id), zap.Error(err))
	}

	// 查询数据库
	articleDO, err := a.articleMysqlRepo.GetArticleById(ctx, id)
	if err != nil {
		logger.Error("根据ID查询文章失败", zap.Uint64("article_id", id), zap.Error(err))
		return nil, common.DatabaseErr
	}

	// 数据不存在：写入空值缓存（防穿透）
	if articleDO == nil {
		_ = a.articleRedisRepo.SetArticleToRedis(ctx, id, &model.ArticleModel{})
		logger.Warn("文章不存在，写入空值缓存", zap.Uint64("article_id", id))
		return nil, common.ArticleNotFoundErr
	}

	// 数据存在：写入正常缓存
	model := convert.ArticleDOToModel(articleDO)
	_ = a.articleRedisRepo.SetArticleToRedis(ctx, id, model)

	logger.Info("根据ID查询文章成功", zap.Uint64("article_id", id))
	return articleDO, nil
}

// UpdateArticle 更新文章
func (a *ArticleDomainService) UpdateArticle(ctx context.Context, articleDO *do.ArticleDO) error {
	// 业务规则校验
	if articleDO.ID <= 0 {
		logger.Warn("更新文章失败：ID不能为空", zap.Uint64("article_id", articleDO.ID))
		return common.ParamErr
	}
	if articleDO.Title == "" {
		logger.Warn("更新文章失败：标题不能为空", zap.Uint64("article_id", articleDO.ID))
		return common.ParamErr
	}
	if len(articleDO.Title) > 128 {
		logger.Warn("更新文章失败：标题超长", zap.Uint64("article_id", articleDO.ID))
		return common.ParamErr
	}
	if articleDO.Sort < 0 {
		logger.Warn("更新文章失败：排序值非法", zap.Uint64("article_id", articleDO.ID))
		return common.ParamErr
	}

	// DO转Model
	articleModel := convert.ArticleDOToModel(articleDO)

	// 更新数据库
	err := a.articleMysqlRepo.UpdateArticle(ctx, articleModel)
	if err != nil {
		logger.Error("更新文章数据库操作失败", zap.Uint64("article_id", articleDO.ID), zap.Error(err))
		return common.DatabaseErr
	}

	// 更新成功，删除缓存（强一致性）
	_ = a.articleRedisRepo.DelArticleFromRedis(ctx, articleDO.ID)
	_ = a.articleRedisRepo.DelArticleListFromRedis(ctx)
	logger.Info("更新文章成功，缓存已删除", zap.Uint64("article_id", articleDO.ID))
	return nil
}

// DeleteArticle 删除文章
func (a *ArticleDomainService) DeleteArticle(ctx context.Context, id uint64) error {
	// 参数校验
	if id <= 0 {
		logger.Warn("删除文章失败：ID非法", zap.Uint64("article_id", id))
		return common.ParamErr
	}

	// 删除数据库数据（软删除）
	err := a.articleMysqlRepo.DeleteArticle(ctx, id)
	if err != nil {
		logger.Error("删除文章数据库操作失败", zap.Uint64("article_id", id), zap.Error(err))
		return common.DatabaseErr
	}

	// 删除成功，清理缓存
	_ = a.articleRedisRepo.DelArticleFromRedis(ctx, id)
	_ = a.articleRedisRepo.DelArticleListFromRedis(ctx)
	logger.Info("删除文章成功，缓存已清理", zap.Uint64("article_id", id))
	return nil
}
