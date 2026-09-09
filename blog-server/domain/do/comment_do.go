/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-22 00:18:44
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-16 16:33:41
 * @FilePath: \yia-blog\blog-server\domain\do\comment_do.go
 * @Description: 评论领域对象与评论查询条件
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package do

import (
	"time"

	"gorm.io/gorm"
)

// CommentDO 评论领域对象。
// UserID 和 AdminID 表示作者类型：用户评论时 UserID 非零，管理员评论时 AdminID 非零。
// 作者快照、请求来源、审核状态、点赞数和时间字段均由服务端流程维护，不直接绑定客户端 JSON。
type CommentDO struct {
	ID        uint64         // 评论ID，由数据库生成
	UserID    uint64         // 用户作者ID；0 表示不是用户作者
	ArticleID uint64         // 评论所属文章ID
	ParentID  uint64         // 0 表示顶级评论，非 0 表示回复的父评论ID
	AdminID   uint64         // 管理员作者ID；0 表示不是管理员作者
	Nickname  string         // 评论创建时保存的作者昵称快照
	Email     string         // 评论创建时保存的作者邮箱快照
	Avatar    string         // 评论创建时保存的作者头像快照
	Content   string         // 评论正文
	IP        string         // 服务端从 HTTP 请求中提取的客户端IP
	UserAgent string         // 服务端从 HTTP 请求中提取的客户端 User-Agent
	Status    int            // 审核状态：0 待审核，1 已通过，2 已拒绝
	Likes     int            // 点赞数，由点赞流程维护
	CreateAt  time.Time      // 评论创建时间
	UpdateAt  time.Time      // 评论最后更新时间
	DeletedAt gorm.DeletedAt // GORM 软删除标记，非空表示已删除
}

// CommentQuery 评论列表查询条件。
// 由 App 从 HTTP DTO 构造，公开范围再由 Domain 强制设置，Repo 转换为 SQL 条件。
type CommentQuery struct {
	// 精确筛选条件
	ArticleID uint64  `form:"articleID" json:"articleID"` // 大于 0 时按所属文章筛选；0 表示不按文章筛选
	ParentID  *uint64 `form:"parentID" json:"parentID"`   // nil 不按父评论筛选；指向 0 只查顶级；指向 N 查某父评论的回复
	Status    *int    `form:"status" json:"status"`       // nil 不按状态筛选；可显式筛选 0 待审核 / 1 已通过 / 2 已拒绝
	AdminID   uint64  `form:"adminID" json:"adminID"`     // 大于 0 时按管理员作者筛选；0 表示不按管理员筛选

	// 模糊搜索条件
	Nickname string `form:"nickname" json:"nickname"` // 搜索评论人昵称
	Content  string `form:"content" json:"content"`   // 搜索评论内容

	// 分页参数，未传或小于等于 0 时由服务端设置默认值
	Page int `form:"page" json:"page"` // 页码，默认 1
	Size int `form:"size" json:"size"` // 每页数量，默认 10

	// 排序参数，进入 Repo 前应限制为允许的字段和值
	OrderBy  string `form:"orderBy" json:"orderBy"`   // 排序字段：create_at、likes、id
	OrderDir string `form:"orderDir" json:"orderDir"` // 排序方向：asc / desc

	// 软删除控制预留字段；当前 Repo 尚未根据该字段调用 Unscoped
	WithDeleted bool `form:"withDeleted" json:"withDeleted"` // 预期仅供管理员查询已删除评论
	// PublishedArticleOnly 由公开领域用例强制开启，客户端不能关闭。
	PublishedArticleOnly bool `form:"-" json:"-"`
}
