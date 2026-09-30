/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 15:22:25
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-08-26 00:00:00
 * @FilePath: \yia-blog\blog-server\domain\service\category_domain_service.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"
	"errors"
	"unicode/utf8"

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
func (c *CategoryDomainService) CreateCategory(ctx context.Context, categoryDO *do.CategoryCreate) error {
	// 业务规则校验
	if categoryDO == nil {
		logger.Warn("创建分类失败：领域对象为空")
		return common.ParamErr
	}
	status := consts.CategoryStatusEnabled
	if categoryDO.Status != nil {
		status = *categoryDO.Status
	}
	if err := validateCategoryFields(categoryDO.Name, &categoryDO.Sort, &status); err != nil {
		return err
	}

	// DO转Model：新建时 ID 为 0，主键交给数据库自增，create_at/update_at 由表默认值填充
	categoryModel := convert.CategoryDOToModel(&do.CategoryDO{Name: categoryDO.Name, Sort: categoryDO.Sort, Status: status})

	// 调用MySQL仓储
	err := c.categoryMysqlRepo.CreateCategory(ctx, categoryModel)
	if err != nil {
		if errors.Is(err, repo.ErrDuplicateKey) {
			return common.CategoryExistErr
		}
		logger.Error("创建分类数据库操作失败", zap.String("name", categoryDO.Name), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("创建分类成功", zap.String("name", categoryDO.Name))
	return nil
}

// ListCategoryByPage 分类分页查询：仅参数校验 + 数据库查询
func (c *CategoryDomainService) ListCategoryByPage(ctx context.Context, query *do.CategoryQuery) ([]do.CategoryDO, int64, error) {
	if err := NormalizeCategoryQuery(query); err != nil {
		return nil, 0, err
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

// NormalizeCategoryQuery 统一数据库查询和缓存键使用的有效分页参数。
func NormalizeCategoryQuery(query *do.CategoryQuery) error {
	if query == nil {
		return common.ParamErr
	}
	if query.PageNum <= 0 {
		query.PageNum = 1
	}
	if query.PageSize <= 0 || query.PageSize > 50 {
		query.PageSize = 10
	}
	return nil
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

// UpdateCategory 更新分类：仅业务校验 + 数据库操作。
// upd.Sort/Status 为 nil 表示本次未提交该字段，领域层刻意不替它们填充原值：
// 未提交的列由 Repo 排除在 UPDATE 白名单之外，UPDATE 只写本次真正修改的列，
// 从而不会像「读全量再整体写回」那样把并发提交的改动覆盖成旧值。
func (c *CategoryDomainService) UpdateCategory(ctx context.Context, upd *do.CategoryUpdate) error {
	// 业务规则校验
	if upd == nil {
		logger.Warn("更新分类失败：领域对象为空")
		return common.ParamErr
	}
	if upd.ID <= 0 {
		logger.Warn("更新分类失败：ID非法", zap.Uint64("id", upd.ID))
		return common.ParamErr
	}
	if err := validateCategoryFields(upd.Name, upd.Sort, upd.Status); err != nil {
		return err
	}

	// 更新前确认分类存在：MySQL 下 RowsAffected==0 无法区分「记录不存在」与「值未变化」，
	// 故存在性在领域层显式校验，Repo 的 RowsAffected 只作并发删除的兜底日志。
	if _, err := c.GetCategoryById(ctx, upd.ID); err != nil {
		return err
	}

	// 更新数据库：直接下传 CategoryUpdate，不经 DO→Model 转换，
	// 否则 Sort/Status 的 nil 会在 Model 的值类型字段上退化成 0，指针语义前功尽弃。
	err := c.categoryMysqlRepo.UpdateCategory(ctx, upd)
	if err != nil {
		if errors.Is(err, repo.ErrDuplicateKey) {
			return common.CategoryExistErr
		}
		logger.Error("更新分类数据库操作失败", zap.Uint64("id", upd.ID), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("更新分类成功", zap.Uint64("id", upd.ID))
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

// validateCategoryFields 使创建和更新遵守相同规则，名称长度按字符与数据库/DTO 对齐。
func validateCategoryFields(name string, sort, status *int) error {
	if name == "" {
		return common.ErrCategoryNameEmpty
	}
	if utf8.RuneCountInString(name) > consts.CategoryNameMaxLength {
		return common.ErrCategoryNameTooLong
	}
	if sort != nil && *sort < 0 {
		return common.ErrCategorySortInvalid
	}
	if status != nil && *status != consts.CategoryStatusDisabled && *status != consts.CategoryStatusEnabled {
		return common.ParamErr.WithMsg("分类状态不合法")
	}
	return nil
}
