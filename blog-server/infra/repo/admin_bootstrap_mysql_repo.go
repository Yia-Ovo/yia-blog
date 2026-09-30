package repo

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type adminBootstrapMysqlRepo struct{ db *gorm.DB }

// NewAdminBootstrapMysqlRepo 将首次初始化固定在同一个 MySQL 连接中。
func NewAdminBootstrapMysqlRepo(db *gorm.DB) domainrepo.AdminBootstrapRepo {
	return &adminBootstrapMysqlRepo{db: db}
}

func (r *adminBootstrapMysqlRepo) WithExclusiveAdmin(ctx context.Context, callback func(domainrepo.AdminMysqlRepo, bool) error) (result error) {
	pool, err := r.db.DB()
	if err != nil {
		return err
	}
	conn, err := pool.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	var database string
	if err := conn.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&database); err != nil {
		return err
	}
	// MySQL 命名锁上限 64 字符；哈希把数据库名纳入范围，避免不同博客库互相阻塞。
	lockName := fmt.Sprintf("%x", sha256.Sum256([]byte(consts.BootstrapLockPrefix+database)))
	var acquired sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", lockName, consts.BootstrapLockWaitSeconds).Scan(&acquired); err != nil {
		// 获取回复丢失时锁可能已在服务器生效；丢弃连接，不让持锁会话回池。
		_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		return err
	}
	if !acquired.Valid || acquired.Int64 != 1 {
		return errors.New("无法取得管理员初始化锁")
	}
	defer func() {
		// 即使业务 context 已取消，也在原连接上尝试解锁。
		cleanup, cancel := context.WithTimeout(context.Background(), consts.BootstrapUnlockTimeout)
		defer cancel()
		var released sql.NullInt64
		err := conn.QueryRowContext(cleanup, "SELECT RELEASE_LOCK(?)", lockName).Scan(&released)
		if err != nil || !released.Valid || released.Int64 != 1 {
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
			if result == nil {
				result = errors.New("管理员初始化锁释放未确认，请先检查账号是否已创建")
			}
		}
	}()

	var existing bool
	// 故意不加 deleted_at 条件：软删除账号也代表这不是初次初始化。
	if err := conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM `admin`)").Scan(&existing); err != nil {
		return err
	}
	lockedDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn: conn, SkipInitializeWithVersion: true,
	}), &gorm.Config{DisableAutomaticPing: true, Logger: r.db.Logger})
	if err != nil {
		return err
	}
	return callback(NewAdminMysqlRepo(lockedDB), existing)
}
