package repo

import (
	"context"
	"errors"
	"strings"
	"testing"

	"blog-server/infra/model"

	"gorm.io/gorm"
)

func TestArticleUpdateSQLProtectsOwnedFields(t *testing.T) {
	db := newSQLTestDB(t)
	result := &sqlCapture{rows: 1}
	captureSQL(t, db, "update", result)
	article := &model.ArticleModel{ID: 7, Title: "updated", Content: "body", UserID: 8, AdminID: 9, Views: 10, Likes: 11, Comments: 12}
	if err := NewArticleMysqlRepo(db).UpdateArticle(context.Background(), article); err != nil {
		t.Fatal(err)
	}
	set, where, ok := strings.Cut(result.sql, " WHERE ")
	if !ok || !strings.Contains(where, "id = ?") || !strings.Contains(where, "deleted_at") {
		t.Fatalf("update can affect the wrong or deleted article: %s", result.sql)
	}
	for _, forbidden := range []string{"user_id", "admin_id", "views", "likes", "comments", "create_at", "deleted_at"} {
		if strings.Contains(set, forbidden) {
			t.Fatalf("system column %q leaked into UPDATE: %s", forbidden, result.sql)
		}
	}
	for _, editable := range []string{"title", "content", "cover", "summary", "category_id", "tag_ids", "status", "sort", "is_top", "comment_status", "publish_at", "update_at"} {
		if !strings.Contains(set, editable) {
			t.Fatalf("editable field %q missing: %s", editable, result.sql)
		}
	}
	// 零值必须参与更新；使用 struct 更新会把关闭评论、取消置顶等操作忽略。
	if len(result.args) != 13 || result.args[1] != 0 || result.args[4] != 0 || result.args[6] != 0 || result.args[7] != 0 {
		t.Fatalf("zero-valued editable fields missing: %v", result.args)
	}
}

func TestArticleUpdateNoRowsNeverCreatesReplacement(t *testing.T) {
	db := newSQLTestDB(t)
	updates, creates := &sqlCapture{}, &sqlCapture{}
	captureSQL(t, db, "update", updates)
	captureSQL(t, db, "create", creates)
	if err := NewArticleMysqlRepo(db).UpdateArticle(context.Background(), &model.ArticleModel{ID: 7, Title: "deleted meanwhile"}); err != nil {
		t.Fatal(err)
	}
	if updates.calls != 1 || creates.calls != 0 {
		t.Fatalf("missing article was recreated: updates=%d creates=%d", updates.calls, creates.calls)
	}
}

func TestArticleUpdateRejectsMissingIdentity(t *testing.T) {
	r := NewArticleMysqlRepo(nil)
	for _, article := range []*model.ArticleModel{nil, {Title: "without ID"}} {
		if err := r.UpdateArticle(context.Background(), article); !errors.Is(err, gorm.ErrInvalidData) {
			t.Fatalf("invalid article returned %v", err)
		}
	}
}
