package repo

import (
	"blog-server/config"
	"blog-server/domain/repo"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type commentRedisRepo struct {
	rdb    *redis.Client
	config *config.RedisConfig // 注入Redis配置
}

func NewCommentRedisRepo(rdb *redis.Client, cfg *config.GlobalConfig) *commentRedisRepo {
	return &commentRedisRepo{
		rdb:    rdb,
		config: &cfg.Redis,
	}
}

const (
	// 评论点赞数
	keyCommentLikeCount = "comment:like:count:%d"

	// 用户点赞记录（Bitmap，每天自动过期，无内存泄漏）
	keyUserLikeRecord = "comment:like:user:%d" // 每日自动过期

	// 文章评论总数
	keyArticleCommentCount = "article:comment:count:%d"
)

// 点赞数过期时间
func (r *commentRedisRepo) getLikeCacheExpire() time.Duration {
	return r.config.LikeCacheExpire
}

// 用户点赞记录过期时间
func (r *commentRedisRepo) getLikeRecordExpire() time.Duration {
	return r.config.LikeRecordExpire
}

// ===================== 评论点赞数 =====================
func (r *commentRedisRepo) GetCommentLikeCount(ctx context.Context, commentID uint64) (int, error) {
	key := fmt.Sprintf(keyCommentLikeCount, commentID)
	val, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(val)
}

func (r *commentRedisRepo) SetCommentLikeCount(ctx context.Context, commentID uint64, count int) error {
	key := fmt.Sprintf(keyCommentLikeCount, commentID)
	return r.rdb.Set(ctx, key, count, r.getLikeCacheExpire()).Err()
}

// DelCommentLikeCount 在持久化点赞事务完成后使旧计数失效。
func (r *commentRedisRepo) DelCommentLikeCount(ctx context.Context, commentID uint64) error {
	return r.rdb.Del(ctx, fmt.Sprintf(keyCommentLikeCount, commentID)).Err()
}

func (r *commentRedisRepo) IncrCommentLikeCount(ctx context.Context, commentID uint64) error {
	key := fmt.Sprintf(keyCommentLikeCount, commentID)
	err := r.rdb.Incr(ctx, key).Err()
	if err == nil {
		r.rdb.Expire(ctx, key, r.getLikeCacheExpire())
	}
	return err
}

func (r *commentRedisRepo) DecrCommentLikeCount(ctx context.Context, commentID uint64) error {
	key := fmt.Sprintf(keyCommentLikeCount, commentID)
	count, _ := r.GetCommentLikeCount(ctx, commentID)
	if count <= 0 {
		return nil
	}
	err := r.rdb.Decr(ctx, key).Err()
	if err == nil {
		r.rdb.Expire(ctx, key, r.getLikeCacheExpire())
	}
	return err
}

// ===================== 旧 Bitmap 兼容接口；当前点赞幂等由 MySQL 关系表保证 =====================
// CheckUserLikeComment 检查是否点赞
func (r *commentRedisRepo) CheckUserLikeComment(ctx context.Context, userID uint64, commentID uint64) (bool, error) {
	key := fmt.Sprintf(keyUserLikeRecord, userID)
	res, err := r.rdb.GetBit(ctx, key, int64(commentID)).Result()
	if err != nil {
		return false, err
	}
	return res == 1, nil

}

// AddUserLikeRecord 添加点赞记录（自动过期）
func (r *commentRedisRepo) AddUserLikeRecord(ctx context.Context, userID uint64, commentID uint64) error {
	key := fmt.Sprintf(keyUserLikeRecord, userID)
	err := r.rdb.SetBit(ctx, key, int64(commentID), 1).Err()
	if err == nil {
		r.rdb.Expire(ctx, key, r.getLikeRecordExpire())
	}
	return err
}

// DelUserLikeRecord 取消点赞
func (r *commentRedisRepo) DelUserLikeRecord(ctx context.Context, userID uint64, commentID uint64) error {
	key := fmt.Sprintf(keyUserLikeRecord, userID)
	return r.rdb.SetBit(ctx, key, int64(commentID), 0).Err()
}

// ===================== 文章评论总数 =====================
func (r *commentRedisRepo) SetArticleCommentCount(ctx context.Context, articleID uint64, count int) error {
	key := fmt.Sprintf(keyArticleCommentCount, articleID)
	return r.rdb.Set(ctx, key, count, r.getLikeCacheExpire()).Err()
}

func (r *commentRedisRepo) GetArticleCommentCount(ctx context.Context, articleID uint64) (int, error) {
	key := fmt.Sprintf(keyArticleCommentCount, articleID)
	val, err := r.rdb.Get(ctx, key).Result()

	// 未命中翻译成领域哨兵 repo.ErrCacheMiss，不把 go-redis 的 redis.Nil 泄漏到 App 层；
	// App 用 errors.Is(err, repo.ErrCacheMiss) 区分未命中与真实故障，两者都回源查库。
	if errors.Is(err, redis.Nil) {
		return 0, repo.ErrCacheMiss
	}
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(val)
}

func (r *commentRedisRepo) DelArticleCommentCache(ctx context.Context, articleID uint64) error {
	key := fmt.Sprintf(keyArticleCommentCount, articleID)
	return r.rdb.Del(ctx, key).Err()
}
