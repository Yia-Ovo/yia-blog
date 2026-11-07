package service

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
	"context"
	"strings"
	"testing"
	"time"
)

type aiAdminRepo struct {
	repo.AdminMysqlRepo
	admin *model.AdminModel
}

func (r aiAdminRepo) FindAdminByID(context.Context, uint64) (*model.AdminModel, error) {
	return r.admin, nil
}

func TestAIAuthorizationAndOwnership(t *testing.T) {
	for _, sample := range []struct {
		name   string
		id     uint64
		status int
		allow  bool
	}{{"owner", 1, 1, true}, {"not allowed", 2, 1, false}, {"disabled", 1, 0, false}, {"anonymous", 0, 1, false}} {
		t.Run(sample.name, func(t *testing.T) {
			s := NewAIDomainService(NewAdminDomainService(aiAdminRepo{admin: &model.AdminModel{ID: sample.id, Status: sample.status}}), do.AIAccessPolicy{AllowedAdminIDs: []uint64{1}}, 3)
			err := s.Authorize(context.Background(), sample.id)
			if (err == nil) != sample.allow {
				t.Fatalf("authorization: %v", err)
			}
		})
	}
	s := NewAIDomainService(nil, do.AIAccessPolicy{}, 3)
	if _, err := s.Conversation(&do.AIState{AdminID: 1}, 2, "test"); err == nil {
		t.Fatal("cross-owner read accepted")
	}
}

func TestAIQuotaDedupeAndRecovery(t *testing.T) {
	s := NewAIDomainService(nil, do.AIAccessPolicy{}, 2)
	state := &do.AIState{AdminID: 1}
	now := time.Date(2026, 11, 7, 15, 50, 0, 0, time.UTC)
	c, created, err := s.PrepareTurn(state, 1, "conversation-00001", "request-000000001", "JWT", now)
	if err != nil || !created {
		t.Fatal(err)
	}
	if _, created, err = s.PrepareTurn(state, 1, c.ID, "request-000000001", "JWT", now); err != nil || created || state.Requests != 1 {
		t.Fatalf("dedupe failed: %v", err)
	}
	if _, _, err = s.PrepareTurn(state, 1, c.ID, "request-000000001", "changed", now); err == nil {
		t.Fatal("changed duplicate accepted")
	}
	if _, _, err = s.PrepareTurn(state, 1, c.ID, "request-000000002", "Redis", now); err == nil {
		t.Fatal("parallel request accepted")
	}
	s.ExpirePending(state, now.Add(consts.AILeaseDuration))
	if c.Messages[1].Status != consts.AIStatusFailed {
		t.Fatal("stale request not failed")
	}
	c, _, err = s.PrepareTurn(state, 1, c.ID, "request-000000002", "Redis", now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	c.Messages[3].Status = consts.AIStatusCompleted
	if _, _, err = s.PrepareTurn(state, 1, c.ID, "request-000000003", "MySQL", now.Add(4*time.Minute)); err == nil {
		t.Fatal("daily quota bypassed")
	}
	// 上海跨天重置额度，服务器本地时区不影响。
	if _, _, err = s.PrepareTurn(state, 1, c.ID, "request-000000003", "MySQL", now.Add(10*time.Minute)); err != nil || state.Requests != 1 {
		t.Fatalf("quota day rollover: %v", err)
	}
}

func TestAIMemoryAndInputLimits(t *testing.T) {
	s := NewAIDomainService(nil, do.AIAccessPolicy{}, 50)
	state := &do.AIState{AdminID: 1}
	if err := s.SetMemory(state, 2, "private"); err == nil {
		t.Fatal("memory ownership bypass")
	}
	if err := s.SetMemory(state, 1, strings.Repeat("字", 4001)); err == nil {
		t.Fatal("oversized memory accepted")
	}
	if _, _, err := s.PrepareTurn(state, 1, "../../outside", "request-000000001", "test", time.Now()); err == nil {
		t.Fatal("invalid id accepted")
	}
}

func TestAIFinishTurnPreservesOwnershipAndTerminalState(t *testing.T) {
	for _, sample := range []struct {
		name   string
		failed bool
		delay  time.Duration
	}{
		{"completed", false, time.Second}, {"failed", true, time.Second}, {"lease expired", false, consts.AILeaseDuration},
	} {
		t.Run(sample.name, func(t *testing.T) {
			s := NewAIDomainService(nil, do.AIAccessPolicy{}, 50)
			state := &do.AIState{AdminID: 1}
			now := time.Now()
			c, _, err := s.PrepareTurn(state, 1, "conversation-00001", "request-000000001", "JWT", now)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.FinishTurn(state, 2, c.ID, "request-000000001", "cross owner", false, nil, now); err == nil {
				t.Fatal("cross-owner completion accepted")
			}
			if err := s.FinishTurn(state, 1, c.ID, "request-000000001", strings.Repeat("字", consts.AIAnswerMaxRunes+1), false, nil, now); err == nil {
				t.Fatal("oversized answer accepted")
			}
			if err := s.FinishTurn(state, 1, c.ID, "request-000000001", "original result", sample.failed, nil, now.Add(sample.delay)); err != nil {
				t.Fatal(err)
			}
			status := consts.AIStatusCompleted
			if sample.failed || sample.delay >= consts.AILeaseDuration {
				status = consts.AIStatusFailed
			}
			if c.Messages[1].Status != status {
				t.Fatal("invalid terminal state")
			}
			original := c.Messages[1].Content
			if sample.delay >= consts.AILeaseDuration && original == "original result" {
				t.Fatal("expired answer was resurrected")
			}
			if err := s.FinishTurn(state, 1, c.ID, "request-000000001", "replacement", !sample.failed, nil, now.Add(sample.delay+time.Second)); err != nil {
				t.Fatal(err)
			}
			if c.Messages[1].Content != original || c.Messages[1].Status != status {
				t.Fatal("terminal state overwritten")
			}
		})
	}
}
