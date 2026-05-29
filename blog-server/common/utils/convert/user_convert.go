/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-14 18:30:10
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 13:46:10
 * @FilePath: \yia-blog\blog-server\common\utils\convert\user_convert.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package convert

import (
	"blog-server/api/dto"
	"blog-server/domain/do"
	"blog-server/infra/model"
)

// ====================== Model ↔ DO（数据库/Redis <-> 领域） ======================

func UserModelToDO(m *model.UserModel) *do.UserDO {
	if m == nil || m.ID == 0 {
		return nil
	}
	return &do.UserDO{
		ID:          m.ID,
		NickName:    m.NickName,
		Password:    m.Password,
		Status:      m.Status,
		LastLoginAt: m.LastLoginAt,
		Avatar:      m.Avatar,
		Gender:      m.Gender,
		Signature:   m.Signature,
		Email:       m.Email,
		CreateAt:    m.CreateAt,
		UpdateAt:    m.UpdateAt,
	}
}

func UserDOToModel(d *do.UserDO) *model.UserModel {
	if d == nil || d.ID == 0 {
		return &model.UserModel{}
	}
	return &model.UserModel{
		ID:          d.ID,
		NickName:    d.NickName,
		Password:    d.Password,
		Status:      d.Status,
		LastLoginAt: d.LastLoginAt,
		Avatar:      d.Avatar,
		Gender:      d.Gender,
		Signature:   d.Signature,
		Email:       d.Email,
		CreateAt:    d.CreateAt,
		UpdateAt:    d.UpdateAt,
	}
}

// ====================== DO → DTO（领域 -> 前端返回） ======================

// UserDOToUserInfoDTO 用于用户详情接口
func UserDOToUserInfoDTO(do *do.UserDO) *dto.UserInfoData {
	if do == nil {
		return nil
	}
	return &dto.UserInfoData{
		ID:          do.ID,
		NickName:    do.NickName,
		Avatar:      do.Avatar,
		Status:      do.Status,
		Gender:      do.Gender,
		Signature:   do.Signature,
		Email:       do.Email,
		CreateAt:    do.CreateAt,
		UpdateAt:    do.UpdateAt,
		LastLoginAt: do.LastLoginAt,
	}
}

// UserDOToLoginUserDTO 用于登录返回（精简字段，JWT 存储）
// 替代你原来冗余的 convert.User
func UserDOToLoginUserDTO(do *do.UserDO) *dto.LoginUserDTO {
	if do == nil {
		return nil
	}
	return &dto.LoginUserDTO{
		UserID:      do.ID,
		NickName:    do.NickName,
		Status:      do.Status,
		LastLoginAt: do.LastLoginAt,
	}
}
