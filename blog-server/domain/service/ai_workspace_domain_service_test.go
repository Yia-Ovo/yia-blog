package service

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"strings"
	"testing"
	"time"
)

func TestCodeJobsRequireOwnerAndExplicitPrivateMode(t *testing.T) {
	s := NewAIDomainService(nil, do.AIAccessPolicy{}, 50)
	state := &do.AIWorkspaceState{AdminID: 1}
	for _, access := range []do.AIAccess{{AdminID: 2, PrivateKnowledge: true}, {AdminID: 1}} {
		if _, err := s.CreateCodeJob(state, access, "code-task-00000001", "修改功能", "general", "", time.Now()); err == nil {
			t.Fatal("unauthorized task accepted")
		}
	}
	owner := do.AIAccess{AdminID: 1, PrivateKnowledge: true}
	if _, err := s.CreateCodeJob(state, owner, "code-task-00000001", "修改功能", "general", "private-data", time.Now()); err == nil {
		t.Fatal("general task received private context")
	}
	j, err := s.CreateCodeJob(state, owner, "code-task-00000001", "修改功能", "general", "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if j.Branch != "codex/ai-code-task-00000001" {
		t.Fatal(j.Branch)
	}
	if _, err = s.CreateCodeJob(state, owner, j.ID, j.Prompt, "general", "", time.Now()); err != nil || len(state.Jobs) != 1 || state.Tasks != 1 {
		t.Fatal("duplicate consumed quota")
	}
}

func TestCodeJobLeaseReviewAndPublishAreBoundToCurrentDiff(t *testing.T) {
	s := NewAIDomainService(nil, do.AIAccessPolicy{}, 50)
	owner := do.AIAccess{AdminID: 1, PrivateKnowledge: true}
	state := &do.AIWorkspaceState{AdminID: 1}
	now := time.Now()
	_, err := s.CreateCodeJob(state, owner, "code-task-00000001", "修改功能", "general", "", now)
	if err != nil {
		t.Fatal(err)
	}
	j, err := s.ClaimCodeJob(state, owner, "lease-secret", now)
	if err != nil || j == nil {
		t.Fatal(err)
	}
	base := strings.Repeat("a", 40)
	if s.ReportCodeJob(state, owner, j.ID, "wrong", "review", "done", "diff", "passed", base, true, now) == nil {
		t.Fatal("wrong lease accepted")
	}
	if err = s.ReportCodeJob(state, owner, j.ID, j.Lease, "review", "done", "diff", "passed", base, true, now); err != nil {
		t.Fatal(err)
	}
	digest := state.Jobs[0].ReviewDigest
	if s.PublishReviewedCodeJob(state, owner, j.ID, strings.Repeat("0", 64), now) == nil {
		t.Fatal("old review accepted")
	}
	if err = s.PublishReviewedCodeJob(state, owner, j.ID, digest, now); err != nil {
		t.Fatal(err)
	}
	j, err = s.ClaimCodeJob(state, owner, "publish-lease", now)
	if err != nil || j.Status != consts.AIJobPublishing {
		t.Fatal(err)
	}
	if s.ReportCodeJob(state, owner, j.ID, j.Lease, "review", "done", "diff", "passed", base, true, now) == nil {
		t.Fatal("publishing regressed to review")
	}
	if err = s.ReportCodeJob(state, owner, j.ID, j.Lease, "completed", "done", "", "", "", true, now); err != nil {
		t.Fatal(err)
	}
}

func TestFailedChecksAndExpiredJobsCannotPublish(t *testing.T) {
	s := NewAIDomainService(nil, do.AIAccessPolicy{}, 50)
	owner := do.AIAccess{AdminID: 1, PrivateKnowledge: true}
	now := time.Now()
	state := &do.AIWorkspaceState{AdminID: 1, Jobs: []do.AICodeJob{{ID: "code-task-00000001", Status: consts.AIJobReview, ReviewDigest: "digest", TestsPassed: false}}}
	if s.PublishReviewedCodeJob(state, owner, state.Jobs[0].ID, "digest", now) == nil {
		t.Fatal("failed tests published")
	}
	state.Jobs[0].Status = consts.AIJobRunning
	state.Jobs[0].LeaseUntil = now.Add(-time.Second)
	state.Jobs[0].Lease = "secret"
	s.ExpireCodeJobs(state, now)
	if state.Jobs[0].Status != consts.AIJobFailed || state.Jobs[0].Lease != "" {
		t.Fatal("lease not expired")
	}
}
