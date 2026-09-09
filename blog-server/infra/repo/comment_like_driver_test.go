package repo

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// likeSQLState 模拟事务与唯一约束，实际 SQL 由生产 GORM Repo 生成并交给驱动执行。
// 它不替代真实 MySQL 的锁与隔离级别验收；用来验证事务边界、身份条件及故障重试。
type likeSQLState struct {
	mu                           sync.Mutex
	likes                        int64
	records                      map[[2]uint64]bool
	present, public              bool
	failCounter, loseCommitReply bool
	commits, rollbacks           int
}

type likeConnector struct{ state *likeSQLState }
type likeDriver struct{}
type likeConn struct {
	state *likeSQLState
	tx    *likeTx
}
type likeTx struct {
	conn    *likeConn
	likes   int64
	records map[[2]uint64]bool
	locked  bool
}

func (c *likeConnector) Connect(context.Context) (driver.Conn, error) {
	return &likeConn{state: c.state}, nil
}
func (*likeConnector) Driver() driver.Driver        { return likeDriver{} }
func (likeDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }
func (*likeConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepared statement")
}
func (*likeConn) Close() error { return nil }
func (c *likeConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *likeConn) BeginTx(ctx context.Context, _ driver.TxOptions) (driver.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.state.mu.Lock()
	if err := ctx.Err(); err != nil {
		c.state.mu.Unlock()
		return nil, err
	}
	copy := make(map[[2]uint64]bool, len(c.state.records))
	for key, value := range c.state.records {
		copy[key] = value
	}
	c.tx = &likeTx{conn: c, likes: c.state.likes, records: copy}
	return c.tx, nil
}

func (tx *likeTx) Commit() error {
	state := tx.conn.state
	state.likes, state.records = tx.likes, tx.records
	state.commits++
	tx.conn.tx = nil
	defer state.mu.Unlock()
	if state.loseCommitReply {
		state.loseCommitReply = false
		return errors.New("commit applied but acknowledgement lost")
	}
	return nil
}

func (tx *likeTx) Rollback() error {
	tx.conn.state.rollbacks++
	tx.conn.tx = nil
	tx.conn.state.mu.Unlock()
	return nil
}

func likeID(value driver.NamedValue) uint64 {
	switch n := value.Value.(type) {
	case int64:
		return uint64(n)
	case uint64:
		return n
	default:
		panic(fmt.Sprintf("unexpected ID argument %T", value.Value))
	}
}

func (c *likeConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.tx == nil || !strings.Contains(query, "FROM `comment`") || !strings.Contains(query, "FOR UPDATE") || !strings.Contains(query, "id = ?") || !strings.Contains(query, "`comment`.`deleted_at` IS NULL") {
		return nil, fmt.Errorf("comment must be locked inside transaction: %s", query)
	}
	rows := &likeRows{}
	if c.state.present && (c.state.public || !strings.Contains(query, "status = ?")) {
		if strings.Contains(query, "status = ?") && (!strings.Contains(query, "EXISTS") || !strings.Contains(query, "`article`.`deleted_at` IS NULL")) {
			return nil, errors.New("public SQL scope missing")
		}
		rows.id = int64(likeID(args[0]))
		c.tx.locked = true
	}
	return rows, nil
}

func (c *likeConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.tx == nil || !c.tx.locked {
		return nil, fmt.Errorf("write outside locked transaction: %s", query)
	}
	switch {
	case strings.HasPrefix(query, "INSERT INTO `comment_like`"):
		if !strings.Contains(query, "`comment_id`,`user_id`") {
			return nil, fmt.Errorf("identity columns missing: %s", query)
		}
		key := [2]uint64{likeID(args[0]), likeID(args[1])}
		if c.tx.records[key] {
			return nil, &mysqlDriver.MySQLError{Number: 1062, Message: "duplicate comment/user"}
		}
		c.tx.records[key] = true
		return driver.RowsAffected(1), nil
	case strings.HasPrefix(query, "DELETE FROM `comment_like`"):
		if !strings.Contains(query, "comment_id = ? AND user_id = ?") {
			return nil, errors.New("unlike missing ownership condition")
		}
		key := [2]uint64{likeID(args[0]), likeID(args[1])}
		if !c.tx.records[key] {
			return driver.RowsAffected(0), nil
		}
		delete(c.tx.records, key)
		return driver.RowsAffected(1), nil
	case strings.HasPrefix(query, "UPDATE `comment`"):
		if c.state.failCounter {
			return nil, errors.New("counter write failed")
		}
		if strings.Contains(query, "likes + 1") {
			c.tx.likes++
		} else if strings.Contains(query, "GREATEST(likes - 1, 0)") {
			if c.tx.likes > 0 {
				c.tx.likes--
			}
		} else {
			return nil, fmt.Errorf("unexpected counter SQL: %s", query)
		}
		return driver.RowsAffected(1), nil
	default:
		return nil, fmt.Errorf("unexpected SQL: %s", query)
	}
}

type likeRows struct {
	id   int64
	read bool
}

func (*likeRows) Columns() []string { return []string{"id"} }
func (*likeRows) Close() error      { return nil }
func (r *likeRows) Next(values []driver.Value) error {
	if r.read || r.id == 0 {
		return io.EOF
	}
	r.read = true
	values[0] = r.id
	return nil
}

func newLikeSQLRepo(t *testing.T) (*commentMysqlRepo, *likeSQLState) {
	t.Helper()
	state := &likeSQLState{records: make(map[[2]uint64]bool), present: true, public: true}
	conn := sql.OpenDB(&likeConnector{state: state})
	conn.SetMaxOpenConns(16)
	t.Cleanup(func() { _ = conn.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return NewCommentMysqlRepo(db), state
}
