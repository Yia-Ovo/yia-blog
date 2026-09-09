package repo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-server/api/dto"
	appservice "blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	domainrepo "blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"

	"github.com/gin-gonic/gin"
)

func insertedColumn(t *testing.T, captured *sqlCapture, column string) interface{} {
	t.Helper()
	_, rest, ok := strings.Cut(captured.sql, "(")
	if !ok {
		t.Fatalf("missing INSERT: %s", captured.sql)
	}
	columns, _, _ := strings.Cut(rest, ")")
	for i, name := range strings.Split(columns, ",") {
		if strings.Trim(name, " `") == column && i < len(captured.args) {
			return captured.args[i]
		}
	}
	t.Fatalf("column %s absent: %s %v", column, captured.sql, captured.args)
	return nil
}

func TestCreateArticleAndCategoryPreserveZeroAndDefaults(t *testing.T) {
	for _, tt := range []struct {
		name, field string
		want        int
	}{
		{"omitted", "", 1}, {"zero", ":0", 0}, {"one", ":1", 1},
	} {
		t.Run("article/"+tt.name, func(t *testing.T) {
			db := newSQLTestDB(t)
			captured := &sqlCapture{rows: 1}
			captureSQL(t, db, "create", captured)
			payload := "{\"title\":\"article\",\"content\":\"body\""
			if tt.field != "" {
				payload += ",\"comment_status\"" + tt.field
			}
			var req dto.CreateArticleReq
			if err := json.Unmarshal([]byte(payload+"}"), &req); err != nil {
				t.Fatal(err)
			}
			app := appservice.NewArticleAppService(domainservice.NewArticleDomainService(NewArticleMysqlRepo(db)), nil, nil, nil)
			if err := app.CreateArticle(context.Background(), 7, &req); err != nil {
				t.Fatal(err)
			}
			if got := insertedColumn(t, captured, "comment_status"); got != tt.want {
				t.Fatalf("comment_status=%v want=%d", got, tt.want)
			}
		})
		t.Run("category/"+tt.name, func(t *testing.T) {
			db := newSQLTestDB(t)
			captured := &sqlCapture{rows: 1}
			captureSQL(t, db, "create", captured)
			payload := "{\"name\":\"tech\""
			if tt.field != "" {
				payload += ",\"status\"" + tt.field
			}
			var req dto.CreateCategoryReq
			if err := json.Unmarshal([]byte(payload+"}"), &req); err != nil {
				t.Fatal(err)
			}
			app := appservice.NewCategoryAppService(domainservice.NewCategoryDomainService(NewCategoryMysqlRepo(db)), nil)
			if err := app.CreateCategory(context.Background(), &req); err != nil {
				t.Fatal(err)
			}
			if got := insertedColumn(t, captured, "status"); got != tt.want {
				t.Fatalf("category status=%v want=%d", got, tt.want)
			}
		})
	}
}

type adminCreateSQLRepo struct{ domainrepo.AdminMysqlRepo }

func (r *adminCreateSQLRepo) FindAdminByName(context.Context, string) (*model.AdminModel, error) {
	return nil, nil
}

func TestCreateAdminPreservesExplicitRole(t *testing.T) {
	for _, role := range []int{consts.AdminRoleSuper, consts.AdminRoleNormal} {
		db := newSQLTestDB(t)
		captured := &sqlCapture{rows: 1}
		captureSQL(t, db, "create", captured)
		r := &adminCreateSQLRepo{AdminMysqlRepo: NewAdminMysqlRepo(db)}
		admin, err := domainservice.NewAdminDomainService(r).CreateAdmin(context.Background(), "admin", "Example123", role)
		if err != nil {
			t.Fatal(err)
		}
		if got := insertedColumn(t, captured, "role"); got != role || admin.Role != role {
			t.Fatalf("role changed: SQL=%v returned=%d wanted=%d", got, admin.Role, role)
		}
	}
}

func TestCreateRejectsInvalidStates(t *testing.T) {
	bad := 2
	if err := domainservice.NewArticleDomainService(nil).CreateArticle(context.Background(), &do.ArticleCreate{Title: "article", Content: "body", CommentStatus: &bad}); !errors.Is(err, common.ParamErr.WithMsg("评论开关不合法")) {
		t.Fatalf("invalid comment status: %v", err)
	}
	if err := domainservice.NewCategoryDomainService(nil).CreateCategory(context.Background(), &do.CategoryCreate{Name: "tech", Status: &bad}); err == nil {
		t.Fatal("invalid category status accepted")
	}
	if _, err := domainservice.NewAdminDomainService(nil).CreateAdmin(context.Background(), "admin", "Example123", bad); !errors.Is(err, common.AdminRoleInvalidErr) {
		t.Fatalf("invalid role: %v", err)
	}
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{\"admin_name\":\"admin\",\"password\":\"Example123\"}"))
	var req dto.AdminCreateReq
	if err := ctx.ShouldBindJSON(&req); err == nil {
		t.Fatal("missing required admin role accepted")
	}
}
