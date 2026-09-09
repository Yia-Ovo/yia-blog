package repo

import (
	"context"
	"strings"
	"testing"

	"blog-server/common/consts"
	"blog-server/domain/do"

	"gorm.io/gorm"
)

func TestPublicCommentSQLFiltersArticlesInCountAndRows(t *testing.T) {
	db := newSQLTestDB(t)
	var statements []string
	if err := db.Callback().Query().After("gorm:query").Register("test:comment_scope", func(tx *gorm.DB) {
		statements = append(statements, tx.Statement.SQL.String())
		if count, ok := tx.Statement.Dest.(*int64); ok {
			*count = 1
			tx.RowsAffected = 1
			tx.Statement.SQL.Reset()
			tx.Statement.Vars = nil
		}
	}); err != nil {
		t.Fatal(err)
	}
	passed := consts.CommentStatusPassed
	_, _, err := NewCommentMysqlRepo(db).ListCommentByPage(context.Background(), &do.CommentQuery{Status: &passed, PublishedArticleOnly: true, Page: 1, Size: 10})
	if err != nil || len(statements) != 2 {
		t.Fatalf("query failed: %v %v", statements, err)
	}
	for _, sql := range statements {
		for _, condition := range []string{"EXISTS (SELECT 1 FROM `article`", "`article`.`id` = `comment`.`article_id`", "`article`.`status` = ?", "`article`.`deleted_at` IS NULL", "`comment`.`deleted_at` IS NULL", "status = ?"} {
			if !strings.Contains(sql, condition) {
				t.Fatalf("missing %q: %s", condition, sql)
			}
		}
	}
}
