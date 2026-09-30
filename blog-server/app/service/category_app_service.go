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
	"encoding/json"
	"strconv"

	"go.uber.org/zap"
)

// CategoryAppService 应用服务：仅做流程编排 + 缓存处理
type CategoryAppService struct {
	categoryDomainService *service.CategoryDomainService // 领域服务（纯业务）
	cache                 *readCacheState                // 缓存失败不影响领域写入结果
}

// NewCategoryAppService 构造函数：依赖外部注入，无强耦合
func NewCategoryAppService(
	domainService *service.CategoryDomainService,
	redisRepo repo.ReadCacheRepo,
) *CategoryAppService {
	return &CategoryAppService{
		categoryDomainService: domainService,
		cache:                 newReadCacheState(redisRepo),
	}
}

// CreateCategory 创建分类：调用领域服务
func (s *CategoryAppService) CreateCategory(ctx context.Context, req *dto.CreateCategoryReq) error {
	if req == nil {
		logger.Warn("创建分类参数为空")
		return common.ParamErr
	}

	// DTO 转换为领域对象
	categoryDO := &do.CategoryCreate{
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

	s.cache.invalidate()
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

	if err := service.NormalizeCategoryQuery(query); err != nil {
		return nil, err
	}
	// 所有筛选条件及归一化后的分页共同组成键，避免不同查询共享列表。
	key, err := json.Marshal(query)
	if err != nil {
		return nil, common.ParamErr
	}
	page, err := readThrough(ctx, s.cache, "list:"+string(key), nil, func(ctx context.Context) (categoryCachedPage, error) {
		items, total, err := s.categoryDomainService.ListCategoryByPage(ctx, query)
		return categoryCachedPage{Items: items, Total: total}, err
	}, func(page categoryCachedPage) bool {
		return page.Total >= 0 && int64(len(page.Items)) <= page.Total && len(page.Items) <= query.PageSize
	})
	if err != nil {
		logger.Error("分页查询分类失败", zap.Error(err))
		return nil, err
	}
	categoryDOList, total := page.Items, page.Total

	// 组装返回
	respList := make([]map[string]interface{}, 0, len(categoryDOList))
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

	// Domain 会把超限条数归一化，响应必须与实际查询采用同一组值。
	pages := (total + int64(query.PageSize) - 1) / int64(query.PageSize)
	return &dto.PageResult{
		Total:    total,
		Pages:    int(pages),
		List:     respList,
		PageNum:  query.PageNum,
		PageSize: query.PageSize,
	}, nil
}

type categoryCachedPage struct {
	Items []do.CategoryDO
	Total int64
}

// GetCategoryById 根据ID查询分类详情
func (s *CategoryAppService) GetCategoryById(ctx context.Context, id uint64) (map[string]interface{}, error) {
	if id <= 0 {
		logger.Warn("分类ID非法")
		return nil, common.ParamErr
	}

	categoryDO, err := readThrough(ctx, s.cache, "detail:"+strconv.FormatUint(id, 10), common.CategoryNotFoundErr, func(ctx context.Context) (*do.CategoryDO, error) {
		return s.categoryDomainService.GetCategoryById(ctx, id)
	}, func(value *do.CategoryDO) bool {
		return value != nil && value.ID == id
	})
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

// UpdateCategory 更新分类：调用领域服务。
// sort/status 为 nil 表示客户端本次未提交该字段。这里刻意不做 ptr.Val 压平——
// 一旦压成 0 就与「显式提交 0」（排最前 / 禁用）无法区分，
// 指针一路保留到 Repo 组装更新列时才解释。
func (s *CategoryAppService) UpdateCategory(ctx context.Context, id uint64, name string, sort *int, status *int) error {
	if id <= 0 {
		logger.Warn("分类ID非法")
		return common.ParamErr
	}

	categoryUpdate := &do.CategoryUpdate{
		ID:     id,
		Name:   name,
		Sort:   sort,
		Status: status,
	}

	err := s.categoryDomainService.UpdateCategory(ctx, categoryUpdate)
	if err != nil {
		logger.Error("更新分类失败", zap.Uint64("category_id", id), zap.Error(err))
		return err
	}

	s.cache.invalidate()
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

	s.cache.invalidate()
	logger.Info("删除分类成功", zap.Uint64("id", id))
	return nil
}
