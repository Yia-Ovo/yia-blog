package repo

import (
	"blog-server/domain/do"
	"context"
)

// ArticlePublicStateReader 是可选的轻量可见性读取能力。
// 独立于 ArticleMysqlRepo，使旧实现可继续通过完整查询执行相同领域规则。
type ArticlePublicStateReader interface {
	// GetArticlePublicState 返回未软删除文章的 ID 和发布状态；不存在返回 (nil, nil)。
	GetArticlePublicState(ctx context.Context, id uint64) (*do.ArticlePublicState, error)
}
