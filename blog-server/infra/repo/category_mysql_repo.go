/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 15:34:17
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-20 19:45:26
 * @FilePath: \yia-blog\blog-server\infra\repo\category_mysql_repo.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import (
	"blog-server/common/gin/response"
	"blog-server/common/utils/convert"
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
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	if err := c.db.WithContext(ctx).Create(category).Error; err != nil {
		logger.Error("CreateCategory failed", zap.String("name", category.Name), zap.Error(err))
		return err
	}
	return nil
}

// --------------------- 2. 分类分页查询 ---------------------
func (c *categoryMysqlRepo) ListCategoryByPage(ctx context.Context, query *do.CategoryQuery) ([]do.CategoryDO, int64, error) {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	var (
		list   []model.CategoryModel
		total  int64
		db     = c.db.WithContext(ctx).Model(&model.CategoryModel{})
		listDO []do.CategoryDO
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
		return listDO, 0, nil
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

	listDO = convert.CategoryModelListToDO(&list)
	return listDO, total, nil
}

// --------------------- 3. 根据ID查询分类 ---------------------
func (c *categoryMysqlRepo) GetCategoryById(ctx context.Context, id uint64) (*do.CategoryDO, error) {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	var category model.CategoryModel
	err := c.db.WithContext(ctx).First(&category, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("GetCategoryById: category not found", zap.Uint64("category_id", id))
		} else {
			logger.Error("GetCategoryById failed", zap.Uint64("category_id", id), zap.Error(err))
		}
		return nil, err
	}

	categoryDO := convert.CategoryModelToDO(&category)
	return categoryDO, nil
}

// --------------------- 4. 删除分类（软删除） ---------------------
func (c *categoryMysqlRepo) DeleteCategory(ctx context.Context, id uint64) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	err := c.db.WithContext(ctx).Delete(&model.CategoryModel{}, id).Error

	if err != nil {
		logger.Error("DeleteCategory failed", zap.Uint64("category_id", id), zap.Error(err))
		return err
	}
	if c.db.RowsAffected == 0 {
		logger.Warn("DeleteCategory no rows affected", zap.Uint64("category_id", id))
	}
	return nil
}

// --------------------- 5. 更新分类 ---------------------
func (c *categoryMysqlRepo) UpdateCategory(ctx context.Context, category *model.CategoryModel) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	err := c.db.WithContext(ctx).Save(category).Error

	if err != nil {
		logger.Error("UpdateCategory failed", zap.Uint64("category_id", category.ID), zap.String("name", category.Name), zap.Error(err))
		return err
	}
	if c.db.RowsAffected == 0 {
		logger.Warn("UpdateCategory no rows affected", zap.Uint64("category_id", category.ID))
	}
	return nil
}
