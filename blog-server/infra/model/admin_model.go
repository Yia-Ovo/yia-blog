/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-30 01:25:35
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-30 01:26:48
 * @FilePath: \yia-blog\blog-server\infra\model\admin_model.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package model

import (
	"time"

	"gorm.io/gorm"
)

type AdminModel struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement;comment:主键ID"` // 手动主键
	AdminName   string         `gorm:"type:varchar(64);not null;uniqueIndex:idx_admin_name;comment:管理员账号(登录名)"`
	Password    string         `gorm:"type:varchar(255);not null;comment:加密密码(bcrypt哈希值)"`
	Avatar      string         `gorm:"type:varchar(255);default:null;comment:管理员头像URL"`
	Role        int            `gorm:"type:tinyint(1);not null;default:1;comment:角色：0-超级管理员，1-普通管理员"`
	Status      int            `gorm:"type:tinyint(1);not null;default:1;comment:状态：1-正常，0-禁用"`
	LastLoginAt *time.Time     `gorm:"type:datetime;default:null;comment:最后登录时间"`
	CreateAt    time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdateAt    time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间"`
	DeletedAt   gorm.DeletedAt `gorm:"type:datetime;default:null;comment:软删除时间"`

	Email  string `gorm:"type:varchar(255);default:null;comment:管理员邮箱"`
	Remark string `gorm:"type:varchar(255);default:null;comment:备注信息"`
}

// TableName 指定数据库表名
func (u *AdminModel) TableName() string {
	return "admin"
}
