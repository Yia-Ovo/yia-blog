/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-16 08:26:43
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-17 12:44:21
 * @FilePath: \yia-blog\blog-server\infra\repo\admin_redis_repo.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-16 08:26:43
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-16 09:51:43
 * @FilePath: \yia-blog\blog-server\infra\repo\admin_redis_repo.go
 * @Description:管理员 Redis 仓储实现
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import (
	"blog-server/common"
	"blog-server/common/utils/logger"
	"blog-server/config"
	"blog-server/infra/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	adminInfoKeyPrefix       = "blog:admin:info:"
	adminOnlineKeyPrefix     = "blog:admin:online:"
	adminBlackTokenKeyPrefix = "blog:admin:logout:black:"
	adminOnlineTokenValue    = "1" // 管理员在线令牌标记值

	adminInfoIDKey   = "admin:info:id:%d"   // 按ID缓存
	adminInfoNameKey = "admin:info:name:%s" // 按用户名缓存
)

type AdminRedisRepo struct {
	rdb    *redis.Client
	config *config.RedisConfig // 注入Redis配置
}

func NewAdminRedisRepo(rdb *redis.Client, cfg *config.GlobalConfig) *AdminRedisRepo {
	return &AdminRedisRepo{rdb: rdb, config: &cfg.Redis}
}

// 私有工具方法：统一构建Redis Key
func buildAdminKey(adminID uint64) string {
	return fmt.Sprintf("%s%d", adminInfoKeyPrefix, adminID)
}
func buildAdminOnlineTokenKey(adminID uint64, token string) string {
	return fmt.Sprintf("%s%d:%s", adminOnlineKeyPrefix, adminID, token)
}
func buildAdminBlackTokenKey(token string) string {
	return adminBlackTokenKeyPrefix + token
}

// isEmptyAdmin 判断是否为【空管理员对象】（防穿透标记）
func isEmptyAdmin(admin *model.AdminModel) bool {
	// 规则：管理员ID为0 或 用户名为空 → 判定为不存在的管理员
	return admin == nil || admin.ID == 0
}

// ====================== 管理员信息缓存 ======================
// GetAdminFromRedis 从Redis获取管理员信息
func (r *AdminRedisRepo) GetAdminFromRedis(ctx context.Context, adminID uint64) (*model.AdminModel, error) {

	key := buildAdminKey(adminID)
	data, err := r.rdb.Get(ctx, key).Result()

	// 1. 缓存不存在
	if errors.Is(err, redis.Nil) {
		logger.Warn("cache admin info not found", zap.Uint64("admin_id", adminID))
		return nil, common.CacheNotExist
	}
	// 2. Redis命令执行失败
	if err != nil {
		logger.Error("redis get admin info failed", zap.Uint64("admin_id", adminID), zap.Error(err))
		return nil, common.RedisCmdErr.WithErr(err)
	}

	// 3. 反序列化
	var admin model.AdminModel
	if err := json.Unmarshal([]byte(data), &admin); err != nil {
		logger.Error("redis unmarshal admin info failed", zap.Uint64("admin_id", adminID), zap.Error(err))
		// 关键：反序列化失败，删除脏缓存
		_ = r.rdb.Del(ctx, key)
		return nil, common.RedisUnmarshalErr.WithErr(err)
	}

	// 核心：判断空值缓存
	if isEmptyAdmin(&admin) {
		logger.Warn("get empty admin from cache", zap.Uint64("admin_id", adminID))
		return nil, nil
	}

	return &admin, nil
}

func (r *AdminRedisRepo) SetAdminToRedis(ctx context.Context, adminID uint64, admin *model.AdminModel) error {

	key := buildAdminKey(adminID)
	var data []byte
	var err error
	var expire time.Duration
	// 核心：空值处理
	if isEmptyAdmin(admin) {
		// 1. 空管理员：序列化空对象 + 短过期
		emptyAdmin := &model.AdminModel{}
		data, err = json.Marshal(emptyAdmin)
		if err != nil {
			logger.Error("redis marshal empty admin failed", zap.Uint64("admin_id", adminID), zap.Error(err))
			return common.RedisMarshalErr.WithErr(err)
		}
	} else {
		// 2. 正常管理员：序列化 + 长过期
		data, err = json.Marshal(admin)
		if err != nil {
			logger.Error("redis marshal admin failed", zap.Uint64("admin_id", adminID), zap.Error(err))
			return common.RedisMarshalErr.WithErr(err)
		}
		expire = r.config.UserCacheExpire // 复用配置
	}

	// 写入 Redis
	if err := r.rdb.Set(ctx, key, data, expire).Err(); err != nil {
		logger.Error("redis set admin info failed", zap.Uint64("admin_id", adminID), zap.Error(err))
		return common.RedisCmdErr.WithErr(err)
	}

	return nil
}

// DelAdminFromRedis 删除管理员缓存
func (r *AdminRedisRepo) DelAdminFromRedis(ctx context.Context, adminID uint64) error {

	key := buildAdminKey(adminID)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("redis delete admin info failed", zap.Uint64("admin_id", adminID), zap.Error(err))
		return common.RedisDeleteErr.WithErr(err)
	}

	logger.Info("delete admin cache success", zap.Uint64("admin_id", adminID))
	return nil
}

// ====================== 管理员在线令牌管理 ======================
// SetAdminOnlineToken 登录写入在线令牌
func (r *AdminRedisRepo) SetAdminOnlineToken(ctx context.Context, adminID uint64, token string, expire time.Duration) error {

	key := buildAdminOnlineTokenKey(adminID, token)
	if err := r.rdb.Set(ctx, key, adminOnlineTokenValue, expire).Err(); err != nil {
		logger.Error("redis set admin online token failed", zap.Uint64("admin_id", adminID), zap.Error(err))
		return common.RedisCmdErr.WithErr(err)
	}

	return nil
}

// DelAdminOnlineToken 单端下线（删除指定token）
func (r *AdminRedisRepo) DelAdminOnlineToken(ctx context.Context, adminID uint64, token string) error {

	key := buildAdminOnlineTokenKey(adminID, token)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("redis delete single admin online token failed", zap.Uint64("admin_id", adminID), zap.Error(err))
		return common.RedisDeleteErr.WithErr(err)
	}
	return nil
}

// ClearAdminAllOnlineToken 登出时清空管理员所有在线Token（全端下线）
func (r *AdminRedisRepo) ClearAdminAllOnlineToken(ctx context.Context, adminID uint64) error {

	pattern := fmt.Sprintf("%s%d:*", adminOnlineKeyPrefix, adminID)
	var cursor uint64
	var keys []string

	// SCAN 扫描匹配键
	for {
		scanKeys, nextCursor, err := r.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			logger.Error("redis scan admin online tokens failed", zap.Uint64("admin_id", adminID), zap.Error(err))
			return common.RedisScanErr.WithErr(err)
		}
		keys = append(keys, scanKeys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		logger.Warn("no admin online token to clear", zap.Uint64("admin_id", adminID))
		return nil
	}

	// 批量删除
	if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
		logger.Error("redis batch delete admin online tokens failed", zap.Uint64("admin_id", adminID), zap.Error(err))
		return common.RedisDeleteErr.WithErr(err)
	}

	logger.Info("clear all admin online tokens success", zap.Uint64("admin_id", adminID), zap.Int("count", len(keys)))
	return nil
}

// ====================== 管理员黑名单令牌管理 ======================
// AddAdminTokenToBlackList 管理员登出加入黑名单
func (r *AdminRedisRepo) AddAdminTokenToBlackList(ctx context.Context, token string, exp time.Duration) error {

	key := buildAdminBlackTokenKey(token)
	if err := r.rdb.Set(ctx, key, adminOnlineTokenValue, exp).Err(); err != nil {
		logger.Warn("redis add admin token to blacklist failed", zap.String("token_mask", maskToken(token)), zap.Error(err))
		return common.RedisCmdErr.WithErr(err)
	}
	return nil
}

// IsAdminTokenBlacklisted 判断管理员Token是否在黑名单
func (r *AdminRedisRepo) IsAdminTokenBlacklisted(ctx context.Context, token string) (bool, error) {

	key := buildAdminBlackTokenKey(token)
	count, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		logger.Warn("redis check admin black token failed", zap.String("token_mask", maskToken(token)), zap.Error(err))
		return false, common.RedisTokenBlackErr.WithErr(err)
	}

	return count > 0, nil
}

// GetAdminByNameFromRedis 根据用户名从缓存获取管理员信息
func (r *AdminRedisRepo) GetAdminByNameFromRedis(ctx context.Context, username string) (*model.AdminModel, error) {
	// 1. 拼接缓存Key
	key := fmt.Sprintf(adminInfoNameKey, username)

	// 2. 从Redis查询数据
	data, err := r.rdb.Get(ctx, key).Bytes()
	if err != nil {
		// 缓存不存在：返回标准缓存不存在错误
		if err == redis.Nil {
			return nil, common.CacheNotExist
		}
		// Redis 异常：打印日志并返回错误
		logger.Error("GetAdminByNameFromRedis 查询缓存失败",
			zap.String("username", username),
			zap.Error(err))
		return nil, err
	}

	// 3. 反序列化（兼容空值缓存，防止缓存穿透）
	var admin *model.AdminModel
	if err := json.Unmarshal(data, &admin); err != nil {
		logger.Error("GetAdminByNameFromRedis 反序列化失败",
			zap.String("username", username),
			zap.Error(err))
		return nil, common.ServerErr
	}

	return admin, nil
}

// SetAdminByNameToRedis 根据用户名缓存管理员信息
func (r *AdminRedisRepo) SetAdminByNameToRedis(ctx context.Context, username string, admin *model.AdminModel) error {
	var expire time.Duration
	// 1. 拼接缓存Key
	key := fmt.Sprintf(adminInfoNameKey, username)

	// 2. 序列化数据
	data, err := json.Marshal(admin)
	if err != nil {
		logger.Error("SetAdminByNameToRedis 序列化失败",
			zap.String("username", username),
			zap.Error(err))
		return common.ServerErr
	}

	// 3. 写入Redis，设置统一过期时间
	expire = r.config.AdminCacheExpire
	if err := r.rdb.Set(ctx, key, data, expire).Err(); err != nil {
		logger.Error("SetAdminByNameToRedis 写入缓存失败",
			zap.String("username", username),
			zap.Error(err))
		return err
	}

	return nil
}
