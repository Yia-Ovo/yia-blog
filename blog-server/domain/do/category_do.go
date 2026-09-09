/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 16:04:25
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-29 08:18:56
 * @FilePath: \yia-blog\blog-server\domain\do\category_do.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package do

import "time"

type CategoryDO struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Sort      int       `json:"sort"`
	Status    int       `json:"status"`
	CreateAt  time.Time `json:"create_at"`
	UpdateAt  time.Time `json:"update_at"`
	IsDeleted bool      `json:"is_deleted"`
}

// CategoryCreate 区分省略状态与显式禁用；主键和创建时间不接受客户端提交。
type CategoryCreate struct {
	Name   string
	Sort   int
	Status *int
}

// CategoryQuery 领域查询条件（给Repo用，和前端无关）
type CategoryQuery struct {
	PageNum  int
	PageSize int
	Name     string
	Status   *int
}

// CategoryUpdate 分类的部分更新入参（给 Repo 用，和前端无关）。
// Sort/Status 用指针表达「本次是否提交了该字段」：nil 表示保持库中原值，
// 非 nil 表示客户端显式指定——含 0，因为这两列的 0 都是合法业务值
// （sort=0 排在最前、status=0 禁用），用值类型无法与「没传」区分。
// 领域层刻意不替 nil 字段填充原值：未提交的列一路保持 nil 直到 Repo 组装更新白名单，
// 使 UPDATE 语句只包含本次真正修改的列，避免「读出原值再整体写回」在并发下
// 把他人刚提交的改动覆盖回旧值（Lost Update）。
type CategoryUpdate struct {
	ID     uint64 // 目标分类 ID，必须为正
	Name   string // 分类名称，领域层要求非空
	Sort   *int   // 排序值；nil = 本次不修改
	Status *int   // 状态 1 正常 / 0 禁用；nil = 本次不修改
}
