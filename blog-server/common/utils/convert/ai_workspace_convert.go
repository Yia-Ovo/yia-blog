package convert

import (
	"blog-server/api/dto"
	"blog-server/domain/do"
	"blog-server/infra/model"
)

func AICodeJobToDTO(j do.AICodeJob) dto.AICodeJob {
	return dto.AICodeJob{ID: j.ID, Prompt: j.Prompt, ContextMode: j.ContextMode, Status: j.Status, Summary: j.Summary, Diff: j.Diff, Checks: j.Checks, Branch: j.Branch, URL: j.URL, TestsPassed: j.TestsPassed, ReviewDigest: j.ReviewDigest, CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt}
}

func AIWorkspaceToModel(s *do.AIWorkspaceState) model.AIWorkspaceModel {
	value := model.AIWorkspaceModel{Version: 1, AdminID: s.AdminID, Day: s.Day, Tasks: s.Tasks, Sync: model.AISyncModel{Phase: s.Sync.Phase, Revision: s.Sync.Revision, Message: s.Sync.Message, UpdatedAt: s.Sync.UpdatedAt}, Jobs: []model.AICodeJobModel{}}
	for _, j := range s.Jobs {
		value.Jobs = append(value.Jobs, model.AICodeJobModel{ID: j.ID, Prompt: j.Prompt, ContextMode: j.ContextMode, Status: j.Status, Context: j.Context, Summary: j.Summary, Diff: j.Diff, Checks: j.Checks, Branch: j.Branch, URL: j.URL, BaseRevision: j.BaseRevision, ReviewDigest: j.ReviewDigest, TestsPassed: j.TestsPassed, CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt, LeaseUntil: j.LeaseUntil, Lease: j.Lease})
	}
	return value
}

func AIWorkspaceFromModel(s model.AIWorkspaceModel) do.AIWorkspaceState {
	value := do.AIWorkspaceState{AdminID: s.AdminID, Day: s.Day, Tasks: s.Tasks, Sync: do.AISyncStatus{Phase: s.Sync.Phase, Revision: s.Sync.Revision, Message: s.Sync.Message, UpdatedAt: s.Sync.UpdatedAt}, Jobs: []do.AICodeJob{}}
	for _, j := range s.Jobs {
		value.Jobs = append(value.Jobs, do.AICodeJob{ID: j.ID, Prompt: j.Prompt, ContextMode: j.ContextMode, Status: j.Status, Context: j.Context, Summary: j.Summary, Diff: j.Diff, Checks: j.Checks, Branch: j.Branch, URL: j.URL, BaseRevision: j.BaseRevision, ReviewDigest: j.ReviewDigest, TestsPassed: j.TestsPassed, CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt, LeaseUntil: j.LeaseUntil, Lease: j.Lease})
	}
	return value
}
