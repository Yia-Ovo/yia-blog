/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-07 03:12:33
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-07 03:13:42
 * @FilePath: \yia-blog\blog-server\domain\do\article_do.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-07 00:00:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-07 03:13:41
 * @FilePath: \yia-blog\blog-server\domain\do\article_do.go
 * @Description: 文章领域 DO 对象
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package do

import "time"

// ArticleDO 文章领域对象
type ArticleDO struct {
	ID            uint64
	AdminID       uint64
	Title         string
	Cover         string
	Summary       string
	Content       string
	CategoryID    uint64
	TagIDs        string
	Status        int
	Sort          int
	Views         uint64
	Likes         int
	Comments      int
	IsTop         int
	CommentStatus int
	PublishAt     *time.Time
	CreateAt      time.Time
	UpdateAt      time.Time
}

// ArticleQuery 文章分页查询条件
type ArticleQuery struct {
	PageNum    int
	PageSize   int
	Title      string  // 文章标题模糊查询
	CategoryID *uint64 // 分类ID
	Status     *int    // 文章状态
	IsTop      *int    // 是否置顶
}
