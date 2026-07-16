/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-06 11:01:05
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-09 08:08:24
 * @FilePath: \yia-blog\blog-server\infra\model\article_model.go
 * @Description: 文章表的 GORM 持久化模型
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package model

import (
	"time"

	"gorm.io/gorm"
)

// ArticleModel 文章表模型。
// UserID 和 AdminID 表示文章作者类型：用户文章时 UserID 非零，管理员文章时 AdminID 非零。
type ArticleModel struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement;comment:主键ID"`                                                     // 数据库自增主键
	UserID         uint64         `gorm:"not null;default:0;index:idx_user_id;comment:用户作者ID(与admin_id二选一)"`                         // 普通用户创建文章时记录用户ID；0 表示不是用户作者
	AdminID        uint64         `gorm:"not null;index:idx_admin_id;comment:管理员作者ID(与user_id二选一)"`                                  // 管理员创建文章时记录管理员ID；0 表示不是管理员作者
	Title          string         `gorm:"type:varchar(128);not null;index:idx_title;comment:文章标题"`                                   // 用于列表和详情展示的文章标题
	Cover          string         `gorm:"type:varchar(255);default:null;comment:文章封面地址"`                                             // 可为空的封面图片 URL
	Summary        string         `gorm:"type:varchar(512);default:null;comment:文章摘要"`                                               // 列表页使用的简短内容摘要
	Content        string         `gorm:"type:longtext;not null;comment:文章正文"`                                                       // 完整文章内容
	CategoryID     uint64         `gorm:"default:0;index:idx_category_id;comment:所属分类ID"`                                            // 逻辑关联 category 表；0 表示未分类
	TagIDs         string         `gorm:"type:varchar(255);default:'';comment:标签ID列表(逗号分隔)"`                                         // 当前以逗号字符串保存多个标签ID
	Status         int            `gorm:"type:tinyint(1);not null;default:0;comment:发布状态(0草稿,1已发布,2下架)"`                             // 控制文章是否对外展示
	Sort           int            `gorm:"type:int;not null;default:0;comment:排序权重(值越小越靠前)"`                                          // 同类文章排序时使用
	Views          uint64         `gorm:"not null;default:0;comment:浏览次数"`                                                           // 由文章浏览统计流程维护
	Likes          int            `gorm:"not null;default:0;comment:点赞数"`                                                            // 由文章点赞流程维护
	Comments       int            `gorm:"not null;default:0;comment:评论数"`                                                            // 文章评论数量的冗余统计字段
	CommentContext []CommentModel `gorm:"foreignKey:ArticleID;references:ID"`                                                        // GORM 评论关联集合，不对应 article 表中的实际列
	IsTop          int            `gorm:"type:tinyint(1);not null;default:0;comment:置顶状态(0不置顶,1置顶)"`                                 // 控制文章是否优先展示
	CommentStatus  int            `gorm:"type:tinyint(1);not null;default:1;comment:评论开关(0禁止,1允许)"`                                  // 控制文章是否允许新增评论
	PublishAt      *time.Time     `gorm:"type:datetime;default:null;comment:发布时间"`                                                   // 草稿未发布时可为 nil
	CreateAt       time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;comment:创建时间"`                             // 文章记录创建时间
	UpdateAt       time.Time      `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间"` // 文章记录最后更新时间
	DeletedAt      gorm.DeletedAt `gorm:"type:datetime;default:null;comment:软删除时间"`                                                  // 非空表示已软删除，普通查询会自动过滤
}

// TableName 指定数据库表名
func (a *ArticleModel) TableName() string {
	return "article"
}
