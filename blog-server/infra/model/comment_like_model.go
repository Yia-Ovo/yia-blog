package model

import "time"

// CommentLikeModel 保存持久化点赞关系，复合主键使同一用户对同一评论最多保留一条记录。
// 取消点赞物理删除关系；计数变化与关系写入必须处于同一 MySQL 事务。
type CommentLikeModel struct {
	CommentID uint64    `gorm:"primaryKey;autoIncrement:false;not null;comment:评论ID"`
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false;not null;comment:点赞用户ID"`
	CreateAt  time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;comment:点赞时间"`
}

// TableName 与手工迁移中的点赞关系表名保持一致。
func (CommentLikeModel) TableName() string { return "comment_like" }
