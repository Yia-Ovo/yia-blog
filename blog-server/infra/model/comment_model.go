/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-06 11:05:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-16 16:34:05
 * @FilePath: \yia-blog\blog-server\infra\model\comment_model.go
 * @Description: 评论模块模型
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package model

import (
	"time"

	"gorm.io/gorm"
)

// CommentModel 评论表模型。
// UserID 和 AdminID 表示评论作者类型：用户评论时 UserID 非零，管理员评论时 AdminID 非零。
// Nickname、Email、Avatar 保存评论创建时的作者资料快照，避免用户资料变化影响历史评论展示。
type CommentModel struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement;comment:主键ID"`                                                     // 数据库自增主键
	UserID    uint64         `gorm:"not null;default:0;index:idx_user_id;comment:用户作者ID(与admin_id二选一)"`                         // 用户评论时记录登录用户ID；0 表示不是用户作者
	ArticleID uint64         `gorm:"not null;index:idx_article_id;comment:所属文章ID"`                                              // 评论所属文章的ID
	ParentID  uint64         `gorm:"not null;default:0;index:idx_parent_id;comment:父评论ID(0表示顶级评论)"`                             // 0 表示顶级评论，非 0 表示回复对应评论
	AdminID   uint64         `gorm:"default:0;index:idx_admin_id;comment:管理员作者ID(与user_id二选一)"`                                 // 管理员评论时记录管理员ID；0 表示不是管理员作者
	Nickname  string         `gorm:"type:varchar(64);not null;comment:作者昵称快照"`                                                  // 创建评论时保存的作者昵称
	Email     string         `gorm:"type:varchar(64);default:null;comment:作者邮箱快照"`                                              // 创建评论时保存的作者邮箱，可为空
	Avatar    string         `gorm:"type:varchar(255);default:null;comment:作者头像快照"`                                             // 创建评论时保存的头像地址，可为空
	Content   string         `gorm:"type:longtext;not null;comment:评论正文"`                                                       // 用户实际发表的评论内容
	IP        string         `gorm:"type:varchar(64);default:null;comment:评论请求IP"`                                              // 由服务端从 HTTP 请求中提取，不接受客户端直接指定
	UserAgent string         `gorm:"type:varchar(512);default:null;comment:评论请求User-Agent"`                                     // 由服务端从 HTTP 请求中提取的客户端信息
	Status    int            `gorm:"type:tinyint(1);not null;default:0;comment:审核状态(0待审核,1已通过,2已拒绝)"`                           // 新评论默认待审核，由审核流程修改
	Likes     int            `gorm:"not null;default:0;comment:点赞数"`                                                            // 由点赞/取消点赞流程维护，不接受创建或编辑请求直接修改
	CreateAt  time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;comment:创建时间"`                             // 评论首次创建时间
	UpdateAt  time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间"` // 评论最后更新时间
	DeletedAt gorm.DeletedAt `gorm:"type:datetime;default:null;comment:软删除时间"`                                                  // 非空表示评论已软删除，普通查询会自动过滤
}

// TableName 指定数据库表名
func (c *CommentModel) TableName() string {
	return "comment"
}
