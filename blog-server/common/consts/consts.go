/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-25 23:37:43
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-17 14:42:13
 * @FilePath: \yia-blog\blog-server\common\consts\consts.go
 * @Description:
 *
 * Copyright (c) 2026 by ${git_name_email}, All Rights Reserved.
 */
package consts

const (
	UserTokenKey    = "User-Token"    // 前端请求头Token键
	CustomerUserKey = "customer_user" // 上下文存储User对象的键
	AdminTokenKey   = "Admin-Token"   // 管理员Token键
	AdminUserKey    = "admin_user"    // 上下文存储AdminUser对象的键
)

const (
	IsEnable  = 1
	IsDisable = -1
)

// 用户状态常量定义
const (
	UserStatusEnabled  = 1 // 正常启用
	UserStatusDisabled = 2 // 禁用
	UserStatusLocked   = 3 // 锁定
)
