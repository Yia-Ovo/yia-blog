/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-06 11:01:05
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-06 11:01:53
 * @FilePath: \yia-blog\blog-server\infra\model\article_model.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package model

import (
	"time"

	"gorm.io/gorm"
)

type ArticleModel struct {
	ID            uint64         `gorm:"primaryKey;autoIncrement;comment:主键ID"`                   // 手动主键
	AdminID       uint64         `gorm:"not null;index:idx_admin_id;comment:作者ID(关联admin表主键)"`    // 文章作者（关联管理员表）
	Title         string         `gorm:"type:varchar(128);not null;index:idx_title;comment:文章标题"` // 文章标题
	Cover         string         `gorm:"type:varchar(255);default:null;comment:文章封面URL"`          // 文章封面图片
	Summary       string         `gorm:"type:varchar(512);default:null;comment:文章摘要"`             // 文章简介/摘要
	Content       string         `gorm:"type:longtext;not null;comment:文章内容"`                     // 文章正文（长文本）
	CategoryID    uint64         `gorm:"default:0;index:idx_category_id;comment:分类ID"`            // 文章分类ID
	TagIDs        string         `gorm:"type:varchar(255);default:'';comment:标签ID(逗号分隔)"`         // 关联标签（多标签用逗号分隔）
	Status        int            `gorm:"type:tinyint(1);not null;default:0;comment:状态：0-草稿，1-已发布，2-下架"`
	Sort          int            `gorm:"type:int;not null;default:0;comment:排序权重(数字越小越靠前)"`
	Views         uint64         `gorm:"not null;default:0;comment:浏览量"`
	Likes         int            `gorm:"not null;default:0;comment:点赞数"`
	Comments      int            `gorm:"not null;default:0;comment:评论数"`
	IsTop         int            `gorm:"type:tinyint(1);not null;default:0;comment:是否置顶：1-是，0-否"`
	CommentStatus int            `gorm:"type:tinyint(1);not null;default:1;comment:评论状态：1-允许，0-禁止"`
	PublishAt     *time.Time     `gorm:"type:datetime;default:null;comment:发布时间"`
	CreateAt      time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdateAt      time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间"`
	DeletedAt     gorm.DeletedAt `gorm:"type:datetime;default:null;comment:软删除时间"`
}

// TableName 指定数据库表名
func (a *ArticleModel) TableName() string {
	return "article"
}
