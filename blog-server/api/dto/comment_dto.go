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

// CommentPageQueryReq 评论分页查询请求，从 query string 绑定，只暴露客户端允许的筛选/排序/分页字段。
// 作者、审核状态等敏感条件的可见范围由服务端决定，不在此结构额外开放。
type CommentPageQueryReq struct {
	ArticleID uint64  `form:"articleID" json:"articleID"` // 大于 0 时按所属文章筛选
	ParentID  *uint64 `form:"parentID" json:"parentID"`   // nil 不筛选；指向 0 查顶级，指向 N 查某父评论的回复
	Status    *int    `form:"status" json:"status"`       // nil 不筛选；0 待审核 / 1 已通过 / 2 已拒绝
	AdminID   uint64  `form:"adminID" json:"adminID"`     // 大于 0 时按管理员作者筛选
	Nickname  string  `form:"nickname" json:"nickname"`   // 昵称模糊搜索
	Content   string  `form:"content" json:"content"`     // 内容模糊搜索
	Page      int     `form:"page" json:"page"`           // 页码，默认 1
	Size      int     `form:"size" json:"size"`           // 每页数量，默认 10
	OrderBy   string  `form:"orderBy" json:"orderBy"`     // 排序字段，由 Repo 白名单校验
	OrderDir  string  `form:"orderDir" json:"orderDir"`   // 排序方向 asc/desc
}
