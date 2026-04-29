/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 15:10:26
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-17 16:50:51
 * @FilePath: \yia-blog\blog-server\api\dto\category_dto.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package dto

// ---------------- 分页通用参数 ----------------
type PageParams struct {
	PageNum  int `json:"pageNum" form:"pageNum" binding:"min=1"`   // 页码，默认1
	PageSize int `json:"pageSize" form:"pageSize" binding:"min=1"` // 每页条数
}

// ---------------- 分类分页查询参数 ----------------
type CategoryPageQuery struct {
	PageParams        // 嵌套通用分页
	PageNum    int    `json:"pageNum" form:"pageNum" binding:"min=1"`   // 页码，默认1
	PageSize   int    `json:"pageSize" form:"pageSize" binding:"min=1"` // 每页条数
	Name       string `json:"name" form:"name"`                         // 分类名称（模糊查询）
	Status     *int   `json:"status" form:"status"`                     // 状态（1正常/0禁用，指针支持不传）
}

// ---------------- 分页通用返回结果 ----------------
type PageResult struct {
	Total    int64       `json:"total"`    // 总条数
	Pages    int         `json:"pages"`    // 总页数
	List     interface{} `json:"list"`     // 数据列表
	PageNum  int         `json:"pageNum"`  // 当前页码
	PageSize int         `json:"pageSize"` // 每页条数
}

// ---------------- 创建分类参数 ----------------
type CreateCategoryReq struct {
	Name   string `json:"name" binding:"required,max=64"` // 名称必填
	Sort   int    `json:"sort" default:"0"`               // 排序
	Status int    `json:"status" default:"1"`             // 状态
}
