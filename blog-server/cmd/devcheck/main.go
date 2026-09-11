// devcheck 只读检查本地联调前置条件，不执行迁移、不创建账号、不修改历史数据。
package main

import (
	"blog-server/common"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/goccy/go-yaml"
	"github.com/redis/go-redis/v9"
)

// localConfig 只读取检查所需字段，避免调用会初始化日志或远程配置的服务启动流程。
type localConfig struct {
	Server struct {
		HTTPPort int `yaml:"http_port"`
	} `yaml:"server"`
	Mysql struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"mysql"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DBIndex  int    `yaml:"db_index"`
	} `yaml:"redis"`
}

type check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type report struct {
	CheckedAt string  `json:"checked_at"`
	Scope     string  `json:"scope"`
	Checks    []check `json:"checks"`
}

type quietRedisLogger struct{}

func (quietRedisLogger) Printf(context.Context, string, ...interface{}) {}

func (r *report) add(name, status, detail string) {
	r.Checks = append(r.Checks, check{name, status, detail})
}

func main() {
	// 驱动的原始错误也可能包含连接信息，只通过脱敏报告输出诊断结果。
	redis.SetLogger(quietRedisLogger{})
	configPath := flag.String("config", "config/config.dev.yaml", "本地配置路径（不会打印密码或密钥）")
	frontend := flag.String("frontend", "http://127.0.0.1:4173", "前端开发或预览地址")
	jsonOutput := flag.Bool("json", false, "输出脱敏 JSON 报告")
	flag.Parse()
	r := report{CheckedAt: time.Now().Format(time.RFC3339), Scope: "只读预检；不代表登录、写作、审核、点赞等真实流程已验收", Checks: []check{}}
	conf, err := readConfig(*configPath)
	if err != nil {
		r.add("config", "fail", err.Error())
	} else if origin, err := parseOrigin(*frontend); err != nil {
		r.add("frontend.config", "fail", err.Error())
	} else {
		r.add("config", "pass", fmt.Sprintf("MySQL %s/%s；Redis %s DB %d；HTTP :%d", net.JoinHostPort(conf.Mysql.Host, strconv.Itoa(conf.Mysql.Port)), conf.Mysql.Database, conf.Redis.Addr, conf.Redis.DBIndex, conf.Server.HTTPPort))
		checkMysql(conf, &r)
		checkRedis(conf, &r)
		checkHTTP(conf.Server.HTTPPort, origin, &r)
	}
	failed := false
	for _, item := range r.Checks {
		failed = failed || item.Status == "fail"
		if !*jsonOutput {
			fmt.Printf("[%s] %s: %s\n", item.Status, item.Name, item.Detail)
		}
	}
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(r); err != nil {
			os.Exit(1)
		}
	} else {
		fmt.Println(r.Scope)
	}
	if failed {
		os.Exit(1)
	}
}

func readConfig(path string) (*localConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("无法读取本地配置文件，请确认 -config 路径")
	}
	var conf localConfig
	if err := yaml.Unmarshal(data, &conf); err != nil {
		// YAML 错误可能包含原文，不能直接输出带凭据的配置行。
		return nil, errors.New("本地 YAML 解析失败，请检查缩进和字段类型；原文未输出")
	}
	if conf.Mysql.Host == "" || conf.Mysql.User == "" || conf.Mysql.Database == "" || conf.Redis.Addr == "" || conf.Mysql.Port < 1 || conf.Mysql.Port > 65535 || conf.Server.HTTPPort < 1 || conf.Server.HTTPPort > 65535 || conf.Redis.DBIndex < 0 {
		return nil, errors.New("本地配置缺少合法的 MySQL、Redis 或 HTTP 连接字段")
	}
	return &conf, nil
}

func parseOrigin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("-frontend 必须是无凭据、查询参数和页面路径的 http(s) 来源地址")
	}
	return u.Scheme + "://" + u.Host, nil
}

func checkMysql(conf *localConfig, r *report) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg := mysql.NewConfig()
	cfg.User, cfg.Passwd, cfg.DBName = conf.Mysql.User, conf.Mysql.Password, conf.Mysql.Database
	cfg.Net, cfg.Addr = "tcp", net.JoinHostPort(conf.Mysql.Host, strconv.Itoa(conf.Mysql.Port))
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 2*time.Second, 2*time.Second, 2*time.Second
	cfg.Logger = log.New(io.Discard, "", 0)
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		r.add("mysql.connect", "fail", "MySQL 连接配置无效；凭据未输出")
		return
	}
	db := sql.OpenDB(connector)
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		r.add("mysql.connect", "fail", safeConnectionError(err))
		return
	}
	r.add("mysql.connect", "pass", "已连接配置指定的数据库")
	// 即使后续扩展查询，数据库也会拒绝此事务中的数据写入。
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		r.add("mysql.readonly", "fail", safeConnectionError(err))
		return
	}
	defer tx.Rollback()
	tables, primary, err := readSchema(ctx, tx, conf.Mysql.Database)
	if err != nil {
		r.add("mysql.schema", "fail", safeConnectionError(err))
		return
	}
	issues := schemaIssues(tables, primary)
	if len(issues) > 0 {
		r.add("mysql.schema", "fail", strings.Join(issues, "；"))
		return
	}
	r.add("mysql.schema", "pass", "六张业务表的必要列、时间列、token_version 及点赞表事务/主键条件符合检查项")
	var emptyCategories, likedComments, totalLikes, relations, superAdmins int64
	queries := []struct {
		query string
		args  []any
	}{
		{"SELECT COUNT(*) FROM category WHERE name = ''", []any{&emptyCategories}},
		{"SELECT COUNT(*), COALESCE(SUM(likes), 0) FROM comment WHERE likes > 0", []any{&likedComments, &totalLikes}},
		{"SELECT COUNT(*) FROM comment_like", []any{&relations}},
		{"SELECT COUNT(*) FROM admin WHERE role = 0 AND status = 1 AND deleted_at IS NULL", []any{&superAdmins}},
	}
	for _, query := range queries {
		if err := tx.QueryRowContext(ctx, query.query).Scan(query.args...); err != nil {
			r.add("mysql.history", "fail", safeConnectionError(err))
			return
		}
	}
	r.add("mysql.history", "pass", fmt.Sprintf("空名分类 %d；有赞评论 %d；累计点赞 %d；持久化关系 %d；启用超管 %d", emptyCategories, likedComments, totalLikes, relations, superAdmins))
	if emptyCategories > 0 || totalLikes != relations {
		r.add("mysql.history.review", "warn", "历史数据需按迁移说明核对；本工具不回填、不重置计数、不删除空行")
	}
	if superAdmins == 0 {
		r.add("admin.bootstrap", "warn", "没有启用的超级管理员，分类管理与完整后台验收需要测试管理员")
	}
}

func safeConnectionError(err error) string {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return fmt.Sprintf("MySQL 错误 %d，请检查服务、权限或表结构；原始错误未输出", mysqlErr.Number)
	}
	return "连接或只读查询失败，请检查服务、地址、凭据和执行权限；原始错误未输出"
}

func checkRedis(conf *localConfig, r *report) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := redis.NewClient(&redis.Options{
		Addr: conf.Redis.Addr, Password: conf.Redis.Password, DB: conf.Redis.DBIndex,
		DialTimeout: 2 * time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second,
		MaxRetries: -1, DialerRetries: 1, ContextTimeoutEnabled: true, DisableIdentity: true,
	})
	defer client.Close()
	if err := client.Ping(ctx).Err(); err != nil {
		r.add("redis.connect", "fail", safeConnectionError(err))
		return
	}
	r.add("redis.connect", "pass", "PING 成功；未写入或清理键")
	// 只做有界扫描，不读取用户资料或包含 token 的键值；结果不等于全库盘点。
	keys, cursor, err := client.Scan(ctx, 0, "comment:like:user:*", 100).Result()
	if err != nil {
		r.add("redis.legacy", "warn", "旧 Bitmap 只读扫描失败，历史关系仍需核对")
		return
	}
	detail := fmt.Sprintf("首批 SCAN 发现旧 Bitmap 键 %d 个；扫描结束=%t；未导出或删除", len(keys), cursor == 0)
	if len(keys) > 0 || cursor != 0 {
		r.add("redis.legacy", "warn", detail+"，完整盘点见点赞切换说明")
	} else {
		r.add("redis.legacy", "pass", detail+"；过期关系仍无法从计数反推")
	}
}

func checkHTTP(port int, origin string, r *report) {
	client := &http.Client{
		Timeout:       3 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Transport:     &http.Transport{DialContext: (&net.Dialer{Timeout: 2 * time.Second}).DialContext},
	}
	defer client.CloseIdleConnections()
	backend := "http://127.0.0.1:" + strconv.Itoa(port)
	for _, target := range []struct{ name, address string }{{"backend.ping", backend + "/ping"}, {"frontend.http", origin + "/"}} {
		res, err := client.Get(target.address)
		if err != nil {
			r.add(target.name, "fail", "HTTP 连接失败，请启动对应服务并确认端口")
			continue
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
		_ = res.Body.Close()
		if res.StatusCode != http.StatusOK {
			r.add(target.name, "fail", fmt.Sprintf("HTTP %d", res.StatusCode))
		} else {
			r.add(target.name, "pass", "HTTP 200；连通不代表业务流程已验收")
		}
	}
	func() {
		req, err := http.NewRequest(http.MethodOptions, backend+"/api/blog/v1/admin/getInfo", nil)
		if err != nil {
			r.add("backend.cors", "fail", "无法构造检查请求")
			return
		}
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization,Admin-Authorization")
		res, err := client.Do(req)
		if err != nil {
			r.add("backend.cors", "fail", "预检连接失败，请检查后端服务")
			return
		}
		_ = res.Body.Close()
		if issue := corsIssue(res.StatusCode, res.Header, origin); issue != "" {
			r.add("backend.cors", "fail", issue)
		} else {
			r.add("backend.cors", "pass", "来源、POST 和双身份鉴权头预检通过")
		}
	}()
	checkProxy(client, origin, r)
}

func checkProxy(client *http.Client, origin string, r *report) {
	// Vite 会自己回答 OPTIONS，不能据此断言代理已连接后端。
	// 无 token 的 GET 在鉴权入口直接被拒绝，不读取账号、不建立会话、不回写缓存。
	req, err := http.NewRequest(http.MethodGet, origin+"/api/blog/v1/user/getInfo", nil)
	if err != nil {
		r.add("frontend.proxy", "fail", "无法构造检查请求")
		return
	}
	req.Header.Set("Origin", origin)
	res, err := client.Do(req)
	if err != nil {
		r.add("frontend.proxy", "fail", "代理连接失败，请检查后端和前端 API_PROXY_TARGET")
		return
	}
	defer res.Body.Close()
	if issue := proxyIssue(res.StatusCode, res.Body); issue != "" {
		r.add("frontend.proxy", "fail", issue)
	} else {
		r.add("frontend.proxy", "pass", "匿名 GET 到达用户鉴权入口，并返回预期的未登录业务码")
	}
}

func proxyIssue(status int, body io.Reader) string {
	if status != http.StatusOK {
		return fmt.Sprintf("代理请求返回 HTTP %d，请检查后端、CORS 和 API_PROXY_TARGET", status)
	}
	var envelope struct {
		Code   *int            `json:"code"`
		Msg    *string         `json:"msg"`
		ErrMsg *string         `json:"err_msg"`
		Data   json.RawMessage `json:"data"`
	}
	decoder := json.NewDecoder(io.LimitReader(body, 64*1024))
	if err := decoder.Decode(&envelope); err != nil || envelope.Code == nil || envelope.Msg == nil || envelope.ErrMsg == nil || envelope.Data == nil {
		return "代理没有返回博客 API 的完整 JSON 信封，不能把静态 HTML 或预检响应当作连接成功"
	}
	if *envelope.Code != common.LoginNotLoginErr.Code || *envelope.ErrMsg != "" || strings.TrimSpace(string(envelope.Data)) != "null" {
		return "匿名鉴权响应不符合未登录和错误隐私契约，请核对代理目标及后端版本"
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return "代理响应包含额外内容，请核对 API 代理配置"
	}
	return ""
}

func corsIssue(status int, header http.Header, origin string) string {
	if status != http.StatusNoContent {
		return fmt.Sprintf("预检应返回 204，实际为 %d；检查 CORS 或代理", status)
	}
	if allowed := header.Get("Access-Control-Allow-Origin"); allowed != origin && allowed != "*" {
		return "CORS 未允许当前前端来源"
	}
	contains := func(value, wanted string) bool {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), wanted) {
				return true
			}
		}
		return false
	}
	if !contains(header.Get("Access-Control-Allow-Methods"), "POST") {
		return "CORS 未允许 POST"
	}
	for _, name := range []string{"Content-Type", "Authorization", "Admin-Authorization"} {
		if !contains(header.Get("Access-Control-Allow-Headers"), name) {
			return "CORS 缺少请求头 " + name
		}
	}
	return ""
}
