/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-08 20:28:25
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-15 11:43:30
 * @FilePath: \yia-blog\blog-server\common\db\redis\redis_init.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-08 20:28:25
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-09 02:02:20
 * @FilePath: \yia-blog\blog-server\infra\redis\redis_client.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package redis

import (
	"blog-server/common/utils/logger"
	"blog-server/config"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var RDB *redis.Client

// InitRedis 初始化Redis客户端
func InitRedis(conf *config.GlobalConfig) *redis.Client {

	redisConf := conf.Redis

	// 对齐GORM日志格式
	fmt.Printf("=== redis init ===\n")
	fmt.Printf("addr: %s, db: %d\n", redisConf.Addr, redisConf.DBIndex)

	// 初始化客户端
	RDB = redis.NewClient(&redis.Options{
		Addr:         redisConf.Addr,
		Password:     redisConf.Password,
		DB:           redisConf.DBIndex,
		DialTimeout:  redisConf.DialTimeout,
		ReadTimeout:  redisConf.ReadTimeout,
		WriteTimeout: redisConf.WriteTimeout,
		// 连接池
		PoolSize:        redisConf.PoolSize,        // 最大连接数（MaxOpen）
		MinIdleConns:    redisConf.MinIdleConns,    // 最小空闲
		MaxIdleConns:    redisConf.MaxIdleConns,    // 最大空闲（MaxIdle）
		PoolTimeout:     redisConf.PoolTimeout,     // 获取连接超时
		ConnMaxIdleTime: redisConf.ConnMaxIdleTime, // 空闲连接超时
	})
	// 校验连接
	_, err := RDB.Ping(context.Background()).Result()
	if err != nil {
		logger.Panic("redis ping failed", zap.Error(err))
	}
	logger.Info("Redis init success")
	return RDB
}

// SetWithExpire 设置带过期时间的Key
func SetWithExpire(key string, value interface{}, expire time.Duration) error {
	return RDB.Set(context.Background(), key, value, expire).Err()
}

// Exists 判断Key是否存在
func Exists(key string) bool {
	return RDB.Exists(context.Background(), key).Val() > 0
}
