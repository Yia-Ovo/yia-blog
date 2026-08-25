/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-01 03:41:21
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-29 04:24:49
 * @FilePath: \yia-blog\blog-server\infra\model\user_model.go
 * @Description: 用户表的 GORM 持久化模型
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package model

import (
	"time"

	"gorm.io/gorm"
)

// UserModel 用户表模型，不内嵌 gorm.Model，以匹配现有表的 create_at/update_at 字段。
type UserModel struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement;comment:主键ID"`                              // 数据库自增主键
	NickName     string         `gorm:"type:varchar(64);not null;uniqueIndex:idx_nick_name;comment:用户登录昵称"` // 登录和展示使用的唯一昵称
	Password     string         `gorm:"type:varchar(255);not null;comment:bcrypt密码哈希" json:"-"` // 只保存 bcrypt 哈希；json:"-" 确保不写入 info 缓存、不返回客户端
	Avatar       string         `gorm:"type:varchar(255);default:null;comment:用户头像地址"`                                             // 可为空的头像 URL
	Gender       int            `gorm:"type:tinyint(1);not null;default:0;comment:性别(0未知,1男,2女)"`                                  // 性别枚举值
	Status       int            `gorm:"type:tinyint(1);not null;default:1;comment:账号状态(1启用,2禁用,3锁定)"`                              // 以 common/consts 中的用户状态常量为准
	TokenVersion uint64         `gorm:"type:bigint unsigned;not null;default:0;comment:令牌版本号"`                                     // 密码或状态变化时递增，使旧 JWT 立即失效
	LastLoginAt  *time.Time     `gorm:"type:datetime;default:null;comment:最后登录时间"`                                                 // 尚未成功登录时为 nil
	CreateAt     time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;comment:创建时间"`                             // 用户记录创建时间
	UpdateAt     time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间"` // 用户记录最后更新时间
	DeletedAt    gorm.DeletedAt `gorm:"type:datetime;default:null;comment:软删除时间"`                                                  // 非空表示已软删除，普通查询会自动过滤

	Signature string `gorm:"type:varchar(255);default:null;comment:个性签名"` // 用户公开展示的个性签名，可为空
	Email     string `gorm:"type:varchar(255);default:null;comment:用户邮箱"` // 可为空的联系邮箱
}

// TableName 指定数据库表名
func (u *UserModel) TableName() string {
	return "user"
}
