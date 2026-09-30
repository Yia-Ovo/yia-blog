package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"blog-server/common"
	"blog-server/common/consts"
)

type readCacheObservedRepo struct {
	*categoryMemoryReadCache
	reads chan struct{}
}

func (r *readCacheObservedRepo) Read(ctx context.Context, key string) ([]byte, string, error) {
	data, generation, err := r.categoryMemoryReadCache.Read(ctx, key)
	select {
	case r.reads <- struct{}{}:
	default:
	}
	return data, generation, err
}

type readCacheResult struct {
	value string
	err   error
}

func assertNoReadCacheFlights(t *testing.T, state *readCacheState) {
	t.Helper()
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.flights) != 0 {
		t.Fatalf("completed load left %d flights", len(state.flights))
	}
}

func TestReadCacheLeaderCancellationDoesNotCancelSharedLoad(t *testing.T) {
	cache := &readCacheObservedRepo{categoryMemoryReadCache: &categoryMemoryReadCache{}, reads: make(chan struct{}, 10)}
	state := newReadCacheState(cache)
	gate := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(gate) }) }
	defer release()
	started := make(chan context.Context, 1)
	var calls atomic.Int32
	load := func(ctx context.Context) (string, error) {
		calls.Add(1)
		started <- ctx
		select {
		case <-gate:
			return "loaded once", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	type ownerKey struct{}
	leaderCtx, cancelLeader := context.WithCancel(context.WithValue(context.Background(), ownerKey{}, "original owner"))
	defer cancelLeader()
	leaderResult := make(chan readCacheResult, 1)
	go func() {
		value, err := readThrough(leaderCtx, state, "shared", nil, load, nil)
		leaderResult <- readCacheResult{value, err}
	}()
	var workCtx context.Context
	select {
	case workCtx = <-started:
	case <-time.After(time.Second):
		t.Fatal("shared load did not start")
	}
	if workCtx.Value(ownerKey{}) != "original owner" {
		t.Fatal("shared load lost request context values")
	}
	if deadline, ok := workCtx.Deadline(); !ok || time.Until(deadline) > consts.ReadCacheLoadTimeout {
		t.Fatal("shared load must have its own finite deadline")
	}
	// 首次读取及回源前的二次缓存检查已经结束，接下来的Read来自另一请求。
	for {
		select {
		case <-cache.reads:
		default:
			goto drained
		}
	}
drained:
	followerResult := make(chan readCacheResult, 1)
	go func() {
		value, err := readThrough(context.Background(), state, "shared", nil, load, nil)
		followerResult <- readCacheResult{value, err}
	}()
	select {
	case <-cache.reads:
	case <-time.After(time.Second):
		t.Fatal("follower did not enter cache lookup")
	}
	cancelLeader()
	select {
	case result := <-leaderResult:
		if result.err == nil {
			t.Fatal("cancelled leader should stop waiting")
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled leader kept waiting for shared load")
	}
	if err := workCtx.Err(); err != nil {
		t.Fatalf("leader cancellation reached shared load: %v", err)
	}
	release()
	select {
	case result := <-followerResult:
		if result.err != nil || result.value != "loaded once" {
			t.Fatalf("follower result=%+v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("follower did not receive shared result")
	}
	if calls.Load() != 1 {
		t.Fatalf("database loads=%d, want 1", calls.Load())
	}
	assertNoReadCacheFlights(t, state)
}

func TestReadCacheRepairGateWaitingIsBounded(t *testing.T) {
	for _, mode := range []string{"operation deadline", "caller deadline"} {
		t.Run(mode, func(t *testing.T) {
			cache := &categoryMemoryReadCache{}
			state := newReadCacheState(cache)
			state.repairGate <- struct{}{}
			state.mu.Lock()
			state.dirty = true
			state.mu.Unlock()
			ctx := context.Background()
			if mode == "caller deadline" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 20*time.Millisecond)
				defer cancel()
			}
			done := make(chan struct{})
			go func() {
				state.repair(ctx)
				close(done)
			}()
			// 未释放占用者时也必须返回，避免把等待锁排除在Redis操作期限外。
			select {
			case <-done:
			case <-time.After(consts.ReadCacheOperationTimeout + 500*time.Millisecond):
				<-state.repairGate
				t.Fatal("repair gate wait exceeded bounded deadline")
			}
			if _, dirty := state.snapshot(); !dirty {
				t.Fatal("timed-out repair unexpectedly cleared dirty state")
			}
			if cache.invalidations != 0 {
				t.Fatalf("blocked repair called backend %d times", cache.invalidations)
			}
			<-state.repairGate
			state.repair(context.Background())
			if _, dirty := state.snapshot(); dirty || cache.invalidations != 1 {
				t.Fatalf("retry failed: dirty=%v invalidations=%d", dirty, cache.invalidations)
			}
		})
	}
}

func TestReadCacheInvalidateGateTimeoutKeepsDirty(t *testing.T) {
	cache := &categoryMemoryReadCache{}
	state := newReadCacheState(cache)
	state.repairGate <- struct{}{}
	done := make(chan struct{})
	go func() {
		state.invalidate()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(consts.ReadCacheOperationTimeout + 500*time.Millisecond):
		<-state.repairGate
		t.Fatal("write-side invalidation queued without a deadline")
	}
	<-state.repairGate
	if revision, dirty := state.snapshot(); revision != 1 || !dirty {
		t.Fatalf("timed-out invalidation revision=%d dirty=%v", revision, dirty)
	}
	if cache.invalidations != 0 {
		t.Fatal("invalidation entered occupied gate")
	}
}

func TestReadCacheFailedLoadCleansFlightAndCanRetry(t *testing.T) {
	for _, mode := range []string{"error", "panic"} {
		t.Run(mode, func(t *testing.T) {
			cache := &categoryMemoryReadCache{}
			state := newReadCacheState(cache)
			failure := errors.New("database unavailable")
			var calls atomic.Int32
			load := func(context.Context) (string, error) {
				if calls.Add(1) == 1 {
					if mode == "panic" {
						panic("test loader panic")
					}
					return "", failure
				}
				return "recovered", nil
			}
			_, err := readThrough(context.Background(), state, "retry", nil, load, nil)
			if mode == "error" && !errors.Is(err, failure) {
				t.Fatalf("load error=%v", err)
			}
			if mode == "panic" && !errors.Is(err, common.ServerErr) {
				t.Fatalf("panic error=%v", err)
			}
			assertNoReadCacheFlights(t, state)
			if cache.writes != 0 {
				t.Fatal("failed load polluted cache")
			}
			value, err := readThrough(context.Background(), state, "retry", nil, load, nil)
			if err != nil || value != "recovered" || calls.Load() != 2 {
				t.Fatalf("retry value=%q err=%v calls=%d", value, err, calls.Load())
			}
			assertNoReadCacheFlights(t, state)
			value, err = readThrough(context.Background(), state, "retry", nil, load, nil)
			if err != nil || value != "recovered" || calls.Load() != 2 {
				t.Fatalf("recovered cache value=%q err=%v calls=%d", value, err, calls.Load())
			}
		})
	}
}
