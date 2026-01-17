/*
 * @Author: 1372594487 1372594487@qq.com
 * @Date: 2026-01-17 16:01:23
 * @LastEditors: '1372594487
 * @LastEditTime: 2026-01-18 00:46:21
 * @Description: File description
 */
package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gogf/gf/util/gconv"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

const (
	ServerName     = "blog"
	ServerFullName = "yia.blog"
)

var (
	etcdKey         = fmt.Sprintf("/configs/%s/system", ServerFullName)
	etcdAddr        string
	localConfigPath string
	GlobalConfig    Config

	// 新增：用于存放 etcd 认证相关的变量
	// etcdUsername   string
	// etcdPassword   string
	// etcdEndpointCA string // CA 证书路径
	// etcdClientCert string // 客户端证书路径
	// etcdClientKey  string // 客户端密钥路径
)

type Config struct {
	// Server
	Server Server `yaml:"server"`
	Mysql  Mysql  `yaml:"mysql"`
	Redis  Redis  `yaml:"redis"`
}

type Server struct {
	HttpPort    int    `yaml:"http_port"`    // HTTP端口
	Env         string `yaml:"env"`          // 运行环境
	EnablePprof bool   `yaml:"enable_pprof"` // 是否启用性能分析
	LogLevel    string `yaml:"log_level"`    // 日志级别
}

type Mysql struct {
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

type Redis struct {
	Addr    string `yaml:"addr"`     // Redis地址
	PWD     string `yaml:"password"` // Redis密码
	DBIndex int    `yaml:"db_index"` // 数据库索引
	MaxIdle int    `yaml:"max_idle"` // 最大空闲连接数
	MaxOpen int    `yaml:"max_open"` // 最大连接数
}

// 初始化命令行参数
func init() {
	flag.StringVar(&localConfigPath, "c", "config/"+ServerName+"_local.yml", "default config path") // 本地配置文件路径
	flag.StringVar(&etcdAddr, "r", os.Getenv("ETCD_ADDR"), "etcd address")                          // etcd地址
}

// GetDsn 方法，返回MySQL的DSN连接字符串
func (m *Mysql) GetDsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local", m.User, m.Password, m.Host, m.Port, m.Database, m.Charset)
}

func InitConfig() *Config {
	var (
		err      error
		tempConf = &Config{}
		vipConf  = viper.New()
	)
	flag.Parse()

	if etcdAddr != "" {
		// 从etcd中获取配置
		if tempConf, err = getFromRemoteAndWatchUpdate(vipConf); err != nil {
			panic(fmt.Sprintf("failed to get config from etcd: %v", err))
		}
	} else {
		// 从本地文件中获取配置
		if tempConf, err = getFromLocal(); err != nil {
			panic(err)
		}
	}
	return tempConf

}

func getFromRemoteAndWatchUpdate(vipConf *viper.Viper) (*Config, error) {
	tempConf := Config{}
	if err := vipConf.AddRemoteProvider("etcd", etcdAddr, etcdKey); err != nil {
		return nil, fmt.Errorf("failed to add remote provider: %v", err)
	}
	if err := vipConf.ReadRemoteConfig(); err != nil {
		return nil, fmt.Errorf("failed to read remote config: %v", err)
	}
	if err := vipConf.Unmarshal(&tempConf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal remote config: %v", err)
	}

	// 启动goroutine监听配置更新
	go func() {
		for {
			time.Sleep(time.Minute * 1)
			if err := vipConf.WatchRemoteConfig(); err == nil {
				_ = vipConf.Unmarshal(&GlobalConfig)
				fmt.Println(">>> etcd config hot-reloaded: ", gconv.String(GlobalConfig))
			}
		}
	}()

	return &tempConf, nil

}

// 从本地文件加载配置
func getFromLocal() (*Config, error) {
	tempConf := Config{}
	if _, err := os.Stat(localConfigPath); err == nil {
		content, err := os.ReadFile(localConfigPath)
		if err != nil {
			return nil, err
		}
		if err = yaml.Unmarshal(content, &tempConf); err != nil {
			return nil, err
		}
		return &tempConf, nil
	}
	return nil, fmt.Errorf("local config file not found ,file_name: %s", localConfigPath)
}
