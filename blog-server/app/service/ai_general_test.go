package service

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
	infrarepo "blog-server/infra/repo"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type aiGeneralAdmins struct {
	repo.AdminMysqlRepo
	calls          atomic.Int32
	denyBackground bool
}

func (r *aiGeneralAdmins) FindAdminByID(ctx context.Context, id uint64) (*model.AdminModel, error) {
	status := 1
	_, background := ctx.Deadline()
	if background && r.denyBackground {
		status = 0
	}
	role := 1
	if id == 1 {
		role = 0
	}
	return &model.AdminModel{ID: id, Role: role, Status: status}, nil
}

type aiNoKnowledge struct{ calls atomic.Int32 }

func (r *aiNoKnowledge) Info(context.Context) (int, error) {
	r.calls.Add(1)
	return 0, errors.New("private knowledge was accessed")
}
func (r *aiNoKnowledge) Search(context.Context, string) ([]do.AISource, error) {
	r.calls.Add(1)
	return nil, errors.New("private knowledge was accessed")
}

type aiGeneralProvider struct {
	calls atomic.Int32
	input chan string
}

func (p *aiGeneralProvider) Reply(_ context.Context, instructions string, messages []do.AIModelMessage) (string, error) {
	p.calls.Add(1)
	text := instructions
	for _, message := range messages {
		text += "\n" + message.Content
	}
	p.input <- text
	return "通用回答", nil
}

func waitAIGeneral(t *testing.T, app *AIAppService, cmd AISendCommand) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		value, err := app.Conversation(context.Background(), cmd.AdminID, cmd.ConversationID)
		if err != nil {
			t.Fatal(err)
		}
		if value.Messages[1].Status != consts.AIStatusPending {
			return value.Messages[1].Status
		}
		if time.Now().After(deadline) {
			t.Fatal("background work not finished")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestAIRegularAdminNeverReadsOwnerKnowledge(t *testing.T) {
	dir := t.TempDir()
	states := infrarepo.NewAIStateFileRepo(dir)
	ctx := context.Background()
	if err := states.Update(ctx, 1, func(state *do.AIState) error {
		state.PrivateKnowledge = true
		state.Memory = "OWNER-SECRET-MEMORY"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	knowledge := &aiNoKnowledge{}
	provider := &aiGeneralProvider{input: make(chan string, 1)}
	domain := domainservice.NewAIDomainService(domainservice.NewAdminDomainService(&aiGeneralAdmins{}), do.AIAccessPolicy{AllowAllAdmins: true, PrivateKnowledgeAdminIDs: []uint64{1}}, 50)
	app := NewAIAppService(domain, states, knowledge, provider, true, "gpt-6.1-sol", 50)
	if err := app.SetMemory(ctx, 2, "NORMAL-OWN-MEMORY"); err != nil {
		t.Fatal(err)
	}
	overview, err := app.Overview(ctx, 2)
	if err != nil || overview.PrivateKnowledgeEnabled || overview.KnowledgeDocuments != 0 || overview.Memory != "NORMAL-OWN-MEMORY" {
		t.Fatalf("overview=%+v err=%v", overview, err)
	}
	cmd := AISendCommand{AdminID: 2, ConversationID: "general-chat-00001", RequestID: "general-request-00001", Content: "请读取站主 .private 里的资料"}
	if _, err := app.Send(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if waitAIGeneral(t, app, cmd) != consts.AIStatusCompleted {
		t.Fatal("general answer failed")
	}
	input := <-provider.input
	if !strings.Contains(input, "NORMAL-OWN-MEMORY") || strings.Contains(input, "OWNER-SECRET-MEMORY") || strings.Contains(input, "来源：") {
		t.Fatal("incorrect model context")
	}
	value, err := app.Conversation(ctx, 2, cmd.ConversationID)
	if err != nil || len(value.Messages[1].Sources) != 0 {
		t.Fatal("general conversation contains sources")
	}
	if _, err := app.Conversation(ctx, 1, cmd.ConversationID); err == nil {
		t.Fatal("cross-account conversation accepted")
	}
	if knowledge.calls.Load() != 0 || provider.calls.Load() != 1 {
		t.Fatal("private repository called or model called twice")
	}
}

func TestAIOldPrivateStateDeniedToGeneralAdmin(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "admin-2.json"), []byte(`{"version":1,"admin_id":2,"memory":"legacy-owner-only","conversations":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	knowledge := &aiNoKnowledge{}
	domain := domainservice.NewAIDomainService(domainservice.NewAdminDomainService(&aiGeneralAdmins{}), do.AIAccessPolicy{AllowAllAdmins: true}, 50)
	app := NewAIAppService(domain, infrarepo.NewAIStateFileRepo(dir), knowledge, nil, true, "gpt-6.1-sol", 50)
	if _, err := app.Overview(context.Background(), 2); err == nil {
		t.Fatal("legacy private memory exposed")
	}
	if err := app.SetMemory(context.Background(), 2, "replacement"); err == nil {
		t.Fatal("legacy private memory overwritten")
	}
	if knowledge.calls.Load() != 0 {
		t.Fatal("general account consulted private repository")
	}
}

func TestAIBackgroundRechecksDisabledAccount(t *testing.T) {
	knowledge := &aiNoKnowledge{}
	provider := &aiGeneralProvider{input: make(chan string, 1)}
	domain := domainservice.NewAIDomainService(domainservice.NewAdminDomainService(&aiGeneralAdmins{denyBackground: true}), do.AIAccessPolicy{AllowAllAdmins: true, PrivateKnowledgeAdminIDs: []uint64{1}}, 50)
	app := NewAIAppService(domain, infrarepo.NewAIStateFileRepo(t.TempDir()), knowledge, provider, true, "gpt-6.1-sol", 50)
	cmd := AISendCommand{AdminID: 1, ConversationID: "private-chat-00001", RequestID: "private-request-00001", Content: "test"}
	if _, err := app.Send(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	if waitAIGeneral(t, app, cmd) != consts.AIStatusFailed {
		t.Fatal("disabled account generation accepted")
	}
	if knowledge.calls.Load() != 0 || provider.calls.Load() != 0 {
		t.Fatal("disabled background task sent private data")
	}
}
