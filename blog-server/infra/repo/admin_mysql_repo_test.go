package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"
	"blog-server/infra/model"

	"gorm.io/gorm"
)

func TestAdminLookupDistinguishesMissingAndFailure(t *testing.T) {
	dbErr := errors.New("database unavailable")
	for _, lookup := range []string{"id", "name"} {
		for _, tc := range []struct {
			name string
			err  error
			want error
		}{
			{"missing", gorm.ErrRecordNotFound, nil},
			{"wrapped missing", fmt.Errorf("query: %w", gorm.ErrRecordNotFound), nil},
			{"failure", dbErr, dbErr},
		} {
			t.Run(lookup+"/"+tc.name, func(t *testing.T) {
				db := newSQLTestDB(t)
				captureSQL(t, db, "query", &sqlCapture{err: tc.err})
				r := NewAdminMysqlRepo(db)
				var admin *model.AdminModel
				var err error
				if lookup == "id" {
					admin, err = r.FindAdminByID(context.Background(), 2)
				} else {
					admin, err = r.FindAdminByName(context.Background(), "admin")
				}
				if admin != nil || !errors.Is(err, tc.want) {
					t.Fatalf("lookup = %+v, %v; want nil, %v", admin, err, tc.want)
				}
			})
		}
	}
}

func TestAdminManagementSQLGuardsRoleAndReportsConflict(t *testing.T) {
	actions := []struct {
		name string
		run  func(domainrepo.AdminMysqlRepo) error
	}{
		{"role", func(r domainrepo.AdminMysqlRepo) error {
			return r.ChangeAdminRole(context.Background(), 2, consts.AdminRoleSuper)
		}},
		{"status", func(r domainrepo.AdminMysqlRepo) error {
			return r.UpdateAdminStatus(context.Background(), 2, consts.AdminStatusDisabled)
		}},
		{"password", func(r domainrepo.AdminMysqlRepo) error { return r.ResetAdminPWDByID(context.Background(), 2, "hash") }},
	}
	for _, action := range actions {
		for _, rows := range []int64{0, 1} {
			t.Run(fmt.Sprintf("%s/rows=%d", action.name, rows), func(t *testing.T) {
				db := newSQLTestDB(t)
				result := &sqlCapture{rows: rows}
				captureSQL(t, db, "update", result)
				err := action.run(NewAdminMysqlRepo(db))
				if rows == 0 && !errors.Is(err, domainrepo.ErrNoRowsAffected) || rows == 1 && err != nil {
					t.Fatalf("rows=%d returned %v", rows, err)
				}
				_, where, ok := strings.Cut(result.sql, " WHERE ")
				if !ok || !strings.Contains(where, "id = ? AND role = ?") || !strings.Contains(where, "deleted_at") {
					t.Fatalf("missing conditional/soft-delete protection: %s", result.sql)
				}
				if !strings.Contains(result.sql, "token_version + ?") {
					t.Fatalf("token version was not incremented atomically: %s", result.sql)
				}
				n := len(result.args)
				if n < 2 || result.args[n-2] != uint64(2) || result.args[n-1] != consts.AdminRoleNormal {
					t.Fatalf("wrong conditional arguments: %v", result.args)
				}
			})
		}
	}
}

func TestAdminOwnPasswordSQLAllowsSuperAdmin(t *testing.T) {
	db := newSQLTestDB(t)
	result := &sqlCapture{rows: 1}
	captureSQL(t, db, "update", result)
	if err := NewAdminMysqlRepo(db).UpdatePWD(context.Background(), 1, "hash"); err != nil {
		t.Fatal(err)
	}
	_, where, ok := strings.Cut(result.sql, " WHERE ")
	if !ok || strings.Contains(where, "role") || !strings.Contains(where, "id = ?") {
		t.Fatalf("own-password change must allow super administrators: %s", result.sql)
	}
}
