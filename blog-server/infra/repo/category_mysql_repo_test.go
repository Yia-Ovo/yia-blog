package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"blog-server/domain/do"
	domainrepo "blog-server/domain/repo"
	"blog-server/infra/model"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func TestCategorySQLPreservesOmittedFieldsAndExplicitZero(t *testing.T) {
	zero := 0
	for _, tc := range []struct {
		name   string
		update *do.CategoryUpdate
		zero   bool
	}{
		{"omitted fields", &do.CategoryUpdate{ID: 7, Name: "new"}, false},
		{"explicit zero", &do.CategoryUpdate{ID: 7, Name: "new", Sort: &zero, Status: &zero}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newSQLTestDB(t)
			result := &sqlCapture{rows: 1}
			captureSQL(t, db, "update", result)
			if err := NewCategoryMysqlRepo(db).UpdateCategory(context.Background(), tc.update); err != nil {
				t.Fatal(err)
			}
			set, where, ok := strings.Cut(result.sql, " WHERE ")
			if !ok || strings.Contains(set, "create_at") || strings.Contains(set, "deleted_at") ||
				!strings.Contains(where, "deleted_at") || !strings.Contains(where, "id = ?") {
				t.Fatalf("category identity/timestamp/soft-delete protection missing: %s", result.sql)
			}
			if strings.Contains(set, "sort") != tc.zero || strings.Contains(set, "status") != tc.zero {
				t.Fatalf("omitted fields leaked into SQL or explicit fields disappeared: %s", result.sql)
			}
			if tc.zero && (len(result.args) != 4 || result.args[1] != 0 || result.args[2] != 0) {
				t.Fatalf("explicit zero was dropped: %v", result.args)
			}
			if !tc.zero && (len(result.args) != 2 || result.args[0] != "new") {
				t.Fatalf("unexpected arguments for name-only update: %v", result.args)
			}
		})
	}
}

func TestCategoryDuplicateKeyTranslation(t *testing.T) {
	dbErr := errors.New("database unavailable")
	for _, operation := range []string{"create", "update"} {
		for _, tc := range []struct {
			name string
			err  error
			want error
		}{
			{"mysql", &mysqldriver.MySQLError{Number: 1062, Message: "duplicate name"}, domainrepo.ErrDuplicateKey},
			{"wrapped mysql", fmt.Errorf("driver: %w", &mysqldriver.MySQLError{Number: 1062}), domainrepo.ErrDuplicateKey},
			{"translated gorm", gorm.ErrDuplicatedKey, domainrepo.ErrDuplicateKey},
			{"database failure", dbErr, dbErr},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				db := newSQLTestDB(t)
				captureSQL(t, db, operation, &sqlCapture{err: tc.err})
				r := NewCategoryMysqlRepo(db)
				var err error
				if operation == "create" {
					err = r.CreateCategory(context.Background(), &model.CategoryModel{Name: "tech", Status: 1})
				} else {
					err = r.UpdateCategory(context.Background(), &do.CategoryUpdate{ID: 1, Name: "tech"})
				}
				if !errors.Is(err, tc.want) {
					t.Fatalf("returned %v, want %v", err, tc.want)
				}
			})
		}
	}
}

func TestCategoryWrappedNotFoundIsMissing(t *testing.T) {
	db := newSQLTestDB(t)
	captureSQL(t, db, "query", &sqlCapture{err: fmt.Errorf("query: %w", gorm.ErrRecordNotFound)})
	category, err := NewCategoryMysqlRepo(db).GetCategoryById(context.Background(), 99)
	if err != nil || category != nil {
		t.Fatalf("wrapped not-found = %+v, %v; want nil, nil", category, err)
	}
}

func TestCategoryDeleteUsesSoftDelete(t *testing.T) {
	db := newSQLTestDB(t)
	result := &sqlCapture{rows: 1}
	captureSQL(t, db, "delete", result)
	if err := NewCategoryMysqlRepo(db).DeleteCategory(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.sql, "UPDATE") || !strings.Contains(result.sql, "category") || !strings.Contains(result.sql, "deleted_at") {
		t.Fatalf("category delete must update the soft-delete marker: %s", result.sql)
	}
}
