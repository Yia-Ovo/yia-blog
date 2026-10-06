package repo

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/infra/model"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestArticlePageStateSQLMatchesFullListWithoutReadingContent(t *testing.T) {
	db := newSQLTestDB(t)
	var statements []sqlCapture
	if err := db.Callback().Query().After("gorm:query").Register("test:article_page_state", func(tx *gorm.DB) {
		statements = append(statements, sqlCapture{sql: tx.Statement.SQL.String(), args: append([]interface{}(nil), tx.Statement.Vars...)})
		switch result := tx.Statement.Dest.(type) {
		case *int64:
			*result = 11
			tx.RowsAffected = 1
			// DryRun 不会自动清理 SQL，模拟真实查询完成后生成独立的分页语句。
			tx.Statement.SQL.Reset()
			tx.Statement.Vars = nil
		case *[]uint64:
			*result = []uint64{9, 7}
			tx.RowsAffected = 2
		default:
			t.Fatalf("unexpected wide query destination %T", result)
		}
	}); err != nil {
		t.Fatal(err)
	}
	status, top, category, user := consts.ArticleStatusPublished, 0, uint64(0), uint64(9)
	state, err := NewArticleMysqlRepo(db).GetArticlePageState(context.Background(), &do.ArticleQuery{
		PageNum: 2, PageSize: 5, Title: "needle", CategoryID: &category, Status: &status, IsTop: &top, UserID: &user,
	})
	if err != nil || state == nil || state.Total != 11 || !reflect.DeepEqual(state.IDs, []uint64{9, 7}) || len(statements) != 2 {
		t.Fatalf("unexpected state: %+v err=%v statements=%+v", state, err, statements)
	}
	for _, statement := range statements {
		for _, condition := range []string{"title LIKE ?", "category_id = ?", "status = ?", "is_top = ?", "user_id = ? AND admin_id = ?", "`article`.`deleted_at` IS NULL"} {
			if !strings.Contains(statement.sql, condition) {
				t.Fatalf("page identity query lost scope %q: %s", condition, statement.sql)
			}
		}
		if len(statement.args) < 6 || !reflect.DeepEqual(statement.args[:6], []interface{}{"%needle%", category, status, top, user, 0}) {
			t.Fatalf("page identity query changed filter values: %v", statement.args)
		}
	}
	if !strings.HasPrefix(statements[0].sql, "SELECT count(*) FROM `article`") {
		t.Fatalf("first query must only count rows: %s", statements[0].sql)
	}
	selected, _, ok := strings.Cut(statements[1].sql, " WHERE ")
	if !ok || selected != "SELECT `id` FROM `article`" || !strings.Contains(statements[1].sql, "ORDER BY sort ASC, publish_at DESC, id DESC LIMIT ? OFFSET ?") || !reflect.DeepEqual(statements[1].args[6:], []interface{}{5, 5}) {
		t.Fatalf("narrow projection, stable ordering or pagination changed: %+v", statements[1])
	}
}

func TestArticlePageStateEmptyScopeSkipsIDs(t *testing.T) {
	db := newSQLTestDB(t)
	result := &sqlCapture{rows: 1}
	captureSQL(t, db, "query", result)
	state, err := NewArticleMysqlRepo(db).GetArticlePageState(context.Background(), &do.ArticleQuery{PageNum: 1, PageSize: 10})
	if err != nil || state == nil || state.Total != 0 || len(state.IDs) != 0 || result.calls != 1 {
		t.Fatalf("empty scope must return no identities: state=%+v err=%v calls=%d", state, err, result.calls)
	}
	for _, unwanted := range []string{"title LIKE", "category_id =", "status =", "is_top =", "user_id ="} {
		if strings.Contains(result.sql, unwanted) {
			t.Fatalf("nil filter became a value restriction: %s", result.sql)
		}
	}
}

func TestArticlePageStateReturnsCountAndIDsErrors(t *testing.T) {
	for _, failureAt := range []int{1, 2} {
		db := newSQLTestDB(t)
		calls := 0
		queryErr := errors.New("database offline")
		if err := db.Callback().Query().After("gorm:query").Register("test:page_state_error", func(tx *gorm.DB) {
			calls++
			if calls == failureAt {
				tx.AddError(queryErr)
				return
			}
			if count, ok := tx.Statement.Dest.(*int64); ok {
				*count = 1
				tx.RowsAffected = 1
				tx.Statement.SQL.Reset()
				tx.Statement.Vars = nil
			}
		}); err != nil {
			t.Fatal(err)
		}
		state, err := NewArticleMysqlRepo(db).GetArticlePageState(context.Background(), &do.ArticleQuery{PageNum: 1, PageSize: 10})
		if state != nil || !errors.Is(err, queryErr) || calls != failureAt {
			t.Fatalf("query error swallowed: state=%+v err=%v calls=%d failureAt=%d", state, err, calls, failureAt)
		}
	}
	if state, err := NewArticleMysqlRepo(nil).GetArticlePageState(context.Background(), nil); state != nil || !errors.Is(err, gorm.ErrInvalidData) {
		t.Fatalf("nil query should fail without touching database: state=%+v err=%v", state, err)
	}
}

func TestArticleExclusionSQLAppliesBeforeCountAndPaging(t *testing.T) {
	db := newSQLTestDB(t)
	q := &do.ArticleQuery{PageNum: 2, PageSize: 3, ExcludeIDs: []uint64{2, 3}}
	count := articleQueryScope(db, q).ToSQL(func(tx *gorm.DB) *gorm.DB { var n int64; return tx.Count(&n) })
	rows := articleQueryScope(db, q).ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Limit(3).Offset(3).Find(&[]model.ArticleModel{}) })
	for _, sql := range []string{count, rows} {
		if !strings.Contains(sql, "id NOT IN (2,3)") || !strings.Contains(sql, "deleted_at") {
			t.Fatalf("lost exclusion/visibility: %s", sql)
		}
	}
}
