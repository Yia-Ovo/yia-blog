package repo

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	domainrepo "blog-server/domain/repo"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var storedArticleImageName = regexp.MustCompile(`^[a-f0-9]{64}\.(png|jpg)$`)

// ArticleImageFileRepo 将内容哈希命名的图片原子保存到独立持久化目录。
type ArticleImageFileRepo struct {
	directory string
	writeMu   sync.Mutex
}

func NewArticleImageFileRepo(directory string) *ArticleImageFileRepo {
	if directory == "" {
		directory = consts.ArticleImageDefaultDirectory
	}
	return &ArticleImageFileRepo{directory: directory}
}

func decodedStoredArticleImage(data []byte) (*do.ArticleImage, string, error) {
	if len(data) == 0 || len(data) > consts.ArticleImageMaxStoredBytes {
		return nil, "", errors.New("invalid normalized image size")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > consts.ArticleImageOutputDimension || cfg.Height > consts.ArticleImageOutputDimension || int64(cfg.Width)*int64(cfg.Height) > consts.ArticleImageMaxPixels {
		return nil, "", errors.New("invalid normalized article image")
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return nil, "", err
	}
	ext, contentType := ".png", "image/png"
	if format == "jpeg" {
		ext, contentType = ".jpg", "image/jpeg"
	}
	return &do.ArticleImage{Data: data, Width: cfg.Width, Height: cfg.Height, ContentType: contentType}, ext, nil
}

func (r *ArticleImageFileRepo) Save(ctx context.Context, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	_, ext, err := decodedStoredArticleImage(data)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	name := hex.EncodeToString(sum[:]) + ext
	// 同一仓储实例串行落盘；不同实例的同名发布由文件系统原子操作处理。
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if existing, err := r.Read(ctx, name); err == nil {
		if bytes.Equal(existing.Data, data) {
			return name, nil
		}
		return "", errors.New("article image hash collision")
	} else if !errors.Is(err, domainrepo.ErrArticleImageNotFound) {
		return "", err
	}
	if err := os.MkdirAll(r.directory, 0750); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(r.directory, ".article-image-*.tmp")
	if err != nil {
		return "", err
	}
	tempName := file.Name()
	defer os.Remove(tempName)
	if _, err := file.Write(data); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := publishArticleImageFile(tempName, filepath.Join(r.directory, name)); err != nil {
		// 不同实例并发保存同一内容时，Windows 不允许替换已有文件，可复用已完整落盘的图片。
		if existing, readErr := r.Read(ctx, name); readErr == nil && bytes.Equal(existing.Data, data) {
			return name, nil
		}
		return "", err
	}
	return name, nil
}

func (r *ArticleImageFileRepo) Read(ctx context.Context, name string) (*do.ArticleImage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !storedArticleImageName.MatchString(name) {
		return nil, domainrepo.ErrArticleImageNotFound
	}
	path := filepath.Join(r.directory, name)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, domainrepo.ErrArticleImageNotFound
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > consts.ArticleImageMaxStoredBytes {
		return nil, domainrepo.ErrArticleImageNotFound
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, consts.ArticleImageMaxStoredBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > consts.ArticleImageMaxStoredBytes {
		return nil, domainrepo.ErrArticleImageNotFound
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != strings.TrimSuffix(name, filepath.Ext(name)) {
		return nil, fmt.Errorf("article image content hash mismatch")
	}
	result, ext, err := decodedStoredArticleImage(data)
	if err != nil || filepath.Ext(name) != ext {
		return nil, domainrepo.ErrArticleImageNotFound
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
