package handler

import (
	appservice "blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	infrarepo "blog-server/infra/repo"
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestArticleImageHTTPTypesCacheAndETag(t *testing.T) {
	for _, format := range []string{"png", "jpg"} {
		t.Run(format, func(t *testing.T) {
			files := infrarepo.NewArticleImageFileRepo(t.TempDir())
			var encoded bytes.Buffer
			source := image.NewNRGBA(image.Rect(0, 0, 20, 10))
			contentType := "image/png"
			if format == "png" {
				if err := png.Encode(&encoded, source); err != nil {
					t.Fatal(err)
				}
			} else {
				contentType = "image/jpeg"
				if err := jpeg.Encode(&encoded, source, nil); err != nil {
					t.Fatal(err)
				}
			}
			name, err := files.Save(context.Background(), encoded.Bytes())
			if err != nil {
				t.Fatal(err)
			}
			h := NewArticleImageHandler(appservice.NewArticleImageAppService(nil, files))
			engine := gin.New()
			engine.GET("/images/:name", h.Image)
			get := func(name, etag string) *httptest.ResponseRecorder {
				r := httptest.NewRequest(http.MethodGet, "/images/"+name, nil)
				r.Header.Set("If-None-Match", etag)
				res := httptest.NewRecorder()
				engine.ServeHTTP(res, r)
				return res
			}
			res := get(name, "")
			if res.Code != 200 || !bytes.Equal(res.Body.Bytes(), encoded.Bytes()) || res.Header().Get("Content-Type") != contentType || res.Header().Get("X-Content-Type-Options") != "nosniff" || res.Header().Get("Cache-Control") != consts.ArticleImageCacheControl {
				t.Fatalf("image response: %v", res.Header())
			}
			etag := res.Header().Get("ETag")
			if etag != `"`+name+`"` {
				t.Fatal("unstable ETag")
			}
			for _, candidate := range []string{etag, "W/" + etag, `"other", ` + etag, "*"} {
				cached := get(name, candidate)
				if cached.Code != 304 || cached.Body.Len() != 0 {
					t.Fatal("conditional read failed")
				}
			}
			if uncached := get(name, `"different"`); uncached.Code != 200 {
				t.Fatal("incorrect cache hit")
			}
			missing := get("not-an-image", "")
			if missing.Header().Get("Cache-Control") != "no-store" || missing.Header().Get("Content-Type") == contentType {
				t.Fatal("invalid image cached")
			}
		})
	}
}

func articleImageMultipart(t *testing.T, data []byte, field string, extra bool) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, "../../untrusted.png")
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
	r := httptest.NewRequest(http.MethodPost, "/", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	return r
}

func TestArticleImageMultipartAndMissingMatchingIdentity(t *testing.T) {
	for _, tc := range []struct {
		name           string
		data           []byte
		field          string
		extra, wantErr bool
	}{
		{"file", []byte("image"), "file", false, false}, {"empty", nil, "file", false, true}, {"too-large", make([]byte, consts.ArticleImageMaxUploadBytes+1), "file", false, true},
		{"extra-target", []byte("image"), "file", true, true}, {"wrong-name", []byte("image"), "avatar", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = articleImageMultipart(t, tc.data, tc.field, tc.extra)
			_, err := readArticleImage(c)
			if (err != nil) != tc.wantErr {
				t.Fatalf("read result %v", err)
			}
		})
	}
	for _, tc := range []struct {
		handler gin.HandlerFunc
		other   string
	}{
		{NewArticleImageHandler(nil).UploadUser, consts.AdminIDKey}, {NewArticleImageHandler(nil).UploadAdmin, consts.UserIDKey},
	} {
		for _, otherPresent := range []bool{false, true} {
			engine := gin.New()
			engine.POST("/", func(c *gin.Context) {
				if otherPresent {
					c.Set(tc.other, uint64(1))
				}
				tc.handler(c)
			})
			res := httptest.NewRecorder()
			engine.ServeHTTP(res, articleImageMultipart(t, []byte("image"), "file", false))
			var body struct{ Code int }
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != common.LoginNotLoginErr.Code {
				t.Fatal("missing/mismatched identity accepted")
			}
		}
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(make([]byte, consts.ArticleImageMaxUploadBytes+consts.ArticleImageMultipartOverhead+1)))
	c.Request.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	if _, err := readArticleImage(c); err == nil {
		t.Fatal("malformed oversized multipart accepted")
	}
}
