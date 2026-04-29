package repo

import (
	"blog-server/infra/model"
	"context"
	"time"
)

type AdminRedisRepo interface {
	GetAdminFromRedis(ctx context.Context, adminID uint64) (*model.AdminModel, error)
	SetAdminToRedis(ctx context.Context, adminID uint64, admin *model.AdminModel) error
	DelAdminFromRedis(ctx context.Context, adminID uint64) error
	SetAdminOnlineToken(ctx context.Context, adminID uint64, token string, expire time.Duration) error
	DelAdminOnlineToken(ctx context.Context, adminID uint64, token string) error
	ClearAdminAllOnlineToken(ctx context.Context, adminID uint64) error
	AddAdminTokenToBlackList(ctx context.Context, token string, exp time.Duration) error
	IsAdminTokenBlacklisted(ctx context.Context, token string) (bool, error)
}
