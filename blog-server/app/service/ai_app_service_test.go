package service

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
	infrarepo "blog-server/infra/repo"
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type aiTestAdmins struct{ repo.AdminMysqlRepo }

func (aiTestAdmins) FindAdminByID(context.Context, uint64) (*model.AdminModel, error) {
	return &model.AdminModel{ID: 1, Status: 1}, nil
}

type aiTestKnowledge struct{}

func (aiTestKnowledge) Info(context.Context) (int, error) { return 1, nil }
func (aiTestKnowledge) Search(context.Context, string) ([]do.AISource, error) {
	return []do.AISource{{Path: ".private/02-面试资料/README.md", Line: 1, Excerpt: "学习进度：软删除与作者 ID"}}, nil
}

type aiTestProvider struct {
	release     chan struct{}
	calls       atomic.Int32
	contextText chan string
}

func (p *aiTestProvider) Reply(ctx context.Context, _ string, messages []do.AIModelMessage) (string, error) {
	p.calls.Add(1)
	p.contextText <- messages[0].Content
	select {
	case <-p.release:
		return "token_version 是账号级撤销机制。", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func TestAIBackgroundPersistenceMemoryAndDuplicate(t *testing.T) {
	states := infrarepo.NewAIStateFileRepo(t.TempDir())
	provider := &aiTestProvider{release: make(chan struct{}), contextText: make(chan string, 1)}
	domain := domainservice.NewAIDomainService(domainservice.NewAdminDomainService(aiTestAdmins{}), do.AIAccessPolicy{AllowedAdminIDs: []uint64{1}, PrivateKnowledgeAdminIDs: []uint64{1}}, 50)
	app := NewAIAppService(domain, states, aiTestKnowledge{}, provider, true, "gpt-6.1-sol", 50)
	if err := app.SetMemory(context.Background(), 1, "我正在准备 Go 后端面试", consts.AIContextProject); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := AISendCommand{AdminID: 1, ConversationID: "conversation-00001", RequestID: "request-000000001", Content: "解释 JWT", ContextMode: consts.AIContextProject}
	result, err := app.Send(ctx, cmd)
	cancel()
	if err != nil || len(result.Messages) != 2 || result.Messages[1].Status != consts.AIStatusPending {
		t.Fatalf("not queued: %v", err)
	}
	select {
	case contextText := <-provider.contextText:
		if !strings.Contains(contextText, "Go 后端面试") || !strings.Contains(contextText, "软删除") {
			t.Fatal("missing memory or retrieval")
		}
	case <-time.After(time.Second):
		t.Fatal("generation did not start")
	}
	close(provider.release)
	deadline := time.Now().Add(2 * time.Second)
	for {
		result, err = app.Conversation(context.Background(), 1, cmd.ConversationID)
		if err != nil {
			t.Fatal(err)
		}
		if result.Messages[1].Status == consts.AIStatusCompleted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("request cancellation stopped background answer")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(result.Messages[1].Content, "账号级撤销") {
		t.Fatal("answer not saved")
	}
	// 第二个 App 实例模拟另一设备读取相同后端数据，不调用模型。
	second := NewAIAppService(domain, states, aiTestKnowledge{}, provider, true, "gpt-6.1-sol", 50)
	overview, err := second.Overview(context.Background(), 1)
	if err != nil || overview.ProjectMemory != "我正在准备 Go 后端面试" || overview.Memory != "" || overview.Used != 1 {
		t.Fatalf("cross-device data missing: %v", err)
	}
	_, err = app.Send(context.Background(), cmd)
	if err != nil || provider.calls.Load() != 1 {
		t.Fatalf("duplicate consumed quota: %v", err)
	}
	if _, err := second.Conversation(context.Background(), 2, cmd.ConversationID); err == nil {
		t.Fatal("unauthorized reader accepted")
	}
}
