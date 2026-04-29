/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-18 17:18:51
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-20 19:11:43
 * @FilePath: \yia-blog\blog-server\domain\repo\category_repo.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import (
	"blog-server/domain/do"
	"blog-server/infra/model"
	"context"
)

// CategoryRepo 分类仓储接口（抽象层，不依赖具体数据库）
type CategoryMysqlRepo interface {
	// CreateCategory 创建分类
	CreateCategory(ctx context.Context, category *model.CategoryModel) error
	// ListCategoryByPage 分页查询分类
	ListCategoryByPage(ctx context.Context, query *do.CategoryQuery) ([]do.CategoryDO, int64, error)
	// GetCategoryById 根据ID查询分类
	GetCategoryById(ctx context.Context, id uint64) (*do.CategoryDO, error)
	// UpdateCategory 更新分类
	UpdateCategory(ctx context.Context, category *model.CategoryModel) error
	// DeleteCategory 软删除分类
	DeleteCategory(ctx context.Context, id uint64) error
}
