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
	"blog-server/common/consts"
	"blog-server/common/utils/logger"
	"blog-server/config"
	"database/sql"
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
		gormLogConfig(conf),
	)

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger, // 使用适配后的日志器
	})
	if err != nil {
		logger.Panic("gorm init failed", zap.Error(err))
	}

	// 测试连接
	sqlDB, err := db.DB()
	if err != nil {
		logger.Panic("[MYSQL] 获取连接池失败", zap.Error(err))
	}
	configureSQLPool(sqlDB, conf.Mysql)
	err = sqlDB.Ping()
	if err != nil {
		logger.Panic("sqlDB ping failed", zap.Error(err))
	}
	logger.Info("GORM init success")
	return db
}

// gormLogConfig 按 show_sql 开启普通 SQL 日志；始终省略参数，避免日志写入密码或用户正文。
func gormLogConfig(conf *config.GlobalConfig) gormlogger.Config {
	level := gormlogger.Warn
	if conf.Mysql.ShowSql {
		level = gormlogger.Info
	}
	return gormlogger.Config{
		SlowThreshold:             time.Second,
		LogLevel:                  level,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
		Colorful:                  conf.Server.Env == "dev" && !conf.Gin.DisableConsoleColor,
	}
}

// configureSQLPool 使用配置限制连接总数、空闲数和寿命；合法的 max_idle=0 表示不保留空闲连接。
func configureSQLPool(db *sql.DB, conf *config.MysqlConfig) {
	maxOpen := conf.MaxOpen
	if maxOpen <= 0 {
		maxOpen = consts.DefaultMySQLMaxOpen
	}
	maxIdle := conf.MaxIdle
	if maxIdle < 0 {
		maxIdle = 0
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	lifetime := conf.ConnMaxLifetime
	if lifetime <= 0 {
		lifetime = consts.DefaultMySQLConnMaxLifetime
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(lifetime)
}
