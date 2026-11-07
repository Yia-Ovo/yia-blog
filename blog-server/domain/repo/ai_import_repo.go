package repo

import (
	"blog-server/domain/do"
	"context"
)

type AIImportRepo interface {
	List(context.Context, uint64) ([]do.AIImportedThread, error)
	Get(context.Context, uint64, string) (*do.AIImportedThread, error)
	Put(context.Context, do.AIImportedThread) error
}
