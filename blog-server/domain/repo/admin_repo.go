/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-30 01:47:30
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-08 14:43:40
 * @FilePath: \yia-blog\blog-server\domain\repo\admin_repo.go
 * @Description: 管理员仓储层接口定义
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import (
	"blog-server/infra/model"
	"context"
	"time"
)

// AdminMysqlRepo 管理员 MySQL 仓储接口
type AdminMysqlRepo interface {
	// UpdatePWD 修改密码
	UpdatePWD(ctx context.Context, adminID uint64, password string) error
	// CreateAdmin 创建管理员
	CreateAdmin(ctx context.Context, adminModel *model.AdminModel) error
	// FindAdminByName 根据用户名查询管理员
	FindAdminByName(ctx context.Context, adminName string) (*model.AdminModel, error)
	// FindAdminByID 根据ID查询管理员
	FindAdminByID(ctx context.Context, adminID uint64) (*model.AdminModel, error)
	// UpdateAdminInfo 更新管理员信息
	UpdateAdminInfo(ctx context.Context, adminModel *model.AdminModel) error
	// UpdateLastLoginInfo 更新最后登录时间
	UpdateLastLoginInfo(ctx context.Context, adminID uint64, loginAt *time.Time) error

	// 管理员列表分页查询
	ListAdmin(ctx context.Context, page, pageSize int) ([]*model.AdminModel, int64, error)
	// 更新账号状态
	UpdateAdminStatus(ctx context.Context, adminID uint64, status int) error
	// 批量清理缓存（账号变更后）
}

// AdminRedisRepo 管理员 Redis 仓储接口
type AdminRedisRepo interface {
	// GetAdminFromRedis 根据ID获取管理员缓存
	GetAdminFromRedis(ctx context.Context, adminID uint64) (*model.AdminModel, error)
	// SetAdminToRedis 设置管理员缓存
	SetAdminToRedis(ctx context.Context, adminID uint64, admin *model.AdminModel) error
	// DelAdminFromRedis 删除管理员缓存
	DelAdminFromRedis(ctx context.Context, adminID uint64) error

	// SetAdminOnlineToken 设置管理员在线令牌
	SetAdminOnlineToken(ctx context.Context, adminID uint64, token string, expire time.Duration) error
	// DelAdminOnlineToken 删除管理员单端在线令牌
	DelAdminOnlineToken(ctx context.Context, adminID uint64, token string) error
	// ClearAdminAllOnlineToken 清空管理员所有在线令牌
	ClearAdminAllOnlineToken(ctx context.Context, adminID uint64) error

	// AddAdminTokenToBlackList 将令牌加入黑名单
	AddAdminTokenToBlackList(ctx context.Context, token string, exp time.Duration) error
	// IsAdminTokenBlacklisted 检查令牌是否在黑名单
	IsAdminTokenBlacklisted(ctx context.Context, token string) (bool, error)
}
