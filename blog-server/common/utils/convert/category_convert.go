/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-18 17:49:49
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-06 11:53:28
 * @FilePath: \yia-blog\blog-server\common\utils\convert\category_convert.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package convert

import (
	"blog-server/api/dto"
	"blog-server/domain/do"
	"blog-server/infra/model"
	"time"
)

// -------------- DTO → 领域查询对象 转换函数 --------------
// CategoryPageDto2Query api分页DTO转领域查询
func CategoryPageDtoQuery(apiQuery dto.CategoryPageQuery) *do.CategoryQuery {
	return &do.CategoryQuery{
		PageNum:  apiQuery.PageNum,
		PageSize: apiQuery.PageSize,
		Name:     apiQuery.Name,
		Status:   apiQuery.Status,
	}
}

func CategoryModelToDO(m *model.CategoryModel) *do.CategoryDO {
	if m == nil || m.ID == 0 {
		return nil
	}
	return &do.CategoryDO{
		ID:        m.ID,
		Name:      m.Name,
		Sort:      m.Sort,
		Status:    m.Status,
		CreateAt:  m.CreateAt,
		UpdateAt:  m.UpdateAt,
		IsDeleted: !m.DeletedAt.IsZero(),
	}
}

// do.CategoryDO → model.CategoryModel
func CategoryDOToModel(d *do.CategoryDO) *model.CategoryModel {
	if d == nil || d.ID == 0 {
		return &model.CategoryModel{} // 空对象，缓存空值用
	}

	var deletedAt time.Time
	if d.IsDeleted {
		deletedAt = time.Now()
	}
	return &model.CategoryModel{
		ID:        d.ID,
		Name:      d.Name,
		Sort:      d.Sort,
		Status:    d.Status,
		CreateAt:  d.CreateAt,
		UpdateAt:  d.UpdateAt,
		DeletedAt: deletedAt,
	}
}

func CategoryModelListToDO(list *[]model.CategoryModel) []do.CategoryDO {
	if list == nil || len(*list) == 0 {
		return []do.CategoryDO{}
	}

	listDO := make([]do.CategoryDO, 0, len(*list))
	for _, m := range *list {
		listDO = append(listDO, do.CategoryDO{
			ID:        m.ID,
			Name:      m.Name,
			Sort:      m.Sort,
			Status:    m.Status,
			CreateAt:  m.CreateAt,
			UpdateAt:  m.UpdateAt,
			IsDeleted: !m.DeletedAt.IsZero(),
		})
	}
	return listDO
}

// CategoryModelListToDOList 模型列表转DO列表
func CategoryModelListToDOList(list []model.CategoryModel) []do.CategoryDO {
	res := make([]do.CategoryDO, 0, len(list))
	for _, v := range list {
		res = append(res, *CategoryModelToDO(&v))
	}
	return res
}

// CategoryDOListToModelList DO列表转模型列表
func CategoryDOListToModelList(list []do.CategoryDO) []model.CategoryModel {
	res := make([]model.CategoryModel, 0, len(list))
	for _, v := range list {
		res = append(res, *CategoryDOToModel(&v))
	}
	return res
}
