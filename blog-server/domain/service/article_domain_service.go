/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-07 03:20:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 16:43:10
 * @FilePath: \yia-blog\blog-server\domain\service\article_domain_service.go
 * @Description: 文章领域服务（纯业务规则，无任何Redis/基础设施依赖）
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/common"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"

	"go.uber.org/zap"
)

// ArticleDomainService 纯净领域服务：仅依赖MySQL仓储，仅处理业务规则
type ArticleDomainService struct {
	articleMysqlRepo repo.ArticleMysqlRepo
}

// NewArticleDomainService 构造函数：仅注入MySQL依赖
func NewArticleDomainService(articleMysqlRepo repo.ArticleMysqlRepo) *ArticleDomainService {
	return &ArticleDomainService{
		articleMysqlRepo: articleMysqlRepo,
	}
}

// CreateArticle 创建文章：仅业务校验 + 数据库操作
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

	logger.Info("创建文章成功", zap.String("title", articleDO.Title))
	return nil
}

// ListArticleByPage 文章分页查询：仅参数校验 + 数据库查询
func (a *ArticleDomainService) ListArticleByPage(ctx context.Context, query *do.ArticleQuery) ([]do.ArticleDO, int64, error) {
	// 分页参数合法性校验
	if query.PageNum < 1 {
		query.PageNum = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 10
	}

	// 直接查询数据库（缓存逻辑在应用层）
	modelList, total, err := a.articleMysqlRepo.ListArticleByPage(ctx, query)
	if err != nil {
		logger.Error("分页查询文章失败", zap.Any("query", query), zap.Error(err))
		return nil, 0, common.DatabaseErr
	}

	logger.Info("分页查询文章成功", zap.Int64("total", total))
	return convert.ArticleModelListToDOList(modelList), total, nil
}

// GetArticleById 根据ID查询文章：仅参数校验 + 数据库查询
func (a *ArticleDomainService) GetArticleById(ctx context.Context, id uint64) (*do.ArticleDO, error) {
	// 参数校验
	if id <= 0 {
		logger.Warn("根据ID查询文章失败：ID非法", zap.Uint64("article_id", id))
		return nil, common.ParamErr
	}

	// 查询数据库
	articleModel, err := a.articleMysqlRepo.GetArticleById(ctx, id)
	if err != nil {
		logger.Error("根据ID查询文章失败", zap.Uint64("article_id", id), zap.Error(err))
		return nil, common.DatabaseErr
	}

	// 数据不存在
	if articleModel == nil {
		logger.Warn("文章不存在", zap.Uint64("article_id", id))
		return nil, common.ArticleNotFoundErr
	}

	logger.Info("根据ID查询文章成功", zap.Uint64("article_id", id))
	return convert.ArticleModelToDO(articleModel), nil
}

// UpdateArticle 更新文章：仅业务校验 + 数据库操作
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

	logger.Info("更新文章成功", zap.Uint64("article_id", articleDO.ID))
	return nil
}

// DeleteArticle 删除文章(软删除)：仅参数校验 + 数据库操作
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

	logger.Info("删除文章成功", zap.Uint64("article_id", id))
	return nil
}
