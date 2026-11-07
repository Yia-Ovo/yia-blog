package repo

import (
	"blog-server/domain/do"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestAIStatePersistenceRollbackAndConcurrency(t *testing.T) {
	dir := t.TempDir()
	r := NewAIStateFileRepo(dir)
	ctx := context.Background()
	if err := r.Update(ctx, 1, func(s *do.AIState) error { s.Memory = "当前练习 Redis"; return nil }); err != nil {
		t.Fatal(err)
	}
	r = NewAIStateFileRepo(dir)
	if err := r.Update(ctx, 1, func(s *do.AIState) error {
		if s.Memory != "当前练习 Redis" {
			t.Fatal("memory lost after reopening")
		}
		s.Memory = "discard"
		return errors.New("rollback")
	}); err == nil {
		t.Fatal("callback error ignored")
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := r.Update(ctx, 1, func(s *do.AIState) error {
				if s.Memory != "当前练习 Redis" {
					return errors.New("rollback failed")
				}
				s.Requests++
				return nil
			}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if err := r.Update(ctx, 1, func(s *do.AIState) error {
		if s.Requests != 10 {
			t.Fatalf("lost concurrent update: %d", s.Requests)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := r.Update(ctx, 2, func(s *do.AIState) error {
		if s.Memory != "" {
			t.Fatal("cross-account leak")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestAIKnowledgeAllowlistAndRedaction(t *testing.T) {
	root := t.TempDir()
	for path, text := range map[string]string{".private/02-面试资料/README.md": "JWT token_version 用于账号级撤销\n密钥 sk-testprivatekey123456789", ".private/05-部署运维/secret.md": "forbidden", ".private/99-归档/old.md": "forbidden", "blog-server/config/config.dev.yaml": "forbidden", ".private/02-面试资料/.source/private.md": "forbidden"} {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	r := NewAIKnowledgeFileRepo(root)
	documents, err := r.Info(context.Background())
	if err != nil || documents != 1 {
		t.Fatalf("allowlist: %d %v", documents, err)
	}
	sources, err := r.Search(context.Background(), "JWT token_version")
	if err != nil || len(sources) != 1 {
		t.Fatalf("search: %v", err)
	}
	if strings.Contains(sources[0].Excerpt, "sk-test") || !strings.Contains(sources[0].Excerpt, "token_version") {
		t.Fatal("redaction damaged relevant data or exposed key")
	}
}

func TestAIResponsesContractAndInterruptedStream(t *testing.T) {
	for _, complete := range []bool{true, false} {
		t.Run(map[bool]string{true: "completed", false: "interrupted"}[complete], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/responses" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Error("invalid provider request")
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body["model"] != "gpt-6.1-sol" || body["store"] != false || body["stream"] != true || body["max_output_tokens"] != nil || body["prompt_cache_key"] == nil {
					t.Error("invalid response options")
				}
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"你好\"}\n\n"))
				if complete {
					_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"你好\"}]}]}}\n\n"))
				}
			}))
			defer server.Close()
			r := NewAIResponsesRepo(server.URL+"/v1", "test-key", "gpt-6.1-sol")
			text, err := r.Reply(context.Background(), "private instructions", []do.AIModelMessage{{Role: "user", Content: "test"}})
			if complete && (err != nil || text != "你好") {
				t.Fatalf("valid response rejected: %v", err)
			}
			if !complete && err == nil {
				t.Fatal("truncated response accepted")
			}
		})
	}
}

func TestAIResponsesDoesNotFollowRedirectOrExposeErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://example.invalid")
		w.WriteHeader(302)
		_, _ = w.Write([]byte("test-key private body"))
	}))
	defer server.Close()
	_, err := NewAIResponsesRepo(server.URL, "test-key", "gpt-6.1-sol").Reply(context.Background(), "", nil)
	if err == nil || strings.Contains(err.Error(), "test-key") || strings.Contains(err.Error(), "private body") {
		t.Fatal("redirect followed or sensitive error exposed")
	}
}
