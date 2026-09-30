package router_test

import (
	appservice "blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	infrarepo "blog-server/infra/repo"
	"blog-server/ioc"
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArticleImageRoutesRequireMatchingAuthentication(t *testing.T) {
	engine := publicTestEngine(&ioc.Container{})
	for _, tc := range []struct{ path, otherHeader string }{
		{"/api/blog/v1/article/uploadImage", consts.AdminTokenKey}, {"/api/blog/v1/article/adminUploadImage", consts.UserTokenKey},
	} {
		for _, token := range []string{"", "Bearer wrong-identity-token"} {
			r := httptest.NewRequest(http.MethodPost, tc.path, nil)
			r.Header.Set(tc.otherHeader, token)
			res := httptest.NewRecorder()
			engine.ServeHTTP(res, r)
			var body struct{ Code int }
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != common.LoginNotLoginErr.Code {
				t.Fatalf("unauthenticated/mismatched upload: %s", res.Body.String())
			}
		}
	}
}

func TestArticleImageRouteIsPublic(t *testing.T) {
	files := infrarepo.NewArticleImageFileRepo(t.TempDir())
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewNRGBA(image.Rect(0, 0, 20, 10))); err != nil {
		t.Fatal(err)
	}
	name, err := files.Save(context.Background(), data.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	engine := publicTestEngine(&ioc.Container{ArticleImageApp: appservice.NewArticleImageAppService(nil, files)})
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, httptest.NewRequest(http.MethodGet, consts.ArticleImageRoute+name, nil))
	if res.Code != 200 || !bytes.Equal(res.Body.Bytes(), data.Bytes()) || res.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("public image failed: %s", res.Body.String())
	}
}
