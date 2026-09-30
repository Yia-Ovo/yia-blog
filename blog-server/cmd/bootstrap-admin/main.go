// bootstrap-admin 仅供服务器维护人员初始化首位管理员，不启动 HTTP/Redis/etcd。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"blog-server/common/consts"
	"blog-server/domain/service"
	"blog-server/infra/repo"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/goccy/go-yaml"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type bootstrapConfig struct {
	Mysql struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"mysql"`
}

func main() {
	if err := run(os.Args[1:], os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("首位超级管理员创建成功；后续账号请在后台管理。")
}

func run(args []string, stdin io.Reader) error {
	flags := flag.NewFlagSet("bootstrap-admin", flag.ContinueOnError)
	configPath := flags.String("config", "/etc/yia-blog/config.prod.yaml", "本地配置路径")
	name := flags.String("name", "", "首位管理员账号")
	passwordStdin := flags.Bool("password-stdin", false, "从标准输入读取一行密码，不接受密码参数")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !*passwordStdin || *name == "" {
		return errors.New("用法：bootstrap-admin -config <配置路径> -name <账号> -password-stdin")
	}
	password, err := readPassword(stdin)
	if err != nil {
		return err
	}
	conf, err := readConfig(*configPath)
	if err != nil {
		return err
	}
	dsn := mysqlDriver.NewConfig()
	dsn.User, dsn.Passwd, dsn.DBName = conf.Mysql.User, conf.Mysql.Password, conf.Mysql.Database
	dsn.Net, dsn.Addr = "tcp", net.JoinHostPort(conf.Mysql.Host, strconv.Itoa(conf.Mysql.Port))
	dsn.ParseTime, dsn.Loc = true, time.Local
	dsn.Timeout, dsn.ReadTimeout, dsn.WriteTimeout = 3*time.Second, 15*time.Second, 15*time.Second
	db, err := gorm.Open(mysql.Open(dsn.FormatDSN()), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		return errors.New("MySQL 连接失败，请检查服务、端口及配置；未输出连接凭据")
	}
	pool, err := db.DB()
	if err != nil {
		return errors.New("无法取得 MySQL 连接池")
	}
	defer pool.Close()
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(0)
	ctx, cancel := context.WithTimeout(context.Background(), consts.BootstrapTimeout)
	defer cancel()
	err = service.NewAdminBootstrapDomainService(repo.NewAdminBootstrapMysqlRepo(db)).Bootstrap(ctx, *name, password)
	if errors.Is(err, service.ErrAdminBootstrapExists) || errors.Is(err, service.ErrAdminBootstrapCredentials) {
		return err
	}
	if err != nil {
		return errors.New("初始化未确认成功，请先检查数据库及账号状态后再试；不会重置已有账号")
	}
	return nil
}

func readPassword(input io.Reader) (string, error) {
	// 最多 bcrypt 上限加 CRLF；限制读入大小，不 TrimSpace 改变用户的密码。
	data, err := io.ReadAll(io.LimitReader(input, consts.BcryptPasswordMaxBytes+3))
	if err != nil {
		return "", errors.New("密码读取失败")
	}
	password := strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r")
	if len(password) > consts.BcryptPasswordMaxBytes || strings.ContainsAny(password, "\r\n\x00") {
		return "", service.ErrAdminBootstrapCredentials
	}
	return password, nil
}

func readConfig(path string) (*bootstrapConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("无法读取本地配置文件")
	}
	var conf bootstrapConfig
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, errors.New("配置 YAML 无法解析；原文未输出")
	}
	ip := net.ParseIP(conf.Mysql.Host)
	if ip == nil || !ip.IsLoopback() || conf.Mysql.Port < 1 || conf.Mysql.Port > 65535 || conf.Mysql.User == "" || conf.Mysql.Database == "" {
		return nil, errors.New("初始化工具仅连接本机回环 IP，且必须配置有效的 MySQL 端口、账号和库名")
	}
	return &conf, nil
}
