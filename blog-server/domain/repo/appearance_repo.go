package repo

import (
	"context"
	"errors"
)

var ErrAppearanceImageNotFound = errors.New("appearance image not found")

// AppearanceRepo 只读取已纳入目录的图片，不接受任意文件路径。
type AppearanceRepo interface {
	List(context.Context) ([]string, error)
	Read(context.Context, string) ([]byte, error)
}
