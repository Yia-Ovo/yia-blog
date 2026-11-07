package repo

import (
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

type AIWorkspaceFileRepo struct {
	directory string
	mu        sync.Mutex
}

func NewAIWorkspaceFileRepo(directory string) *AIWorkspaceFileRepo {
	return &AIWorkspaceFileRepo{directory: directory}
}

func (r *AIWorkspaceFileRepo) Update(ctx context.Context, adminID uint64, change func(*do.AIWorkspaceState) error) error {
	if r.directory == "" || adminID == 0 {
		return errors.New("workspace storage unavailable")
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
	state := do.AIWorkspaceState{AdminID: adminID, Jobs: []do.AICodeJob{}}
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || info.Size() > 16<<20 {
			return errors.New("invalid workspace state")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var value model.AIWorkspaceModel
		if json.Unmarshal(data, &value) != nil || value.Version != 1 || value.AdminID != adminID {
			return errors.New("invalid workspace owner")
		}
		state = convert.AIWorkspaceFromModel(value)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	before, err := json.Marshal(convert.AIWorkspaceToModel(&state))
	if err != nil {
		return err
	}
	if err := change(&state); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := json.Marshal(convert.AIWorkspaceToModel(&state))
	if err != nil {
		return err
	}
	if len(data) > 16<<20 {
		return errors.New("workspace state too large")
	}
	if string(data) == string(before) {
		return nil
	}
	file, err := os.CreateTemp(r.directory, ".workspace-*")
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
