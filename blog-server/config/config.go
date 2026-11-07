/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-21 11:32:32
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-05 18:14:21
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
	"net"
	"os"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/spf13/viper"
)

const (
	ServerName     = "blog"
	ServerFullName = "yia.blog"
)

type GlobalConfig struct {
	AI      AIConfig      `yaml:"ai"`      // 私有助手默认关闭，密钥仅供后端使用。
	Storage StorageConfig `yaml:"storage"` // 可写资源目录，与构建产物分离。
	Server  ServerConfig  `yaml:"server"`  // 服务器基础配置
	Gin     GinConfig     `yaml:"gin"`     // Gin 专属配置
	Mysql   *MysqlConfig  `yaml:"mysql"`
	Redis   RedisConfig   `yaml:"redis"`
	Logger  LoggerConfig  `yaml:"logger"`
	JWT     JWTConfig     `yaml:"jwt"`
}

// StorageConfig 的目录由部署方设置，空值使用进程工作目录下的 uploads 对应资源子目录。
type StorageConfig struct {
	ArticleImageDir string `yaml:"article_image_dir"`
	AvatarDir       string `yaml:"avatar_dir"`
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

type JWTConfig struct {
	CustomSecret string        `yaml:"custom_secret"` // 自定义密钥（从配置文件读取）
	AdminSecret  string        `yaml:"admin_secret"`  // 管理员密钥（从配置文件读取）
	ExpireHour   time.Duration `yaml:"expire_hour"`   // Token 过期时长（如 2h、30m）
	Issuer       string        `yaml:"issuer"`        // 签发者
}

type ServerConfig struct {
	HttpHost        string        `yaml:"http_host"`        // 监听地址；空值沿用全部网卡，代理部署时设为 127.0.0.1。
	HttpPort        int           `yaml:"http_port"`        // HTTP端口
	Env             string        `yaml:"env"`              // 运行环境
	EnablePprof     bool          `yaml:"enable_pprof"`     // 仅允许 dev/local/test 环境在回环监听地址开启。
	LogLevel        string        `yaml:"log_level"`        // 日志级别
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"` // 关闭期限，配置使用 5s/1m 等时长字符串；非正值采用服务端默认值。

}

type GinConfig struct {
	Mode                string `yaml:"mode"`                  // 运行模式：debug/release/test
	DisableConsoleColor bool   `yaml:"disable_console_color"` // 是否禁用控制台颜色（生产环境建议true）
	MaxMultipartMemory  int64  `yaml:"max_multipart_memory"`  // 上传文件最大内存限制(MB)
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
	Dialect         string        `yaml:"dialect"`           // 数据库方言
	User            string        `yaml:"user"`              // 用户名
	Password        string        `yaml:"password"`          // 密码
	Host            string        `yaml:"host"`              // 主机地址
	Port            int           `yaml:"port"`              // 端口
	Database        string        `yaml:"database"`          // 数据库名
	Charset         string        `yaml:"charset"`           // 字符集
	ShowSql         bool          `yaml:"show_sql"`          // 是否显示SQL
	MaxOpen         int           `yaml:"max_open"`          // 最大连接数
	MaxIdle         int           `yaml:"max_idle"`          // 最大空闲连接数
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"` // 连接最大存活时间，如 5m；未配置时使用服务端默认值。
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`     // Redis地址
	Password string `yaml:"password"` // Redis密码
	DBIndex  int    `yaml:"db_index"` // 数据库索引
	//超时配置
	DialTimeout  time.Duration `yaml:"dial_timeout"`  // 连接超时时间（秒）
	ReadTimeout  time.Duration `yaml:"read_timeout"`  // 读取超时时间（秒）
	WriteTimeout time.Duration `yaml:"write_timeout"` // 写入超时时间（秒）
	// ====================== 连接池核心配置 ======================
	PoolSize        int           `yaml:"pool_size"`          // 最大连接数
	MinIdleConns    int           `yaml:"min_idle_conns"`     // 最小空闲连接
	MaxIdleConns    int           `yaml:"max_idle_conns"`     // 最大空闲连接
	PoolTimeout     time.Duration `yaml:"pool_timeout"`       // 获取连接超时时间
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"` // 空闲连接超时关闭
	// 缓存过期配置
	UserCacheExpire      time.Duration `yaml:"user_cache_expire"`       // 用户缓存过期时间（秒）
	AdminCacheExpire     time.Duration `yaml:"admin_cache_expire"`      // 管理员缓存过期时间（秒）
	UserEmptyCacheExpire time.Duration `yaml:"user_empty_cache_expire"` // 用户缓存过期时间（秒）
	LikeCacheExpire      time.Duration `yaml:"like_cache_expire"`       // 点赞缓存过期时间（秒）
	LikeRecordExpire     time.Duration `yaml:"like_record_expire"`      // 点赞记录缓存过期时间（秒）

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

	flag.StringVar(&localConfigPath, "c", "config/config."+CurrentEnv+".yaml", "default config path") // 本地配置文件路径
	fmt.Println("localConfigPath:", localConfigPath)
	flag.StringVar(&etcdAddr, "r", os.Getenv("ETCD_ADDR"), "etcd address") // etcd地址
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
	// 解析命令行参数
	flag.Parse()
	// 设置默认值
	setDefaultConfig(vipConf)

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

	fmt.Printf("=== 加载的配置信息 ===\n")
	fmt.Printf("server.http_port: %d\n", tempConf.Server.HttpPort)
	fmt.Printf("gin.mode: %s\n", tempConf.Gin.Mode)
	fmt.Printf("mysql.host: %s\n", tempConf.Mysql.Host)
	fmt.Printf("redis.addr: %s\n", tempConf.Redis.Addr)
	fmt.Printf("=====================\n")

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
	tempConf := GlobalConfig{}
	if _, err := os.Stat(localConfigPath); err == nil {
		content, err := os.ReadFile(localConfigPath)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(content, &tempConf)
		if err != nil {
			return nil, err
		}
		return &tempConf, nil
	}
	return nil, fmt.Errorf("local config file not found ,file_name: %s", localConfigPath)
}

// 补充到 config.go 中（建议放在 InitConfig 上方）
func setDefaultConfig(vipConf *viper.Viper) {

	vipConf.SetConfigType("yaml")
	// ============== 1. Server 服务器默认配置 ==============
	vipConf.SetDefault("server.http_port", 8080)     // 默认HTTP端口
	vipConf.SetDefault("server.env", "dev")          // 默认环境
	vipConf.SetDefault("server.enable_pprof", false) // 默认关闭pprof（生产环境禁用）
	vipConf.SetDefault("server.log_level", "info")   // 默认日志级别

	// ============== 2. Gin 框架默认配置 ==============
	vipConf.SetDefault("gin.mode", "release")             // 生产默认release模式
	vipConf.SetDefault("gin.disable_console_color", true) // 生产默认禁用控制台颜色
	vipConf.SetDefault("gin.max_multipart_memory", 10)    // 默认10MB上传限制

	// Gin CORS 默认配置（生产环境严格限制）
	vipConf.SetDefault("gin.cors.enabled", false)
	vipConf.SetDefault("gin.cors.allow_origins", []string{"https://your-blog.com"})
	vipConf.SetDefault("gin.cors.allow_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	vipConf.SetDefault("gin.cors.allow_headers", []string{"Content-Type", "Authorization", "Admin-Authorization", "X-Requested-With"})
	vipConf.SetDefault("gin.cors.allow_credentials", true)
	vipConf.SetDefault("gin.cors.max_age", 3600) // 1小时

	// Gin Gzip 默认配置
	vipConf.SetDefault("gin.gzip.enable_gzip", true)
	vipConf.SetDefault("gin.gzip.min_size", 1024) // 1KB最小压缩阈值
	vipConf.SetDefault("gin.gzip.gzip_exclude_types", []string{
		"image/*", "video/*", "audio/*", "application/zip", "application/x-rar-compressed",
	})

	// ============== 3. MySQL 默认配置 ==============
	vipConf.SetDefault("mysql.dialect", "mysql")
	vipConf.SetDefault("mysql.user", "root")
	vipConf.SetDefault("mysql.password", "123456")
	vipConf.SetDefault("mysql.host", "127.0.0.1")
	vipConf.SetDefault("mysql.port", 3306)
	vipConf.SetDefault("mysql.database", "yia_blog_dev")
	vipConf.SetDefault("mysql.charset", "utf8mb4") // 生产默认utf8mb4
	vipConf.SetDefault("mysql.show_sql", false)    // 生产默认关闭SQL打印
	vipConf.SetDefault("mysql.max_open", 20)       // 最大连接数
	vipConf.SetDefault("mysql.max_idle", 10)       // 最大空闲连接数

	// ============== 4. Redis 默认配置 ==============
	vipConf.SetDefault("redis.addr", "127.0.0.1:6379")
	vipConf.SetDefault("redis.password", "")
	vipConf.SetDefault("redis.db_index", 0)  // 默认使用0号数据库
	vipConf.SetDefault("redis.max_idle", 10) // 最大空闲连接数
	vipConf.SetDefault("redis.max_open", 20) // 最大连接数

	// ============== 5. Logger 日志默认配置 ==============
	vipConf.SetDefault("logger.path", "./logs")
	vipConf.SetDefault("logger.file_name", "blog_server.log")
	vipConf.SetDefault("logger.max_size", 10)    // 单个文件100MB
	vipConf.SetDefault("logger.max_backups", 10) // 最多30个备份
	vipConf.SetDefault("logger.max_age", 7)      // 保留7天
	vipConf.SetDefault("logger.compress", false) // 生产默认压缩，开发默认不压缩
}

// AllowsLocalPprof 将性能分析限制在显式开发环境和回环监听地址。
func (s ServerConfig) AllowsLocalPprof() bool {
	if s.Env != "dev" && s.Env != "local" && s.Env != "test" {
		return false
	}
	ip := net.ParseIP(s.HttpHost)
	return ip != nil && ip.IsLoopback()
}

// 配置合法性校验（必填项+格式校验）
func validateConfig(conf *GlobalConfig) error {
	if conf.Server.EnablePprof && !conf.Server.AllowsLocalPprof() {
		return fmt.Errorf("server.enable_pprof requires explicit dev/local/test env and a loopback IP in server.http_host")
	}
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
