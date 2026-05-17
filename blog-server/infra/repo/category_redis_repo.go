/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-06 11:26:21
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-06 11:26:46
 * @FilePath: \yia-blog\blog-server\infra\repo\category_redis_repo.go
 * @Description:
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

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	categoryInfoKeyPrefix = "blog:category:info:" // 单个分类缓存前缀
	categoryListKeyPrefix = "blog:category:list"  // 分类列表缓存前缀
)

type CategoryRedisRepo struct {
	rdb    *redis.Client
	config *config.RedisConfig // 注入Redis配置
}

func NewCategoryRedisRepo(rdb *redis.Client, cfg *config.GlobalConfig) *CategoryRedisRepo {
	return &CategoryRedisRepo{rdb: rdb, config: &cfg.Redis}
}

// 私有工具方法：统一构建Redis Key
func buildCategoryKey(categoryID uint64) string {
	return fmt.Sprintf("%s%d", categoryInfoKeyPrefix, categoryID)
}

// isEmptyCategory 判断是否为【空分类对象】（防穿透标记）
func isEmptyCategory(category *model.CategoryModel) bool {
	// 规则：分类ID为0 → 判定为不存在的分类
	return category == nil || category.ID == 0
}

// ====================== 单个分类信息缓存 ======================
// GetCategoryFromRedis 从Redis获取单个分类信息
func (r *CategoryRedisRepo) GetCategoryFromRedis(ctx context.Context, categoryID uint64) (*model.CategoryModel, error) {

	key := buildCategoryKey(categoryID)
	data, err := r.rdb.Get(ctx, key).Result()

	// 1. 缓存不存在
	if errors.Is(err, redis.Nil) {
		logger.Warn("cache category info not found", zap.Uint64("category_id", categoryID))
		return nil, common.CacheNotExist
	}
	// 2. Redis命令执行失败
	if err != nil {
		logger.Error("redis get category info failed", zap.Uint64("category_id", categoryID), zap.Error(err))
		return nil, common.RedisCmdErr.WithErr(err)
	}

	// 3. 反序列化
	var category model.CategoryModel
	if err := json.Unmarshal([]byte(data), &category); err != nil {
		logger.Error("redis unmarshal category info failed", zap.Uint64("category_id", categoryID), zap.Error(err))
		// 反序列化失败，删除脏缓存
		_ = r.rdb.Del(ctx, key)
		return nil, common.RedisUnmarshalErr.WithErr(err)
	}

	// 核心：判断空值缓存
	if isEmptyCategory(&category) {
		logger.Warn("get empty category from cache", zap.Uint64("category_id", categoryID))
		return nil, nil
	}

	return &category, nil
}

// SetCategoryToRedis 将单个分类写入Redis
func (r *CategoryRedisRepo) SetCategoryToRedis(ctx context.Context, categoryID uint64, category *model.CategoryModel) error {

	key := buildCategoryKey(categoryID)
	var data []byte
	var err error
	expire := r.config.UserCacheExpire // 复用配置的缓存过期时间

	// 核心：空值处理（缓存穿透防护）
	if isEmptyCategory(category) {
		// 空分类：序列化空对象 + 短过期
		emptyCategory := &model.CategoryModel{}
		data, err = json.Marshal(emptyCategory)
		if err != nil {
			logger.Error("redis marshal empty category failed", zap.Uint64("category_id", categoryID), zap.Error(err))
			return common.RedisMarshalErr.WithErr(err)
		}
	} else {
		// 正常分类：序列化数据
		data, err = json.Marshal(category)
		if err != nil {
			logger.Error("redis marshal category failed", zap.Uint64("category_id", categoryID), zap.Error(err))
			return common.RedisMarshalErr.WithErr(err)
		}
	}

	// 写入 Redis
	if err := r.rdb.Set(ctx, key, data, expire).Err(); err != nil {
		logger.Error("redis set category info failed", zap.Uint64("category_id", categoryID), zap.Error(err))
		return common.RedisCmdErr.WithErr(err)
	}

	return nil
}

// DelCategoryFromRedis 删除单个分类缓存
func (r *CategoryRedisRepo) DelCategoryFromRedis(ctx context.Context, categoryID uint64) error {

	key := buildCategoryKey(categoryID)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("redis delete category info failed", zap.Uint64("category_id", categoryID), zap.Error(err))
		return common.RedisDeleteErr.WithErr(err)
	}

	logger.Info("delete category cache success", zap.Uint64("category_id", categoryID))
	return nil
}

// ====================== 分类列表缓存 ======================
// GetCategoryListFromRedis 从Redis获取分类列表
func (r *CategoryRedisRepo) GetCategoryListFromRedis(ctx context.Context) ([]model.CategoryModel, error) {

	key := categoryListKeyPrefix
	data, err := r.rdb.Get(ctx, key).Result()

	// 缓存不存在
	if errors.Is(err, redis.Nil) {
		logger.Warn("cache category list not found")
		return nil, common.CacheNotExist
	}
	// Redis执行失败
	if err != nil {
		logger.Error("redis get category list failed", zap.Error(err))
		return nil, common.RedisCmdErr.WithErr(err)
	}

	// 反序列化为列表
	var categoryList []model.CategoryModel
	if err := json.Unmarshal([]byte(data), &categoryList); err != nil {
		logger.Error("redis unmarshal category list failed", zap.Error(err))
		_ = r.rdb.Del(ctx, key)
		return nil, common.RedisUnmarshalErr.WithErr(err)
	}

	return categoryList, nil
}

// SetCategoryListToRedis 将分类列表写入Redis
func (r *CategoryRedisRepo) SetCategoryListToRedis(ctx context.Context, categoryList []model.CategoryModel) error {

	key := categoryListKeyPrefix
	expire := r.config.UserCacheExpire

	// 序列化列表
	data, err := json.Marshal(categoryList)
	if err != nil {
		logger.Error("redis marshal category list failed", zap.Error(err))
		return common.RedisMarshalErr.WithErr(err)
	}

	// 写入缓存
	if err := r.rdb.Set(ctx, key, data, expire).Err(); err != nil {
		logger.Error("redis set category list failed", zap.Error(err))
		return common.RedisCmdErr.WithErr(err)
	}

	return nil
}

// DelCategoryListFromRedis 删除分类列表缓存（新增/编辑/删除分类时调用）
func (r *CategoryRedisRepo) DelCategoryListFromRedis(ctx context.Context) error {

	key := categoryListKeyPrefix
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("redis delete category list failed", zap.Error(err))
		return common.RedisDeleteErr.WithErr(err)
	}

	logger.Info("delete category list cache success")
	return nil
}
