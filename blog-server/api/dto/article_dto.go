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

// CreateArticleReq 创建文章请求
type CreateArticleReq struct {
	AdminID       uint64     `json:"admin_id"`
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
	PageNum    int     `json:"page_num"`
	PageSize   int     `json:"page_size"`
	Title      string  `json:"title"`
	CategoryID *uint64 `json:"category_id"`
	Status     *int    `json:"status"`
	IsTop      *int    `json:"is_top"`
}
