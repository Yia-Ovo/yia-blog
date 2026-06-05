/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-10 04:57:16
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-05 18:15:13
 * @FilePath: \yia-blog\blog-server\infra\repo\user_redis_repo.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import (
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

type userRedisRepo struct {
	rdb    *redis.Client
	config *config.RedisConfig
}

func NewUserRedisRepo(rdb *redis.Client, cfg *config.GlobalConfig) *userRedisRepo {
	return &userRedisRepo{rdb: rdb, config: &cfg.Redis}
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
	return user == nil || user.ID == 0
}

// ====================== 用户信息缓存 ======================
// GetUserFromRedis 从Redis获取用户信息
func (r *userRedisRepo) GetUserFromRedis(ctx context.Context, userID uint64) (*model.UserModel, error) {
	key := buildUserKey(userID)
	data, err := r.rdb.Get(ctx, key).Result()

	if errors.Is(err, redis.Nil) {
		logger.Warn("[REDIS] 缓存用户信息不存在", zap.Uint64("user_id", userID))
		return nil, err
	}
	if err != nil {
		logger.Error("[REDIS] 查询用户缓存异常", zap.Uint64("user_id", userID), zap.Error(err))
		return nil, err
	}

	var user model.UserModel
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		logger.Error("[REDIS] 用户缓存数据反序列化失败", zap.Uint64("user_id", userID), zap.Error(err))
		_ = r.DelUserFromRedis(ctx, userID)
		return nil, err
	}

	if isEmptyUser(&user) {
		logger.Warn("[REDIS] 命中空值防穿透缓存", zap.Uint64("user_id", userID))
		return nil, nil
	}

	return &user, nil
}

// SetUserToRedis 将用户信息写入Redis缓存
func (r *userRedisRepo) SetUserToRedis(ctx context.Context, userID uint64, user *model.UserModel) error {
	key := buildUserKey(userID)
	var data []byte
	var err error

	// 从 config 取过期时间
	expire := r.config.UserCacheExpire

	if isEmptyUser(user) {
		emptyUser := &model.UserModel{}
		data, err = json.Marshal(emptyUser)
		if err != nil {
			logger.Error("[REDIS] 空用户缓存序列化失败", zap.Uint64("user_id", userID), zap.Error(err))
			return err
		}
		// 从配置读取空值过期时间
		expire = r.config.UserEmptyCacheExpire
	} else {
		data, err = json.Marshal(user)
		if err != nil {
			logger.Error("[REDIS] 用户信息缓存序列化失败", zap.Uint64("user_id", userID), zap.Error(err))
			return err
		}
	}

	if err := r.rdb.Set(ctx, key, data, expire).Err(); err != nil {
		logger.Error("[REDIS] 写入用户缓存失败", zap.Uint64("user_id", userID), zap.Error(err))
		return err
	}

	return nil
}

// DelUserFromRedis 删除用户缓存
func (r *userRedisRepo) DelUserFromRedis(ctx context.Context, userID uint64) error {
	key := buildUserKey(userID)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("[REDIS] 删除用户缓存失败", zap.Uint64("user_id", userID), zap.Error(err))
		return err
	}
	return nil
}

// ====================== 在线令牌管理 ======================
func (r *userRedisRepo) SetOnlineToken(ctx context.Context, userID uint64, token string, expire time.Duration) error {
	key := buildOnlineTokenKey(userID, token)
	if err := r.rdb.Set(ctx, key, onlineTokenValue, expire).Err(); err != nil {
		logger.Error("[REDIS] 写入用户在线令牌失败", zap.Uint64("user_id", userID), zap.Error(err))
		return err
	}
	return nil
}

func (r *userRedisRepo) DelUserOnlineToken(ctx context.Context, userID uint64, token string) error {
	key := buildOnlineTokenKey(userID, token)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("[REDIS] 删除单个在线令牌失败", zap.Uint64("user_id", userID), zap.Error(err))
		return err
	}
	return nil
}

func (r *userRedisRepo) ClearUserAllOnlineToken(ctx context.Context, userID uint64) error {
	pattern := fmt.Sprintf("%s%d:*", userOnlineKeyPrefix, userID)
	var cursor uint64
	var keys []string

	for {
		scanKeys, nextCursor, err := r.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			logger.Error("[REDIS] 扫描用户在线令牌异常", zap.Uint64("user_id", userID), zap.Error(err))
			return err
		}
		keys = append(keys, scanKeys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		logger.Warn("[REDIS] 未查询到需要清理的在线令牌", zap.Uint64("user_id", userID))
		return nil
	}

	if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
		logger.Error("[REDIS] 批量清理在线令牌失败", zap.Uint64("user_id", userID), zap.Error(err))
		return err
	}

	logger.Info("[REDIS] 清理用户在线令牌完成", zap.Uint64("user_id", userID), zap.Int("count", len(keys)))
	return nil
}

// ====================== 黑名单令牌管理 ======================
func (r *userRedisRepo) AddTokenToBlackList(ctx context.Context, token string, exp time.Duration) error {
	key := buildBlackTokenKey(token)
	if err := r.rdb.Set(ctx, key, onlineTokenValue, exp).Err(); err != nil {
		logger.Warn("[REDIS] 令牌加入黑名单写入异常", zap.String("token_mask", maskToken(token)), zap.Error(err))
		return err
	}
	return nil
}

func (r *userRedisRepo) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := buildBlackTokenKey(token)
	count, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		logger.Warn("[REDIS] 校验黑名单令牌异常", zap.String("token_mask", maskToken(token)), zap.Error(err))
		return false, err
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
