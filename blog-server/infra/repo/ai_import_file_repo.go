package repo

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/infra/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
)

var aiImportID = regexp.MustCompile(`^[a-z0-9-]{16,64}$`)

// AIImportFileRepo 保存单进程同步副本；重复上传原子替换，不向静态目录导出。
type AIImportFileRepo struct {
	directory string
	mu        sync.Mutex
}

func NewAIImportFileRepo(directory string) *AIImportFileRepo {
	return &AIImportFileRepo{directory: directory}
}

func (r *AIImportFileRepo) Get(ctx context.Context, adminID uint64, id string) (*do.AIImportedThread, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r.directory == "" || adminID == 0 || !aiImportID.MatchString(id) {
		return nil, common.ParamErr
	}
	path := filepath.Join(r.directory, fmt.Sprintf("admin-%d", adminID), id+".json")
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, common.ParamErr.WithMsg("本地对话尚未同步或已移除。")
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > consts.AIImportMaxBytes {
		return nil, errors.New("invalid imported thread file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, consts.AIImportMaxBytes+1))
	if err != nil || len(data) > consts.AIImportMaxBytes {
		return nil, errors.New("imported thread exceeds size limit")
	}
	var stored model.AIImportedThreadModel
	if json.Unmarshal(data, &stored) != nil || stored.Version != 1 || stored.ID != id || len(stored.Messages) > consts.AIImportMaxMessages {
		return nil, errors.New("invalid imported thread format")
	}
	value := &do.AIImportedThread{AdminID: adminID, ID: id, Title: RedactAIKnowledge(stored.Title), Workspace: stored.Workspace, UpdatedAt: stored.UpdatedAt, Messages: []do.AIMessage{}}
	for _, m := range stored.Messages {
		if (m.Role != "user" && m.Role != "assistant") || len(m.Content) > 100000 {
			return nil, errors.New("invalid imported message")
		}
		value.Messages = append(value.Messages, do.AIMessage{ID: m.ID, Role: m.Role, Content: RedactAIKnowledge(m.Content), Status: consts.AIStatusCompleted, CreateAt: m.CreateAt})
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return value, nil
}

func (r *AIImportFileRepo) List(ctx context.Context, adminID uint64) ([]do.AIImportedThread, error) {
	result := []do.AIImportedThread{}
	if r.directory == "" {
		return result, nil
	}
	entries, err := os.ReadDir(filepath.Join(r.directory, fmt.Sprintf("admin-%d", adminID)))
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	if len(entries) > consts.AIImportMaxThreads {
		return nil, errors.New("too many imported threads")
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() && filepath.Ext(entry.Name()) == ".json" {
			value, err := r.Get(ctx, adminID, entry.Name()[:len(entry.Name())-5])
			if err != nil {
				return nil, err
			}
			value.Messages = nil
			result = append(result, *value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return result, nil
}

func (r *AIImportFileRepo) Put(ctx context.Context, value do.AIImportedThread) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.directory == "" || value.AdminID == 0 || !aiImportID.MatchString(value.ID) {
		return common.ParamErr
	}
	directory := filepath.Join(r.directory, fmt.Sprintf("admin-%d", value.AdminID))
	for _, dir := range []string{r.directory, directory} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		info, err := os.Lstat(dir)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("invalid imported thread directory")
		}
	}
	path := filepath.Join(directory, value.ID+".json")
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return errors.New("invalid imported thread target")
		}
		old, err := r.Get(ctx, value.AdminID, value.ID)
		if err != nil {
			return err
		}
		// 延迟上传不能用旧快照覆盖服务器中的新历史。
		if old.UpdatedAt.After(value.UpdatedAt) {
			return nil
		}
	} else if errors.Is(err, os.ErrNotExist) {
		entries, err := os.ReadDir(directory)
		if err != nil {
			return err
		}
		if len(entries) >= consts.AIImportMaxThreads {
			return common.ParamErr.WithMsg("同步会话数量已达上限。")
		}
	} else {
		return err
	}
	stored := model.AIImportedThreadModel{Version: 1, ID: value.ID, Title: RedactAIKnowledge(value.Title), Workspace: value.Workspace, UpdatedAt: value.UpdatedAt, Messages: []model.AIMessageModel{}}
	for _, m := range value.Messages {
		stored.Messages = append(stored.Messages, model.AIMessageModel{ID: m.ID, Role: m.Role, Content: RedactAIKnowledge(m.Content), Status: consts.AIStatusCompleted, CreateAt: m.CreateAt})
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	if len(data) > consts.AIImportMaxBytes {
		return common.ParamErr.WithMsg("同步会话超过 8MB，请减少历史范围。")
	}
	if current, err := os.ReadFile(path); err == nil && string(current) == string(data) {
		return nil
	}
	file, err := os.CreateTemp(directory, ".import-*")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if err = file.Chmod(0600); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Rename(temp, path)
}
