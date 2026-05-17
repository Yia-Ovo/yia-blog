/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-15 18:39:09
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-14 10:47:03
 * @FilePath: \yia-blog\blog-server\common\utils\convert\admin_convert.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package convert

import (
	"blog-server/domain/do"
	"blog-server/infra/model"
)

// Admin 管理员信息（适配管理员认证）
type Admin struct {
	ID        uint64 `json:"admin_id"` // 管理员ID
	AdminName string `json:"name"`     // 管理员姓名
	Role      int    `json:"role_id"`  // 角色ID（RBAC权限用）
	Avatar    string `json:"avatar"`   // 头像
}

// AdminModelToAdmin Model转前端脱敏对象
func AdminModelToAdmin(model *model.AdminModel) *Admin {
	return &Admin{
		ID:        model.ID,
		AdminName: model.AdminName,
		Role:      model.Role,
		Avatar:    model.Avatar,
	}
}

// AdminModelToDO Model转DO
func AdminModelToDO(model *model.AdminModel) *do.AdminDO {
	if model == nil {
		return nil
	}
	return &do.AdminDO{
		ID:          model.ID,
		AdminName:   model.AdminName,
		Password:    model.Password,
		Role:        model.Role,
		Avatar:      model.Avatar,
		Status:      model.Status,
		LastLoginAt: model.LastLoginAt,
		CreateAt:    model.CreateAt,
		UpdateAt:    model.UpdateAt,
	}
}

// AdminDOToModel DO转Model
func AdminDOToModel(do *do.AdminDO) *model.AdminModel {
	return &model.AdminModel{
		ID:          do.ID,
		AdminName:   do.AdminName,
		Password:    do.Password,
		Role:        do.Role,
		Avatar:      do.Avatar,
		Status:      do.Status,
		LastLoginAt: do.LastLoginAt,
		CreateAt:    do.CreateAt,
		UpdateAt:    do.UpdateAt,
	}
}

func AdminDOToAdmin(do *do.AdminDO) *Admin {
	return &Admin{
		ID:        do.ID,
		AdminName: do.AdminName,
		Role:      do.Role,
		Avatar:    do.Avatar,
	}
}
