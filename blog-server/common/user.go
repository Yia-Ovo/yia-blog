/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-26 00:29:52
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-03-06 12:29:14
 * @FilePath: \yia-blog\blog-server\common\user.go
 * @Description:
 *
 * Copyright (c) 2026 by ${git_name_email}, All Rights Reserved.
 */
package common

import (
	"blog-server/domain/do"
	"time"
)

type User struct {
	UserID        int64      `json:"user_id"`
	NickName      string     `json:"nick_name"`
	Status        int        `json:"status"`
	LastLoginIP   string     `json:"last_login_ip,omitempty"`
	LastLoginTime *time.Time `json:"last_login_time,omitempty"`
}

// UserDOToUser 函数，解决 nil 赋值和字段引用错误
// UserDOToUser 将领域层UserDO转换为通用User模型
func UserDOToUser(do *do.UserDO) *User {
	// 1. 优先处理入参为nil的情况：避免空指针解引用
	if do == nil {
		return nil
	}

	// 2. 初始化基础字段（无空指针风险的字段）
	user := &User{
		UserID:      do.ID,
		NickName:    do.NickName,
		Status:      do.Status,
		LastLoginIP: "", // UserDO中未定义LastLoginIP，暂时赋空（后续可在UserDO中补充该字段）
	}

	// 3. 处理可能为nil的时间字段：LastLoginAt（核心修复空指针）
	// 逻辑：如果UserDO的LastLoginAt不为nil，且不是零值，就直接赋值；否则置为nil（前端返回null）
	if do.LastLoginAt != nil && !do.LastLoginAt.IsZero() {
		user.LastLoginTime = do.LastLoginAt // 直接赋值*time.Time，类型匹配
	} else {
		user.LastLoginTime = nil // 无登录时间时，返回null给前端
	}

	return user
}
