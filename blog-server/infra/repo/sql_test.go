package repo

import (
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// newSQLTestDB 用真实 MySQL 方言生成 SQL，但不建立连接或执行数据库语句。
func newSQLTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "test:test@tcp(127.0.0.1:1)/test?parseTime=true",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true,
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return db
}

type sqlCapture struct {
	sql   string
	args  []interface{}
	calls int
	rows  int64
	err   error
}

// captureSQL 在 GORM 生成语句后记录 SQL，并模拟驱动返回的行数/错误。
func captureSQL(t *testing.T, db *gorm.DB, operation string, result *sqlCapture) {
	t.Helper()
	capture := func(tx *gorm.DB) {
		result.sql = tx.Statement.SQL.String()
		result.args = append([]interface{}(nil), tx.Statement.Vars...)
		result.calls++
		tx.RowsAffected = result.rows
		if result.err != nil {
			tx.AddError(result.err)
		}
	}
	var err error
	switch operation {
	case "update":
		err = db.Callback().Update().After("gorm:update").Register("test:capture", capture)
	case "query":
		err = db.Callback().Query().After("gorm:query").Register("test:capture", capture)
	case "create":
		err = db.Callback().Create().After("gorm:create").Register("test:capture", capture)
	case "delete":
		err = db.Callback().Delete().After("gorm:delete").Register("test:capture", capture)
	default:
		t.Fatalf("unsupported SQL operation %q", operation)
	}
	if err != nil {
		t.Fatal(err)
	}
}
