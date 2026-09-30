package router_test

import (
	"blog-server/common"
	"blog-server/ioc"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAvatarUploadRoutesRequireMatchingAuthentication(t *testing.T) {
	engine := publicTestEngine(&ioc.Container{})
	for _, path := range []string{"/api/blog/v1/admin/uploadAvatar", "/api/blog/v1/user/uploadAvatar"} {
		r := httptest.NewRequest(http.MethodPost, path, nil)
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, r)
		var body struct{ Code int }
		if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Code != common.LoginNotLoginErr.Code {
			t.Fatalf("unauthenticated upload accepted: %s", res.Body.String())
		}
	}
}
