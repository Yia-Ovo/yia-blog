package repo

import (
	"blog-server/common/consts"
	"blog-server/common/utils/convert"
	"blog-server/domain/do"
	"blog-server/infra/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// AIStateFileRepo 适合当前单进程部署；多副本上线前须换为事务数据库仓储。
type AIStateFileRepo struct {
	directory string
	mu        sync.Mutex
}

func NewAIStateFileRepo(directory string) *AIStateFileRepo {
	return &AIStateFileRepo{directory: directory}
}

func (r *AIStateFileRepo) Update(ctx context.Context, adminID uint64, change func(*do.AIState) error) error {
	if adminID == 0 || r.directory == "" {
		return errors.New("invalid AI storage configuration")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(r.directory, 0700); err != nil {
		return err
	}
	path := filepath.Join(r.directory, fmt.Sprintf("admin-%d.json", adminID))
	if info, statErr := os.Lstat(path); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || info.Size() > consts.AIStateMaxBytes {
			return errors.New("invalid AI state file")
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	state := do.AIState{AdminID: adminID, Conversations: []do.AIConversation{}}
	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) > consts.AIStateMaxBytes {
			return errors.New("AI state exceeds size limit")
		}
		var stored model.AIStateModel
		if err = json.Unmarshal(data, &stored); err != nil {
			return err
		}
		if (stored.Version != 1 && stored.Version != 2 && stored.Version != 3) || stored.AdminID != adminID {
			return errors.New("invalid AI state owner or version")
		}
		state = convert.AIModelToState(stored)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	before, err := json.Marshal(convert.AIStateToModel(&state))
	if err != nil {
		return err
	}
	if err := change(&state); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err = json.Marshal(convert.AIStateToModel(&state))
	if err != nil {
		return err
	}
	if len(data) > consts.AIStateMaxBytes {
		return errors.New("AI state exceeds size limit")
	}
	if string(before) == string(data) {
		return nil
	}
	file, err := os.CreateTemp(r.directory, ".ai-state-*")
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
	return os.Rename(temp, path)
}
