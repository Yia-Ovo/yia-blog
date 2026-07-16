/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 13:47:04
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-20 18:17:16
 * @FilePath: \yia-blog\blog-server\infra\model\category_model.go
 * @Description: 分类表的 GORM 持久化模型
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package model

import (
	"time"
)

// CategoryModel 分类表模型。
// 注意：DeletedAt 当前是 time.Time，不会触发 GORM 的自动软删除；模型也未声明 TableName()。
type CategoryModel struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID"`                                                     // 数据库自增主键
	Name      string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_name;comment:分类名称"`                               // 分类名称，数据库唯一索引防止重名
	Sort      int       `gorm:"type:int;not null;default:0;comment:排序权重(值越小越靠前)"`                                          // 分类列表的展示顺序
	Status    int       `gorm:"type:tinyint(1);not null;default:1;comment:分类状态(0禁用,1启用)"`                                  // 禁用后不应作为可选分类展示
	CreateAt  time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;comment:创建时间"`                             // 分类记录创建时间
	UpdateAt  time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间"` // 分类记录最后更新时间
	DeletedAt time.Time `gorm:"type:datetime;default:null;comment:删除时间"`                                                   // 当前只是普通时间列，GORM 不会自动写入或过滤
}

// 当前未实现 TableName，GORM 默认会将该模型映射到 category_models 表。
