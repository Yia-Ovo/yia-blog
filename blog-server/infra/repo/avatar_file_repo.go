package repo

import (
	"blog-server/common/consts"
	domainrepo "blog-server/domain/repo"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

var storedAvatarName = regexp.MustCompile(`^[a-f0-9]{64}\.png$`)

// AvatarFileRepo 将内容哈希命名的 PNG 保存在独立的持久化目录，不暴露文件系统路径。
type AvatarFileRepo struct {
	directory string
	writeMu   sync.Mutex
}

func NewAvatarFileRepo(directory string) *AvatarFileRepo {
	if directory == "" {
		directory = consts.AvatarDefaultDirectory
	}
	return &AvatarFileRepo{directory: directory}
}

func (r *AvatarFileRepo) Save(ctx context.Context, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(data) == 0 || len(data) > consts.AvatarMaxStoredBytes {
		return "", errors.New("invalid normalized avatar size")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width != config.Height || config.Width < 1 || config.Width > consts.AvatarOutputDimension {
		return "", errors.New("invalid normalized avatar")
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		return "", errors.New("invalid normalized PNG content")
	}
	sum := sha256.Sum256(data)
	name := hex.EncodeToString(sum[:]) + ".png"
	// 避免 Windows 上同时替换和读取同一个文件时的共享冲突；锁只保护落盘阶段。
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if existing, err := r.Read(ctx, name); err == nil {
		if bytes.Equal(existing, data) {
			return name, nil
		}
		return "", errors.New("avatar hash collision")
	} else if !errors.Is(err, domainrepo.ErrAvatarNotFound) {
		return "", err
	}
	if err := os.MkdirAll(r.directory, 0750); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(r.directory, ".avatar-*.tmp")
	if err != nil {
		return "", err
	}
	tempName := file.Name()
	defer os.Remove(tempName)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return "", err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return "", err
	}
	if err = file.Close(); err != nil {
		return "", err
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	if err = os.Rename(tempName, filepath.Join(r.directory, name)); err != nil {
		// Windows 不能覆盖已存在的文件；并发相同内容上传可复用完整的目标文件。
		if existing, readErr := r.Read(ctx, name); readErr == nil && bytes.Equal(existing, data) {
			return name, nil
		}
		return "", err
	}
	return name, nil
}

func (r *AvatarFileRepo) Read(ctx context.Context, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !storedAvatarName.MatchString(name) {
		return nil, domainrepo.ErrAvatarNotFound
	}
	path := filepath.Join(r.directory, name)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, domainrepo.ErrAvatarNotFound
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > consts.AvatarMaxStoredBytes {
		return nil, domainrepo.ErrAvatarNotFound
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, consts.AvatarMaxStoredBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > consts.AvatarMaxStoredBytes {
		return nil, domainrepo.ErrAvatarNotFound
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:])+".png" != name {
		return nil, fmt.Errorf("avatar content hash mismatch")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width != config.Height || config.Width < 1 || config.Width > consts.AvatarOutputDimension {
		return nil, domainrepo.ErrAvatarNotFound
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		return nil, domainrepo.ErrAvatarNotFound
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}
