package handler

import (
	"blog-server/app/service"
	"blog-server/assets"
	"blog-server/common/consts"
	domainservice "blog-server/domain/service"
	infrarepo "blog-server/infra/repo"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAppearancePublicMetadataAndImage(t *testing.T) {
	h := NewAppearanceHandler(service.NewAppearanceAppService(domainservice.NewAppearanceDomainService(infrarepo.NewAppearanceFileRepo(assets.DailyArtwork))))
	engine := gin.New()
	engine.GET("/api/blog/v1/appearance/today", h.Today)
	engine.GET("/api/blog/v1/appearance/images/:id", h.Image)
	get := func(url, etag string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, req)
		return res
	}
	res := get("/api/blog/v1/appearance/today", "")
	var body struct {
		Code int
		Data struct {
			Date         string
			ImageURL     string    `json:"image_url"`
			NextChangeAt time.Time `json:"next_change_at"`
			ServerTime   time.Time `json:"server_time"`
		}
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if res.Code != 200 || body.Code != 200 || res.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("metadata: %s", res.Body.String())
	}
	if !strings.HasPrefix(body.Data.ImageURL, consts.AppearanceImageRoute) || body.Data.NextChangeAt.Sub(body.Data.ServerTime) <= 0 || body.Data.NextChangeAt.Sub(body.Data.ServerTime) > 24*time.Hour {
		t.Fatalf("invalid metadata: %+v", body.Data)
	}
	image := get(body.Data.ImageURL, "")
	if image.Code != 200 || image.Header().Get("Content-Type") != "image/jpeg" || image.Body.Len() == 0 || image.Header().Get("Cache-Control") != consts.AppearanceImageCacheControl {
		t.Fatalf("image headers: %v", image.Header())
	}
	if image.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing image type guard")
	}
	cached := get(body.Data.ImageURL, image.Header().Get("ETag"))
	if cached.Code != 304 || cached.Body.Len() != 0 {
		t.Fatal("conditional image request failed")
	}
	invalid := get(consts.AppearanceImageRoute+"not-in-catalog.jpg", "")
	var failure struct{ Code int }
	if err := json.Unmarshal(invalid.Body.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if invalid.Code != 200 || failure.Code != 400 || invalid.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("invalid image: %s", invalid.Body.String())
	}
}
