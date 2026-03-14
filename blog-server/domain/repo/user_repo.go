/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:21:26
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-03-13 16:48:32
 * @FilePath: \yia-blog\blog-server\domain\repo\user_repo.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import (
	"blog-server/domain/do"
	"context"
	"time"
)

type UserRepo interface {
	CreateUser(ctx context.Context, userDO *do.UserDO) error
	FindUserByNickname(ctx context.Context, nickName string) (*do.UserDO, error)
	FindUserByID(ctx context.Context, id int64) (*do.UserDO, error)
	UpdateUserInfo(ctx context.Context, userDO *do.UserDO) error
	UpdateLastLoginInfo(ctx context.Context, id int64, loginAt *time.Time) error
}
