/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 13:47:04
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-20 18:17:16
 * @FilePath: \yia-blog\blog-server\infra\model\category_model.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package model

import (
	"time"
)

type CategoryModel struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;comment:主键ID"`
	Name      string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_name;comment:分类名称"` // 唯一索引
	Sort      int       `gorm:"type:int;not null;default:0;comment:排序值，越小越靠前"`
	Status    int       `gorm:"type:tinyint(1);not null;default:1;comment:状态：1-正常，0-禁用"`
	CreateAt  time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdateAt  time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间"`
	DeletedAt time.Time `gorm:"type:datetime;default:null;comment:软删除时间"`
}
