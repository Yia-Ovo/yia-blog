/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-07-16 16:23:04
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-16 16:23:18
 * @FilePath: \yia-blog\blog-server\api\dto\comment_dto.go
 * @Description: 评论接口请求 DTO，只暴露客户端允许提交的字段
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package dto

import "time"

// CommentCreateReq 创建评论请求。
// 作者身份、昵称、头像、IP、User-Agent、审核状态和点赞数均由服务端生成，不允许客户端提交。
type CommentCreateReq struct {
	ArticleID uint64 `json:"article_id" binding:"required"`       // 评论所属文章ID
	ParentID  uint64 `json:"parent_id"`                           // 0 表示顶级评论，非 0 表示回复对应评论
	Content   string `json:"content" binding:"required,max=1000"` // 评论正文，最多 1000 个字符
}

// CommentUpdateReq 用户编辑评论请求，只允许修改自己评论的正文。
type CommentUpdateReq struct {
	ID      uint64 `json:"id" binding:"required"`               // 要编辑的评论ID
	Content string `json:"content" binding:"required,max=1000"` // 新的评论正文
}

// CommentDeleteReq 用户删除评论请求，评论归属由 DomainService 校验。
type CommentDeleteReq struct {
	ID uint64 `json:"id" binding:"required"` // 要删除的评论ID
}

// CommentPageQueryReq 公开评论分页查询请求，从 query string 绑定，只暴露访客允许使用的筛选/排序/分页字段。
// 审核状态不在此开放：可见范围由 DomainService 强制为"仅已通过"，避免访客借筛选条件翻出待审核/已拒绝内容。
type CommentPageQueryReq struct {
	ArticleID uint64  `form:"articleID" json:"articleID"` // 大于 0 时按所属文章筛选
	ParentID  *uint64 `form:"parentID" json:"parentID"`   // nil 不筛选；指向 0 查顶级，指向 N 查某父评论的回复
	Nickname  string  `form:"nickname" json:"nickname"`   // 昵称模糊搜索
	Content   string  `form:"content" json:"content"`     // 内容模糊搜索
	Page      int     `form:"page" json:"page"`           // 页码，默认 1
	Size      int     `form:"size" json:"size"`           // 每页数量，默认 10
	OrderBy   string  `form:"orderBy" json:"orderBy"`     // 排序字段，由 Repo 白名单校验
	OrderDir  string  `form:"orderDir" json:"orderDir"`   // 排序方向 asc/desc
}

// CommentVO 评论公开展示对象，是公开读接口的输出白名单。
// 结构上只保留访客可见字段：作者邮箱、IP、User-Agent、审核状态、作者ID 和软删除标记
// 都不在此结构中，因此不可能随响应泄露出去。
// 脱敏靠"换结构"而不是给 CommentDO 打 json:"-"，避免序列化细节渗入领域层，
// 也便于将来给管理端单独定义包含审核字段的 VO。
type CommentVO struct {
	ID        uint64    `json:"id"`         // 评论ID
	ArticleID uint64    `json:"article_id"` // 所属文章ID
	ParentID  uint64    `json:"parent_id"`  // 0 表示顶级评论，非 0 表示回复的父评论ID
	Nickname  string    `json:"nickname"`   // 作者昵称快照
	Avatar    string    `json:"avatar"`     // 作者头像快照
	Content   string    `json:"content"`    // 评论正文
	Likes     int       `json:"likes"`      // 点赞数
	CreateAt  time.Time `json:"create_at"`  // 评论创建时间
}
