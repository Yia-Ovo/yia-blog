package repo

import (
	"blog-server/domain/do"
	"context"
)

// ArticlePageStateReader 以轻量查询核对分页成员、顺序和总数；查询范围由 Domain 决定。
type ArticlePageStateReader interface {
	GetArticlePageState(ctx context.Context, query *do.ArticleQuery) (*do.ArticlePageState, error)
}
