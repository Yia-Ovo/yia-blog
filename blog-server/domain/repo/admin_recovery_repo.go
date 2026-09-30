package repo

import (
	"context"
	"errors"
)

// ErrAdminRecoveryConflict 表示读取后账号发生变化，恢复不得覆盖并发修改。
var ErrAdminRecoveryConflict = errors.New("管理员已变化，密码恢复未写入，请重新检查账号")

// AdminRecoveryAccount 仅用于本机维护恢复；PasswordHash 不得输出或序列化。
type AdminRecoveryAccount struct {
	ID           uint64
	Name         string
	Role         int
	Status       int
	PasswordHash string `json:"-"`
	TokenVersion uint64
}

// AdminRecoveryRepo 只提供查询已有账号和有条件替换密码，不允许创建或变更权限。
type AdminRecoveryRepo interface {
	FindByName(ctx context.Context, name string) (*AdminRecoveryAccount, error)
	ReplacePassword(ctx context.Context, expected *AdminRecoveryAccount, newHash string) error
}
