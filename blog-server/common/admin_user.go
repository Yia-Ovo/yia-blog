/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-03-13 17:59:38
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-03-13 18:00:08
 * @FilePath: \yia-blog\blog-server\common\admin_user.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package common

// AdminUser 管理员信息（适配管理员认证）
type AdminUser struct {
	AdminID int64  `json:"admin_id"` // 管理员ID
	Name    string `json:"name"`     // 管理员姓名
	RoleID  int64  `json:"role_id"`  // 角色ID（RBAC权限用）
}
