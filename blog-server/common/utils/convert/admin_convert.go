/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-15 18:39:09
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-10 15:56:45
 * @FilePath: \yia-blog\blog-server\common\utils\convert\admin_convert.go
 * @Description: 管理员实体类型转换工具
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package convert

import (
	"blog-server/api/dto"
	"blog-server/domain/do"
	"blog-server/infra/model"
)

// AdminModelToDO Model转DO
func AdminModelToDO(model *model.AdminModel) *do.AdminDO {
	if model == nil {
		return nil
	}
	return &do.AdminDO{
		ID:           model.ID,
		AdminName:    model.AdminName,
		Password:     model.Password,
		Role:         model.Role,
		Avatar:       model.Avatar,
		Email:        model.Email,
		Remark:       model.Remark,
		Status:       model.Status,
		TokenVersion: model.TokenVersion,
		LastLoginAt:  model.LastLoginAt,
		CreateAt:     model.CreateAt,
		UpdateAt:     model.UpdateAt,
	}
}

// AdminDOToModel 领域对象转 GORM 模型。
// 入参 nil 返回 nil，由调用方在第一次解引用前判空（convert 层统一 nil 契约）。
// 空值防穿透缓存不依赖本函数返回空对象：isEmptyAdmin 同时接受 nil 与 ID==0。
func AdminDOToModel(do *do.AdminDO) *model.AdminModel {
	if do == nil {
		return nil
	}
	return &model.AdminModel{
		ID:           do.ID,
		AdminName:    do.AdminName,
		Password:     do.Password,
		Role:         do.Role,
		Avatar:       do.Avatar,
		Email:        do.Email,
		Remark:       do.Remark,
		Status:       do.Status,
		TokenVersion: do.TokenVersion,
		LastLoginAt:  do.LastLoginAt,
		CreateAt:     do.CreateAt,
		UpdateAt:     do.UpdateAt,
	}
}

// AdminDOToAdminBaseDTO DO转基础DTO（登录用）
func AdminDOToAdminBaseDTO(do *do.AdminDO) dto.AdminBaseDTO {
	return dto.AdminBaseDTO{
		ID:           do.ID,
		AdminName:    do.AdminName,
		Avatar:       do.Avatar,
		Role:         do.Role,
		Status:       do.Status,
		TokenVersion: do.TokenVersion,
		Email:        do.Email,
		Remark:       do.Remark,
	}
}

// AdminDOToAdminInfoData DO转详情DTO
func AdminDOToAdminInfoData(do *do.AdminDO) *dto.AdminInfoData {
	return &dto.AdminInfoData{
		AdminBaseDTO: AdminDOToAdminBaseDTO(do),
		LastLoginAt:  do.LastLoginAt,
		CreateAt:     do.CreateAt,
		UpdateAt:     do.UpdateAt,
	}
}
