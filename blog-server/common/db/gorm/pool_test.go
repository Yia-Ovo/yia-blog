package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"blog-server/common/consts"
	"blog-server/config"
)

type poolConnector struct{}

func (poolConnector) Connect(context.Context) (driver.Conn, error) { return poolConn{}, nil }
func (poolConnector) Driver() driver.Driver                        { return poolDriver{} }

type poolDriver struct{}

func (poolDriver) Open(string) (driver.Conn, error) { return poolConn{}, nil }

type poolConn struct{}

func (poolConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not used") }
func (poolConn) Close() error                        { return nil }
func (poolConn) Begin() (driver.Tx, error)           { return nil, errors.New("not used") }

func TestSQLPoolAppliesLimitsAndConnectionLifetime(t *testing.T) {
	db := sql.OpenDB(poolConnector{})
	defer db.Close()
	configureSQLPool(db, &config.MysqlConfig{MaxOpen: 2, MaxIdle: 1, ConnMaxLifetime: 10 * time.Millisecond})
	first, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if extra, err := db.Conn(ctx); !errors.Is(err, context.DeadlineExceeded) {
		if extra != nil {
			extra.Close()
		}
		t.Fatalf("pool did not enforce connection limit: %v", err)
	}
	first.Close()
	second.Close()
	if stats := db.Stats(); stats.MaxOpenConnections != 2 || stats.Idle > 1 {
		t.Fatalf("pool limits not applied: %+v", stats)
	}
	// 预留连接期间已超出寿命，归还或再次取出时必须淘汰。
	replacement, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	replacement.Close()
	if db.Stats().MaxLifetimeClosed == 0 {
		t.Fatal("expired connections were retained")
	}
}

func TestSQLPoolDefaultsAndZeroIdle(t *testing.T) {
	for _, tc := range []struct{ open, idle, wantOpen, wantIdle int }{{0, 0, consts.DefaultMySQLMaxOpen, 0}, {-1, -1, consts.DefaultMySQLMaxOpen, 0}, {1, 5, 1, 1}} {
		db := sql.OpenDB(poolConnector{})
		configureSQLPool(db, &config.MysqlConfig{MaxOpen: tc.open, MaxIdle: tc.idle})
		conn, err := db.Conn(context.Background())
		if err != nil {
			db.Close()
			t.Fatal(err)
		}
		conn.Close()
		stats := db.Stats()
		db.Close()
		if stats.MaxOpenConnections != tc.wantOpen || stats.Idle != tc.wantIdle {
			t.Fatalf("invalid pool defaults: %+v", stats)
		}
	}
}
