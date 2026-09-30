package repo

import (
	"blog-server/domain/do"
	"context"
	"errors"
)

var ErrArticleImageNotFound = errors.New("article image not found")

// ArticleImageRepo 只保存规范化的 PNG/JPEG；名称由内容哈希生成，不接受用户文件名。
type ArticleImageRepo interface {
	Save(ctx context.Context, data []byte) (string, error)
	Read(ctx context.Context, name string) (*do.ArticleImage, error)
}
