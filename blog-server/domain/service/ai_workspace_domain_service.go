package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

var aiCommitHash = regexp.MustCompile(`^[a-f0-9]{40}$`)

func (s *AIDomainService) WorkspaceOwners() []uint64 {
	return slices.Clone(s.policy.PrivateKnowledgeAdminIDs)
}

func (s *AIDomainService) AuthorizeWorkspace(state *do.AIWorkspaceState, access do.AIAccess) error {
	if s.PrivateWorkspace(access) != nil || state.AdminID != access.AdminID {
		return common.PermissionErr
	}
	return nil
}

func (s *AIDomainService) CreateCodeJob(state *do.AIWorkspaceState, access do.AIAccess, id, prompt, mode, contextText string, now time.Time) (*do.AICodeJob, error) {
	if err := s.AuthorizeWorkspace(state, access); err != nil {
		return nil, err
	}
	mode, err := s.ContextMode(access, mode)
	if err != nil {
		return nil, err
	}
	prompt = strings.TrimSpace(prompt)
	if mode == consts.AIContextGeneral && contextText != "" {
		return nil, common.PermissionErr
	}
	if !aiIDPattern.MatchString(id) || prompt == "" || !utf8.ValidString(prompt) || utf8.RuneCountInString(prompt) > 4000 || utf8.RuneCountInString(contextText) > 18000 {
		return nil, common.ParamErr
	}
	for i := range state.Jobs {
		job := &state.Jobs[i]
		if job.ID == id {
			if job.Prompt != prompt || job.ContextMode != mode {
				return nil, common.ParamErr
			}
			return job, nil
		}
	}
	day := now.In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006-01-02")
	if state.Day != day {
		state.Day = day
		state.Tasks = 0
	}
	if state.Tasks >= consts.AIJobDailyLimit {
		return nil, common.Errno{Code: 429, Msg: "今日代码任务次数已达上限。"}
	}
	if len(state.Jobs) >= consts.AIJobMaxRecords {
		return nil, common.ParamErr.WithMsg("代码任务记录已达上限。")
	}
	state.Tasks++
	state.Jobs = append(state.Jobs, do.AICodeJob{ID: id, Prompt: prompt, ContextMode: mode, Context: contextText, Status: consts.AIJobQueued, Branch: "codex/ai-" + id, CreatedAt: now, UpdatedAt: now})
	return &state.Jobs[len(state.Jobs)-1], nil
}

// PublishReviewedCodeJob 将发布许可绑定到刚刚查看的差异；合并与部署不属于这个动作。
func (s *AIDomainService) PublishReviewedCodeJob(state *do.AIWorkspaceState, access do.AIAccess, id, digest string, now time.Time) error {
	if err := s.AuthorizeWorkspace(state, access); err != nil {
		return err
	}
	for _, j := range state.Jobs {
		if j.ID == id {
			if digest == "" || subtle.ConstantTimeCompare([]byte(digest), []byte(j.ReviewDigest)) != 1 {
				return common.ParamErr.WithMsg("差异已变化，请重新查看后再发布分支。")
			}
			return s.publishCodeJob(state, access, id, now)
		}
	}
	return common.ParamErr
}

func (s *AIDomainService) ExpireCodeJobs(state *do.AIWorkspaceState, now time.Time) {
	for i := range state.Jobs {
		j := &state.Jobs[i]
		if (j.Status == consts.AIJobRunning || j.Status == consts.AIJobPublishing) && !j.LeaseUntil.After(now) {
			j.Status = consts.AIJobFailed
			j.Summary = "执行服务租约已到期，请检查差异后重新创建任务。"
			j.Lease = ""
			j.UpdatedAt = now
		}
	}
}

func (s *AIDomainService) ClaimCodeJob(state *do.AIWorkspaceState, access do.AIAccess, lease string, now time.Time) (*do.AICodeJob, error) {
	if err := s.AuthorizeWorkspace(state, access); err != nil {
		return nil, err
	}
	if lease == "" || len(lease) > 64 {
		return nil, common.ParamErr
	}
	s.ExpireCodeJobs(state, now)
	for i := range state.Jobs {
		if state.Jobs[i].Status == consts.AIJobRunning || state.Jobs[i].Status == consts.AIJobPublishing {
			return nil, nil
		}
	}
	for i := range state.Jobs {
		j := &state.Jobs[i]
		if j.Status != consts.AIJobQueued && j.Status != consts.AIJobPublishQueued {
			continue
		}
		if j.Status == consts.AIJobPublishQueued {
			j.Status = consts.AIJobPublishing
		} else {
			j.Status = consts.AIJobRunning
		}
		j.Lease = lease
		j.LeaseUntil = now.Add(consts.AIJobLeaseDuration)
		j.UpdatedAt = now
		value := *j
		return &value, nil
	}
	return nil, nil
}

func (s *AIDomainService) publishCodeJob(state *do.AIWorkspaceState, access do.AIAccess, id string, now time.Time) error {
	if err := s.AuthorizeWorkspace(state, access); err != nil {
		return err
	}
	for i := range state.Jobs {
		j := &state.Jobs[i]
		if j.ID != id {
			continue
		}
		if j.Status == consts.AIJobPublishQueued || j.Status == consts.AIJobPublishing || j.Status == consts.AIJobCompleted {
			return nil
		}
		if j.Status != consts.AIJobReview || !j.TestsPassed {
			return common.ParamErr.WithMsg("必须先完成测试并审核差异，才能发布分支。")
		}
		j.Status = consts.AIJobPublishQueued
		j.UpdatedAt = now
		return nil
	}
	return common.ParamErr
}

func (s *AIDomainService) ReportCodeJob(state *do.AIWorkspaceState, access do.AIAccess, id, lease, phase, summary, diff, checks, base string, passed bool, now time.Time) error {
	if err := s.AuthorizeWorkspace(state, access); err != nil {
		return err
	}
	if utf8.RuneCountInString(summary) > 4000 || utf8.RuneCountInString(diff) > 100000 || utf8.RuneCountInString(checks) > 12000 {
		return common.ParamErr
	}
	for i := range state.Jobs {
		j := &state.Jobs[i]
		if j.ID != id {
			continue
		}
		if (j.Status != consts.AIJobRunning && j.Status != consts.AIJobPublishing) || !j.LeaseUntil.After(now) || lease == "" || subtle.ConstantTimeCompare([]byte(lease), []byte(j.Lease)) != 1 {
			return common.PermissionErr
		}
		if phase == consts.AIJobReview {
			if j.Status != consts.AIJobRunning || !aiCommitHash.MatchString(base) || strings.TrimSpace(diff) == "" {
				return common.ParamErr
			}
			j.Diff = diff
			j.Checks = checks
			j.TestsPassed = passed
			j.BaseRevision = base
			digest := sha256.Sum256([]byte(base + "\n" + diff + "\n" + checks))
			j.ReviewDigest = hex.EncodeToString(digest[:])
		}
		if phase == consts.AIJobCompleted {
			if j.Status != consts.AIJobPublishing || !j.TestsPassed {
				return common.PermissionErr
			}
			j.URL = "https://github.com/Yia-Ovo/yia-blog/compare/main..." + j.Branch
		}
		if phase != consts.AIJobFailed && phase != consts.AIJobReview && phase != consts.AIJobCompleted {
			return common.ParamErr
		}
		j.Status = phase
		j.Summary = summary
		j.Lease = ""
		j.UpdatedAt = now
		return nil
	}
	return common.ParamErr
}

func (s *AIDomainService) ReportSync(state *do.AIWorkspaceState, access do.AIAccess, phase, revision, message string, now time.Time) error {
	if err := s.AuthorizeWorkspace(state, access); err != nil {
		return err
	}
	if utf8.RuneCountInString(message) > 1000 || (revision != "" && !aiCommitHash.MatchString(revision)) {
		return common.ParamErr
	}
	if phase != "requested" && phase != "syncing" && phase != "synced" && phase != "error" {
		return common.ParamErr
	}
	state.Sync = do.AISyncStatus{Phase: phase, Revision: revision, Message: message, UpdatedAt: now}
	return nil
}
