package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-server/config"

	"github.com/gin-gonic/gin"
)

func TestCORSPreflightAndRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	conf := &config.GlobalConfig{}
	conf.Gin.CORS = config.CORSConfig{AllowOrigins: []string{"http://localhost:3000"}, AllowMethods: []string{"GET", "POST"}, AllowHeaders: []string{"Content-Type", "Authorization", "Admin-Authorization"}, AllowCredentials: true, MaxAge: 3600}
	for _, tt := range []struct {
		name, method, origin, requestedMethod, headers string
		wantStatus, wantCalls                          int
		wantAllow                                      bool
	}{
		{"preflight", "OPTIONS", "http://localhost:3000", "POST", "content-type, ADMIN-authorization, authorization", 204, 0, true},
		{"rejected-origin", "OPTIONS", "https://other.example", "POST", "content-type", 403, 0, false},
		{"rejected-method", "OPTIONS", "http://localhost:3000", "DELETE", "", 403, 0, false},
		{"rejected-header", "OPTIONS", "http://localhost:3000", "POST", "X-Forbidden", 403, 0, false},
		{"actual", "POST", "http://localhost:3000", "", "", 200, 1, true},
		{"actual-rejected", "POST", "https://other.example", "", "", 403, 0, false},
		{"no-origin", "GET", "", "", "", 200, 1, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			engine := gin.New()
			engine.Use(Cors(conf))
			engine.Any("/", func(c *gin.Context) { calls++; c.Status(http.StatusOK) })
			req := httptest.NewRequest(tt.method, "/", nil)
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Access-Control-Request-Method", tt.requestedMethod)
			req.Header.Set("Access-Control-Request-Headers", tt.headers)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			if w.Code != tt.wantStatus || calls != tt.wantCalls {
				t.Fatalf("HTTP=%d calls=%d headers=%v", w.Code, calls, w.Header())
			}
			if !strings.Contains(strings.Join(w.Header().Values("Vary"), ","), "Origin") {
				t.Fatalf("response must vary even when Origin is absent: %v", w.Header())
			}
			if tt.wantAllow {
				if w.Header().Get("Access-Control-Allow-Origin") != tt.origin || w.Header().Get("Access-Control-Allow-Methods") != "GET, POST" || w.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Authorization, Admin-Authorization" || w.Header().Get("Access-Control-Allow-Credentials") != "true" {
					t.Fatalf("incomplete CORS response: %v", w.Header())
				}
				if !strings.Contains(strings.Join(w.Header().Values("Vary"), ","), "Origin") {
					t.Fatalf("missing Vary: %v", w.Header())
				}
				if tt.method == http.MethodOptions && w.Header().Get("Access-Control-Max-Age") != "3600" {
					t.Fatal("missing preflight cache TTL")
				}
			} else if w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Access-Control-Allow-Credentials") != "" {
				t.Fatalf("unexpected CORS grant: %v", w.Header())
			}
		})
	}
}
