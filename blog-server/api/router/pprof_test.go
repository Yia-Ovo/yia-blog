package router_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"blog-server/api/router"
	"blog-server/config"
	"blog-server/ioc"

	"github.com/gin-gonic/gin"
)

func pprofEngine(server config.ServerConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	router.NewRouter(&config.GlobalConfig{Server: server}, &ioc.Container{}, func() error { return nil }).Register(r)
	return r
}

func TestPprofRegistrationBoundary(t *testing.T) {
	for _, tc := range []struct {
		name   string
		server config.ServerConfig
		want   int
	}{
		{"off", config.ServerConfig{Env: "dev", HttpHost: "127.0.0.1"}, 404},
		{"production", config.ServerConfig{EnablePprof: true, Env: "prod", HttpHost: "127.0.0.1"}, 404},
		{"all_interfaces", config.ServerConfig{EnablePprof: true, Env: "dev", HttpHost: "0.0.0.0"}, 404},
		{"empty_environment", config.ServerConfig{EnablePprof: true, HttpHost: "127.0.0.1"}, 404},
		{"local", config.ServerConfig{EnablePprof: true, Env: "local", HttpHost: "127.0.0.1"}, 200},
		{"ipv6", config.ServerConfig{EnablePprof: true, Env: "test", HttpHost: "::1"}, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := pprofEngine(tc.server)
			req := httptest.NewRequest(http.MethodGet, "http://localhost/debug/pprof/", nil)
			req.RemoteAddr = "127.0.0.1:1234"
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status=%d want=%d", w.Code, tc.want)
			}
		})
	}
}

func TestPprofRejectsNonLocalAndForwardedRequests(t *testing.T) {
	r := pprofEngine(config.ServerConfig{EnablePprof: true, Env: "dev", HttpHost: "127.0.0.1"})
	for _, tc := range []struct {
		name, peer, host, header, value string
		want                            int
	}{
		{"ipv4", "127.0.0.1:1234", "localhost", "", "", 200},
		{"ipv6", "[::1]:1234", "[::1]:8081", "", "", 200},
		{"remote", "192.0.2.1:1234", "localhost", "", "", 403},
		{"malformed_peer", "localhost", "localhost", "", "", 403},
		{"remote_host", "127.0.0.1:1234", "example.com", "", "", 403},
		{"spoofed_ip", "192.0.2.1:1234", "localhost", "X-Forwarded-For", "127.0.0.1", 403},
		{"local_proxy", "127.0.0.1:1234", "localhost", "X-Forwarded-For", "192.0.2.1", 403},
		{"empty_forwarded", "127.0.0.1:1234", "localhost", "Forwarded", "", 403},
		{"real_ip", "127.0.0.1:1234", "localhost", "X-Real-IP", "192.0.2.1", 403},
		{"origin", "127.0.0.1:1234", "localhost", "Origin", "https://example.com", 403},
		{"cross_site", "127.0.0.1:1234", "localhost", "Sec-Fetch-Site", "cross-site", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://localhost/debug/pprof/", nil)
			req.RemoteAddr, req.Host = tc.peer, tc.host
			if tc.header != "" {
				req.Header.Set(tc.header, tc.value)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status=%d want=%d", w.Code, tc.want)
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("missing no-store")
			}
		})
	}
}

func TestPprofServesRealProfiles(t *testing.T) {
	r := pprofEngine(config.ServerConfig{EnablePprof: true, Env: "dev", HttpHost: "127.0.0.1"})
	for _, tc := range []struct{ path, marker string }{
		{"/debug/pprof/", "Types of profiles available"},
		{"/debug/pprof/goroutine?debug=1", "goroutine profile"},
		{"/debug/pprof/heap?debug=1", "heap profile"},
		{"/debug/pprof/symbol", "num_symbols"},
		{"/debug/pprof/profile?seconds=1", ""},
	} {
		req := httptest.NewRequest(http.MethodGet, "http://localhost"+tc.path, nil)
		req.RemoteAddr = "127.0.0.1:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 200 || w.Body.Len() == 0 || !strings.Contains(w.Body.String(), tc.marker) {
			t.Fatalf("%s: status=%d length=%d", tc.path, w.Code, w.Body.Len())
		}
	}
	paths := map[string]bool{}
	for _, route := range r.Routes() {
		paths[route.Method+" "+route.Path] = true
	}
	for _, path := range []string{"trace", "cmdline", "allocs", "block", "mutex", "threadcreate"} {
		if !paths["GET /debug/pprof/"+path] {
			t.Fatalf("missing route %s", path)
		}
	}
}
