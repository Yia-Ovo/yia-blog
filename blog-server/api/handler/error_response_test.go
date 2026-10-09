package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-server/common/consts"
	"github.com/gin-gonic/gin"
)

func TestBindingErrorsDoNotExposeInternalDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name   string
		handle gin.HandlerFunc
	}{
		{"register", NewUserHandler(nil).Register},
		{"article", NewArticleHandler(nil).CreateArticle},
		{"category", NewCategoryHandler(nil).CreateCategory},
		{"admin_password", NewAdminHandler(nil).UpdatePWD},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, payload := range []string{`"private-marker"`, `{"private-marker":`} {
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Set(consts.UserIDKey, uint64(1))
				c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
				c.Request.Header.Set("Content-Type", "application/json")
				tc.handle(c)
				var body map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if w.Code != 200 || len(body) != 4 || body["code"] != float64(400) || body["msg"] != "Param Error" || body["err_msg"] != "" {
					t.Fatalf("unexpected response: %s", w.Body.String())
				}
			}
		})
	}
}

func TestPublicMessageAndInternalIdentityError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name        string
		handle      gin.HandlerFunc
		setIdentity bool
		wantCode    float64
		wantMsg     string
	}{
		{"article_no_identity", NewArticleHandler(nil).CreateArticle, false, 11009, "请先登录"},
		{"admin_no_identity", NewAdminHandler(nil).GetInfo, false, 11009, "请先登录管理员账号"},
		{"user_bad_identity", NewUserHandler(nil).GetInfo, true, 500, "Internal Server Error"},
		{"admin_bad_identity", NewAdminHandler(nil).GetInfo, true, 500, "Internal Server Error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.setIdentity {
				c.Set(consts.UserIDKey, "private-type-detail")
				c.Set(consts.AdminIDKey, "private-type-detail")
				c.Set(consts.CustomerUserKey, struct{}{})
			}
			tc.handle(c)
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != http.StatusOK || len(body) != 4 || body["code"] != tc.wantCode || body["msg"] != tc.wantMsg || body["err_msg"] != "" {
				t.Fatalf("unexpected response: %s", w.Body.String())
			}
		})
	}
}
