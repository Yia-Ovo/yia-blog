package repo

import (
	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"sync"
)

var appearanceID = regexp.MustCompile(`^[a-f0-9]{20}\.jpg$`)

// AppearanceFileRepo 从受控 FS 读取内容哈希命名的 JPEG；目录只加载一次。
type AppearanceFileRepo struct {
	files   fs.FS
	once    sync.Once
	ids     []string
	allowed map[string]struct{}
	err     error
}

func NewAppearanceFileRepo(files fs.FS) *AppearanceFileRepo {
	return &AppearanceFileRepo{files: files}
}

func (r *AppearanceFileRepo) load() {
	data, err := fs.ReadFile(r.files, consts.AppearanceCatalogPath)
	if err != nil {
		r.err = err
		return
	}
	if err := json.Unmarshal(data, &r.ids); err != nil {
		r.err = err
		return
	}
	r.allowed = make(map[string]struct{}, len(r.ids))
	for _, id := range r.ids {
		if !appearanceID.MatchString(id) {
			r.err = fmt.Errorf("invalid appearance catalog ID")
			return
		}
		if _, exists := r.allowed[id]; exists {
			r.err = fmt.Errorf("duplicate appearance catalog ID")
			return
		}
		r.allowed[id] = struct{}{}
	}
	sort.Strings(r.ids)
}

func (r *AppearanceFileRepo) List(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.once.Do(r.load)
	return append([]string(nil), r.ids...), r.err
}

func (r *AppearanceFileRepo) Read(ctx context.Context, id string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.once.Do(r.load)
	if r.err != nil {
		return nil, r.err
	}
	if _, ok := r.allowed[id]; !ok {
		return nil, domainrepo.ErrAppearanceImageNotFound
	}
	data, err := fs.ReadFile(r.files, consts.AppearanceAssetPrefix+id)
	if err != nil {
		return nil, err
	}
	// URL 长期缓存的前提是文件名与内容匹配，避免换图后命中旧缓存。
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:])[:20]+".jpg" != id {
		return nil, fmt.Errorf("appearance content hash mismatch")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}
