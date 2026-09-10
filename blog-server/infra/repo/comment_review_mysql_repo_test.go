package repo

import (
	"context"
	"errors"
	"strings"
	"testing"

	"blog-server/domain/do"
	"gorm.io/gorm"
)

func TestCommentReviewSQLProtectsContentAndUpdatesOnlyStatus(t *testing.T) {
	db := newSQLTestDB(t)
	result := &sqlCapture{rows: 1}
	captureSQL(t, db, "update", result)
	rows, err := NewCommentMysqlRepo(db).ReviewComment(context.Background(), &do.CommentReview{CommentID: 3, ExpectedContent: "Text ", ExpectedStatus: 0, Status: 1})
	if err != nil || rows != 1 {
		t.Fatalf("review failed: rows=%d err=%v", rows, err)
	}
	set, where, found := strings.Cut(result.sql, " WHERE ")
	if !found || !strings.Contains(where, "id = ? AND status = ? AND BINARY content = BINARY ?") || !strings.Contains(where, "deleted_at") {
		t.Fatalf("stale/soft-deleted content can be reviewed: %s", result.sql)
	}
	for _, field := range []string{"content", "likes", "user_id", "admin_id", "article_id", "create_at", "deleted_at"} {
		if strings.Contains(set, field) {
			t.Fatalf("review overwrites %s: %s", field, result.sql)
		}
	}
	if len(result.args) != 5 || result.args[0] != 1 || result.args[2] != uint64(3) || result.args[3] != 0 || result.args[4] != "Text " {
		t.Fatalf("review preconditions changed: %v", result.args)
	}
}

func TestCommentReviewSQLReturnsZeroRowsAndDatabaseErrors(t *testing.T) {
	for _, dbErr := range []error{nil, errors.New("database unavailable")} {
		db := newSQLTestDB(t)
		captureSQL(t, db, "update", &sqlCapture{err: dbErr})
		rows, err := NewCommentMysqlRepo(db).ReviewComment(context.Background(), &do.CommentReview{CommentID: 3, ExpectedContent: "old", ExpectedStatus: 0, Status: 1})
		if rows != 0 || !errors.Is(err, dbErr) {
			t.Fatalf("lost conditional result: rows=%d err=%v", rows, err)
		}
	}
	if _, err := NewCommentMysqlRepo(nil).ReviewComment(context.Background(), nil); !errors.Is(err, gorm.ErrInvalidData) {
		t.Fatalf("invalid review reached DB: %v", err)
	}
}
