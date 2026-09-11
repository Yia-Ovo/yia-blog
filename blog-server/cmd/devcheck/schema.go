package main

import (
	"context"
	"database/sql"
	"sort"
	"strings"
)

type column struct {
	Type         string
	Nullable     string
	DefaultValue sql.NullString
	Extra        string
}

type table struct {
	Engine  string
	Columns map[string]column
}

// 必要列来自当前 infra/model；这里只核对结构，不调用 AutoMigrate。
var requiredColumns = map[string]string{
	"user":         "id nick_name password avatar gender status token_version last_login_at create_at update_at deleted_at signature email",
	"admin":        "id admin_name password avatar role status token_version last_login_at create_at update_at deleted_at email remark",
	"article":      "id user_id admin_id title cover summary content category_id tag_ids status sort views likes comments is_top comment_status publish_at create_at update_at deleted_at",
	"category":     "id name sort status create_at update_at deleted_at",
	"comment":      "id user_id article_id parent_id admin_id nickname email avatar content ip user_agent status likes create_at update_at deleted_at",
	"comment_like": "comment_id user_id create_at",
}

func readSchema(ctx context.Context, tx *sql.Tx, database string) (map[string]*table, []string, error) {
	tables := make(map[string]*table)
	rows, err := tx.QueryContext(ctx, `SELECT TABLE_NAME, COALESCE(ENGINE, '') FROM information_schema.TABLES
WHERE TABLE_SCHEMA = ? AND TABLE_NAME IN ('user', 'admin', 'article', 'category', 'comment', 'comment_like')`, database)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var name, engine string
		if err := rows.Scan(&name, &engine); err != nil {
			_ = rows.Close()
			return nil, nil, err
		}
		tables[name] = &table{Engine: engine, Columns: make(map[string]column)}
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return nil, nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT TABLE_NAME, COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT, EXTRA
FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME IN ('user', 'admin', 'article', 'category', 'comment', 'comment_like')`, database)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var name, field string
		var col column
		if err := rows.Scan(&name, &field, &col.Type, &col.Nullable, &col.DefaultValue, &col.Extra); err != nil {
			_ = rows.Close()
			return nil, nil, err
		}
		if item := tables[name]; item != nil {
			item.Columns[field] = col
		}
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return nil, nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT COLUMN_NAME FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = ? AND TABLE_NAME = 'comment_like' AND INDEX_NAME = 'PRIMARY' ORDER BY SEQ_IN_INDEX`, database)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var primary []string
	for rows.Next() {
		var field string
		if err := rows.Scan(&field); err != nil {
			return nil, nil, err
		}
		primary = append(primary, field)
	}
	return tables, primary, rows.Err()
}

func schemaIssues(tables map[string]*table, primary []string) []string {
	var issues []string
	for name, fields := range requiredColumns {
		item := tables[name]
		if item == nil {
			issues = append(issues, "缺表 "+name)
			continue
		}
		for _, field := range strings.Fields(fields) {
			if _, ok := item.Columns[field]; !ok {
				issues = append(issues, "缺列 "+name+"."+field)
			}
		}
		if (name == "comment" || name == "comment_like") && !strings.EqualFold(item.Engine, "InnoDB") {
			issues = append(issues, name+" 必须使用 InnoDB")
		}
		if name == "user" || name == "admin" {
			col, ok := item.Columns["token_version"]
			if ok && (!strings.HasPrefix(col.Type, "bigint") || !strings.Contains(col.Type, "unsigned") || col.Nullable != "NO" || !col.DefaultValue.Valid || col.DefaultValue.String != "0") {
				issues = append(issues, name+".token_version 应为非空 BIGINT UNSIGNED DEFAULT 0")
			}
		}
		if name == "category" {
			col, ok := item.Columns["update_at"]
			if ok && !strings.Contains(strings.ToLower(col.Extra), "on update current_timestamp") {
				issues = append(issues, "category.update_at 缺少 ON UPDATE CURRENT_TIMESTAMP")
			}
		}
		if name == "comment_like" {
			if col, ok := item.Columns["create_at"]; ok && (col.Nullable != "NO" || !col.DefaultValue.Valid || !strings.HasPrefix(strings.ToLower(col.DefaultValue.String), "current_timestamp")) {
				issues = append(issues, "comment_like.create_at 应为非空且默认 CURRENT_TIMESTAMP")
			}
			for _, field := range []string{"comment_id", "user_id"} {
				col, ok := item.Columns[field]
				if ok && (!strings.HasPrefix(col.Type, "bigint") || !strings.Contains(col.Type, "unsigned") || col.Nullable != "NO") {
					issues = append(issues, name+"."+field+" 应为非空 BIGINT UNSIGNED")
				}
			}
			if strings.Join(primary, ",") != "comment_id,user_id" {
				issues = append(issues, "comment_like 主键应为 (comment_id, user_id)")
			}
		}
	}
	sort.Strings(issues)
	return issues
}
