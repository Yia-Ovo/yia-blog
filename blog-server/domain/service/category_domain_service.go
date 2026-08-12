/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 15:22:25
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 16:24:51
 * @FilePath: \yia-blog\blog-server\domain\service\category_domain_service.go
 * @Description:
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
	"context"

	"go.uber.org/zap"
)

// CategoryDomainService 纯净领域服务：仅依赖MySQL仓储，仅处理业务规则
type CategoryDomainService struct {
	categoryMysqlRepo repo.CategoryMysqlRepo
}

// NewCategoryDomainService 构造函数：仅注入MySQL依赖
func NewCategoryDomainService(categoryMysqlRepo repo.CategoryMysqlRepo) *CategoryDomainService {
	return &CategoryDomainService{
		categoryMysqlRepo: categoryMysqlRepo,
	}
}

// CreateCategory 创建分类：仅业务校验 + 数据库操作
func (c *CategoryDomainService) CreateCategory(ctx context.Context, categoryDO *do.CategoryDO) error {
	// 业务规则校验
	if categoryDO.Name == "" {
		logger.Warn("创建分类失败：分类名称不能为空")
		return common.ParamErr
	}

	// DO转Model
	categoryModel := convert.CategoryDOToModel(categoryDO)

	// 调用MySQL仓储
	err := c.categoryMysqlRepo.CreateCategory(ctx, categoryModel)
	if err != nil {
		logger.Error("创建分类数据库操作失败", zap.String("name", categoryDO.Name), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("创建分类成功", zap.String("name", categoryDO.Name))
	return nil
}

// ListCategoryByPage 分类分页查询：仅参数校验 + 数据库查询
func (c *CategoryDomainService) ListCategoryByPage(ctx context.Context, query *do.CategoryQuery) ([]do.CategoryDO, int64, error) {
	// 分页参数合法性校验
	if query.PageNum <= 0 {
		query.PageNum = 1
	}
	if query.PageSize <= 0 || query.PageSize > 50 {
		query.PageSize = 10
	}

	// 查询数据库
	modelList, total, err := c.categoryMysqlRepo.ListCategoryByPage(ctx, query)
	if err != nil {
		logger.Error("分页查询分类失败", zap.Any("query", query), zap.Error(err))
		return nil, 0, common.DatabaseErr
	}

	logger.Info("分页查询分类成功", zap.Int64("total", total))
	return convert.CategoryModelListToDOList(modelList), total, nil
}

// GetCategoryById 根据ID查询分类：仅参数校验 + 数据库查询
func (c *CategoryDomainService) GetCategoryById(ctx context.Context, id uint64) (*do.CategoryDO, error) {
	// 参数校验
	if id <= 0 {
		logger.Warn("根据ID查询分类失败：ID非法", zap.Uint64("id", id))
		return nil, common.ParamErr
	}

	// 查询数据库
	// 1. 先接收 model
	model, err := c.categoryMysqlRepo.GetCategoryById(ctx, id)
	if err != nil {
		logger.Error("根据ID查询分类失败", zap.Uint64("id", id), zap.Error(err))
		return nil, common.DatabaseErr
	}

	if model == nil {
		// Repo 约定：查不到返回 (nil, nil)，与 DB 故障（err != nil）区分开
		logger.Warn("分类不存在", zap.Uint64("id", id))
		return nil, common.CategoryNotFoundErr
	}

	// 2. 再调用 convert 转成 DO
	logger.Info("根据ID查询分类成功", zap.Uint64("id", id))
	return convert.CategoryModelToDO(model), nil
}

// UpdateCategory 更新分类：仅业务校验 + 数据库操作
func (c *CategoryDomainService) UpdateCategory(ctx context.Context, categoryDO *do.CategoryDO) error {
	// 业务规则校验
	if categoryDO.ID <= 0 {
		logger.Warn("更新分类失败：ID非法", zap.Uint64("id", categoryDO.ID))
		return common.ParamErr
	}
	if categoryDO.Name == "" {
		logger.Warn("更新分类失败：名称不能为空", zap.Uint64("id", categoryDO.ID))
		return common.ParamErr
	}

	// 更新前确认分类存在：MySQL 下 RowsAffected==0 无法区分「记录不存在」与「值未变化」，
	// 故存在性在领域层显式校验，Repo 的 RowsAffected 只作并发删除的兜底日志。
	if _, err := c.GetCategoryById(ctx, categoryDO.ID); err != nil {
		return err
	}

	// DO转Model
	categoryModel := convert.CategoryDOToModel(categoryDO)

	// 更新数据库
	err := c.categoryMysqlRepo.UpdateCategory(ctx, categoryModel)
	if err != nil {
		logger.Error("更新分类数据库操作失败", zap.Uint64("id", categoryDO.ID), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("更新分类成功", zap.Uint64("id", categoryDO.ID))
	return nil
}

// DeleteCategory 删除分类(软删除)：仅参数校验 + 数据库操作
func (c *CategoryDomainService) DeleteCategory(ctx context.Context, id uint64) error {
	// 参数校验
	if id <= 0 {
		logger.Warn("删除分类失败：ID非法", zap.Uint64("id", id))
		return common.ParamErr
	}

	// 删除前确认分类存在，避免删除不存在的分类仍返回成功（同 UpdateCategory）
	if _, err := c.GetCategoryById(ctx, id); err != nil {
		return err
	}

	// 删除数据库数据（软删除）
	err := c.categoryMysqlRepo.DeleteCategory(ctx, id)
	if err != nil {
		logger.Error("删除分类数据库操作失败", zap.Uint64("id", id), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("删除分类成功", zap.Uint64("id", id))
	return nil
}
