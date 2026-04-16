/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-25 23:37:43
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-16 08:02:41
 * @FilePath: \yia-blog\blog-server\common\consts\consts.go
 * @Description:
 *
 * Copyright (c) 2026 by ${git_name_email}, All Rights Reserved.
 */
package consts

const (
	UserTokenBlackListPrefix  = "user:token:blacklist:"  // 用户Token黑名单
	AdminTokenBlackListPrefix = "admin:token:blacklist:" // 管理员Token黑名单
)

const (
	UserTokenKey    = "Authorization"       // 用户Token请求头
	AdminTokenKey   = "Admin-Authorization" // 管理员Token请求头
	CustomerUserKey = "customer_user"       // Gin Context中存储用户信息的key
	AdminUserKey    = "admin_user"          // Gin Context中存储管理员信息的key
	UserIDKey       = "user_id"             // Gin Context中存储用户ID的key
	AdminIDKey      = "admin_id"            // Gin Context中存储管理员ID的key
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
