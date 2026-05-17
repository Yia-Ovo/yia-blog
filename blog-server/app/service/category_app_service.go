/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 15:17:22
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-06 17:11:22
 * @FilePath: \yia-blog\blog-server\app\service\category_app_service.go
 * @Description:
 *
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

type CategoryAppService struct {
	categoryDomainSrv *service.CategoryDomainService
}

// NewCategoryAppService 创建分类应用服务实例（IOC注入）
func NewCategoryAppService(
	categoryMysqlRepo repo.CategoryMysqlRepo,
	categoryRedisRepo repo.CategoryRedisRepo,
) *CategoryAppService {
	// 初始化领域服务
	categoryDomainSrv := service.NewCategoryDomainService(categoryMysqlRepo, categoryRedisRepo)

	return &CategoryAppService{
		categoryDomainSrv: categoryDomainSrv,
	}
}

// CreateCategory 创建分类
func (s *CategoryAppService) CreateCategory(ctx context.Context, req *dto.CreateCategoryReq) error {
	// 1. 应用层基础参数校验
	if req == nil {
		return common.ParamErr.WithMsg("创建分类请求参数不能为空")
	}

	// 2. DTO 转换为 DO（领域对象）
	categoryDO := &do.CategoryDO{
		Name:   req.Name,
		Sort:   req.Sort,
		Status: req.Status,
	}

	// 3. 调用领域服务（核心逻辑交给领域层）
	err := s.categoryDomainSrv.CreateCategory(ctx, categoryDO)
	if err != nil {
		logger.Error("创建分类应用服务失败",
			zap.String("category_name", req.Name),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// ListCategoryByPage 分页查询分类
func (s *CategoryAppService) ListCategoryByPage(ctx context.Context, req *dto.CategoryPageQuery) (*dto.PageResult, error) {
	// 1. DTO 转换为领域查询对象
	query := &do.CategoryQuery{
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
		Name:     req.Name,
		Status:   req.Status,
	}

	// 2. 调用领域服务
	categoryDOList, total, err := s.categoryDomainSrv.ListCategoryByPage(ctx, query)
	if err != nil {
		logger.Error("分页查询分类应用服务失败",
			zap.Int("page_num", req.PageNum),
			zap.Int("page_size", req.PageSize),
			zap.Error(err),
		)
		return nil, err
	}

	// 3. DO 列表转换为响应数据
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

	// 4. 封装通用分页返回结果
	pages := (total + int64(req.PageSize) - 1) / int64(req.PageSize)
	result := &dto.PageResult{
		Total:    total,
		Pages:    int(pages),
		List:     respList,
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
	}

	return result, nil
}

// GetCategoryById 根据ID查询分类详情
func (s *CategoryAppService) GetCategoryById(ctx context.Context, id uint64) (map[string]interface{}, error) {
	// 1. 调用领域服务
	categoryDO, err := s.categoryDomainSrv.GetCategoryById(ctx, id)
	if err != nil {
		logger.Error("根据ID查询分类应用服务失败",
			zap.Uint64("category_id", id),
			zap.Error(err),
		)
		return nil, err
	}

	// 2. 组装返回数据（后续可替换为自定义 CategoryResp DTO）
	resp := map[string]interface{}{
		"id":        categoryDO.ID,
		"name":      categoryDO.Name,
		"sort":      categoryDO.Sort,
		"status":    categoryDO.Status,
		"create_at": categoryDO.CreateAt,
		"update_at": categoryDO.UpdateAt,
		"is_delete": categoryDO.IsDeleted,
	}

	return resp, nil
}

// UpdateCategory 更新分类
func (s *CategoryAppService) UpdateCategory(ctx context.Context, id uint64, name string, sort int, status int) error {
	// 1. 应用层参数校验
	if id <= 0 {
		return common.ParamErr.WithMsg("分类ID不合法")
	}

	// 2. 组装 DO
	categoryDO := &do.CategoryDO{
		ID:     id,
		Name:   name,
		Sort:   sort,
		Status: status,
	}

	// 3. 调用领域服务
	err := s.categoryDomainSrv.UpdateCategory(ctx, categoryDO)
	if err != nil {
		logger.Error("更新分类应用服务失败",
			zap.Uint64("category_id", id),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// DeleteCategory 删除分类
func (s *CategoryAppService) DeleteCategory(ctx context.Context, id uint64) error {
	// 1. 调用领域服务
	err := s.categoryDomainSrv.DeleteCategory(ctx, id)
	if err != nil {
		logger.Error("删除分类应用服务失败",
			zap.Uint64("category_id", id),
			zap.Error(err),
		)
		return err
	}

	return nil
}
