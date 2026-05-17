/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-07 02:05:38
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-17 09:51:01
 * @FilePath: \yia-blog\blog-server\infra\repo\article_redis_repo.go
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
	articleInfoKeyPrefix = "blog:article:info:" // 单个文章缓存前缀
	articleListKeyPrefix = "blog:article:list"  // 文章列表缓存前缀
)

type ArticleRedisRepo struct {
	rdb    *redis.Client
	config *config.RedisConfig // 注入Redis配置
}

func NewArticleRedisRepo(rdb *redis.Client, cfg *config.GlobalConfig) *ArticleRedisRepo {
	return &ArticleRedisRepo{rdb: rdb, config: &cfg.Redis}
}

// 私有工具方法：统一构建Redis Key
func buildArticleKey(articleID uint64) string {
	return fmt.Sprintf("%s%d", articleInfoKeyPrefix, articleID)
}

// isEmptyArticle 判断是否为【空文章对象】（防穿透标记）
func isEmptyArticle(article *model.ArticleModel) bool {
	// 规则：文章ID为0 → 判定为不存在的文章
	return article == nil || article.ID == 0
}

// ====================== 单个文章信息缓存 ======================
// GetArticleFromRedis 从Redis获取单个文章信息
func (r *ArticleRedisRepo) GetArticleFromRedis(ctx context.Context, articleID uint64) (*model.ArticleModel, error) {

	key := buildArticleKey(articleID)
	data, err := r.rdb.Get(ctx, key).Result()

	// 1. 缓存不存在
	if errors.Is(err, redis.Nil) {
		logger.Warn("cache article info not found", zap.Uint64("article_id", articleID))
		return nil, common.CacheNotExist
	}
	// 2. Redis命令执行失败
	if err != nil {
		logger.Error("redis get article info failed", zap.Uint64("article_id", articleID), zap.Error(err))
		return nil, common.RedisCmdErr.WithErr(err)
	}

	// 3. 反序列化
	var article model.ArticleModel
	if err := json.Unmarshal([]byte(data), &article); err != nil {
		logger.Error("redis unmarshal article info failed", zap.Uint64("article_id", articleID), zap.Error(err))
		// 反序列化失败，删除脏缓存
		_ = r.rdb.Del(ctx, key)
		return nil, common.RedisUnmarshalErr.WithErr(err)
	}

	// 核心：判断空值缓存
	if isEmptyArticle(&article) {
		logger.Warn("get empty article from cache", zap.Uint64("article_id", articleID))
		return nil, nil
	}

	return &article, nil
}

// SetArticleToRedis 将单个文章写入Redis
func (r *ArticleRedisRepo) SetArticleToRedis(ctx context.Context, articleID uint64, article *model.ArticleModel) error {

	key := buildArticleKey(articleID)
	var data []byte
	var err error
	expire := r.config.UserCacheExpire // 复用配置的缓存过期时间

	// 核心：空值处理（缓存穿透防护）
	if isEmptyArticle(article) {
		// 空文章：序列化空对象 + 短过期
		emptyArticle := &model.ArticleModel{}
		data, err = json.Marshal(emptyArticle)
		if err != nil {
			logger.Error("redis marshal empty article failed", zap.Uint64("article_id", articleID), zap.Error(err))
			return common.RedisMarshalErr.WithErr(err)
		}
	} else {
		// 正常文章：序列化数据
		data, err = json.Marshal(article)
		if err != nil {
			logger.Error("redis marshal article failed", zap.Uint64("article_id", articleID), zap.Error(err))
			return common.RedisMarshalErr.WithErr(err)
		}
	}

	// 写入 Redis
	if err := r.rdb.Set(ctx, key, data, expire).Err(); err != nil {
		logger.Error("redis set article info failed", zap.Uint64("article_id", articleID), zap.Error(err))
		return common.RedisCmdErr.WithErr(err)
	}

	return nil
}

// DelArticleFromRedis 删除单个文章缓存
func (r *ArticleRedisRepo) DelArticleFromRedis(ctx context.Context, articleID uint64) error {

	key := buildArticleKey(articleID)
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("redis delete article info failed", zap.Uint64("article_id", articleID), zap.Error(err))
		return common.RedisDeleteErr.WithErr(err)
	}

	logger.Info("delete article cache success", zap.Uint64("article_id", articleID))
	return nil
}

// ====================== 文章列表缓存 ======================
// GetArticleListFromRedis 从Redis获取文章列表
func (r *ArticleRedisRepo) GetArticleListFromRedis(ctx context.Context) ([]model.ArticleModel, error) {

	key := articleListKeyPrefix
	data, err := r.rdb.Get(ctx, key).Result()

	// 缓存不存在
	if errors.Is(err, redis.Nil) {
		logger.Warn("cache article list not found")
		return nil, common.CacheNotExist
	}
	// Redis执行失败
	if err != nil {
		logger.Error("redis get article list failed", zap.Error(err))
		return nil, common.RedisCmdErr.WithErr(err)
	}

	// 反序列化为列表
	var articleList []model.ArticleModel
	if err := json.Unmarshal([]byte(data), &articleList); err != nil {
		logger.Error("redis unmarshal article list failed", zap.Error(err))
		_ = r.rdb.Del(ctx, key)
		return nil, common.RedisUnmarshalErr.WithErr(err)
	}

	return articleList, nil
}

// SetArticleListToRedis 将文章列表写入Redis
func (r *ArticleRedisRepo) SetArticleListToRedis(ctx context.Context, articleList []model.ArticleModel) error {

	key := articleListKeyPrefix
	expire := r.config.UserCacheExpire

	// 序列化列表
	data, err := json.Marshal(articleList)
	if err != nil {
		logger.Error("redis marshal article list failed", zap.Error(err))
		return common.RedisMarshalErr.WithErr(err)
	}

	// 写入缓存
	if err := r.rdb.Set(ctx, key, data, expire).Err(); err != nil {
		logger.Error("redis set article list failed", zap.Error(err))
		return common.RedisCmdErr.WithErr(err)
	}

	return nil
}

// DelArticleListFromRedis 删除文章列表缓存（新增/编辑/删除文章时调用）
func (r *ArticleRedisRepo) DelArticleListFromRedis(ctx context.Context) error {

	key := articleListKeyPrefix
	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("redis delete article list failed", zap.Error(err))
		return common.RedisDeleteErr.WithErr(err)
	}

	logger.Info("delete article list cache success")
	return nil
}
