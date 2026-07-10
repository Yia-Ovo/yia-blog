/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 07:04:03
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-09 07:52:05
 * @FilePath: \yia-blog\blog-server\common\db\gorm\gorm_init.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package gorm

import (
	"blog-server/common/utils/logger"
	"blog-server/config"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// zapWriterAdapter 将 zapcore.WriteSyncer 适配为 GORM 的 Writer 接口
type zapWriterAdapter struct {
	writer zapcore.WriteSyncer
}

// Printf 实现 GORM Writer 接口的 Printf 方法
func (a *zapWriterAdapter) Printf(format string, args ...interface{}) {
	a.writer.Write([]byte(fmt.Sprintf(format, args...)))
}

// InitGorm 初始化GORM客户端，连接MySQL
func InitGorm(conf *config.GlobalConfig) *gorm.DB {
	// 配置MySQL连接信息
	dsn := conf.Mysql.GetDsn()
	logger.Info("=== gorm init ===\n",
		zap.String("mysql_host", conf.Mysql.Host),
		zap.Int("mysql_port", conf.Mysql.Port),
		zap.String("mysql_database", conf.Mysql.Database),
		zap.String("mysql_user", conf.Mysql.User),
	)

	// 创建GORM日志适配器
	gormLogger := gormlogger.New(
		&zapWriterAdapter{writer: logger.GetWriter()}, // 使用适配器包装你的logger.Writer
		gormlogger.Config{
			SlowThreshold:             time.Second,     // 慢SQL阈值
			LogLevel:                  gormlogger.Info, // 日志级别
			IgnoreRecordNotFoundError: true,            // 忽略ErrRecordNotFound错误
			Colorful:                  true,            // 彩色日志
		},
	)

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger, // 使用适配后的日志器
	})
	if err != nil {
		logger.Panic("gorm init failed", zap.Error(err))
	}

	// 测试连接
	sqlDB, _ := db.DB()
	err = sqlDB.Ping()
	if err != nil {
		logger.Panic("sqlDB ping failed", zap.Error(err))
	}
	logger.Info("GORM init success")
	return db
}
