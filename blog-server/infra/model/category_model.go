/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 13:47:04
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-26 16:25:05
 * @FilePath: \yia-blog\blog-server\infra\model\category_model.go
 * @Description: 分类表的 GORM 持久化模型
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package model

import (
	"time"

	"gorm.io/gorm"
)

// CategoryModel 分类表模型，不内嵌 gorm.Model，以匹配现有表的 create_at/update_at 字段。
type CategoryModel struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement;comment:主键ID"`                                                     // 数据库自增主键
	Name      string         `gorm:"type:varchar(64);not null;uniqueIndex:idx_name;comment:分类名称"`                               // 分类名称；idx_name 为单列唯一索引，软删除后旧行仍占用该名称
	Sort      int            `gorm:"type:int;not null;default:0;comment:排序权重(值越小越靠前)"`                                          // 分类列表的展示顺序
	Status    int            `gorm:"type:tinyint(1);not null;default:1;comment:分类状态(0禁用,1启用)"`                                  // 禁用后不应作为可选分类展示
	CreateAt  time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;comment:创建时间"`                             // 分类记录创建时间
	UpdateAt  time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间"` // 分类记录最后更新时间
	DeletedAt gorm.DeletedAt `gorm:"type:datetime;default:null;comment:软删除时间"`                                                  // 非空表示已软删除，普通查询会自动过滤
}

// TableName 指定数据库表名
func (u *CategoryModel) TableName() string {
	return "category"
}
