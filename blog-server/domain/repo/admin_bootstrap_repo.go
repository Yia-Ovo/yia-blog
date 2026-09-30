package repo

import "context"

// AdminBootstrapRepo 为一次性初始化提供独占会话，不暴露 SQL 或数据库连接类型。
type AdminBootstrapRepo interface {
	// WithExclusiveAdmin 在同一数据库的初始化锁内查询全部管理员（包含软删除）。
	// existing 只表示记录是否存在；是否允许初始化由 Domain 决定。
	// callback 中的仓储使用持锁连接，callback 完成后释放锁；查询失败不得调用 callback。
	WithExclusiveAdmin(ctx context.Context, callback func(admins AdminMysqlRepo, existing bool) error) error
}
