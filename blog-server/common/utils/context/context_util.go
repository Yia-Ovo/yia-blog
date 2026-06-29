/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-06-10 01:38:09
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-10 01:38:21
 * @FilePath: \yia-blog\blog-server\common\utils\context\context_util.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-06-10 01:38:09
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-10 01:38:16
 * @FilePath: \yia-blog\blog-server\common\utils\context\context_util.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package context

import (
	"blog-server/common/consts"
	"context"
)

// IsAdmin 判断当前请求是否为管理员身份
func IsAdmin(ctx context.Context) bool {
	_, ok := ctx.Value(consts.AdminIDKey).(uint64)
	return ok
}

// GetUserID 获取当前登录的用户ID（普通用户）
func GetUserID(ctx context.Context) uint64 {
	uid, _ := ctx.Value(consts.UserIDKey).(uint64)
	return uid
}

// GetAdminID 获取当前登录的管理员ID
func GetAdminID(ctx context.Context) uint64 {
	aid, _ := ctx.Value(consts.AdminIDKey).(uint64)
	return aid
}
