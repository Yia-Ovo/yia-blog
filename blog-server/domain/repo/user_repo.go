/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:21:26
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-14 18:50:32
 * @FilePath: \yia-blog\blog-server\domain\repo\user_repo.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import (
	"blog-server/domain/do"
	"blog-server/infra/model"
	"context"
	"time"
)

type UserMysqlRepo interface {
	CreateUser(ctx context.Context, userDO *do.UserDO) error
	FindUserByNickname(ctx context.Context, nickName string) (*do.UserDO, error)
	FindUserByID(ctx context.Context, userID uint64) (*do.UserDO, error)
	UpdateUserInfo(ctx context.Context, userDO *do.UserDO) error
	UpdateLastLoginInfo(ctx context.Context, userID uint64, loginAt *time.Time) error
	UpdatePWD(ctx context.Context, userID uint64, password string) error
}

type UserRedisRepo interface {
	SetOnlineToken(ctx context.Context, userID uint64, token string, exp time.Duration) error
	DelUserOnlineToken(ctx context.Context, userID uint64, token string) error
	ClearUserAllOnlineToken(ctx context.Context, userID uint64) error
	AddTokenToBlackList(ctx context.Context, token string, exp time.Duration) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
	GetUserFromRedis(ctx context.Context, userID uint64) (*model.UserModel, error)
	SetUserToRedis(ctx context.Context, userID uint64, user *model.UserModel) error
	DelUserFromRedis(ctx context.Context, userID uint64) error
}
