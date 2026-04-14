/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-14 18:30:10
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-15 00:47:37
 * @FilePath: \yia-blog\blog-server\common\utils\convert\user_convert.go
 * @Description:用户模型定义 + 全类型转换（DO/Model/通用User）
 查询流向 (Read)
	MySQL (Repo) -> do.UserDO (Domain) -> 转换 -> 业务逻辑使用
	Redis (Repo) -> model.UserModel (Infra) -> 转换 -> 业务逻辑使用
 缓存流向 (Cache)
	Domain 层拿到 DO 后，不能直接把 DO 塞给 Redis。
	正确流程：Domain 层把 DO 转换 为 Model -> 调用 Repo -> 写入 Redis。
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
*/

package convert

import (
	"blog-server/domain/do"
	"blog-server/infra/model"
	"time"
)

type User struct {
	UserID        uint64     `json:"user_id"`
	NickName      string     `json:"nick_name"`
	Status        int        `json:"status"`
	LastLoginIP   string     `json:"last_login_ip,omitempty"`
	LastLoginTime *time.Time `json:"last_login_time,omitempty"`
}

// ====================== 1. Redis/数据库 Model ↔ 领域 DO（缓存核心） ======================
// UserModelToDO model.UserModel → do.UserDO
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
		CreateAt:    m.CreateAt,
		UpdateAt:    m.UpdateAt,
		Email:       m.Email,
	}
}

// UserDO2Model do.UserDO → model.UserModel
func UserDOToModel(d *do.UserDO) *model.UserModel {
	if d == nil || d.ID == 0 {
		return &model.UserModel{} // 空对象，缓存空值用
	}
	return &model.UserModel{
		ID:          d.ID,
		NickName:    d.NickName,
		Password:    d.Password,
		Status:      d.Status,
		LastLoginAt: d.LastLoginAt,
		CreateAt:    d.CreateAt,
		UpdateAt:    d.UpdateAt,
		// 补充你所有字段
	}
}

// ====================== 2. 领域 DO → 前端通用 User ======================
// UserDOToUser do.UserDO → convert.User
func UserDOToUser(do *do.UserDO) *User {
	if do == nil {
		return nil
	}

	user := &User{
		UserID:      do.ID,
		NickName:    do.NickName,
		Status:      do.Status,
		LastLoginIP: "",
	}

	if do.LastLoginAt != nil && !do.LastLoginAt.IsZero() {
		user.LastLoginTime = do.LastLoginAt
	} else {
		user.LastLoginTime = nil
	}

	return user
}
