/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-16 08:26:43
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-01 10:19:11
 * @FilePath: \yia-blog\blog-server\infra\repo\admin_redis_repo.go
 * @Description: 管理员 Redis 仓储实现
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
	adminInfoKeyPrefix       = "blog:admin:info:"
	adminOnlineKeyPrefix     = "blog:admin:online:"
	adminBlackTokenKeyPrefix = "blog:admin:logout:black:"
	adminOnlineTokenValue    = "1" // 管理员在线令牌标记值
)

type adminRedisRepo struct {
	rdb    *redis.Client
	config *config.RedisConfig
}

func NewAdminRedisRepo(rdb *redis.Client, cfg *config.GlobalConfig) *adminRedisRepo {
	return &adminRedisRepo{rdb: rdb, config: &cfg.Redis}
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
	return admin == nil || admin.ID == 0
}

// ====================== 管理员信息缓存 ======================
// GetAdminFromRedis 从Redis获取管理员信息
func (r *adminRedisRepo) GetAdminFromRedis(ctx context.Context, adminID uint64) (*model.AdminModel, error) {
	key := buildAdminKey(adminID)
	data, err := r.rdb.Get(ctx, key).Result()

	if errors.Is(err, redis.Nil) {
		logger.Warn("[REDIS] 缓存管理员信息不存在", zap.Uint64("admin_id", adminID))
		return nil, err
	}
	if err != nil {
		logger.Error("[REDIS] 查询管理员缓存异常", zap.Uint64("admin_id", adminID), zap.Error(err))
		return nil, err
	}

	var admin model.AdminModel
	if err := json.Unmarshal([]byte(data), &admin); err != nil {
		logger.Error("[REDIS] 管理员缓存数据反序列化失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		_ = r.DelAdminFromRedis(ctx, adminID)
		return nil, err
	}

	if isEmptyAdmin(&admin) {
		logger.Warn("[REDIS] 命中空值防穿透缓存", zap.Uint64("admin_id", adminID))
		return nil, nil
	}

	return &admin, nil
}

// SetAdminToRedis 将管理员信息写入Redis缓存
func (r *adminRedisRepo) SetAdminToRedis(ctx context.Context, adminID uint64, admin *model.AdminModel) error {
	key := buildAdminKey(adminID)
	var data []byte
	var err error

	expire := r.config.AdminCacheExpire

	if isEmptyAdmin(admin) {
		emptyAdmin := &model.AdminModel{}
		data, err = json.Marshal(emptyAdmin)
		if err != nil {
			logger.Error("[REDIS] 空管理员缓存序列化失败", zap.Uint64("admin_id", adminID), zap.Error(err))
			return err
		}
		expire = r.config.UserEmptyCacheExpire
	} else {
		data, err = json.Marshal(admin)
		if err != nil {
			logger.Error("[REDIS] 管理员信息缓存序列化失败", zap.Uint64("admin_id", adminID), zap.Error(err))
			return err
		}
	}

	if err := r.rdb.Set(ctx, key, data, expire).Err(); err != nil {
		logger.Error("[REDIS] 写入管理员缓存失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}

	return nil
}

// DelAdminFromRedis 删除管理员缓存
func (r *adminRedisRepo) DelAdminFromRedis(ctx context.Context, adminID uint64) error {
	key := buildAdminKey(adminID)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("[REDIS] 删除管理员缓存失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}
	logger.Info("[REDIS] 删除管理员缓存成功", zap.Uint64("admin_id", adminID))
	return nil
}

// ====================== 在线令牌管理 ======================
func (r *adminRedisRepo) SetAdminOnlineToken(ctx context.Context, adminID uint64, token string, expire time.Duration) error {
	key := buildAdminOnlineTokenKey(adminID, token)
	if err := r.rdb.Set(ctx, key, adminOnlineTokenValue, expire).Err(); err != nil {
		logger.Error("[REDIS] 写入管理员在线令牌失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}
	return nil
}

func (r *adminRedisRepo) DelAdminOnlineToken(ctx context.Context, adminID uint64, token string) error {
	key := buildAdminOnlineTokenKey(adminID, token)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("[REDIS] 删除单个在线令牌失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}
	return nil
}

func (r *adminRedisRepo) ClearAdminAllOnlineToken(ctx context.Context, adminID uint64) error {
	pattern := fmt.Sprintf("%s%d:*", adminOnlineKeyPrefix, adminID)
	var cursor uint64
	var keys []string

	for {
		scanKeys, nextCursor, err := r.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			logger.Error("[REDIS] 扫描管理员在线令牌异常", zap.Uint64("admin_id", adminID), zap.Error(err))
			return err
		}
		keys = append(keys, scanKeys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		logger.Warn("[REDIS] 未查询到需要清理的在线令牌", zap.Uint64("admin_id", adminID))
		return nil
	}

	if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
		logger.Error("[REDIS] 批量清理在线令牌失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return err
	}

	logger.Info("[REDIS] 清理管理员在线令牌完成", zap.Uint64("admin_id", adminID), zap.Int("count", len(keys)))
	return nil
}

func (r *adminRedisRepo) IsAdminOnlineToken(ctx context.Context, adminID uint64, token string) (bool, error) {
	key := buildAdminOnlineTokenKey(adminID, token)
	count, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		logger.Warn("[REDIS] 校验在线令牌异常", zap.Uint64("admin_id", adminID), zap.String("token_mask", maskToken(token)), zap.Error(err))
		return false, err
	}
	return count > 0, nil
}

// ====================== 黑名单令牌管理 ======================
func (r *adminRedisRepo) AddAdminTokenToBlackList(ctx context.Context, token string, exp time.Duration) error {
	key := buildAdminBlackTokenKey(token)
	if err := r.rdb.Set(ctx, key, adminOnlineTokenValue, exp).Err(); err != nil {
		logger.Warn("[REDIS] 令牌加入黑名单写入异常", zap.String("token_mask", maskToken(token)), zap.Error(err))
		return err
	}
	return nil
}

func (r *adminRedisRepo) IsAdminTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := buildAdminBlackTokenKey(token)
	count, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		logger.Warn("[REDIS] 校验黑名单令牌异常", zap.String("token_mask", maskToken(token)), zap.Error(err))
		return false, err
	}
	return count > 0, nil
}
