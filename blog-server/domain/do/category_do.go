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

// CategoryQuery 领域查询条件（给Repo用，和前端无关）
type CategoryQuery struct {
	PageNum  int
	PageSize int
	Name     string
	Status   *int
}
