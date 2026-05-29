/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-29 17:00:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 16:55:51
 * @FilePath: \yia-blog\blog-server\app\service\article_app_service.go
 * @Description: 文章应用服务 - 业务编排、缓存处理、流程调度
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/domain/service"
	"context"

	"go.uber.org/zap"
)

// ArticleAppService 应用服务：仅做流程编排 + 缓存处理
type ArticleAppService struct {
	articleDomainService *service.ArticleDomainService // 领域服务（纯业务）
	articleRedisRepo     repo.ArticleRedisRepo         // 缓存仓储（应用层编排）
}

// NewArticleAppService 构造函数：依赖外部注入，无强耦合
func NewArticleAppService(
	domainService *service.ArticleDomainService,
	redisRepo repo.ArticleRedisRepo,
) *ArticleAppService {
	return &ArticleAppService{
		articleDomainService: domainService,
		articleRedisRepo:     redisRepo,
	}
}

// CreateArticle 创建文章：调用领域服务
func (s *ArticleAppService) CreateArticle(ctx context.Context, req *dto.CreateArticleReq) error {
	if req == nil {
		logger.Warn("创建文章参数为空")
		return common.ParamErr
	}

	// DTO 转换为领域对象
	articleDO := &do.ArticleDO{
		AdminID:       req.AdminID,
		Title:         req.Title,
		Cover:         req.Cover,
		Summary:       req.Summary,
		Content:       req.Content,
		CategoryID:    req.CategoryID,
		TagIDs:        req.TagIDs,
		Status:        req.Status,
		Sort:          req.Sort,
		IsTop:         req.IsTop,
		CommentStatus: req.CommentStatus,
		PublishAt:     req.PublishAt,
	}

	// 1. 执行核心业务逻辑
	err := s.articleDomainService.CreateArticle(ctx, articleDO)
	if err != nil {
		logger.Error("创建文章失败", zap.String("title", req.Title), zap.Error(err))
		return err
	}

	logger.Info("创建文章成功", zap.String("title", req.Title))
	return nil
}

// ListArticleByPage 分页查询文章
func (s *ArticleAppService) ListArticleByPage(ctx context.Context, req *dto.ArticlePageQuery) (*dto.PageResult, error) {
	if req == nil {
		logger.Warn("文章分页查询参数为空")
		return nil, common.ParamErr
	}

	query := &do.ArticleQuery{
		PageNum:    req.PageNum,
		PageSize:   req.PageSize,
		Title:      req.Title,
		CategoryID: req.CategoryID,
		Status:     req.Status,
		IsTop:      req.IsTop,
	}

	// 调用领域服务
	articleDOList, total, err := s.articleDomainService.ListArticleByPage(ctx, query)
	if err != nil {
		logger.Error("分页查询文章失败", zap.Error(err))
		return nil, err
	}

	// 组装返回
	var respList []map[string]interface{}
	for _, articleDO := range articleDOList {
		respList = append(respList, map[string]interface{}{
			"id":          articleDO.ID,
			"title":       articleDO.Title,
			"cover":       articleDO.Cover,
			"summary":     articleDO.Summary,
			"category_id": articleDO.CategoryID,
			"sort":        articleDO.Sort,
			"status":      articleDO.Status,
			"is_top":      articleDO.IsTop,
			"views":       articleDO.Views,
			"likes":       articleDO.Likes,
			"comments":    articleDO.Comments,
			"publish_at":  articleDO.PublishAt,
			"created_at":  articleDO.CreateAt,
			"updated_at":  articleDO.UpdateAt,
		})
	}

	pages := (total + int64(req.PageSize) - 1) / int64(req.PageSize)
	return &dto.PageResult{
		Total:    total,
		Pages:    int(pages),
		List:     respList,
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	}, nil
}

// GetArticleById 根据ID查询文章详情
func (s *ArticleAppService) GetArticleById(ctx context.Context, id uint64) (map[string]interface{}, error) {
	if id <= 0 {
		logger.Warn("文章ID非法")
		return nil, common.ParamErr
	}

	articleDO, err := s.articleDomainService.GetArticleById(ctx, id)
	if err != nil {
		logger.Error("查询文章详情失败", zap.Uint64("id", id), zap.Error(err))
		return nil, err
	}

	return map[string]interface{}{
		"id":             articleDO.ID,
		"admin_id":       articleDO.AdminID,
		"title":          articleDO.Title,
		"cover":          articleDO.Cover,
		"summary":        articleDO.Summary,
		"content":        articleDO.Content,
		"category_id":    articleDO.CategoryID,
		"tag_ids":        articleDO.TagIDs,
		"status":         articleDO.Status,
		"sort":           articleDO.Sort,
		"views":          articleDO.Views,
		"likes":          articleDO.Likes,
		"comments":       articleDO.Comments,
		"is_top":         articleDO.IsTop,
		"comment_status": articleDO.CommentStatus,
		"publish_at":     articleDO.PublishAt,
		"create_at":      articleDO.CreateAt,
		"update_at":      articleDO.UpdateAt,
	}, nil
}

// UpdateArticle 更新文章：调用领域服务
func (s *ArticleAppService) UpdateArticle(ctx context.Context, req *dto.UpdateArticleReq) error {
	if req.ID <= 0 {
		logger.Warn("文章ID非法")
		return common.ParamErr
	}

	articleDO := &do.ArticleDO{
		ID:            req.ID,
		Title:         req.Title,
		Cover:         req.Cover,
		Summary:       req.Summary,
		Content:       req.Content,
		CategoryID:    req.CategoryID,
		TagIDs:        req.TagIDs,
		Status:        req.Status,
		Sort:          req.Sort,
		IsTop:         req.IsTop,
		CommentStatus: req.CommentStatus,
		PublishAt:     req.PublishAt,
	}

	err := s.articleDomainService.UpdateArticle(ctx, articleDO)
	if err != nil {
		logger.Error("更新文章失败", zap.Uint64("id", req.ID), zap.Error(err))
		return err
	}

	logger.Info("更新文章成功", zap.Uint64("id", req.ID))
	return nil
}

// DeleteArticle 删除文章：调用领域服务
func (s *ArticleAppService) DeleteArticle(ctx context.Context, id uint64) error {
	if id <= 0 {
		logger.Warn("文章ID非法")
		return common.ParamErr
	}

	err := s.articleDomainService.DeleteArticle(ctx, id)
	if err != nil {
		logger.Error("删除文章失败", zap.Uint64("id", id), zap.Error(err))
		return err
	}

	logger.Info("删除文章成功", zap.Uint64("id", id))
	return nil
}
