package repo

import (
	"context"
	"strings"
	"testing"

	"blog-server/domain/do"
	"gorm.io/gorm"
)

func TestManagedArticleSQLScopesCountAndRowsByUser(t *testing.T) {
	db := newSQLTestDB(t)
	var statements []sqlCapture
	if err := db.Callback().Query().After("gorm:query").Register("test:managed_page", func(tx *gorm.DB) {
		statements = append(statements, sqlCapture{sql: tx.Statement.SQL.String(), args: append([]interface{}(nil), tx.Statement.Vars...)})
		if count, ok := tx.Statement.Dest.(*int64); ok {
			*count = 2
			tx.RowsAffected = 1
			tx.Statement.SQL.Reset()
			tx.Statement.Vars = nil
		}
	}); err != nil {
		t.Fatal(err)
	}
	userID := uint64(7)
	_, _, err := NewArticleMysqlRepo(db).ListArticleByPage(context.Background(), &do.ArticleQuery{PageNum: 1, PageSize: 10, UserID: &userID})
	if err != nil || len(statements) != 2 {
		t.Fatalf("missing queries: %+v %v", statements, err)
	}
	for _, s := range statements {
		if !strings.Contains(s.sql, "user_id = ? AND admin_id = ?") || !strings.Contains(s.sql, "deleted_at") || len(s.args) < 2 || s.args[0] != userID || s.args[1] != 0 {
			t.Fatalf("owner scope omitted from count or rows: %+v", s)
		}
	}
}
