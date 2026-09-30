package repo

import (
	"context"
	"errors"
	"time"

	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"
	"blog-server/infra/model"

	"gorm.io/gorm"
)

type adminRecoveryMysqlRepo struct{ db *gorm.DB }

// NewAdminRecoveryMysqlRepo 使用本地命令提供的连接；调用方必须关闭 SQL 日志以保护哈希。
func NewAdminRecoveryMysqlRepo(db *gorm.DB) domainrepo.AdminRecoveryRepo {
	return &adminRecoveryMysqlRepo{db: db}
}

func (r *adminRecoveryMysqlRepo) FindByName(ctx context.Context, name string) (*domainrepo.AdminRecoveryAccount, error) {
	var admin model.AdminModel
	err := r.db.WithContext(ctx).
		Select("id", "admin_name", "role", "status", "password", "token_version").
		Where("admin_name = ? AND deleted_at IS NULL", name).Take(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domainrepo.AdminRecoveryAccount{
		ID: admin.ID, Name: admin.AdminName, Role: admin.Role, Status: admin.Status,
		PasswordHash: admin.Password, TokenVersion: admin.TokenVersion,
	}, nil
}

func (r *adminRecoveryMysqlRepo) ReplacePassword(ctx context.Context, expected *domainrepo.AdminRecoveryAccount, newHash string) error {
	if expected == nil || expected.ID == 0 {
		return domainrepo.ErrAdminRecoveryConflict
	}
	// 仅替换读取时的版本；角色、状态、删除情况也在写入时再次约束。
	result := r.db.WithContext(ctx).Model(&model.AdminModel{}).
		Where("id = ? AND admin_name = ? AND role = ? AND status = ? AND deleted_at IS NULL AND token_version = ? AND password = ?",
			expected.ID, expected.Name, consts.AdminRoleSuper, consts.AdminStatusEnabled, expected.TokenVersion, expected.PasswordHash).
		Updates(map[string]interface{}{
			"password":      newHash,
			"token_version": gorm.Expr("token_version + ?", 1),
			"update_at":     time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return domainrepo.ErrAdminRecoveryConflict
	}
	return nil
}
