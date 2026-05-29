/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 15:17:22
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 16:56:56
 * @FilePath: \yia-blog\blog-server\app\service\category_app_service.go
 * @Description: 分类应用服务 - 业务编排、缓存处理、流程调度
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

// CategoryAppService 应用服务：仅做流程编排 + 缓存处理
type CategoryAppService struct {
	categoryDomainService *service.CategoryDomainService // 领域服务（纯业务）
	categoryRedisRepo     repo.CategoryRedisRepo         // 缓存仓储（应用层编排）
}

// NewCategoryAppService 构造函数：依赖外部注入，无强耦合
func NewCategoryAppService(
	domainService *service.CategoryDomainService,
	redisRepo repo.CategoryRedisRepo,
) *CategoryAppService {
	return &CategoryAppService{
		categoryDomainService: domainService,
		categoryRedisRepo:     redisRepo,
	}
}

// CreateCategory 创建分类：调用领域服务
func (s *CategoryAppService) CreateCategory(ctx context.Context, req *dto.CreateCategoryReq) error {
	if req == nil {
		logger.Warn("创建分类参数为空")
		return common.ParamErr
	}

	// DTO 转换为领域对象
	categoryDO := &do.CategoryDO{
		Name:   req.Name,
		Sort:   req.Sort,
		Status: req.Status,
	}

	// 1. 执行核心业务逻辑
	err := s.categoryDomainService.CreateCategory(ctx, categoryDO)
	if err != nil {
		logger.Error("创建分类失败", zap.String("name", req.Name), zap.Error(err))
		return err
	}

	logger.Info("创建分类成功", zap.String("name", req.Name))
	return nil
}

// ListCategoryByPage 分页查询分类
func (s *CategoryAppService) ListCategoryByPage(ctx context.Context, req *dto.CategoryPageQuery) (*dto.PageResult, error) {
	if req == nil {
		logger.Warn("分类分页查询参数为空")
		return nil, common.ParamErr
	}

	query := &do.CategoryQuery{
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
		Name:     req.Name,
		Status:   req.Status,
	}

	// 调用领域服务
	categoryDOList, total, err := s.categoryDomainService.ListCategoryByPage(ctx, query)
	if err != nil {
		logger.Error("分页查询分类失败", zap.Error(err))
		return nil, err
	}

	// 组装返回
	var respList []map[string]interface{}
	for _, categoryDO := range categoryDOList {
		respList = append(respList, map[string]interface{}{
			"id":         categoryDO.ID,
			"name":       categoryDO.Name,
			"sort":       categoryDO.Sort,
			"status":     categoryDO.Status,
			"created_at": categoryDO.CreateAt,
			"updated_at": categoryDO.UpdateAt,
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

// GetCategoryById 根据ID查询分类详情
func (s *CategoryAppService) GetCategoryById(ctx context.Context, id uint64) (map[string]interface{}, error) {
	if id <= 0 {
		logger.Warn("分类ID非法")
		return nil, common.ParamErr
	}

	categoryDO, err := s.categoryDomainService.GetCategoryById(ctx, id)
	if err != nil {
		logger.Error("查询分类详情失败", zap.Uint64("category_id", id), zap.Error(err))
		return nil, err
	}

	return map[string]interface{}{
		"id":        categoryDO.ID,
		"name":      categoryDO.Name,
		"sort":      categoryDO.Sort,
		"status":    categoryDO.Status,
		"create_at": categoryDO.CreateAt,
		"update_at": categoryDO.UpdateAt,
		"is_delete": categoryDO.IsDeleted,
	}, nil
}

// UpdateCategory 更新分类：调用领域服务
func (s *CategoryAppService) UpdateCategory(ctx context.Context, id uint64, name string, sort int, status int) error {
	if id <= 0 {
		logger.Warn("分类ID非法")
		return common.ParamErr
	}

	categoryDO := &do.CategoryDO{
		ID:     id,
		Name:   name,
		Sort:   sort,
		Status: status,
	}

	err := s.categoryDomainService.UpdateCategory(ctx, categoryDO)
	if err != nil {
		logger.Error("更新分类失败", zap.Uint64("category_id", id), zap.Error(err))
		return err
	}

	logger.Info("更新分类成功", zap.Uint64("id", id))
	return nil
}

// DeleteCategory 删除分类：调用领域服务
func (s *CategoryAppService) DeleteCategory(ctx context.Context, id uint64) error {
	if id <= 0 {
		logger.Warn("分类ID非法")
		return common.ParamErr
	}

	err := s.categoryDomainService.DeleteCategory(ctx, id)
	if err != nil {
		logger.Error("删除分类失败", zap.Uint64("category_id", id), zap.Error(err))
		return err
	}

	logger.Info("删除分类成功", zap.Uint64("id", id))
	return nil
}
