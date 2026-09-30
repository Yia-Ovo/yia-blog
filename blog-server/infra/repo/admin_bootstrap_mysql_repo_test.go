package repo

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	domainrepo "blog-server/domain/repo"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type bootstrapSQLState struct {
	lockResult driver.Value
	existing   bool
	queryErr   error
	releaseErr error
	lockedConn *bootstrapSQLConn
	queryCount int
	releases   int
	closed     int
}

type bootstrapSQLConnector struct{ state *bootstrapSQLState }
type bootstrapSQLDriver struct{}
type bootstrapSQLConn struct{ state *bootstrapSQLState }
type bootstrapSQLRows struct {
	columns []string
	values  []driver.Value
	done    bool
}

func (c bootstrapSQLConnector) Connect(context.Context) (driver.Conn, error) {
	return &bootstrapSQLConn{state: c.state}, nil
}
func (bootstrapSQLConnector) Driver() driver.Driver { return bootstrapSQLDriver{} }
func (bootstrapSQLDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use connector")
}
func (*bootstrapSQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepared statement")
}
func (*bootstrapSQLConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }
func (c *bootstrapSQLConn) Close() error            { c.state.closed++; return nil }
func (r *bootstrapSQLRows) Columns() []string       { return r.columns }
func (*bootstrapSQLRows) Close() error              { return nil }
func (r *bootstrapSQLRows) Next(dest []driver.Value) error {
	if r.done || r.values == nil {
		return io.EOF
	}
	copy(dest, r.values)
	r.done = true
	return nil
}

func (c *bootstrapSQLConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	rows := &bootstrapSQLRows{columns: []string{"value"}}
	switch {
	case query == "SELECT DATABASE()":
		rows.values = []driver.Value{"bootstrap_test"}
	case strings.HasPrefix(query, "SELECT GET_LOCK"):
		c.state.lockedConn = c
		rows.values = []driver.Value{c.state.lockResult}
	case strings.HasPrefix(query, "SELECT RELEASE_LOCK"):
		if c != c.state.lockedConn {
			return nil, errors.New("unlock used a different connection")
		}
		c.state.releases++
		if c.state.releaseErr != nil {
			return nil, c.state.releaseErr
		}
		rows.values = []driver.Value{int64(1)}
	case query == "SELECT EXISTS(SELECT 1 FROM `admin`)":
		if c != c.state.lockedConn {
			return nil, errors.New("existence check did not use the locked connection")
		}
		c.state.queryCount++
		if c.state.queryErr != nil {
			return nil, c.state.queryErr
		}
		rows.values = []driver.Value{c.state.existing}
	case strings.HasPrefix(query, "SELECT * FROM `admin`"):
		if c != c.state.lockedConn {
			return nil, errors.New("callback repository lost the pinned connection")
		}
		rows.columns = []string{"id"}
	default:
		return nil, errors.New("unexpected query: " + query)
	}
	return rows, nil
}

func newBootstrapSQLTestDB(t *testing.T, state *bootstrapSQLState) *gorm.DB {
	t.Helper()
	pool := sql.OpenDB(bootstrapSQLConnector{state: state})
	pool.SetMaxOpenConns(2)
	t.Cleanup(func() { _ = pool.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: pool, SkipInitializeWithVersion: true}), &gorm.Config{
		DisableAutomaticPing: true, Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestBootstrapRepoChecksAllRowsAndPinsCallbackAndUnlock(t *testing.T) {
	for _, existing := range []bool{false, true} {
		state := &bootstrapSQLState{lockResult: int64(1), existing: existing}
		db := newBootstrapSQLTestDB(t, state)
		callbackErr := errors.New("domain rejected initialization")
		err := NewAdminBootstrapMysqlRepo(db).WithExclusiveAdmin(context.Background(), func(admins domainrepo.AdminMysqlRepo, found bool) error {
			if found != existing {
				t.Fatal("existing account result was changed")
			}
			if _, err := admins.FindAdminByName(context.Background(), "owner"); err != nil {
				t.Fatal(err)
			}
			return callbackErr
		})
		if !errors.Is(err, callbackErr) || state.queryCount != 1 || state.releases != 1 {
			t.Fatalf("err=%v checks=%d releases=%d", err, state.queryCount, state.releases)
		}
	}
}

func TestBootstrapRepoDoesNotProceedAfterLockOrQueryFailure(t *testing.T) {
	for _, state := range []*bootstrapSQLState{
		{lockResult: int64(0)}, {lockResult: nil},
		{lockResult: int64(1), queryErr: errors.New("table lookup failed")},
	} {
		db := newBootstrapSQLTestDB(t, state)
		called := false
		err := NewAdminBootstrapMysqlRepo(db).WithExclusiveAdmin(context.Background(), func(domainrepo.AdminMysqlRepo, bool) error {
			called = true
			return nil
		})
		if err == nil || called {
			t.Fatal("failure reached the initialization callback")
		}
		if state.queryErr != nil && state.releases != 1 {
			t.Fatal("query failure leaked the lock")
		}
	}
}

func TestBootstrapRepoDiscardsConnectionWhenUnlockFails(t *testing.T) {
	state := &bootstrapSQLState{lockResult: int64(1), releaseErr: errors.New("reply lost")}
	db := newBootstrapSQLTestDB(t, state)
	err := NewAdminBootstrapMysqlRepo(db).WithExclusiveAdmin(context.Background(), func(domainrepo.AdminMysqlRepo, bool) error { return nil })
	if err == nil || state.closed != 1 {
		t.Fatal("an unconfirmed locked connection was returned to the pool")
	}
}
