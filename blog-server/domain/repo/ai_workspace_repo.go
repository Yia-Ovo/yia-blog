package repo

import (
	"blog-server/domain/do"
	"context"
)

type AIWorkspaceRepo interface {
	Update(context.Context, uint64, func(*do.AIWorkspaceState) error) error
}
