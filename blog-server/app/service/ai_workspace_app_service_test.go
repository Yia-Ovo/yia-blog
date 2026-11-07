package service

import (
	"blog-server/domain/do"
	domainservice "blog-server/domain/service"
	infrarepo "blog-server/infra/repo"
	"context"
	"strings"
	"testing"
)

func TestCodeJobWorkerAuthenticatesAndDoesNotReadPrivateSourcesInGeneralMode(t *testing.T) {
	dir := t.TempDir()
	knowledge := &aiNoKnowledge{}
	domain := domainservice.NewAIDomainService(domainservice.NewAdminDomainService(&aiGeneralAdmins{}), do.AIAccessPolicy{AllowAllAdmins: true, PrivateKnowledgeAdminIDs: []uint64{1}}, 50)
	app := NewAIAppService(domain, infrarepo.NewAIStateFileRepo(dir+"/state"), knowledge, nil, true, "gpt-6.1-sol", 50)
	key := strings.Repeat("k", 48)
	app.ConfigureWorkspace(infrarepo.NewAIWorkspaceFileRepo(dir+"/workspace"), true, key)
	ctx := context.Background()
	if _, err := app.CreateCodeJob(ctx, CreateCodeJobCommand{AdminID: 2, ID: "code-task-00000001", Prompt: "change"}); err == nil {
		t.Fatal("normal admin allowed")
	}
	if _, err := app.CreateCodeJob(ctx, CreateCodeJobCommand{AdminID: 1, ID: "code-task-00000001", Prompt: "change", ContextMode: "general"}); err != nil {
		t.Fatal(err)
	}
	if knowledge.calls.Load() != 0 {
		t.Fatal("general task read private sources")
	}
	if _, err := app.ClaimCodeJob(ctx, "admin-token"); err == nil {
		t.Fatal("admin token used as worker key")
	}
	job, err := app.ClaimCodeJob(ctx, key)
	if err != nil || job == nil {
		t.Fatal(err)
	}
	if job.Context != "" || job.AdminID != 1 {
		t.Fatal("invalid worker context")
	}
	view, err := app.Workspace(ctx, 1)
	if err != nil || len(view.Jobs) != 1 {
		t.Fatal(err)
	}
	if view.Jobs[0].Status != "running" {
		t.Fatal("task not persisted")
	}
}
