package repo

import (
	"blog-server/common/consts"
	"blog-server/infra/model"
	"context"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestArticlePublicStateSQLExcludesContentAndSoftDeletedRows(t *testing.T) {
	db := newSQLTestDB(t)
	result := &sqlCapture{rows: 1}
	captureSQL(t, db, "query", result)
	if err := db.Callback().Query().After("test:capture").Register("test:public_state_row", func(tx *gorm.DB) {
		article := tx.Statement.Dest.(*model.ArticleModel)
		article.ID = 7
		article.Status = consts.ArticleStatusPublished
	}); err != nil {
		t.Fatal(err)
	}
	state, err := NewArticleMysqlRepo(db).GetArticlePublicState(context.Background(), 7)
	if err != nil || state == nil || state.ID != 7 || state.Status != consts.ArticleStatusPublished {
		t.Fatalf("unexpected state: %+v err=%v", state, err)
	}
	selected, where, ok := strings.Cut(result.sql, " WHERE ")
	if !ok || selected != "SELECT `id`,`status` FROM `article`" {
		t.Fatalf("visibility query must not read content or other wide columns: %s", result.sql)
	}
	if !strings.Contains(where, "`article`.`id` = ?") || !strings.Contains(where, "`article`.`deleted_at` IS NULL") || len(result.args) != 2 || result.args[0] != uint64(7) {
		t.Fatalf("visibility query lost identity or soft-delete constraints: %s args=%v", result.sql, result.args)
	}
}

func TestArticlePublicStateDistinguishesMissingAndDatabaseFailure(t *testing.T) {
	databaseErr := errors.New("database offline")
	for _, tc := range []struct {
		name     string
		queryErr error
		want     error
	}{
		{name: "missing or soft deleted", queryErr: gorm.ErrRecordNotFound},
		{name: "database failure", queryErr: databaseErr, want: databaseErr},
		{name: "cancelled", queryErr: context.Canceled, want: context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newSQLTestDB(t)
			captureSQL(t, db, "query", &sqlCapture{err: tc.queryErr})
			state, err := NewArticleMysqlRepo(db).GetArticlePublicState(context.Background(), 7)
			if state != nil || !errors.Is(err, tc.want) {
				t.Fatalf("missing and unavailable database must remain distinct: state=%+v err=%v want=%v", state, err, tc.want)
			}
		})
	}
}
