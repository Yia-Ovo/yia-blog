package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gingzip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

type accessTrackingBody struct {
	io.Reader
	reads  int
	closed bool
}

func (b *accessTrackingBody) Read(p []byte) (int, error) { b.reads++; return b.Reader.Read(p) }
func (b *accessTrackingBody) Close() error               { b.closed = true; return nil }

func TestAccessLogLeavesBodyForHandlerAndDoesNotReplaceWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(AccessLogMiddleware(nil))
	body := &accessTrackingBody{Reader: strings.NewReader(`{"content":"private text"}`)}
	engine.POST("/body", func(c *gin.Context) {
		if body.reads != 0 || body.closed || c.Request.Body != body {
			t.Error("middleware consumed, closed or replaced request body")
		}
		var input struct {
			Content string `json:"content"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.Content != "private text" {
			t.Fatalf("handler lost body: %v %+v", err, input)
		}
		c.String(http.StatusAccepted, "preserved")
	})
	req := httptest.NewRequest(http.MethodPost, "/body", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = body
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted || w.Body.String() != "preserved" {
		t.Fatalf("response changed: %d %s", w.Code, w.Body.String())
	}
}

func TestAccessLogPreservesGzipAndNilBodyRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(AccessLogMiddleware(nil), gingzip.Gzip(gingzip.DefaultCompression))
	plain := strings.Repeat("博客内容", 1000)
	engine.GET("/gzip", func(c *gin.Context) { c.String(http.StatusOK, plain) })
	req := httptest.NewRequest(http.MethodGet, "/gzip", nil)
	req.Body = nil
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("gzip response changed: status=%d headers=%v", w.Code, w.Header())
	}
	reader, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil || string(decoded) != plain {
		t.Fatalf("gzip body corrupted: %v", err)
	}
}
