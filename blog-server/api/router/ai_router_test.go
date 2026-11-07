package router_test

import (
	"blog-server/api/dto"
	appservice "blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/jwt"
	"blog-server/ioc"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAIEndpointsRequireAdminAuthentication(t *testing.T) {
	engine := publicTestEngine(&ioc.Container{AIApp: appservice.NewAIAppService(nil, nil, nil, nil, false, "", 50)})
	for _, path := range []string{"/overview", "/conversation?id=conversation-00001", "/imports", "/import?id=import-00000000001", "/send", "/memory", "/delete", "/import", "/workspace", "/job", "/job/publish"} {
		for _, ordinaryUser := range []bool{false, true} {
			method := "GET"
			if path == "/send" || path == "/memory" || path == "/delete" || path == "/import" || path == "/job" || path == "/job/publish" {
				method = "POST"
			}
			request := httptest.NewRequest(method, "/api/blog/v1/admin/ai"+path, strings.NewReader(`{}`))
			if ordinaryUser {
				request.Header.Set("Authorization", "Bearer ordinary-user-token")
			}
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if !strings.Contains(response.Body.String(), `"code":11009`) {
				t.Fatalf("unprotected AI path %s: %s", path, response.Body.String())
			}
		}
	}
}

func TestAIImportRoutesRejectAuthenticatedRegularAdmin(t *testing.T) {
	c, _ := managementHTTPContainer(t)
	// 若路由漏挂超管中间件，禁用 App 将返回 503，从而与预期拒绝码区分。
	c.AIApp = appservice.NewAIAppService(nil, nil, nil, nil, false, "", 50)
	engine := publicTestEngine(c)
	token, err := jwt.GenerateAdminToken(&dto.AdminBaseDTO{ID: 9})
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []struct{ method, path string }{{"GET", "/admin/ai/imports"}, {"GET", "/admin/ai/import?id=import-00000000001"}, {"POST", "/admin/ai/import"}, {"GET", "/admin/ai/workspace"}, {"POST", "/admin/ai/job"}, {"POST", "/admin/ai/job/publish"}} {
		w, code := managementRequest(t, engine, request.method, request.path, consts.AdminTokenKey, token, `{}`)
		if code != common.PermissionDeniedErr.Code {
			t.Fatalf("ordinary administrator passed super route: %s", w.Body.String())
		}
	}
}
