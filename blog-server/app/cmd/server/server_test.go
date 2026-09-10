package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"blog-server/config"
	"github.com/goccy/go-yaml"
)

func TestShutdownUsesDurationAndClosesActiveConnectionsOnTimeout(t *testing.T) {
	var cfg config.GlobalConfig
	if err := yaml.Unmarshal([]byte("server:\n  shutdown_timeout: 30ms\n"), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Server.ShutdownTimeout != 30*time.Millisecond {
		t.Fatalf("duration decode changed: %v", cfg.Server.ShutdownTimeout)
	}
	started, release := make(chan struct{}), make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { close(started); <-release }))
	defer ts.Close()
	defer close(release)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		resp, err := ts.Client().Get(ts.URL)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not start")
	}
	app := &App{httpSrv: ts.Config}
	begin := time.Now()
	err := app.shutdown(cfg.Server.ShutdownTimeout)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(begin) > 2*time.Second {
		t.Fatalf("shutdown timeout not honored: %v elapsed=%v", err, time.Since(begin))
	}
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("active connection was not forcibly closed")
	}
}

func TestShutdownWithoutActiveRequestsCompletesNormally(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Second, time.Second} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
		app := &App{httpSrv: ts.Config}
		if err := app.shutdown(timeout); err != nil {
			ts.Close()
			t.Fatalf("graceful shutdown failed: %v", err)
		}
		ts.Close()
	}
}
