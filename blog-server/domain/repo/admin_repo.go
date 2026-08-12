/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-30 01:47:30
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-08-12 12:09:16
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
	// UpdatePWD 管理员修改自己的密码（不限角色，超管同样适用）
	UpdatePWD(ctx context.Context, adminID uint64, password string) error
	// ResetAdminPWDByID 超管重置他人密码。SQL 侧固定约束目标为普通管理员，
	// 目标被并发提升为超管或已不存在时返回 ErrNoRowsAffected。
	// 不可用于「改自己密码」——那会误伤超管，请走 UpdatePWD。
	ResetAdminPWDByID(ctx context.Context, adminID uint64, password string) error
	// CreateAdmin 创建管理员
	CreateAdmin(ctx context.Context, adminModel *model.AdminModel) error
	// FindAdminByName 根据用户名查询管理员
	FindAdminByName(ctx context.Context, adminName string) (*model.AdminModel, error)
	// FindAdminByID 根据ID查询管理员
	FindAdminByID(ctx context.Context, adminID uint64) (*model.AdminModel, error)
	// UpdateAdminInfo 更新管理员个人资料（昵称/头像/邮箱/备注）。
	// 只写这四列与 update_at：role 由 ChangeAdminRole 维护、status 由 UpdateAdminStatus 维护，
	// 均不从本方法写入，避免读-改-写把旧权限值覆盖回去；也不触碰 token_version，改资料不踢登录态。
	UpdateAdminInfo(ctx context.Context, adminModel *model.AdminModel) error
	// UpdateLastLoginInfo 更新最后登录时间
	UpdateLastLoginInfo(ctx context.Context, adminID uint64, loginAt *time.Time) error
	// ChangeAdminRole 变更管理员角色，并递增 token_version 使目标的旧 token 立即失效。
	// 不接收操作者 id——操作者身份属于领域规则，由 DomainService 校验，不下沉到持久化层。
	ChangeAdminRole(ctx context.Context, targetID uint64, role int) error

	// 管理员列表分页查询
	ListAdmin(ctx context.Context, page, pageSize int) ([]*model.AdminModel, int64, error)
	// UpdateAdminStatus 更新账号状态（启用/禁用）。SQL 侧固定约束目标为普通管理员，
	// 目标被并发提升为超管或已不存在时返回 ErrNoRowsAffected；同时递增 token_version 使其旧 token 立即失效。
	UpdateAdminStatus(ctx context.Context, adminID uint64, status int) error
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
	// IsAdminOnlineToken 检查管理员在线token是否存在
	IsAdminOnlineToken(ctx context.Context, adminID uint64, token string) (bool, error)
}
