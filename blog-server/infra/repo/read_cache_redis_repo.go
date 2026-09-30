package repo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	randv2 "math/rand/v2"
	"time"

	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"

	"github.com/redis/go-redis/v9"
)

// ReadCacheRedisRepo 以随机代次隔离读缓存，只有当前代次允许回填。
// 代次不设 TTL；即使被驱逐或 Redis 重启后丢失，也生成新随机值而非从零计数。
type ReadCacheRedisRepo struct {
	rdb     *redis.Client
	prefix  string
	initErr error
}

var _ domainrepo.ReadCacheRepo = (*ReadCacheRedisRepo)(nil)

// NewReadCacheRedisRepo 为文章或分类构造独立缓存。无效依赖由方法返回错误，避免启动 panic。
func NewReadCacheRedisRepo(rdb *redis.Client, namespace string) *ReadCacheRedisRepo {
	r := &ReadCacheRedisRepo{rdb: rdb}
	if rdb == nil {
		r.initErr = errors.New("read cache: nil Redis client")
	} else if namespace != consts.ReadCacheArticleNamespace && namespace != consts.ReadCacheCategoryNamespace {
		r.initErr = errors.New("read cache: unsupported namespace")
	} else {
		r.prefix = consts.ReadCacheKeyPrefix + namespace + ":"
	}
	return r
}

// Lua 将初始化、取代次和取内容放在同一条命令中，避免读到新代次配旧内容。
// 当前部署使用单 Redis 实例；动态内容键不适用于 Redis Cluster 跨槽脚本。
const readCacheReadScript = `
local generation = redis.call('GET', KEYS[1])
if not generation then
  redis.call('SET', KEYS[1], ARGV[1], 'NX')
  generation = redis.call('GET', KEYS[1])
end
if string.len(generation) ~= 32 or not string.match(generation, '^[0-9a-f]+$') then
  return redis.error_reply('invalid read cache generation')
end
local dataKey = ARGV[2] .. generation .. ':' .. ARGV[3]
if redis.call('STRLEN', dataKey) > tonumber(ARGV[4]) then
  return redis.error_reply('read cache value too large')
end
local value = redis.call('GET', dataKey)
return {generation, value}
`

const readCacheWriteScript = `
if redis.call('GET', KEYS[1]) ~= ARGV[1] then
  return 0
end
redis.call('SET', KEYS[2], ARGV[2], 'PX', ARGV[3])
return 1
`

func (r *ReadCacheRedisRepo) ready() error {
	if r == nil {
		return errors.New("read cache: nil repository")
	}
	if r.initErr != nil {
		return r.initErr
	}
	if r.rdb == nil || r.prefix == "" {
		return errors.New("read cache: uninitialized repository")
	}
	return nil
}

func readCacheGeneration() (string, error) {
	var entropy [16]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return "", fmt.Errorf("read cache: generate version: %w", err)
	}
	return hex.EncodeToString(entropy[:]), nil
}

func validReadCacheGeneration(generation string) bool {
	if len(generation) != 32 {
		return false
	}
	for _, c := range generation {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func readCacheQueryHash(key string) string {
	digest := sha256.Sum256([]byte(key))
	return hex.EncodeToString(digest[:])
}

func readCacheTTL(negative bool) time.Duration {
	if negative {
		return consts.ReadCacheNegativeTTL
	}
	jitter := consts.ReadCacheTTL / consts.ReadCacheJitterDivisor
	return consts.ReadCacheTTL - jitter + time.Duration(randv2.Int64N(int64(jitter)*2+1))
}

// Read 只将真正的内容缺失映射为 ErrCacheMiss，Redis 故障或坏代次不提供可回填代次。
func (r *ReadCacheRedisRepo) Read(ctx context.Context, key string) ([]byte, string, error) {
	if err := r.ready(); err != nil {
		return nil, "", err
	}
	candidate, err := readCacheGeneration()
	if err != nil {
		return nil, "", err
	}
	result, err := r.rdb.Eval(ctx, readCacheReadScript, []string{r.prefix + "generation"},
		candidate, r.prefix+"data:", readCacheQueryHash(key), consts.ReadCacheMaxBytes).Slice()
	if err != nil {
		return nil, "", fmt.Errorf("read cache: read: %w", err)
	}
	if len(result) != 2 {
		return nil, "", errors.New("read cache: unexpected read response")
	}
	generation, ok := result[0].(string)
	if !ok || !validReadCacheGeneration(generation) {
		return nil, "", errors.New("read cache: invalid generation response")
	}
	if result[1] == nil {
		return nil, generation, domainrepo.ErrCacheMiss
	}
	data, ok := result[1].(string)
	if !ok {
		return nil, "", errors.New("read cache: invalid data response")
	}
	if len(data) > consts.ReadCacheMaxBytes {
		return nil, "", errors.New("read cache: value too large")
	}
	return []byte(data), generation, nil
}

// Write 在 Redis 内原子比较代次并回填，防止查库期间的修改被慢响应覆盖。
func (r *ReadCacheRedisRepo) Write(ctx context.Context, key, generation string, data []byte, negative bool) (bool, error) {
	if err := r.ready(); err != nil {
		return false, err
	}
	if !validReadCacheGeneration(generation) {
		return false, errors.New("read cache: invalid generation")
	}
	if len(data) > consts.ReadCacheMaxBytes {
		return false, errors.New("read cache: value too large")
	}
	ttl := readCacheTTL(negative).Milliseconds()
	if ttl <= 0 {
		return false, errors.New("read cache: non-positive TTL")
	}
	result, err := r.rdb.Eval(ctx, readCacheWriteScript,
		[]string{r.prefix + "generation", r.prefix + "data:" + generation + ":" + readCacheQueryHash(key)},
		generation, data, ttl).Int64()
	if err != nil {
		return false, fmt.Errorf("read cache: write: %w", err)
	}
	if result != 0 && result != 1 {
		return false, errors.New("read cache: unexpected write response")
	}
	return result == 1, nil
}

// Invalidate 只替换一个持久代次键，旧快照在有限 TTL 内自动回收。
func (r *ReadCacheRedisRepo) Invalidate(ctx context.Context) error {
	if err := r.ready(); err != nil {
		return err
	}
	generation, err := readCacheGeneration()
	if err != nil {
		return err
	}
	if err := r.rdb.Set(ctx, r.prefix+"generation", generation, 0).Err(); err != nil {
		return fmt.Errorf("read cache: invalidate: %w", err)
	}
	return nil
}
