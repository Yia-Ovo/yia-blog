package handler

import (
	"blog-server/app/service"
	domainservice "blog-server/domain/service"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWeatherClientIPTrustBoundary(t *testing.T) {
	for _, tt := range []struct{ remote, header, want string }{
		{"8.8.8.8:1234", "1.1.1.1", "8.8.8.8"},
		{"127.0.0.1:1234", "8.8.8.8", "8.8.8.8"},
		{"[::1]:1234", "1.1.1.1, 8.8.8.8", "8.8.8.8"},
		{"127.0.0.1:1234", "8.8.8.8, 127.0.0.1", "8.8.8.8"},
		{"127.0.0.1:1234", "8.8.8.8, 10.0.0.1", "10.0.0.1"},
		{"127.0.0.1:1234", "invalid", ""},
		{"127.0.0.1:1234", "", "127.0.0.1"},
		{"bad", "8.8.8.8", ""},
		{"127.0.0.1:1234", strings.Repeat("8.8.8.8,", 200), ""},
	} {
		if got := weatherClientIP(tt.remote, tt.header); got != tt.want {
			t.Errorf("%q/%q: got %q want %q", tt.remote, tt.header, got, tt.want)
		}
	}
}

func TestWeatherLocalHTTPDegradesWithoutUpstreamAndDoesNotCacheShared(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	app := service.NewWeatherAppService(domainservice.NewWeatherDomainService(), nil)
	engine.GET("/weather/current", NewWeatherHandler(app).Current)
	for _, target := range []string{"/weather/current", "/weather/current?city=invalid"} {
		req := httptest.NewRequest("GET", target, nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code != 200 || rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("wrong HTTP semantics: %d %v", rec.Code, rec.Header())
		}
		if target == "/weather/current" && !strings.Contains(rec.Body.String(), "\"reason\":\"location_unavailable\"") {
			t.Fatal(rec.Body.String())
		}
		if target != "/weather/current" && !strings.Contains(rec.Body.String(), "\"code\":400") {
			t.Fatal(rec.Body.String())
		}
	}
}
