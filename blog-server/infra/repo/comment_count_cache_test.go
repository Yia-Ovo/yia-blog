package repo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"blog-server/config"
	domainrepo "blog-server/domain/repo"

	"github.com/redis/go-redis/v9"
)

func TestCommentCountCacheDecode(t *testing.T) {
	readErr := errors.New("test Redis read failure")
	maxInt := int(^uint(0) >> 1)
	for _, tc := range []struct {
		name       string
		value      string
		commandErr error
		wantCount  int
		wantMiss   bool
		wantError  bool
	}{
		{name: "zero is a hit", value: "0"},
		{name: "positive is a hit", value: "17", wantCount: 17},
		{name: "maximum int is a hit", value: strconv.Itoa(maxInt), wantCount: maxInt},
		{name: "missing key", commandErr: redis.Nil, wantMiss: true},
		{name: "wrapped missing key", commandErr: fmt.Errorf("wrapped: %w", redis.Nil), wantMiss: true},
		{name: "read failure", commandErr: readErr, wantError: true},
		{name: "empty value is corrupt", value: "", wantError: true},
		{name: "malformed value is corrupt", value: "not-a-count", wantError: true},
		{name: "overflow is corrupt", value: strings.Repeat("9", 64), wantError: true},
		{name: "negative value is corrupt", value: "-1", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			client := readCacheMockClient(t, func(_ context.Context, cmd redis.Cmder) error {
				reads++
				args := cmd.Args()
				if cmd.Name() != "get" || len(args) != 2 || args[1] != "article:comment:count:42" {
					t.Fatalf("unexpected comment count command: %v", args)
				}
				if tc.commandErr != nil {
					return tc.commandErr
				}
				cmd.(*redis.StringCmd).SetVal(tc.value)
				return nil
			})
			count, err := NewCommentRedisRepo(client, &config.GlobalConfig{}).GetArticleCommentCount(context.Background(), 42)
			if reads != 1 {
				t.Fatalf("cache reads = %d, want 1", reads)
			}
			if tc.wantMiss {
				if count != 0 || !errors.Is(err, domainrepo.ErrCacheMiss) || errors.Is(err, redis.Nil) {
					t.Fatalf("missing key was not translated to the domain sentinel: count=%d err=%v", count, err)
				}
				return
			}
			if tc.wantError {
				if err == nil || errors.Is(err, domainrepo.ErrCacheMiss) {
					t.Fatalf("invalid cache value became a hit or miss: count=%d err=%v", count, err)
				}
				if tc.commandErr != nil && !errors.Is(err, tc.commandErr) {
					t.Fatalf("Redis failure was lost: %v", err)
				}
				return
			}
			if err != nil || count != tc.wantCount {
				t.Fatalf("cache hit = (%d, %v), want (%d, nil)", count, err, tc.wantCount)
			}
		})
	}
}
