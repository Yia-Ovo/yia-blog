package repo

import (
	"blog-server/infra/model"
	"context"
	"time"
)

type AdminMysqlRepo interface {
	CreateAdmin(ctx context.Context, adminModel *model.AdminModel) error
	FindAdminByName(ctx context.Context, adminName string) (*model.AdminModel, error)
	FindAdminByID(ctx context.Context, adminID uint64) (*model.AdminModel, error)
	UpdateAdminInfo(ctx context.Context, adminModel *model.AdminModel) error
	UpdateLastLoginInfo(ctx context.Context, adminID uint64, loginAt *time.Time) error
}
