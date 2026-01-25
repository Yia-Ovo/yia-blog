/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-26 00:29:52
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-01-26 00:30:13
 * @FilePath: \yia-blog\blog-server\common\auth.go
 * @Description:
 *
 * Copyright (c) 2026 by ${git_name_email}, All Rights Reserved.
 */
package common

type AdminUser struct {
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
}

type User struct {
	UserID   int64  `json:"user_id"`
	NickName string `json:"nick_name"`
}
