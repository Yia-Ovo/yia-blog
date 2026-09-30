package handler

import (
	appservice "blog-server/app/service"
	"blog-server/common/consts"
	infrarepo "blog-server/infra/repo"
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAvatarPublicImageHeadersAndConditionalRead(t *testing.T) {
	files := infrarepo.NewAvatarFileRepo(t.TempDir())
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, image.NewNRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	name, err := files.Save(context.Background(), pngBytes.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	h := NewAvatarHandler(appservice.NewAvatarAppService(nil, files, nil, nil))
	engine := gin.New()
	engine.GET("/image/:name", h.Image)
	get := func(value, etag string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/image/"+value, nil)
		r.Header.Set("If-None-Match", etag)
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, r)
		return res
	}
	res := get(name, "")
	if res.Code != 200 || !bytes.Equal(res.Body.Bytes(), pngBytes.Bytes()) || res.Header().Get("Content-Type") != "image/png" || res.Header().Get("X-Content-Type-Options") != "nosniff" || res.Header().Get("Cache-Control") != consts.AvatarImageCacheControl {
		t.Fatalf("image response: %v", res.Header())
	}
	cached := get(name, res.Header().Get("ETag"))
	if cached.Code != 304 || cached.Body.Len() != 0 {
		t.Fatal("conditional request failed")
	}
	missing := get("not-an-image", "")
	if missing.Header().Get("Cache-Control") != "no-store" || missing.Header().Get("Content-Type") == "image/png" {
		t.Fatal("invalid image cached or served as PNG")
	}
}

func avatarMultipartRequest(t *testing.T, data []byte, extra bool) *http.Request {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	part, err := writer.CreateFormFile("file", "portrait.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if extra {
		if err := writer.WriteField("id", "another-user"); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/", &buffer)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	return r
}

func TestAvatarMultipartBoundariesAndMissingIdentity(t *testing.T) {
	for _, tc := range []struct {
		name           string
		data           []byte
		extra, wantErr bool
	}{
		{"file", []byte("opaque bytes"), false, false}, {"empty", nil, false, true}, {"too-large", make([]byte, consts.AvatarMaxUploadBytes+1), false, true}, {"extra-target", []byte("image"), true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = avatarMultipartRequest(t, tc.data, tc.extra)
			data, err := readAvatar(c)
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected read: %d %v", len(data), err)
			}
		})
	}
	for _, handler := range []gin.HandlerFunc{NewAvatarHandler(nil).UploadAdmin, NewAvatarHandler(nil).UploadUser} {
		engine := gin.New()
		engine.POST("/", handler)
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, avatarMultipartRequest(t, []byte("image"), false))
		var body struct{ Code int }
		if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Code == 200 {
			t.Fatal("missing identity accepted")
		}
	}
	// multipart 请求头过大也必须被总请求上限拦截。
	res := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(res)
	c.Request = httptest.NewRequest(http.MethodPost, "/", io.LimitReader(bytes.NewReader(make([]byte, consts.AvatarMaxUploadBytes+consts.AvatarMultipartOverhead+1)), consts.AvatarMaxUploadBytes+consts.AvatarMultipartOverhead+1))
	c.Request.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	if _, err := readAvatar(c); err == nil {
		t.Fatal("malformed multipart accepted")
	}
}
