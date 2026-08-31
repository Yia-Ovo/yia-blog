/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-18 17:49:49
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-08-26 00:00:00
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

	"gorm.io/gorm"
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
		IsDeleted: m.DeletedAt.Valid,
	}
}

// CategoryDOToModel 领域对象转 GORM 模型。
// ID==0 是「新建、主键待数据库自增」的合法状态，不能当作无效 DO 拦截，
// 否则创建路径的 Name/Sort/Status 会被整体丢弃、写入空行。
// 仅 d 为 nil 时返回 nil，与 CategoryModelToDO 的 nil 约定保持对称，由调用方判空。
func CategoryDOToModel(d *do.CategoryDO) *model.CategoryModel {
	if d == nil {
		return nil
	}

	var deletedAt gorm.DeletedAt
	if d.IsDeleted {
		deletedAt = gorm.DeletedAt{
			Time:  time.Now(),
			Valid: true,
		}
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
			IsDeleted: m.DeletedAt.Valid,
		})
	}
	return listDO
}

// CategoryModelListToDOList 模型列表转DO列表。
// CategoryModelToDO 对 nil / ID==0 的元素返回 nil，此处跳过而非解引用，避免 panic。
func CategoryModelListToDOList(list []*model.CategoryModel) []do.CategoryDO {
	res := make([]do.CategoryDO, 0, len(list))
	for _, v := range list {
		d := CategoryModelToDO(v)
		if d == nil {
			continue
		}
		res = append(res, *d)
	}
	return res
}
