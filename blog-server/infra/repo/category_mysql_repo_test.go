package repo

import (
	"context"
	"strings"
	"testing"

	"blog-server/domain/do"
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
