package logger

import (
	"blog-server/config"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.Logger

// 创建一个原子级别的日志级别配置，并将其设置为调试级别(DebugLevel)
// zap.AtomicLevelAt 用于创建一个指定级别的原子级别配置
// atom 变量将用于控制日志的输出级别
var atom = zap.NewAtomicLevelAt(zap.DebugLevel)

type stdoutWriteSyncer struct{}

func (s stdoutWriteSyncer) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p)
}

func (s stdoutWriteSyncer) Sync() error {
	return nil // 在 Windows 上忽略 Sync 错误
}

func InitLogger(conf *config.Config) {

	// 配置日志输出和轮转
	writeSyncer := getLogWriter(conf)

	// 配置日志编码器
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:   "[msg]",
		LevelKey:     "[level]",
		TimeKey:      "[time]",
		CallerKey:    "[caller]",
		EncodeTime:   zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	// 创建核心日志组件
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writeSyncer,
		atom, // 使用原子级别配置，支持动态修改日志级别
	)

	// 创建 logger
	// 创建一个新的logger实例
	// core: zap的核心配置，定义了日志的编码方式、级别等信息
	// AddCaller(): 添加调用者信息（文件名和行号）到日志中
	// AddCallerSkip(1): 跳过1层调用栈，避免显示zap库内部的调用信息
	// AddStacktrace(zap.ErrorLevel): 在错误级别及以上的日志中自动添加堆栈跟踪信息
	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zap.ErrorLevel))
}

func SetLevel(level string) {
	tLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		fmt.Printf("invalid level, input: %s", level)
		return
	}
	atom.SetLevel(tLevel)
}

// getLogWriter 配置日志输出位置和轮转
func getLogWriter(conf *config.Config) zapcore.WriteSyncer {
	// 确保日志目录存在
	if err := os.MkdirAll("./logs", 0755); err != nil {
		fmt.Printf("failed to create log directory: %v\n", err)
		return stdoutWriteSyncer{}
	}

	// 读取配置
	// 使用 lumberjack 实现日志轮转
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filepath.Join(conf.Logger.Path, conf.Logger.FileName), // 日志文件路径
		MaxSize:    conf.Logger.MaxSize,                                   // 单个文件最大尺寸，单位 MB
		MaxBackups: conf.Logger.MaxBackups,                                // 最多保留 5 个备份
		MaxAge:     conf.Logger.MaxAge,                                    // 最大保留天数
		Compress:   conf.Logger.Compress,                                  // 是否压缩
	}

	// 同时输出到文件和控制台
	return zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(lumberJackLogger),
		stdoutWriteSyncer{},
	)
}

// Sync 同步日志缓冲区
func Sync() {
	if logger != nil {
		if err := logger.Sync(); err != nil {
			// 忽略 "sync /dev/stdout: The handle is invalid" 错误，这是 Windows 系统下的已知问题
			if err.Error() != "sync /dev/stdout: The handle is invalid" {
				_, _ = os.Stderr.WriteString("Failed to sync logger: " + err.Error() + "\n")
			}
		}
	}
}

// GetLogger 获取日志实例
func GetLogger() *zap.Logger {
	return logger
}

func Debug(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	logger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}
