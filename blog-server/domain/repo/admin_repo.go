/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-30 01:47:30
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-14 10:57:08
 * @FilePath: \yia-blog\blog-server\domain\repo\admin_repo.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import (
	"blog-server/infra/model"
	"context"
	"time"
)

type AdminMysqlRepo interface {
	UpdatePWD(ctx context.Context, adminID uint64, password string) error
	CreateAdmin(ctx context.Context, adminModel *model.AdminModel) error
	FindAdminByName(ctx context.Context, adminName string) (*model.AdminModel, error)
	FindAdminByID(ctx context.Context, adminID uint64) (*model.AdminModel, error)
	UpdateAdminInfo(ctx context.Context, adminModel *model.AdminModel) error
	UpdateLastLoginInfo(ctx context.Context, adminID uint64, loginAt *time.Time) error
}

type AdminRedisRepo interface {
	GetAdminFromRedis(ctx context.Context, adminID uint64) (*model.AdminModel, error)
	SetAdminToRedis(ctx context.Context, adminID uint64, admin *model.AdminModel) error
	DelAdminFromRedis(ctx context.Context, adminID uint64) error
	SetAdminOnlineToken(ctx context.Context, adminID uint64, token string, expire time.Duration) error
	DelAdminOnlineToken(ctx context.Context, adminID uint64, token string) error
	ClearAdminAllOnlineToken(ctx context.Context, adminID uint64) error
	AddAdminTokenToBlackList(ctx context.Context, token string, exp time.Duration) error
	IsAdminTokenBlacklisted(ctx context.Context, token string) (bool, error)
	GetAdminByNameFromRedis(ctx context.Context, username string) (*model.AdminModel, error)
	SetAdminByNameToRedis(ctx context.Context, username string, admin *model.AdminModel) error
}
