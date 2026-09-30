package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/logger"
	"blog-server/domain/repo"

	"go.uber.org/zap"
)

// readCacheState 只在应用层编排缓存；同进程请求共享回源，Redis generation 隔离更新前后的数据。
type readCacheState struct {
	backend    repo.ReadCacheRepo
	mu         sync.Mutex
	repairGate chan struct{}
	revision   uint64
	dirty      bool
	flights    map[string]*readCacheFlight
}

type readCacheFlight struct {
	done chan struct{}
	data []byte
	err  error
}

type readCacheEnvelope[T any] struct {
	Complete bool `json:"complete"`
	Missing  bool `json:"missing"`
	Value    T    `json:"value"`
}

func newReadCacheState(backend repo.ReadCacheRepo) *readCacheState {
	return &readCacheState{backend: backend, repairGate: make(chan struct{}, 1), flights: make(map[string]*readCacheFlight)}
}

func (s *readCacheState) snapshot() (uint64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revision, s.dirty
}

// invalidate 在数据库成功后使用独立短期限；失败后本进程暂停读取旧缓存，不能把已提交写入报成失败。
func (s *readCacheState) invalidate() {
	if s == nil || s.backend == nil {
		return
	}
	s.mu.Lock()
	s.revision++
	s.dirty = true
	s.mu.Unlock()
	s.repair(context.Background())
}

func (s *readCacheState) repair(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, consts.ReadCacheOperationTimeout)
	defer cancel()
	select {
	case s.repairGate <- struct{}{}:
		defer func() { <-s.repairGate }()
	case <-ctx.Done():
		return
	}
	revision, dirty := s.snapshot()
	if !dirty {
		return
	}
	if err := s.backend.Invalidate(ctx); err != nil {
		logger.Warn("[APP] 读取缓存失效失败，本进程暂时回源", zap.Error(err))
		return
	}
	s.mu.Lock()
	if s.revision == revision {
		s.dirty = false
	}
	s.mu.Unlock()
}

func cacheRead(ctx context.Context, s *readCacheState, key string) ([]byte, string, error) {
	bounded, cancel := context.WithTimeout(ctx, consts.ReadCacheOperationTimeout)
	defer cancel()
	return s.backend.Read(bounded, key)
}

func sameCacheError(err, target error) bool {
	if target == nil {
		return false
	}
	var got, want common.Errno
	if errors.As(err, &got) && errors.As(target, &want) {
		return got.Code == want.Code
	}
	return errors.Is(err, target)
}

func decodeReadCache[T any](raw []byte, notFound error, valid func(T) bool) (T, error, bool) {
	var entry readCacheEnvelope[T]
	if len(raw) == 0 || len(raw) > consts.ReadCacheMaxBytes || json.Unmarshal(raw, &entry) != nil || !entry.Complete {
		var zero T
		return zero, nil, false
	}
	if entry.Missing {
		if notFound == nil {
			var zero T
			return zero, nil, false
		}
		var zero T
		return zero, notFound, true
	}
	if valid != nil && !valid(entry.Value) {
		var zero T
		return zero, nil, false
	}
	return entry.Value, nil, true
}

// readThrough 将缓存命中、未命中和故障回源收口；notFound 只缓存明确的业务不存在，其他错误不缓存。
func readThrough[T any](ctx context.Context, state *readCacheState, key string, notFound error, load func(context.Context) (T, error), valid func(T) bool) (T, error) {
	var zero T
	if ctx.Err() != nil {
		return zero, common.ServerErr.WithErr(ctx.Err())
	}
	if state == nil || state.backend == nil {
		return load(ctx)
	}
	if _, dirty := state.snapshot(); dirty {
		state.repair(ctx)
	}
	revision, dirty := state.snapshot()
	generation := ""
	if !dirty {
		raw, observed, err := cacheRead(ctx, state, key)
		generation = observed
		if err == nil {
			if value, failure, ok := decodeReadCache(raw, notFound, valid); ok {
				return value, failure
			}
		} else if !errors.Is(err, repo.ErrCacheMiss) {
			generation = ""
		}
	}
	// generation 与本机 revision 都参与 flight key，新写入之后的请求不会加入旧回源。
	flightKey := fmt.Sprintf("%d:%s:%s", revision, generation, key)
	state.mu.Lock()
	flight := state.flights[flightKey]
	if flight == nil {
		flight = &readCacheFlight{done: make(chan struct{})}
		state.flights[flightKey] = flight
		// 首个浏览器取消不应取消其他等待者；后台回源有独立上限，不无限运行。
		workCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), consts.ReadCacheLoadTimeout)
		go func() {
			defer cancel()
			defer func() {
				if failure := recover(); failure != nil {
					flight.err = common.ServerErr
					logger.Error("[APP] 缓存回源异常", zap.Any("panic", failure))
				}
				state.mu.Lock()
				delete(state.flights, flightKey)
				close(flight.done)
				state.mu.Unlock()
			}()
			cacheable := generation != ""
			currentRevision, currentDirty := state.snapshot()
			if cacheable && !currentDirty && currentRevision == revision {
				raw, currentGeneration, err := cacheRead(workCtx, state, key)
				if currentGeneration != generation {
					cacheable = false
				}
				if err == nil {
					if _, failure, ok := decodeReadCache(raw, notFound, valid); ok {
						flight.data, flight.err = raw, failure
						return
					}
				} else if !errors.Is(err, repo.ErrCacheMiss) {
					cacheable = false
				}
			} else {
				cacheable = false
			}
			value, err := load(workCtx)
			missing := sameCacheError(err, notFound)
			if err != nil && !missing {
				flight.err = err
				return
			}
			entry := readCacheEnvelope[T]{Complete: true, Missing: missing, Value: value}
			encoded, encodeErr := json.Marshal(entry)
			if encodeErr != nil {
				flight.err = common.ServerErr.WithErr(encodeErr)
				return
			}
			flight.data, flight.err = encoded, err
			currentRevision, currentDirty = state.snapshot()
			if cacheable && !currentDirty && currentRevision == revision && len(encoded) <= consts.ReadCacheMaxBytes && (missing || valid == nil || valid(value)) {
				bounded, done := context.WithTimeout(workCtx, consts.ReadCacheOperationTimeout)
				defer done()
				if _, writeErr := state.backend.Write(bounded, key, generation, encoded, missing); writeErr != nil {
					logger.Warn("[APP] 读取缓存回填失败，保留数据库结果", zap.Error(writeErr))
				}
			}
		}()
	}
	state.mu.Unlock()
	select {
	case <-ctx.Done():
		return zero, common.ServerErr.WithErr(ctx.Err())
	case <-flight.done:
		if flight.err != nil {
			return zero, flight.err
		}
		// 每个等待者独立反序列化，避免返回共享 map/slice 后发生并发修改。
		var entry readCacheEnvelope[T]
		if err := json.Unmarshal(flight.data, &entry); err != nil {
			return zero, common.ServerErr.WithErr(err)
		}
		return entry.Value, nil
	}
}
