package service

import (
	"blog-server/domain/do"
	"blog-server/infra/model"
	"context"
	"testing"
)

func TestAIAccessSeparatesChatFromPrivateKnowledge(t *testing.T) {
	for _, sample := range []struct {
		name             string
		id               uint64
		role, status     int
		allowed, private bool
	}{
		{"owner", 1, 0, 1, true, true},
		{"regular administrator", 2, 1, 1, true, false},
		{"another super administrator", 3, 0, 1, true, false},
		{"owner downgraded", 1, 1, 1, true, false},
		{"disabled", 2, 1, 0, false, false},
		{"invalid role", 2, 9, 1, false, false},
		{"anonymous", 0, 1, 1, false, false},
	} {
		t.Run(sample.name, func(t *testing.T) {
			s := NewAIDomainService(NewAdminDomainService(aiAdminRepo{admin: &model.AdminModel{ID: sample.id, Role: sample.role, Status: sample.status}}), do.AIAccessPolicy{AllowAllAdmins: true, PrivateKnowledgeAdminIDs: []uint64{1}}, 50)
			access, err := s.Access(context.Background(), sample.id)
			if (err == nil) != sample.allowed || access.PrivateKnowledge != sample.private {
				t.Fatalf("access=%+v err=%v", access, err)
			}
		})
	}
}

func TestAIPrivateHistoryCannotBecomeGeneralHistory(t *testing.T) {
	s := NewAIDomainService(nil, do.AIAccessPolicy{}, 50)
	state := &do.AIState{AdminID: 1, PrivateKnowledge: true, Memory: "owner-only"}
	if err := s.AuthorizeState(state, do.AIAccess{AdminID: 1}); err == nil {
		t.Fatal("downgraded private state accepted")
	}
	if err := s.AuthorizeState(state, do.AIAccess{AdminID: 2, PrivateKnowledge: true}); err == nil {
		t.Fatal("different account accepted")
	}
	if state.Memory != "owner-only" || !state.PrivateKnowledge {
		t.Fatal("rejected access changed private state")
	}
}
