package repo

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"
	"blog-server/infra/model"

	"gorm.io/gorm"
)

func TestAdminRecoverySQLOnlyReplacesPasswordAndVersion(t *testing.T) {
	for _, rows := range []int64{0, 1, 2} {
		db := newSQLTestDB(t)
		result := &sqlCapture{rows: rows}
		captureSQL(t, db, "update", result)
		expected := &domainrepo.AdminRecoveryAccount{ID: 7, Name: "super_admin", TokenVersion: 19, PasswordHash: "old-hash"}
		err := NewAdminRecoveryMysqlRepo(db).ReplacePassword(context.Background(), expected, "new-hash")
		if rows == 1 && err != nil || rows != 1 && !errors.Is(err, domainrepo.ErrAdminRecoveryConflict) {
			t.Fatalf("rows=%d returned %v", rows, err)
		}
		_, tail, found := strings.Cut(result.sql, " SET ")
		set, where, whereFound := strings.Cut(tail, " WHERE ")
		if !found || !whereFound || strings.Count(set, ",") != 2 {
			t.Fatalf("expected exactly three updated columns: %s", result.sql)
		}
		for _, column := range []string{"`password`", "`token_version`", "`update_at`"} {
			if !strings.Contains(set, column) {
				t.Fatalf("missing permitted updated column %s", column)
			}
		}
		for _, guard := range []string{"id = ?", "admin_name = ?", "role = ?", "status = ?", "deleted_at IS NULL", "token_version = ?", "password = ?"} {
			if !strings.Contains(where, guard) {
				t.Fatalf("missing recovery guard %s: %s", guard, result.sql)
			}
		}
		if !strings.Contains(set, "token_version + ?") || len(result.args) != 9 {
			t.Fatalf("missing atomic version increment or unexpected argument count: %s", result.sql)
		}
		want := []interface{}{uint64(7), "super_admin", consts.AdminRoleSuper, consts.AdminStatusEnabled, uint64(19), "old-hash"}
		if !reflect.DeepEqual(result.args[3:], want) || result.args[0] != "new-hash" || result.args[1] != 1 {
			t.Fatal("conditional update did not preserve expected identity and credential version")
		}
	}
}

func TestAdminRecoverySQLPreservesDatabaseFailure(t *testing.T) {
	db := newSQLTestDB(t)
	dbErr := errors.New("database unavailable")
	result := &sqlCapture{err: dbErr}
	captureSQL(t, db, "update", result)
	err := NewAdminRecoveryMysqlRepo(db).ReplacePassword(context.Background(), &domainrepo.AdminRecoveryAccount{ID: 7}, "new-hash")
	if !errors.Is(err, dbErr) {
		t.Fatalf("database failure was hidden: %v", err)
	}
	for _, invalid := range []*domainrepo.AdminRecoveryAccount{nil, {ID: 0}} {
		calls := result.calls
		if err := NewAdminRecoveryMysqlRepo(db).ReplacePassword(context.Background(), invalid, "new-hash"); !errors.Is(err, domainrepo.ErrAdminRecoveryConflict) || result.calls != calls {
			t.Fatal("invalid target reached a database write")
		}
	}
}

func TestAdminRecoveryLookupUsesLimitedColumnsAndExcludesDeletedAccounts(t *testing.T) {
	for _, lookupErr := range []error{nil, gorm.ErrRecordNotFound, errors.New("database unavailable")} {
		db := newSQLTestDB(t)
		result := &sqlCapture{rows: 1, err: lookupErr}
		captureSQL(t, db, "query", result)
		if err := db.Callback().Query().After("test:capture").Register("test:recovery-row", func(tx *gorm.DB) {
			if tx.Error == nil {
				*tx.Statement.Dest.(*model.AdminModel) = model.AdminModel{ID: 7, AdminName: "super_admin", Role: 0, Status: 1, Password: "old-hash", TokenVersion: 19}
			}
		}); err != nil {
			t.Fatal(err)
		}
		account, err := NewAdminRecoveryMysqlRepo(db).FindByName(context.Background(), "super_admin")
		switch {
		case errors.Is(lookupErr, gorm.ErrRecordNotFound):
			if account != nil || err != nil {
				t.Fatal("missing/deleted account must return nil without error")
			}
		case lookupErr != nil:
			if account != nil || !errors.Is(err, lookupErr) {
				t.Fatal("lookup failure was hidden")
			}
		default:
			if err != nil || account == nil || account.ID != 7 || account.Name != "super_admin" || account.TokenVersion != 19 || account.PasswordHash != "old-hash" {
				t.Fatal("recovery snapshot lost identity or credential fields")
			}
		}
		projection, _, _ := strings.Cut(result.sql, " FROM ")
		if strings.Contains(projection, "*") || strings.Count(projection, ",") != 5 {
			t.Fatalf("recovery query fetched unexpected columns: %s", result.sql)
		}
		for _, column := range []string{"`id`", "`admin_name`", "`role`", "`status`", "`password`", "`token_version`"} {
			if !strings.Contains(projection, column) {
				t.Fatalf("missing recovery snapshot column %s", column)
			}
		}
		if !strings.Contains(result.sql, "admin_name = ? AND deleted_at IS NULL") || len(result.args) != 2 || result.args[0] != "super_admin" {
			t.Fatalf("recovery query lost exact account/soft-delete guards: %s", result.sql)
		}
	}
}
