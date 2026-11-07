package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/convert"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

// ConfigureWorkspace 在 IOC 装配独立任务仓储；WorkerKey 不复用模型密钥或管理员令牌。
func (s *AIAppService) ConfigureWorkspace(storage repo.AIWorkspaceRepo, enabled bool, key string) {
	s.workspace, s.workerEnabled, s.workerKey = storage, enabled && len(key) >= 32, key
}

func (s *AIAppService) workspaceAccess(ctx context.Context, id uint64) (do.AIAccess, error) {
	access, err := s.access(ctx, id)
	if err != nil {
		return access, err
	}
	if err = s.domain.PrivateWorkspace(access); err != nil {
		return access, err
	}
	if s.workspace == nil {
		return access, common.ServerErr.WithMsg("开发工作区尚未配置。")
	}
	return access, nil
}

// Workspace 每次读取都核验现行身份；查看差异不触发执行。
func (s *AIAppService) Workspace(ctx context.Context, id uint64) (*dto.AIWorkspace, error) {
	access, err := s.workspaceAccess(ctx, id)
	if err != nil {
		return nil, err
	}
	result := &dto.AIWorkspace{Enabled: s.workerEnabled, DailyLimit: consts.AIJobDailyLimit, Jobs: []dto.AICodeJob{}}
	err = s.workspace.Update(ctx, id, func(state *do.AIWorkspaceState) error {
		if err := s.domain.AuthorizeWorkspace(state, access); err != nil {
			return err
		}
		s.domain.ExpireCodeJobs(state, time.Now())
		for _, job := range state.Jobs {
			result.Jobs = append(result.Jobs, convert.AICodeJobToDTO(job))
		}
		return nil
	})
	sort.Slice(result.Jobs, func(i, j int) bool { return result.Jobs[i].CreatedAt.After(result.Jobs[j].CreatedAt) })
	return result, aiStorageError(err)
}

// CreateCodeJobCommand 的账号来自鉴权，不能由浏览器指定执行身份。
type CreateCodeJobCommand struct {
	AdminID                 uint64
	ID, Prompt, ContextMode string
}

func (s *AIAppService) CreateCodeJob(ctx context.Context, cmd CreateCodeJobCommand) (*dto.AICodeJob, error) {
	access, err := s.workspaceAccess(ctx, cmd.AdminID)
	if err != nil {
		return nil, err
	}
	if !s.workerEnabled {
		return nil, common.ServerErr.WithMsg("服务器执行服务尚未启用。")
	}
	mode, err := s.domain.ContextMode(access, cmd.ContextMode)
	if err != nil {
		return nil, err
	}
	var contextText strings.Builder
	if mode == consts.AIContextProject {
		sources, err := s.knowledge.Search(ctx, cmd.Prompt)
		if err != nil {
			return nil, aiStorageError(err)
		}
		for _, source := range sources {
			contextText.WriteString(source.Path + "\n" + source.Excerpt + "\n")
		}
	}
	var result dto.AICodeJob
	err = s.workspace.Update(ctx, cmd.AdminID, func(state *do.AIWorkspaceState) error {
		job, err := s.domain.CreateCodeJob(state, access, cmd.ID, cmd.Prompt, mode, contextText.String(), time.Now())
		if err == nil {
			result = convert.AICodeJobToDTO(*job)
		}
		return err
	})
	return &result, aiStorageError(err)
}

func (s *AIAppService) PublishCodeJob(ctx context.Context, id uint64, jobID, digest string) error {
	access, err := s.workspaceAccess(ctx, id)
	if err != nil {
		return err
	}
	if !s.workerEnabled {
		return common.ServerErr.WithMsg("服务器执行服务尚未启用。")
	}
	return aiStorageError(s.workspace.Update(ctx, id, func(state *do.AIWorkspaceState) error {
		return s.domain.PublishReviewedCodeJob(state, access, jobID, digest, time.Now())
	}))
}

func (s *AIAppService) workerAuthenticate(key string) error {
	if !s.enabled || !s.workerEnabled || len(key) < 32 || subtle.ConstantTimeCompare([]byte(key), []byte(s.workerKey)) != 1 {
		return common.PermissionErr
	}
	return nil
}

// ClaimCodeJob 只交给独立执行服务，领取时仍核验当前管理员而非任务创建时的角色。
func (s *AIAppService) ClaimCodeJob(ctx context.Context, key string) (*dto.AIWorkerJob, error) {
	if err := s.workerAuthenticate(key); err != nil {
		return nil, err
	}
	var result *dto.AIWorkerJob
	for _, id := range s.domain.WorkspaceOwners() {
		access, err := s.workspaceAccess(ctx, id)
		if err != nil {
			continue
		}
		var random [24]byte
		if _, err = rand.Read(random[:]); err != nil {
			return nil, common.ServerErr
		}
		err = s.workspace.Update(ctx, id, func(state *do.AIWorkspaceState) error {
			job, err := s.domain.ClaimCodeJob(state, access, hex.EncodeToString(random[:]), time.Now())
			if err == nil && job != nil {
				result = &dto.AIWorkerJob{AdminID: id, ID: job.ID, Prompt: job.Prompt, Context: job.Context, ContextMode: job.ContextMode, Status: job.Status, Lease: job.Lease, Branch: job.Branch, BaseRevision: job.BaseRevision, Diff: job.Diff}
			}
			return err
		})
		if err != nil {
			return nil, aiStorageError(err)
		}
		if result != nil {
			return result, nil
		}
	}
	return nil, nil
}

type CodeJobReportCommand struct {
	AdminID                                               uint64
	ID, Lease, Phase, Summary, Diff, Checks, BaseRevision string
	TestsPassed                                           bool
}

func (s *AIAppService) ReportCodeJob(ctx context.Context, key string, cmd CodeJobReportCommand) error {
	if err := s.workerAuthenticate(key); err != nil {
		return err
	}
	access, err := s.workspaceAccess(ctx, cmd.AdminID)
	if err != nil {
		return err
	}
	return aiStorageError(s.workspace.Update(ctx, cmd.AdminID, func(state *do.AIWorkspaceState) error {
		return s.domain.ReportCodeJob(state, access, cmd.ID, cmd.Lease, cmd.Phase, cmd.Summary, cmd.Diff, cmd.Checks, cmd.BaseRevision, cmd.TestsPassed, time.Now())
	}))
}
