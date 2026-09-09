/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-14 01:14:58
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-14 01:15:43
 * @FilePath: \yia-blog\blog-server\api\dto\article_dto.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-07 04:10:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-14 01:15:34
 * @FilePath: \yia-blog\blog-server\api\dto\article_dto.go
 * @Description: 文章DTO
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package dto

import "time"

// CreateArticleReq 创建文章请求；CommentStatus 未传时由 Domain 默认允许，显式 0 表示关闭。
type CreateArticleReq struct {
	Title         string     `json:"title"`
	Cover         string     `json:"cover"`
	Summary       string     `json:"summary"`
	Content       string     `json:"content"`
	CategoryID    uint64     `json:"category_id"`
	TagIDs        string     `json:"tag_ids"`
	Status        int        `json:"status"`
	Sort          int        `json:"sort"`
	IsTop         int        `json:"is_top"`
	CommentStatus *int       `json:"comment_status"`
	PublishAt     *time.Time `json:"publish_at"`
}

// UpdateArticleReq 更新文章请求
type UpdateArticleReq struct {
	ID            uint64     `json:"id"`
	Title         string     `json:"title"`
	Cover         string     `json:"cover"`
	Summary       string     `json:"summary"`
	Content       string     `json:"content"`
	CategoryID    uint64     `json:"category_id"`
	TagIDs        string     `json:"tag_ids"`
	Status        int        `json:"status"`
	Sort          int        `json:"sort"`
	IsTop         int        `json:"is_top"`
	CommentStatus int        `json:"comment_status"`
	PublishAt     *time.Time `json:"publish_at"`
}

// ArticlePageQuery 文章分页查询
type ArticlePageQuery struct {
	PageNum    int     `json:"page_num" form:"page_num"`
	PageSize   int     `json:"page_size" form:"page_size"`
	Title      string  `json:"title" form:"title"`
	CategoryID *uint64 `json:"category_id" form:"category_id"`
	Status     *int    `json:"status" form:"status"` // 公开用例始终限定已发布，不能通过筛选读取草稿。
	IsTop      *int    `json:"is_top" form:"is_top"`
}

// AuthorInfo 文章作者信息（读取时按 user_id/admin_id 实时关联填充，作者不存在时为 null）
type AuthorInfo struct {
	Type      string `json:"type"`      // 作者类型：user-普通用户，admin-管理员
	Name      string `json:"name"`      // 作者名（用户昵称 / 管理员名）
	Avatar    string `json:"avatar"`    // 头像URL
	Signature string `json:"signature"` // 简介/签名（用户签名 / 管理员备注）
}
