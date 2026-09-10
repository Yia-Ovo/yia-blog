package gorm

import (
	"context"
	"testing"

	"blog-server/config"
	gormlogger "gorm.io/gorm/logger"
)

func TestSQLLoggingUsesConfigurationAndOmitsParameters(t *testing.T) {
	for _, tc := range []struct {
		show    bool
		env     string
		noColor bool
		level   gormlogger.LogLevel
		color   bool
	}{
		{false, "prod", false, gormlogger.Warn, false},
		{true, "prod", false, gormlogger.Info, false},
		{true, "dev", false, gormlogger.Info, true},
		{true, "dev", true, gormlogger.Info, false},
	} {
		c := gormLogConfig(&config.GlobalConfig{Mysql: &config.MysqlConfig{ShowSql: tc.show}, Server: config.ServerConfig{Env: tc.env}, Gin: config.GinConfig{DisableConsoleColor: tc.noColor}})
		if c.LogLevel != tc.level || c.Colorful != tc.color || !c.ParameterizedQueries || !c.IgnoreRecordNotFoundError {
			t.Fatalf("SQL logging config mismatch: %+v", c)
		}
		logger := gormlogger.New(testLogWriter{}, c)
		filter, ok := logger.(interface {
			ParamsFilter(context.Context, string, ...interface{}) (string, []interface{})
		})
		if !ok {
			t.Fatal("GORM logger has no parameter filter")
		}
		_, params := filter.ParamsFilter(context.Background(), "SELECT ?", "private-password-marker")
		if len(params) != 0 {
			t.Fatalf("SQL log parameters exposed: %v", params)
		}
	}
}

type testLogWriter struct{}

func (testLogWriter) Printf(string, ...interface{}) {}
