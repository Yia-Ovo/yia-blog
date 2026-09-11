package main

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestCredentialErrorsStayRedacted(t *testing.T) {
	secret := "private-test-password"
	path := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(path, []byte("mysql:\n  password: ["+secret), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readConfig(path); err == nil || strings.Contains(err.Error(), secret) {
		t.Fatal("解析错误必须保留失败状态且不能包含 YAML 凭据原文")
	}
	for _, err := range []error{errors.New(secret), &mysql.MySQLError{Number: 1045, Message: secret}} {
		if strings.Contains(safeConnectionError(err), secret) {
			t.Fatal("连接错误不能泄露驱动返回的敏感原文")
		}
	}
}

func TestOriginRejectsCredentialsAndPageURLs(t *testing.T) {
	for _, raw := range []string{"file:///tmp/index.html", "http://user:secret@localhost:4173", "http://localhost:4173/login", "http://localhost:4173/?token=secret", "http://localhost:4173/#token"} {
		if _, err := parseOrigin(raw); err == nil {
			t.Fatalf("应拒绝非来源地址 %q", raw)
		}
	}
	if origin, err := parseOrigin("http://127.0.0.1:4173/"); err != nil || origin != "http://127.0.0.1:4173" {
		t.Fatal("应移除末尾斜杠后用于精确 Origin 匹配")
	}
}

func TestCORSRequiresAdminHeaderAndRealPreflight(t *testing.T) {
	origin := "http://127.0.0.1:4173"
	header := make(http.Header)
	header.Set("Access-Control-Allow-Origin", origin)
	header.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	header.Set("Access-Control-Allow-Headers", "content-type, authorization")
	if !strings.Contains(corsIssue(204, header, origin), "Admin-Authorization") {
		t.Fatal("缺少管理员头应阻止通过预检")
	}
	header.Set("Access-Control-Allow-Headers", "content-type, authorization, admin-authorization")
	if issue := corsIssue(204, header, origin); issue != "" {
		t.Fatal(issue)
	}
	if corsIssue(200, header, origin) == "" {
		t.Fatal("不能把静态页面的 HTTP 200 当作 API 代理预检通过")
	}
	header.Set("Access-Control-Allow-Origin", "http://localhost:8081")
	if corsIssue(204, header, origin) == "" {
		t.Fatal("必须检查前端实际来源")
	}
}

func TestLikeMigrationRequiresEngineAndCompositeKey(t *testing.T) {
	base := func() map[string]*table {
		return map[string]*table{
			"comment": {Engine: "InnoDB", Columns: map[string]column{}},
			"comment_like": {Engine: "InnoDB", Columns: map[string]column{
				"comment_id": {Type: "bigint unsigned", Nullable: "NO"},
				"user_id":    {Type: "bigint unsigned", Nullable: "NO"},
				"create_at":  {Type: "datetime", Nullable: "NO", DefaultValue: sql.NullString{String: "CURRENT_TIMESTAMP", Valid: true}},
			}},
		}
	}
	for _, tc := range []struct {
		name    string
		primary []string
		mutate  func(map[string]*table)
		want    string
	}{
		{"missing table", nil, func(tables map[string]*table) { delete(tables, "comment_like") }, "缺表 comment_like"},
		{"single key", []string{"comment_id"}, func(map[string]*table) {}, "comment_like 主键应为 (comment_id, user_id)"},
		{"extra key", []string{"comment_id", "user_id", "create_at"}, func(map[string]*table) {}, "comment_like 主键应为 (comment_id, user_id)"},
		{"nontransactional", []string{"comment_id", "user_id"}, func(tables map[string]*table) { tables["comment"].Engine = "MyISAM" }, "comment 必须使用 InnoDB"},
		{"signed id", []string{"comment_id", "user_id"}, func(tables map[string]*table) {
			tables["comment_like"].Columns["user_id"] = column{Type: "bigint", Nullable: "NO"}
		}, "comment_like.user_id 应为非空 BIGINT UNSIGNED"},
		{"missing time default", []string{"comment_id", "user_id"}, func(tables map[string]*table) {
			tables["comment_like"].Columns["create_at"] = column{Type: "datetime", Nullable: "NO"}
		}, "comment_like.create_at 应为非空且默认 CURRENT_TIMESTAMP"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tables := base()
			tc.mutate(tables)
			if !strings.Contains(strings.Join(schemaIssues(tables, tc.primary), "；"), tc.want) {
				t.Fatalf("未识别迁移缺陷：%s", tc.want)
			}
		})
	}
	for _, issue := range schemaIssues(base(), []string{"comment_id", "user_id"}) {
		if strings.Contains(issue, "comment_like") || strings.Contains(issue, "InnoDB") {
			t.Fatalf("合法点赞结构被误报：%s", issue)
		}
	}
}

func TestProxyDoesNotTrustVitePreflight(t *testing.T) {
	var method, path, auth, adminAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		auth, adminAuth = r.Header.Get("Authorization"), r.Header.Get("Admin-Authorization")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	var result report
	checkProxy(server.Client(), server.URL, &result)
	if method != http.MethodGet || path != "/api/blog/v1/user/getInfo" || auth != "" || adminAuth != "" {
		t.Fatal("代理验证必须使用无身份的实际 GET 请求")
	}
	if len(result.Checks) != 1 || result.Checks[0].Status != "fail" {
		t.Fatal("后端不可用时，Vite 的 OPTIONS 不能使代理验证通过")
	}
}

func TestProxyRequiresBlogUnauthenticatedEnvelope(t *testing.T) {
	valid := `{"code":11009,"msg":"Not login","err_msg":"","data":null}`
	if issue := proxyIssue(200, strings.NewReader(valid)); issue != "" {
		t.Fatal(issue)
	}
	for _, body := range []string{
		"<!doctype html><title>Vite</title>",
		`{"code":11009}`,
		`{"code":200,"msg":"OK","err_msg":"","data":null}`,
		`{"code":11009,"msg":"Not login","err_msg":"private error","data":null}`,
		valid + `{"extra":true}`,
	} {
		if proxyIssue(200, strings.NewReader(body)) == "" {
			t.Fatal("错误、缺字段或静态响应不能通过代理验证")
		}
	}
	if proxyIssue(204, strings.NewReader(valid)) == "" {
		t.Fatal("OPTIONS 状态码不能通过真实代理检查")
	}
}

func TestTokenVersionAndCategoryDatabaseDefaults(t *testing.T) {
	tables := map[string]*table{
		"user":     {Columns: map[string]column{"token_version": {Type: "bigint unsigned", Nullable: "YES"}}},
		"category": {Columns: map[string]column{"update_at": {Type: "datetime"}}},
	}
	issues := strings.Join(schemaIssues(tables, nil), "；")
	if !strings.Contains(issues, "user.token_version 应为") || !strings.Contains(issues, "category.update_at 缺少") {
		t.Fatal("应识别旧表的令牌版本和自动更新时间缺陷")
	}
	tables["user"].Columns["token_version"] = column{Type: "bigint(20) unsigned", Nullable: "NO", DefaultValue: sql.NullString{String: "0", Valid: true}}
	tables["category"].Columns["update_at"] = column{Type: "datetime", Extra: "DEFAULT_GENERATED on update CURRENT_TIMESTAMP"}
	issues = strings.Join(schemaIssues(tables, nil), "；")
	if strings.Contains(issues, "user.token_version 应为") || strings.Contains(issues, "category.update_at 缺少") {
		t.Fatal("应兼容 MySQL 显示宽度及默认值元数据格式")
	}
}
