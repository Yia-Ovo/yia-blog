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

// ---------------- 更新分类参数 ----------------
// UpdateCategoryReq 更新分类的请求体，只接受客户端真正拥有的三个字段。
// Sort/Status 用指针区分「未提交该字段」（nil，保持库中原值）与「提交了 0」：
// 这两列的 0 都是合法业务值（sort=0 排在最前、status=0 禁用），
// 若用值类型 int，客户端只提交 {id, name} 改个名字，
// 也会被当成「显式提交了 0」而把分类静默改成禁用 + 排序归零。
// create_at / deleted_at 等服务端字段不接受客户端提交，由 Repo 的更新白名单排除。
type UpdateCategoryReq struct {
	ID     uint64 `json:"id" binding:"required"`          // 目标分类 ID，必填
	Name   string `json:"name" binding:"required,max=64"` // 分类名称，必填，最长 64
	Sort   *int   `json:"sort"`                           // 排序值；nil 表示本次不修改
	Status *int   `json:"status"`                         // 状态 1 正常 / 0 禁用；nil 表示本次不修改
}
