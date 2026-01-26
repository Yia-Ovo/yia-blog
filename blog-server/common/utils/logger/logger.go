/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-27 12:00:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-01-27 01:37:26
 * @FilePath: \yia-blog\blog-server\common\utils\logger\logger.go
 * @Description: 基于zap+lumberjack的日志组件，适配全局配置
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package logger

import (
	"blog-server/config"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.Logger
	// 原子级别配置：支持动态修改日志级别，初始值从配置读取（而非固定Debug）
	atom = zap.NewAtomicLevel()
	// 日志输出同步器（文件+控制台）
	logWriteSyncer zapcore.WriteSyncer
)

// stdoutWriteSyncer 标准输出同步器（跨平台兼容）
type stdoutWriteSyncer struct{}

func (s stdoutWriteSyncer) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p)
}

func (s stdoutWriteSyncer) Sync() error {
	// 跨平台兼容：Windows系统忽略Sync错误，Linux/Mac正常同步
	if runtime.GOOS == "windows" {
		return nil
	}
	return os.Stdout.Sync()
}

// InitLogger 初始化日志组件（修复参数类型，适配GlobalConfig）
// 参数：conf - 全局配置实例（来自config.GetGlobalConfig()）
func InitLogger(conf *config.GlobalConfig) {
	if conf == nil {
		panic("初始化日志失败：全局配置为nil，请先执行config.InitConfig()")
	}

	// 1. 从配置设置初始日志级别
	setInitialLogLevel(conf.Server.LogLevel)

	// 2. 获取日志输出同步器（文件+控制台）
	writeSyncer := getLogWriter(conf)
	logWriteSyncer = writeSyncer // 赋值给全局变量

	// 3. 配置日志编码器（优化字段命名，更符合行业规范）
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "msg",                                                  // 日志消息字段名
		LevelKey:       "level",                                                // 日志级别字段名
		TimeKey:        "time",                                                 // 时间字段名
		CallerKey:      "caller",                                               // 调用者字段名
		StacktraceKey:  "stack",                                                // 堆栈跟踪字段名
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"), // 毫秒级时间格式
		EncodeLevel:    zapcore.LowercaseLevelEncoder,                          // 小写级别（debug/info/warn/error）
		EncodeCaller:   zapcore.ShortCallerEncoder,                             // 短路径调用者（如logger/logger.go:50）
		EncodeDuration: zapcore.MillisDurationEncoder,                          // 耗时毫秒级
	}

	// 4. 区分环境设置编码器：开发环境用ConsoleEncoder（易读），生产环境用JSONEncoder（易解析）
	var encoder zapcore.Encoder
	if conf.Server.Env == "dev" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig) // 开发环境：控制台友好格式
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig) // 生产环境：JSON格式，便于日志收集
	}

	// 5. 创建zap核心组件
	core := zapcore.NewCore(
		encoder,
		writeSyncer,
		atom, // 动态级别控制
	)

	// 6. 构建最终logger实例（优化调用栈跳过，补充开发环境特性）
	options := []zap.Option{
		zap.AddCaller(),                   // 添加调用者信息
		zap.AddCallerSkip(1),              // 跳过当前层，显示业务代码的调用位置
		zap.AddStacktrace(zap.ErrorLevel), // 仅Error/Fatal/Panic级别输出堆栈
	}
	// 开发环境额外开启开发模式（更详细的错误提示）
	if conf.Server.Env == "dev" {
		options = append(options, zap.Development())
	}

	logger = zap.New(core, options...)

	// 替换zap全局logger（可选，便于第三方库使用）
	zap.ReplaceGlobals(logger)

	Info("日志组件初始化成功",
		zap.String("env", conf.Server.Env),
		zap.String("log_level", atom.Level().String()),
		zap.String("log_path", filepath.Join(conf.Logger.Path, conf.Logger.FileName)),
	)
}

// setInitialLogLevel 从配置设置初始日志级别
func setInitialLogLevel(levelStr string) {
	if levelStr == "" {
		levelStr = "info" // 默认级别
	}
	level, err := zapcore.ParseLevel(levelStr)
	if err != nil {
		fmt.Printf("日志级别配置无效：%s，使用默认级别info\n", levelStr)
		level = zapcore.InfoLevel
	}
	atom.SetLevel(level)
}

// getLogWriter 配置日志输出（文件轮转+控制台），修复硬编码问题
func getLogWriter(conf *config.GlobalConfig) zapcore.WriteSyncer {
	loggerConf := conf.Logger

	// 1. 确保日志目录存在（使用配置的路径，而非硬编码./logs）
	if err := os.MkdirAll(loggerConf.Path, 0755); err != nil {
		fmt.Printf("创建日志目录失败：%v，降级输出到控制台\n", err)
		return stdoutWriteSyncer{}
	}

	// 2. 配置lumberjack日志轮转（严格读取配置）
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filepath.Join(loggerConf.Path, loggerConf.FileName), // 日志文件完整路径
		MaxSize:    loggerConf.MaxSize,                                  // 单个文件最大MB
		MaxBackups: loggerConf.MaxBackups,                               // 最大备份文件数
		MaxAge:     loggerConf.MaxAge,                                   // 最大保留天数
		Compress:   loggerConf.Compress,                                 // 是否压缩备份文件
		LocalTime:  true,                                                // 使用本地时间命名备份文件（而非UTC）
	}

	// 3. 开发环境：输出到文件+控制台；生产环境：仅输出到文件（减少控制台开销）
	if conf.Server.Env == "dev" {
		return zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(lumberJackLogger),
			stdoutWriteSyncer{},
		)
	}
	return zapcore.AddSync(lumberJackLogger)
}

// SetLevel 动态修改日志级别（兼容配置的log_level）
func SetLevel(level string) {
	tLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		Warn("日志级别设置失败", zap.String("input_level", level), zap.Error(err))
		return
	}
	oldLevel := atom.Level()
	atom.SetLevel(tLevel)
	Info("日志级别已修改", zap.String("old_level", oldLevel.String()), zap.String("new_level", tLevel.String()))
}

// 获取日志Writer（适配gin.RecoveryWithWriter）
func GetWriter() zapcore.WriteSyncer {
	if logWriteSyncer != nil {
		return logWriteSyncer
	}
	// 兜底：返回标准输出同步器
	return stdoutWriteSyncer{}
}

// Sync 同步日志缓冲区（优化错误处理）
func Sync() {
	if logger != nil {
		err := logger.Sync()
		if err != nil {
			// 过滤Windows系统的已知无害错误
			if runtime.GOOS != "windows" || err.Error() != "sync /dev/stdout: The handle is invalid" {
				fmt.Fprintf(os.Stderr, "日志缓冲区同步失败：%v\n", err)
			}
		}
	}
}

// GetLogger 获取原始zap.Logger实例（便于自定义扩展）
func GetLogger() *zap.Logger {
	if logger == nil {
		// 未初始化时返回一个临时的控制台logger，避免nil panic
		return zap.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			stdoutWriteSyncer{},
			zapcore.DebugLevel,
		), zap.AddCaller())
	}
	return logger
}

// ========== 封装常用日志方法（保持原有接口兼容） ==========
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	GetLogger().Panic(msg, fields...)
}
