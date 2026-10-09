package ioc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"blog-server/api/dto"
	appservice "blog-server/app/service"
	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
	infrarepo "blog-server/infra/repo"

	"github.com/redis/go-redis/v9"
)

// cacheFaultServer 只模拟本测试需要的 RESP 命令；不执行 Lua，也不连接真实 Redis。
type cacheFaultServer struct {
	listener      net.Listener
	stop          chan struct{}
	wg            sync.WaitGroup
	mu            sync.Mutex
	connections   map[net.Conn]struct{}
	mode          string
	reads         atomic.Int32
	writes        atomic.Int32
	invalidations atomic.Int32
}

func newCacheFaultServer(t *testing.T, mode string) *cacheFaultServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &cacheFaultServer{listener: listener, stop: make(chan struct{}), connections: make(map[net.Conn]struct{}), mode: mode}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			s.mu.Lock()
			select {
			case <-s.stop:
				s.mu.Unlock()
				_ = conn.Close()
				return
			default:
			}
			s.connections[conn] = struct{}{}
			s.wg.Add(1)
			s.mu.Unlock()
			go func() {
				defer s.wg.Done()
				defer func() {
					_ = conn.Close()
					s.mu.Lock()
					delete(s.connections, conn)
					s.mu.Unlock()
				}()
				s.serve(conn)
			}()
		}
	}()
	t.Cleanup(func() {
		close(s.stop)
		_ = listener.Close()
		s.mu.Lock()
		for conn := range s.connections {
			_ = conn.Close()
		}
		s.mu.Unlock()
		s.wg.Wait()
	})
	return s
}

// readCacheRESPCommand 读取有界的 RESP 数组，确保测试线程不会因损坏的输入无限分配。
func readCacheRESPCommand(r *bufio.Reader) ([]string, error) {
	header, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(header, "*") || !strings.HasSuffix(header, "\r\n") {
		return nil, errors.New("expected RESP array")
	}
	count, err := strconv.Atoi(strings.TrimSuffix(header[1:], "\r\n"))
	if err != nil || count < 1 || count > 64 {
		return nil, errors.New("invalid RESP array length")
	}
	args := make([]string, count)
	for i := range args {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(header, "$") || !strings.HasSuffix(header, "\r\n") {
			return nil, errors.New("expected RESP bulk")
		}
		length, err := strconv.Atoi(strings.TrimSuffix(header[1:], "\r\n"))
		if err != nil || length < 0 || length > 4<<20 {
			return nil, errors.New("invalid RESP bulk length")
		}
		data := make([]byte, length+2)
		if _, err := io.ReadFull(r, data); err != nil {
			return nil, err
		}
		if string(data[length:]) != "\r\n" {
			return nil, errors.New("invalid RESP bulk terminator")
		}
		args[i] = string(data[:length])
	}
	return args, nil
}

func (s *cacheFaultServer) serve(conn net.Conn) {
	r := bufio.NewReader(conn)
	for {
		args, err := readCacheRESPCommand(r)
		if err != nil {
			return
		}
		switch strings.ToLower(args[0]) {
		case "hello":
			reply := "*0\r\n"
			if len(args) > 1 && args[1] == "3" {
				reply = "%0\r\n"
			}
			if _, err := io.WriteString(conn, reply); err != nil {
				return
			}
		case "client", "ping":
			if _, err := io.WriteString(conn, "+OK\r\n"); err != nil {
				return
			}
		case "eval":
			if len(args) < 3 {
				return
			}
			if args[2] == "2" {
				s.writes.Add(1)
				_, _ = io.WriteString(conn, "-READONLY test replica rejects writes\r\n")
				continue
			}
			s.reads.Add(1)
			switch s.mode {
			case "disconnect":
				return
			case "stall":
				<-s.stop
				return
			default:
				generation := strings.Repeat("a", 32)
				// 正常握手后返回真实协议未命中；失效失败测试若误读缓存，则返回过期快照。
				if s.mode == "invalidate_stall" {
					stale := "{\"complete\":true,\"value\":{\"ID\":1,\"Name\":\"stale\"}}"
					_, _ = fmt.Fprintf(conn, "*2\r\n$32\r\n%s\r\n$%d\r\n%s\r\n", generation, len(stale), stale)
				} else {
					_, _ = fmt.Fprintf(conn, "*2\r\n$32\r\n%s\r\n$-1\r\n", generation)
				}
			}
		case "set":
			s.invalidations.Add(1)
			if s.mode == "invalidate_stall" {
				<-s.stop
				return
			}
			if _, err := io.WriteString(conn, "+OK\r\n"); err != nil {
				return
			}
		default:
			if _, err := io.WriteString(conn, "-ERR unsupported test command\r\n"); err != nil {
				return
			}
		}
	}
}

func networkReadCache(t *testing.T, server *cacheFaultServer, protocol int) *infrarepo.ReadCacheRedisRepo {
	t.Helper()
	// 模拟鉴权客户端较长的默认超时；内容缓存必须由真实工厂改为独立的 200ms 预算。
	source := redis.NewClient(&redis.Options{
		Addr: server.listener.Addr().String(), Protocol: protocol, DisableIdentity: true,
		DialTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
	})
	t.Cleanup(func() { _ = source.Close() })
	client := newReadCacheClient(source)
	t.Cleanup(func() { _ = client.Close() })
	return infrarepo.NewReadCacheRedisRepo(client, consts.ReadCacheCategoryNamespace)
}

type networkCategoryRepo struct {
	domainrepo.CategoryMysqlRepo
	readErr error
	reads   atomic.Int32
	creates atomic.Int32
}

func (r *networkCategoryRepo) GetCategoryById(_ context.Context, id uint64) (*model.CategoryModel, error) {
	r.reads.Add(1)
	if r.readErr != nil {
		return nil, r.readErr
	}
	return &model.CategoryModel{ID: id, Name: "fresh", Status: 1}, nil
}
func (r *networkCategoryRepo) CreateCategory(context.Context, *model.CategoryModel) error {
	r.creates.Add(1)
	return nil
}

func assertCacheFaultBound(t *testing.T, started time.Time) {
	t.Helper()
	// 留出 CI 调度余量，同时能检测错误地沿用鉴权客户端 5s 超时的回归。
	if elapsed := time.Since(started); elapsed > 1500*time.Millisecond {
		t.Fatalf("cache fault exceeded local fallback budget: %v", elapsed)
	}
}

func TestReadCacheNetworkReadFaultFallsBack(t *testing.T) {
	for _, protocol := range []int{2, 3} {
		for _, mode := range []string{"disconnect", "stall"} {
			t.Run(fmt.Sprintf("RESP%d/%s", protocol, mode), func(t *testing.T) {
				server := newCacheFaultServer(t, mode)
				cache := networkReadCache(t, server, protocol)
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				started := time.Now()
				data, generation, err := cache.Read(ctx, "probe")
				assertCacheFaultBound(t, started)
				if err == nil || errors.Is(err, domainrepo.ErrCacheMiss) || data != nil || generation != "" {
					t.Fatalf("network failure became a hit/miss: data=%q generation=%q err=%v", data, generation, err)
				}
				db := &networkCategoryRepo{}
				app := appservice.NewCategoryAppService(domainservice.NewCategoryDomainService(db), cache)
				started = time.Now()
				value, err := app.GetCategoryById(ctx, 1)
				assertCacheFaultBound(t, started)
				if err != nil || value["name"] != "fresh" || db.reads.Load() != 1 {
					t.Fatalf("fallback lost database result: value=%v err=%v reads=%d", value, err, db.reads.Load())
				}
				if server.reads.Load() < 2 || server.writes.Load() != 0 {
					t.Fatal("fault was not reached or unsafe writeback occurred")
				}
			})
		}
	}
}

func TestReadCacheNetworkWriteFailurePreservesDatabaseResult(t *testing.T) {
	server := newCacheFaultServer(t, "write_error")
	cache := networkReadCache(t, server, 3)
	db := &networkCategoryRepo{}
	app := appservice.NewCategoryAppService(domainservice.NewCategoryDomainService(db), cache)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for range 2 {
		value, err := app.GetCategoryById(ctx, 1)
		if err != nil || value["name"] != "fresh" {
			t.Fatalf("write failure replaced success: %v %v", value, err)
		}
	}
	if db.reads.Load() != 2 || server.writes.Load() != 2 {
		t.Fatalf("expected two rejected writebacks: DB=%d writes=%d", db.reads.Load(), server.writes.Load())
	}
}

func TestReadCacheNetworkInvalidationFailureSkipsOldSnapshot(t *testing.T) {
	server := newCacheFaultServer(t, "invalidate_stall")
	cache := networkReadCache(t, server, 3)
	db := &networkCategoryRepo{}
	app := appservice.NewCategoryAppService(domainservice.NewCategoryDomainService(db), cache)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	started := time.Now()
	if err := app.CreateCategory(ctx, &dto.CreateCategoryReq{Name: "fresh"}); err != nil {
		t.Fatal(err)
	}
	assertCacheFaultBound(t, started)
	if db.creates.Load() != 1 || server.invalidations.Load() != 1 {
		t.Fatal("database success did not attempt invalidation")
	}
	started = time.Now()
	value, err := app.GetCategoryById(ctx, 1)
	assertCacheFaultBound(t, started)
	if err != nil || value["name"] != "fresh" || db.reads.Load() != 1 {
		t.Fatalf("dirty cache reused old data: %v %v", value, err)
	}
	if server.invalidations.Load() != 2 || server.reads.Load() != 0 || server.writes.Load() != 0 {
		t.Fatal("failed repair must keep this process away from stale cache")
	}
}

func TestReadCacheNetworkAndDatabaseFailureDoesNotFakeSuccess(t *testing.T) {
	server := newCacheFaultServer(t, "disconnect")
	db := &networkCategoryRepo{readErr: errors.New("test database unavailable")}
	app := appservice.NewCategoryAppService(domainservice.NewCategoryDomainService(db), networkReadCache(t, server, 3))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	value, err := app.GetCategoryById(ctx, 1)
	if err == nil || value != nil || db.reads.Load() != 1 || server.reads.Load() != 1 || server.writes.Load() != 0 {
		t.Fatalf("double failure became success or empty cache: %v %v", value, err)
	}
}
