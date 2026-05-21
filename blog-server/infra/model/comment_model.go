/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-06 11:05:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-21 11:40:11
 * @FilePath: \yia-blog\blog-server\infra\model\common_model.go
 * @Description: 评论模块模型
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package model

import (
	"time"

	"gorm.io/gorm"
)

// CommentModel 评论表模型
// 支持：二级评论/回复、关联文章、游客评论、管理员评论、审核状态、IP记录
type CommentModel struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement;comment:主键ID"`                       // 手动主键
	ArticleID uint64         `gorm:"not null;index:idx_article_id;comment:文章ID(关联article表主键)"`    // 关联文章ID
	ParentID  uint64         `gorm:"not null;default:0;index:idx_parent_id;comment:父评论ID 0-顶级评论"` // 父评论ID（0=顶级评论，非0=回复评论）
	AdminID   uint64         `gorm:"default:0;index:idx_admin_id;comment:管理员ID 0-游客评论"`           // 管理员评论ID（0=普通游客/用户）
	Nickname  string         `gorm:"type:varchar(64);not null;comment:评论用户昵称"`                    // 评论人昵称
	Email     string         `gorm:"type:varchar(64);default:null;comment:评论用户邮箱"`                // 评论人邮箱（可选）
	Avatar    string         `gorm:"type:varchar(255);default:null;comment:评论用户头像URL"`            // 评论人头像（可选）
	Content   string         `gorm:"type:longtext;not null;comment:评论内容"`                         // 评论正文
	IP        string         `gorm:"type:varchar(64);default:null;comment:评论IP地址"`                // 评论IP
	UserAgent string         `gorm:"type:varchar(512);default:null;comment:评论设备/浏览器信息"`           // 设备信息
	Status    int            `gorm:"type:tinyint(1);not null;default:0;comment:状态：0-待审核，1-已通过，2-已拒绝"`
	Likes     int            `gorm:"not null;default:0;comment:评论点赞数"`
	CreateAt  time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdateAt  time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间"`
	DeletedAt gorm.DeletedAt `gorm:"type:datetime;default:null;comment:软删除时间"`
}

// TableName 指定数据库表名
func (c *CommentModel) TableName() string {
	return "comment"
}
