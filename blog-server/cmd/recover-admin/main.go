// recover-admin 仅供本地维护人员恢复已有超管，不注册 HTTP 路由。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"blog-server/config"
	domainrepo "blog-server/domain/repo"
	"blog-server/domain/service"
	"blog-server/infra/repo"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/goccy/go-yaml"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type quietRedisLogger struct{}

func (quietRedisLogger) Printf(context.Context, string, ...interface{}) {}

func main() {
	id, name, status, message := run(os.Args[1:], os.Stdin)
	result, _ := json.Marshal(map[string]any{
		"id": id, "name": name, "status": status, "message": message,
		"finished_at": time.Now().Format(time.RFC3339),
	})
	// 输出仅包含结果与账号标识；密码、哈希和连接凭据不进入报告。
	fmt.Println("RESULT " + string(result))
	if status == "password_changed_cache_pending" {
		os.Exit(2)
	}
	if status != "success" && status != "ready" && status != "password_matches" {
		os.Exit(1)
	}
}

func run(args []string, input io.Reader) (uint64, string, string, string) {
	flags := flag.NewFlagSet("recover-admin", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "config/config.local.yaml", "本地配置")
	name := flags.String("name", "", "已有超管账号")
	checkOnly := flags.Bool("check", false, "只读检查")
	passwordStdin := flags.Bool("password-stdin", false, "密码仅从标准输入读取")
	verifyStdin := flags.Bool("verify-stdin", false, "只读核对标准输入的密码")
	parseErr := flags.Parse(args)
	modeCount := 0
	for _, enabled := range []bool{*checkOnly, *passwordStdin, *verifyStdin} {
		if enabled {
			modeCount++
		}
	}
	if parseErr != nil || flags.NArg() != 0 || *name == "" || modeCount != 1 {
		return 0, "", "failed", "参数错误：指定配置、账号，并且只选择 -check、-password-stdin 或 -verify-stdin 之一。"
	}
	conf, err := readConfig(*configPath)
	if err != nil {
		return 0, *name, "failed", err.Error()
	}
	dsn := mysqlDriver.NewConfig()
	dsn.User, dsn.Passwd, dsn.DBName = conf.Mysql.User, conf.Mysql.Password, conf.Mysql.Database
	dsn.Net, dsn.Addr = "tcp", net.JoinHostPort(conf.Mysql.Host, strconv.Itoa(conf.Mysql.Port))
	dsn.ParseTime, dsn.Loc = true, time.Local
	dsn.Timeout, dsn.ReadTimeout, dsn.WriteTimeout = 3*time.Second, 10*time.Second, 10*time.Second
	dsn.Logger = log.New(io.Discard, "", 0)
	db, err := gorm.Open(mysql.Open(dsn.FormatDSN()), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		return 0, *name, "failed", "无法连接本地 MySQL；未输出连接凭据。"
	}
	pool, err := db.DB()
	if err != nil {
		return 0, *name, "failed", "无法取得数据库连接。"
	}
	defer pool.Close()
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(0)
	redis.SetLogger(quietRedisLogger{})
	rdb := redis.NewClient(&redis.Options{
		Addr: conf.Redis.Addr, Password: conf.Redis.Password, DB: conf.Redis.DBIndex,
		DialTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second,
		MaxRetries: -1, DialerRetries: 1, ContextTimeoutEnabled: true,
	})
	defer rdb.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return 0, *name, "failed", "无法连接本地 Redis；尚未修改密码。"
	}
	domain := service.NewAdminRecoveryDomainService(repo.NewAdminRecoveryMysqlRepo(db))
	id, err := domain.CheckTarget(ctx, *name)
	if err != nil {
		return 0, *name, "failed", safeDomainError(err)
	}
	if *checkOnly {
		return id, *name, "ready", "本地数据库与 Redis 可用，目标是启用的超级管理员；未修改密码。"
	}
	newPassword, err := readPassword(input)
	if err != nil {
		return id, *name, "failed", err.Error()
	}
	if *verifyStdin {
		matches, err := domain.CheckPassword(ctx, *name, newPassword)
		newPassword = ""
		if err != nil {
			return id, *name, "failed", "只读验证失败；未修改密码或登录状态。"
		}
		if !matches {
			return id, *name, "password_mismatch", "本次输入与数据库中的当前密码不匹配；未修改密码或登录状态。"
		}
		return id, *name, "password_matches", "本次输入与数据库中的当前密码匹配；未修改密码或登录状态。"
	}
	updatedID, err := domain.Recover(ctx, *name, newPassword)
	newPassword = ""
	if err != nil {
		return id, *name, "failed_or_unconfirmed", safeDomainError(err)
	}
	// 本地命令承担应用编排；数据库提交后只清本账号缓存，不回滚已完成的密码更新。
	cleanup, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cleanupCancel()
	cache := repo.NewAdminRedisRepo(rdb, conf)
	infoErr := cache.DelAdminFromRedis(cleanup, updatedID)
	onlineErr := cache.ClearAdminAllOnlineToken(cleanup, updatedID)
	if infoErr != nil || onlineErr != nil {
		return updatedID, *name, "password_changed_cache_pending", "密码已修改，旧登录版本已失效；本账号缓存清理未全部完成，请勿重复重置。"
	}
	return updatedID, *name, "success", "密码已重置，旧登录已失效。请使用新密码登录。"
}

func safeDomainError(err error) string {
	switch {
	case errors.Is(err, service.ErrAdminRecoveryTarget):
		return "目标必须是已有、未删除、启用的超级管理员。"
	case errors.Is(err, service.ErrAdminRecoveryCredentials):
		return "账号或密码格式不合法：密码须为 6–20 字节，同时包含英文字母与数字。"
	case errors.Is(err, domainrepo.ErrAdminRecoveryConflict):
		return "账号在操作期间发生变化，本次未写入，请重新检查。"
	default:
		return "恢复未确认完成，请先用新密码尝试登录并检查账号状态；未输出数据库错误详情。"
	}
}

func readPassword(input io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(input, 75))
	if err != nil {
		return "", errors.New("无法读取新密码。")
	}
	value := strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r")
	if len(value) == 0 || len(value) > 20 || strings.ContainsAny(value, "\r\n\x00") {
		return "", errors.New("密码须为 6–20 字节且不能包含换行或空字符。")
	}
	return value, nil
}

func readConfig(path string) (*config.GlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("无法读取本地配置文件。")
	}
	var conf config.GlobalConfig
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, errors.New("本地 YAML 格式错误；原文未输出。")
	}
	if conf.Server.Env != "local" && conf.Server.Env != "dev" {
		return nil, errors.New("此工具仅用于 local/dev 本机环境。")
	}
	if conf.Mysql == nil || conf.Mysql.Port < 1 || conf.Mysql.Port > 65535 || conf.Mysql.User == "" || conf.Mysql.Database == "" {
		return nil, errors.New("MySQL 配置不完整。")
	}
	mysqlIP := net.ParseIP(conf.Mysql.Host)
	redisHost, redisPort, redisErr := net.SplitHostPort(conf.Redis.Addr)
	port, portErr := strconv.Atoi(redisPort)
	redisIP := net.ParseIP(redisHost)
	if mysqlIP == nil || !mysqlIP.IsLoopback() || redisErr != nil || portErr != nil || port < 1 || port > 65535 ||
		redisIP == nil || !redisIP.IsLoopback() || conf.Redis.DBIndex < 0 {
		return nil, errors.New("恢复工具仅允许连接本机回环 IP 的 MySQL 和 Redis。")
	}
	return &conf, nil
}
