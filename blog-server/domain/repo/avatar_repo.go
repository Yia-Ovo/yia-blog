package repo

import (
	"context"
	"errors"
)

var ErrAvatarNotFound = errors.New("avatar image not found")

// AvatarRepo 保存已经规范化的 PNG；名称由内容哈希生成，不接受客户端文件名。
type AvatarRepo interface {
	Save(ctx context.Context, png []byte) (string, error)
	Read(ctx context.Context, name string) ([]byte, error)
}
