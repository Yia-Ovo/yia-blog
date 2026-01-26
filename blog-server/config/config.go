/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-21 11:32:32
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-01-27 00:52:48
 * @FilePath: \yia-blog\blog-server\config\config.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package config

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

const (
	ServerName     = "blog"
	ServerFullName = "yia.blog"
)

type GlobalConfig struct {
	Server ServerConfig `yaml:"server"` // 服务器基础配置
	Gin    GinConfig    `yaml:"gin"`    // Gin 专属配置
	Mysql  MysqlConfig  `yaml:"mysql"`
	Redis  RedisConfig  `yaml:"redis"`
	Logger LoggerConfig `yaml:"logger"`
	// 可追加数据库、Redis等其他配置
}

var (
	etcdKey         = fmt.Sprintf("/configs/%s/system", ServerFullName)
	etcdAddr        string
	localConfigPath string
	tempConfig      *GlobalConfig
	CurrentEnv      string // 程序当前运行的环境（dev/prod/test）

	// 新增：用于存放 etcd 认证相关的变量
	// etcdUsername   string
	// etcdPassword   string
	// etcdEndpointCA string // CA 证书路径
	// etcdClientCert string // 客户端证书路径
	// etcdClientKey  string // 客户端密钥路径
)

type ServerConfig struct {
	HttpPort    int    `yaml:"http_port"`    // HTTP端口
	Env         string `yaml:"env"`          // 运行环境
	EnablePprof bool   `yaml:"enable_pprof"` // 是否启用性能分析
	LogLevel    string `yaml:"log_level"`    // 日志级别

}

type GinConfig struct {
	Mode                string `yaml:"mode"`                  // 运行模式：debug/release/test
	DisableConsoleColor bool   `yaml:"disable_console_color"` // 是否禁用控制台颜色（生产环境建议true）
	MaxMultipartMemory  int64  `yaml:"max_multipart_memory"`  // 上传文件最大内存限制(MB)
	RouterPrefix        string `yaml:"router_prefix"`         // 全局路由前缀（如/api/v1）
	// CORS跨域配置
	CORS CORSConfig `yaml:"cors"`
	// 其他配置
	Gzip GzipConfig `yaml:"gzip"`
}

type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"`           // 是否开启跨域
	AllowOrigins     []string `yaml:"allow_origins"`     // 允许的源（生产环境指定具体域名，而非*）
	AllowMethods     []string `yaml:"allow_methods"`     // 允许的方法
	AllowHeaders     []string `yaml:"allow_headers"`     // 允许的请求头
	AllowCredentials bool     `yaml:"allow_credentials"` // 是否允许携带Cookie
	MaxAge           int      `yaml:"max_age"`           // 预检请求缓存时间(秒)
}

type GzipConfig struct {
	//压缩中间件
	EnableGzip       bool     `yaml:"enable_gzip"`        // 是否启用Gzip压缩
	MinSize          int      `yaml:"min_size"`           // 最小压缩阈值，单位字节
	GzipExcludeTypes []string `yaml:"gzip_exclude_types"` // 不压缩的内容类型
}

type MysqlConfig struct {
	Dialect  string `yaml:"dialect"`  // 数据库方言
	User     string `yaml:"user"`     // 用户名
	Password string `yaml:"password"` // 密码
	Host     string `yaml:"host"`     // 主机地址
	Port     int    `yaml:"port"`     // 端口
	Database string `yaml:"database"` // 数据库名
	Charset  string `yaml:"charset"`  // 字符集
	ShowSql  bool   `yaml:"show_sql"` // 是否显示SQL
	MaxOpen  int    `yaml:"max_open"` // 最大连接数
	MaxIdle  int    `yaml:"max_idle"` // 最大空闲连接数
}

type RedisConfig struct {
	Addr    string `yaml:"addr"`     // Redis地址
	PWD     string `yaml:"password"` // Redis密码
	DBIndex int    `yaml:"db_index"` // 数据库索引
	MaxIdle int    `yaml:"max_idle"` // 最大空闲连接数
	MaxOpen int    `yaml:"max_open"` // 最大连接数
}

type LoggerConfig struct {
	Path       string `yaml:"path"`        // 日志文件路径
	FileName   string `yaml:"file_name"`   // 日志文件名
	MaxSize    int    `yaml:"max_size"`    // 单个日志文件最大尺寸，单位MB
	MaxBackups int    `yaml:"max_backups"` // 最大备份数量
	MaxAge     int    `yaml:"max_age"`     // 最大保留天数
	Compress   bool   `yaml:"compress"`    // 是否压缩
}

// 初始化命令行参数
func init() {
	CurrentEnv = os.Getenv("APP_ENV")
	if CurrentEnv == "" {
		CurrentEnv = "dev" // 默认开发环境
	}

	flag.StringVar(&localConfigPath, "c", "config."+CurrentEnv+".yaml", "default config path") // 本地配置文件路径
	flag.StringVar(&etcdAddr, "r", os.Getenv("ETCD_ADDR"), "etcd address")                     // etcd地址
}

// GetDsn 方法，返回MySQL的DSN连接字符串
func (m *MysqlConfig) GetDsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local", m.User, m.Password, m.Host, m.Port, m.Database, m.Charset)
}

func InitConfig() *GlobalConfig {
	var (
		err      error
		tempConf = &GlobalConfig{}
		vipConf  = viper.New()
	)
	// 设置默认值
	setDefaultConfig()
	// 解析命令行参数
	flag.Parse()

	if etcdAddr != "" {
		// 从etcd中获取配置
		if tempConf, err = getFromRemoteAndWatchUpdate(vipConf); err != nil {
			panic(fmt.Sprintf("failed to get config from etcd: %v", err))
		}
	} else {
		// 从本地文件中获取配置
		if tempConf, err = getFromLocal(); err != nil {
			panic(fmt.Sprintf("failed to get config from local: %v", err))
		}
	}

	// 配置合法性校验
	if err := validateConfig(tempConf); err != nil {
		panic(fmt.Sprintf("config validate failed: %v", err))
	}

	// 校验配置文件里的env和环境变量的env是否一致
	if tempConf.Server.Env != "" && tempConf.Server.Env != CurrentEnv {
		panic(fmt.Sprintf(
			"config env mismatch! APP_ENV=%s, but config server.env=%s",
			CurrentEnv, tempConf.Server.Env,
		))
	}
	// 如果配置文件里的env为空，用环境变量的env填充
	if tempConf.Server.Env == "" {
		tempConf.Server.Env = CurrentEnv
	}

	// 赋值给全局配置
	tempConfig = tempConf
	// 更新全局环境标识（确保和配置一致）
	CurrentEnv = tempConf.Server.Env

	return tempConf

}

func GetGlobalConfig() *GlobalConfig {
	return tempConfig
}

func getFromRemoteAndWatchUpdate(vipConf *viper.Viper) (*GlobalConfig, error) {
	tempConf := &GlobalConfig{} // 改为指针
	if err := vipConf.AddRemoteProvider("etcd", etcdAddr, etcdKey); err != nil {
		return nil, fmt.Errorf("failed to add remote provider: %v", err)
	}
	if err := vipConf.ReadRemoteConfig(); err != nil {
		return nil, fmt.Errorf("failed to read remote config: %v", err)
	}
	if err := vipConf.Unmarshal(tempConf); err != nil { // 直接Unmarshal到指针
		return nil, fmt.Errorf("failed to unmarshal remote config: %v", err)
	}

	// 优化：用context控制goroutine退出（避免泄露）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 函数退出时取消（如果有需要）

	// 启动goroutine监听配置更新（增加错误处理+退出机制）
	go func(ctx context.Context) {
		ticker := time.NewTicker(time.Minute * 1)
		defer ticker.Stop() // 退出时停止ticker

		for {
			select {
			case <-ctx.Done():
				fmt.Println(">>> etcd config watch goroutine exit")
				return
			case <-ticker.C:
				if err := vipConf.WatchRemoteConfig(); err != nil {
					fmt.Printf(">>> watch etcd config error: %v\n", err)
					continue // 出错不退出，继续下一次监听
				}
				// 优化：先Unmarshal到临时变量，避免配置更新失败导致程序异常
				temp := &GlobalConfig{}
				if err := vipConf.Unmarshal(temp); err != nil {
					fmt.Printf(">>> unmarshal etcd config error: %v\n", err)
					continue
				}
				// 校验新配置合法后再替换
				if err := validateConfig(temp); err == nil {
					tempConfig = temp
					fmt.Println(">>> etcd config hot-reloaded success")
				} else {
					fmt.Printf(">>> new etcd config invalid: %v, skip reload\n", err)
				}
			}
		}
	}(ctx)

	return tempConf, nil
}

// 从本地文件加载配置
func getFromLocal() (*GlobalConfig, error) {
	// 初始化viper本地配置
	vipConf := viper.New()
	vipConf.SetConfigFile(localConfigPath)
	vipConf.SetConfigType("yaml") // 强制指定yaml格式

	// 读取配置文件
	if err := vipConf.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read local config error: %v, file: %s", err, localConfigPath)
	}

	// Unmarshal到结构体
	tempConf := &GlobalConfig{}
	if err := vipConf.Unmarshal(tempConf); err != nil {
		return nil, fmt.Errorf("unmarshal local config error: %v", err)
	}

	return tempConf, nil
}

// 补充到 config.go 中（建议放在 InitConfig 上方）
func setDefaultConfig() {
	// ============== 1. Server 服务器默认配置 ==============
	viper.SetDefault("server.http_port", 8080)     // 默认HTTP端口（修正原错误的port→http_port）
	viper.SetDefault("server.env", "dev")          // 默认环境
	viper.SetDefault("server.enable_pprof", false) // 默认关闭pprof（生产环境禁用）
	viper.SetDefault("server.log_level", "info")   // 默认日志级别

	// ============== 2. Gin 框架默认配置 ==============
	viper.SetDefault("gin.mode", "release")             // 生产默认release模式
	viper.SetDefault("gin.disable_console_color", true) // 生产默认禁用控制台颜色
	viper.SetDefault("gin.max_multipart_memory", 10)    // 默认10MB上传限制
	viper.SetDefault("gin.router_prefix", "/api/v1")    // 默认路由前缀

	// Gin CORS 默认配置（生产环境严格限制）
	viper.SetDefault("gin.cors.enabled", false)
	viper.SetDefault("gin.cors.allow_origins", []string{"https://your-blog.com"})
	viper.SetDefault("gin.cors.allow_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("gin.cors.allow_headers", []string{"Content-Type", "Authorization", "X-Requested-With"})
	viper.SetDefault("gin.cors.allow_credentials", true)
	viper.SetDefault("gin.cors.max_age", 3600) // 24小时

	// Gin Gzip 默认配置
	viper.SetDefault("gin.gzip.enable_gzip", true)
	viper.SetDefault("gin.gzip.min_size", 1024) // 1KB最小压缩阈值
	viper.SetDefault("gin.gzip.gzip_exclude_types", []string{
		"image/*", "video/*", "audio/*", "application/zip", "application/x-rar-compressed",
	})

	// ============== 3. MySQL 默认配置 ==============
	viper.SetDefault("mysql.dialect", "mysql")
	viper.SetDefault("mysql.user", "root")
	viper.SetDefault("mysql.password", "123456")
	viper.SetDefault("mysql.host", "127.0.0.1")
	viper.SetDefault("mysql.port", 3306)
	viper.SetDefault("mysql.database", "yia_blog_dev")
	viper.SetDefault("mysql.charset", "utf8mb4") // 生产默认utf8mb4
	viper.SetDefault("mysql.show_sql", false)    // 生产默认关闭SQL打印
	viper.SetDefault("mysql.max_open", 20)       // 最大连接数
	viper.SetDefault("mysql.max_idle", 10)       // 最大空闲连接数

	// ============== 4. Redis 默认配置 ==============
	viper.SetDefault("redis.addr", "127.0.0.1:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db_index", 0)  // 默认使用0号数据库
	viper.SetDefault("redis.max_idle", 10) // 最大空闲连接数
	viper.SetDefault("redis.max_open", 20) // 最大连接数

	// ============== 5. Logger 日志默认配置 ==============
	viper.SetDefault("logger.path", "./logs")
	viper.SetDefault("logger.file_name", "blog_server.log")
	viper.SetDefault("logger.max_size", 10)    // 单个文件100MB
	viper.SetDefault("logger.max_backups", 10) // 最多30个备份
	viper.SetDefault("logger.max_age", 7)      // 保留7天
	viper.SetDefault("logger.compress", false) // 生产默认压缩，开发默认不压缩
}

// 配置合法性校验（必填项+格式校验）
func validateConfig(conf *GlobalConfig) error {
	// 1. 端口校验（1-65535）
	if conf.Server.HttpPort <= 0 || conf.Server.HttpPort > 65535 {
		return fmt.Errorf("server.http_port invalid: %d (must 1-65535)", conf.Server.HttpPort)
	}

	// 2. Gin模式校验
	validGinModes := map[string]bool{"debug": true, "release": true, "test": true}
	if !validGinModes[conf.Gin.Mode] {
		return fmt.Errorf("gin.mode invalid: %s (must debug/release/test)", conf.Gin.Mode)
	}

	// 3. MySQL必填项校验
	if conf.Mysql.Host == "" || conf.Mysql.User == "" || conf.Mysql.Database == "" {
		return fmt.Errorf("mysql config missing: host/user/database is required")
	}

	// 4. Redis地址校验
	if conf.Redis.Addr == "" {
		return fmt.Errorf("redis.addr is required")
	}

	// 5. 日志路径校验（确保目录存在）
	if conf.Logger.Path != "" {
		if err := os.MkdirAll(conf.Logger.Path, 0755); err != nil {
			return fmt.Errorf("create logger path error: %v", err)
		}
	}

	return nil
}
