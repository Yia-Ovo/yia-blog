/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-10 04:57:16
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-16 09:51:58
 * @FilePath: \yia-blog\blog-server\infra\repo\user_redis_repo.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import (
	"blog-server/common"
	"blog-server/common/gin/response"
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
	userInfoKeyPrefix   = "blog:user:info:"
	userOnlineKeyPrefix = "blog:user:online:"
	blackTokenKeyPrefix = "blog:user:logout:black:"
	onlineTokenValue    = "1" // 在线令牌标记值
)

type UserRedisRepo struct {
	rdb    *redis.Client
	config *config.RedisConfig // 注入Redis配置
}

func NewUserRedisRepo(rdb *redis.Client, cfg *config.GlobalConfig) *UserRedisRepo {
	return &UserRedisRepo{rdb: rdb, config: &cfg.Redis}
}

// 私有工具方法：统一构建Redis Key
func buildUserKey(userID uint64) string {
	return fmt.Sprintf("%s%d", userInfoKeyPrefix, userID)
}
func buildOnlineTokenKey(userID uint64, token string) string {
	return fmt.Sprintf("%s%d:%s", userOnlineKeyPrefix, userID, token)
}
func buildBlackTokenKey(token string) string {
	return blackTokenKeyPrefix + token
}

// isEmptyUser 判断是否为【空用户对象】（防穿透标记）
func isEmptyUser(user *model.UserModel) bool {
	// 规则：用户ID为0 或 用户名空 → 判定为不存在的用户
	return user == nil || user.ID == 0
}

// ====================== 用户信息缓存 ======================
// GetUserFromRedis 从Redis获取用户信息
func (r *UserRedisRepo) GetUserFromRedis(ctx context.Context, userID uint64) (*model.UserModel, error) {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	key := buildUserKey(userID)
	data, err := r.rdb.Get(ctx, key).Result()

	// 1. 缓存不存在
	if errors.Is(err, redis.Nil) {
		logger.Warn("cache user info not found", zap.Uint64("user_id", userID))
		return nil, common.CacheNotExist
	}
	// 2. Redis命令执行失败
	if err != nil {
		logger.Error("redis get user info failed", zap.Uint64("user_id", userID), zap.Error(err))
		return nil, common.RedisCmdErr.WithErr(err)
	}

	// 3. 反序列化
	var user model.UserModel
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		logger.Error("redis unmarshal user info failed", zap.Uint64("user_id", userID), zap.Error(err))
		// 关键：反序列化失败，删除脏缓存，避免下次继续报错
		_ = r.rdb.Del(ctx, key)
		return nil, common.RedisUnmarshalErr.WithErr(err)
	}

	// ====================== 核心：判断空值缓存 ======================
	if isEmptyUser(&user) {
		logger.Warn("get empty user from cache", zap.Uint64("user_id", userID))
		// 返回空用户 + 无错误（上层直接判定用户不存在）
		return nil, nil
	}

	return &user, nil
}

func (r *UserRedisRepo) SetUserToRedis(ctx context.Context, userID uint64, user *model.UserModel) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	key := buildUserKey(userID)
	var data []byte
	var err error
	var expire time.Duration
	// ====================== 核心：空值处理 ======================
	if isEmptyUser(user) {
		// 1. 空用户：序列化空对象 + 短过期时间
		emptyUser := &model.UserModel{} // 空结构体
		data, err = json.Marshal(emptyUser)
		if err != nil {
			logger.Error("redis marshal empty user failed", zap.Uint64("user_id", userID), zap.Error(err))
			return common.RedisMarshalErr.WithErr(err)
		}
	} else {
		// 2. 正常用户：序列化 + 长过期时间
		data, err = json.Marshal(user)
		if err != nil {
			logger.Error("redis marshal user failed", zap.Uint64("user_id", userID), zap.Error(err))
			return common.RedisMarshalErr.WithErr(err)
		}
		expire = r.config.UserCacheExpire // 从配置读取
	}

	// 写入 Redis
	if err := r.rdb.Set(ctx, key, data, expire).Err(); err != nil {
		logger.Error("redis set user info failed", zap.Uint64("user_id", userID), zap.Error(err))
		return common.RedisCmdErr.WithErr(err)
	}

	return nil
}

// DelUserFromRedis 删除用户缓存（用户信息更新/删除时调用）
func (r *UserRedisRepo) DelUserFromRedis(ctx context.Context, userID uint64) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	key := buildUserKey(userID)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("redis delete user info failed", zap.Uint64("user_id", userID), zap.Error(err))
		return common.RedisDeleteErr.WithErr(err)
	}

	logger.Info("delete user cache success", zap.Uint64("user_id", userID))
	return nil
}

// ====================== 在线令牌管理 ======================
// SetOnlineToken 登录写入在线令牌
func (r *UserRedisRepo) SetOnlineToken(ctx context.Context, userID uint64, token string, expire time.Duration) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	key := buildOnlineTokenKey(userID, token)
	if err := r.rdb.Set(ctx, key, onlineTokenValue, expire).Err(); err != nil {
		logger.Error("redis set online token failed", zap.Uint64("user_id", userID), zap.Error(err))
		return common.RedisCmdErr.WithErr(err)
	}

	return nil
}

// DelUserOnlineToken 单端下线（删除指定token）
func (r *UserRedisRepo) DelUserOnlineToken(ctx context.Context, userID uint64, token string) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	key := buildOnlineTokenKey(userID, token)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("redis delete single online token failed", zap.Uint64("user_id", userID), zap.Error(err))
		return common.RedisDeleteErr.WithErr(err)
	}
	return nil
}

// ClearUserAllOnlineToken 登出时清空用户所有在线Token（全端下线）
func (r *UserRedisRepo) ClearUserAllOnlineToken(ctx context.Context, userID uint64) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()
	pattern := fmt.Sprintf("%s%d:*", userOnlineKeyPrefix, userID)
	var cursor uint64
	var keys []string

	// SCAN 扫描匹配键（替代Keys命令，生产环境标准用法）
	for {
		scanKeys, nextCursor, err := r.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			logger.Error("redis scan online tokens failed", zap.Uint64("user_id", userID), zap.Error(err))
			return common.RedisScanErr.WithErr(err)
		}
		keys = append(keys, scanKeys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		logger.Warn("no online token to clear", zap.Uint64("user_id", userID))
		return nil
	}

	// 批量删除
	if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
		logger.Error("redis batch delete online tokens failed", zap.Uint64("user_id", userID), zap.Error(err))
		return common.RedisDeleteErr.WithErr(err)
	}

	logger.Info("clear all user online tokens success", zap.Uint64("user_id", userID), zap.Int("count", len(keys)))
	return nil
}

// ====================== 黑名单令牌管理 ======================
// AddTokenToBlackList 【核心Logout】把当前token加入黑名单
func (r *UserRedisRepo) AddTokenToBlackList(ctx context.Context, token string, exp time.Duration) error {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	key := buildBlackTokenKey(token)
	if err := r.rdb.Set(ctx, key, onlineTokenValue, exp).Err(); err != nil {
		logger.Warn("redis add token to blacklist failed", zap.String("token_mask", maskToken(token)), zap.Error(err))
		return common.RedisCmdErr.WithErr(err)
	}
	return nil
}

// IsTokenBlacklisted 判断Token是否在黑名单
func (r *UserRedisRepo) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	ctx, cancel := response.WithDBTimeout(ctx)
	defer cancel()

	key := buildBlackTokenKey(token)
	count, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		logger.Warn("redis check black token failed", zap.String("token_mask", maskToken(token)), zap.Error(err))
		return false, common.RedisTokenBlackErr.WithErr(err)
	}

	return count > 0, nil
}

// ====================== 安全工具：Token脱敏 ======================
func maskToken(token string) string {
	if len(token) < 8 {
		return "***"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
