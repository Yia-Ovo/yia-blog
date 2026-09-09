package repo

import (
	"context"
	"errors"
	"strings"
	"testing"

	"blog-server/domain/do"
	domainrepo "blog-server/domain/repo"

	"gorm.io/gorm"
)

func TestUserProfileUpdateOnlyWritesProvidedColumns(t *testing.T) {
	name, empty, unknown := "new-name", "", 0
	for _, tt := range []struct {
		name    string
		update  *do.UserInfoUpdate
		columns []string
		values  []interface{}
	}{
		{"nickname", &do.UserInfoUpdate{NickName: &name}, []string{"nick_name"}, []interface{}{name}},
		{"clear", &do.UserInfoUpdate{Avatar: &empty, Email: &empty, Gender: &unknown, Signature: &empty}, []string{"avatar", "email", "gender", "signature"}, []interface{}{"", "", 0, ""}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := newSQLTestDB(t)
			captured := &sqlCapture{rows: 1}
			captureSQL(t, db, "update", captured)
			if err := NewUserMysqlRepo(db).UpdateUserInfo(context.Background(), 7, tt.update); err != nil {
				t.Fatal(err)
			}
			set, where, ok := strings.Cut(captured.sql, " WHERE ")
			if !ok || !strings.Contains(where, "id = ?") || !strings.Contains(where, "deleted_at") {
				t.Fatalf("missing identity/deletion scope: %s", captured.sql)
			}
			for _, protected := range []string{"status", "password", "token_version", "last_login_at", "create_at"} {
				if strings.Contains(set, protected) {
					t.Fatalf("account column %q overwritten: %s", protected, captured.sql)
				}
			}
			if len(captured.args) != len(tt.columns)+2 {
				t.Fatalf("unexpected columns or lost zero values: %s %v", captured.sql, captured.args)
			}
			for i, column := range tt.columns {
				if !strings.Contains(set, "`"+column+"`=?") || captured.args[i] != tt.values[i] {
					t.Fatalf("provided column %q lost: %s %v", column, captured.sql, captured.args)
				}
			}
		})
	}
}

func TestUserProfileEmptyUpdateDoesNotWrite(t *testing.T) {
	db := newSQLTestDB(t)
	captured := &sqlCapture{}
	captureSQL(t, db, "update", captured)
	if err := NewUserMysqlRepo(db).UpdateUserInfo(context.Background(), 7, &do.UserInfoUpdate{}); err != nil || captured.calls != 0 {
		t.Fatalf("empty update wrote data: calls=%d err=%v", captured.calls, err)
	}
}

func TestUserProfileZeroRowsDistinguishesRetryFromDeletion(t *testing.T) {
	for _, exists := range []bool{true, false} {
		db := newSQLTestDB(t)
		captureSQL(t, db, "update", &sqlCapture{})
		if err := db.Callback().Query().After("gorm:query").Register("test:user_exists", func(tx *gorm.DB) {
			if exists {
				*tx.Statement.Dest.(*int64) = 1
			}
			tx.RowsAffected = 1
		}); err != nil {
			t.Fatal(err)
		}
		name := "same"
		err := NewUserMysqlRepo(db).UpdateUserInfo(context.Background(), 7, &do.UserInfoUpdate{NickName: &name})
		if (exists && err != nil) || (!exists && !errors.Is(err, domainrepo.ErrNoRowsAffected)) {
			t.Fatalf("wrong zero-row result: exists=%v err=%v", exists, err)
		}
	}
}
