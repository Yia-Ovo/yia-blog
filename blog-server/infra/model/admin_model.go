/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-30 01:25:35
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-29 04:25:04
 * @FilePath: \yia-blog\blog-server\infra\model\admin_model.go
 * @Description: 管理员表的 GORM 持久化模型
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package model

import (
	"time"

	"gorm.io/gorm"
)

// AdminModel 管理员表模型，不内嵌 gorm.Model，以匹配现有表的 create_at/update_at 字段。
type AdminModel struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement;comment:主键ID"`                                // 数据库自增主键
	AdminName    string         `gorm:"type:varchar(64);not null;uniqueIndex:idx_admin_name;comment:管理员登录账号"` // 管理员登录时使用的唯一账号
	Password     string         `gorm:"type:varchar(255);not null;comment:bcrypt密码哈希" json:"-"` // 只保存 bcrypt 哈希；json:"-" 确保不写入 info 缓存、不返回客户端
	Avatar       string         `gorm:"type:varchar(255);default:null;comment:管理员头像地址"`                                            // 可为空的头像 URL
	Role         int            `gorm:"type:tinyint(1);not null;default:1;comment:角色(0超级管理员,1普通管理员)"`                              // 权限角色，以 common/consts 中的管理员角色常量为准
	Status       int            `gorm:"type:tinyint(1);not null;default:1;comment:账号状态(0禁用,1启用)"`                                  // 禁用后鉴权中间件会拒绝该管理员
	TokenVersion uint64         `gorm:"type:bigint unsigned;not null;default:0;comment:令牌版本号"`                                     // 安全信息变化时递增，用于让旧 JWT 立即失效
	LastLoginAt  *time.Time     `gorm:"type:datetime;default:null;comment:最后登录时间"`                                                 // 尚未成功登录时为 nil
	CreateAt     time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;comment:创建时间"`                             // 管理员记录创建时间
	UpdateAt     time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间"` // 管理员记录最后更新时间
	DeletedAt    gorm.DeletedAt `gorm:"type:datetime;default:null;comment:软删除时间"`                                                  // 非空表示已软删除，普通查询会自动过滤

	Email  string `gorm:"type:varchar(255);default:null;comment:管理员邮箱"` // 可为空的联系邮箱
	Remark string `gorm:"type:varchar(255);default:null;comment:管理员备注"` // 后台内部备注
}

// TableName 指定数据库表名
func (u *AdminModel) TableName() string {
	return "admin"
}
