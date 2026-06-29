/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-01 03:41:21
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-10 15:24:16
 * @FilePath: \yia-blog\blog-server\infra\model\user_model.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package model

import (
	"time"

	"gorm.io/gorm"
)

// UserModel 用户数据库实体，【不继承gorm.Model】，手动定义字段匹配已创建的user表
// 字段名和表中完全一致：create_at/update_at，而非created_at/updated_at
type UserModel struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement;comment:主键ID"` // 手动主键
	NickName    string         `gorm:"type:varchar(64);not null;uniqueIndex:idx_nick_name;comment:用户昵称"`
	Password    string         `gorm:"type:varchar(255);not null;comment:加密密码(bcrypt哈希值)"`
	Avatar      string         `gorm:"type:varchar(255);default:null;comment:用户头像URL"`
	Gender      int            `gorm:"type:tinyint(1);not null;default:0;comment:性别：0-未知，1-男，2-女"`
	Status      int            `gorm:"type:tinyint(1);not null;default:1;comment:状态：1-正常，0-禁用"`
	LastLoginAt *time.Time     `gorm:"type:datetime;default:null;comment:最后登录时间"`
	CreateAt    time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdateAt    time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间"`
	DeletedAt   gorm.DeletedAt `gorm:"type:datetime;default:null;comment:软删除时间"`

	Signature string `gorm:"type:varchar(255);default:null;comment:个性签名"`
	Email     string `gorm:"type:varchar(255);default:null;comment:邮箱"`
}

// TableName 指定数据库表名
func (u *UserModel) TableName() string {
	return "user"
}
