package repo

import (
	"blog-server/domain/do"
	"context"
)

// AIStateRepo 在锁内读、修改并原子保存管理员资料，回调失败时不提交。
type AIStateRepo interface {
	Update(context.Context, uint64, func(*do.AIState) error) error
}

type AIKnowledgeRepo interface {
	Search(context.Context, string) ([]do.AISource, error)
	Info(context.Context) (int, error)
}

type AIModelRepo interface {
	Reply(context.Context, string, []do.AIModelMessage) (string, error)
}
