/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 15:34:17
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-29 00:18:17
 * @FilePath: \yia-blog\blog-server\infra\repo\category_mysql_repo.go
 * @Description:
 *
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

type categoryMysqlRepo struct {
	db *gorm.DB
}

func NewCategoryMysqlRepo(db *gorm.DB) *categoryMysqlRepo {
	return &categoryMysqlRepo{db: db}
}

// --------------------- 1. 创建分类 ---------------------
func (c *categoryMysqlRepo) CreateCategory(ctx context.Context, category *model.CategoryModel) error {

	if err := c.db.WithContext(ctx).Create(category).Error; err != nil {
		logger.Error("CreateCategory failed", zap.String("name", category.Name), zap.Error(err))
		return err
	}
	return nil
}

// --------------------- 2. 分类分页查询 ---------------------
func (c *categoryMysqlRepo) ListCategoryByPage(ctx context.Context, query *do.CategoryQuery) ([]*model.CategoryModel, int64, error) {

	var (
		list  []*model.CategoryModel
		total int64
		db    = c.db.WithContext(ctx).Model(&model.CategoryModel{})
	)

	// 1. 拼接查询条件（动态SQL）
	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}

	// 2. 查询总条数
	if err := db.Count(&total).Error; err != nil {
		logger.Error("ListCategoryByPage count failed", zap.Any("query", query), zap.Error(err))
		return nil, 0, err
	}
	if total == 0 {
		return list, 0, nil
	}

	// 3. 分页查询
	offset := (query.PageNum - 1) * query.PageSize
	err := db.
		Order("sort ASC, id DESC").
		Limit(query.PageSize).
		Offset(offset).
		Find(&list).Error
	if err != nil {
		logger.Error("ListCategoryByPage find failed", zap.Any("query", query), zap.Error(err))
		return nil, 0, err
	}
	return list, total, nil
}

// --------------------- 3. 根据ID查询分类 ---------------------
func (c *categoryMysqlRepo) GetCategoryById(ctx context.Context, id uint64) (*model.CategoryModel, error) {

	var category model.CategoryModel
	err := c.db.WithContext(ctx).First(&category, id).Error

	if err != nil {
		// 记录不存在不是故障：返回 (nil, nil) 由 Domain 翻译成 CategoryNotFoundErr，
		// 避免 gorm.ErrRecordNotFound 渗入领域层，也避免「不存在」被误报成 DatabaseErr。
		if err == gorm.ErrRecordNotFound {
			logger.Warn("GetCategoryById: category not found", zap.Uint64("category_id", id))
			return nil, nil
		}
		logger.Error("GetCategoryById failed", zap.Uint64("category_id", id), zap.Error(err))
		return nil, err
	}

	return &category, nil
}

// --------------------- 4. 删除分类（软删除） ---------------------
func (c *categoryMysqlRepo) DeleteCategory(ctx context.Context, id uint64) error {

	result := c.db.WithContext(ctx).Delete(&model.CategoryModel{}, id)
	if result.Error != nil {
		logger.Error("DeleteCategory failed", zap.Uint64("category_id", id), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		// 存在性由 Domain 先行校验；此处为 0 说明校验后被并发删除，只记日志不报错。
		logger.Warn("DeleteCategory no rows affected", zap.Uint64("category_id", id))
	}
	return nil
}

// --------------------- 5. 更新分类 ---------------------
func (c *categoryMysqlRepo) UpdateCategory(ctx context.Context, category *model.CategoryModel) error {
	result := c.db.WithContext(ctx).
		Model(&model.CategoryModel{}).
		Where("id = ?", category.ID). // 显式 WHERE，不依赖「Model 非零字段转条件」的隐式规则
		Updates(map[string]interface{}{ // 用 map：status=0 等零值也会更新
			"name":   category.Name,
			"sort":   category.Sort,
			"status": category.Status,
		}) // create_at/deleted_at 不在白名单，保持原值
	if result.Error != nil {
		logger.Error("UpdateCategory failed", zap.Uint64("category_id", category.ID), zap.Error(result.Error))
		return result.Error
	}
	// 存在性由 Domain 先行校验；这里为 0 说明校验后被并发软删除，或本次值与原值完全相同。
	// MySQL 不区分这两种情况，故只记日志不报错。
	if result.RowsAffected == 0 { // 取本次 result，不再读 c.db
		logger.Warn("UpdateCategory no rows affected", zap.Uint64("category_id", category.ID))
	}
	return nil
}
