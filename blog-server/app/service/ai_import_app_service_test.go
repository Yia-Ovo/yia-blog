package service

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
	infrarepo "blog-server/infra/repo"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type aiImportAdmins struct {
	repo.AdminMysqlRepo
	role, status int
}

func (r *aiImportAdmins) FindAdminByID(_ context.Context, id uint64) (*model.AdminModel, error) {
	return &model.AdminModel{ID: id, Role: r.role, Status: r.status}, nil
}

func importedSample() do.AIImportedThread {
	return do.AIImportedThread{AdminID: 1, ID: "import-00000000001", Title: "项目问答", Workspace: "yia-blog", UpdatedAt: time.Now().UTC(), Messages: []do.AIMessage{{ID: "local-user-1", Role: consts.AIRoleUser, Content: "JWT"}, {ID: "local-assistant-1", Role: consts.AIRoleAssistant, Content: "secret=abcdefghijklmnopqr"}}}
}

func TestAIImportOnlyLiveOwnerSuperAdmin(t *testing.T) {
	for _, sample := range []struct {
		name         string
		id           uint64
		role, status int
		owners       []uint64
		allowed      bool
	}{
		{"owner", 1, 0, 1, []uint64{1}, true}, {"normal admin including owner ID", 1, 1, 1, []uint64{1}, false}, {"ordinary admin", 2, 1, 1, []uint64{1}, false}, {"other super", 3, 0, 1, []uint64{1}, false}, {"disabled owner", 1, 0, 0, []uint64{1}, false}, {"owner removed from list", 1, 0, 1, nil, false},
	} {
		t.Run(sample.name, func(t *testing.T) {
			directory := t.TempDir()
			ctx := context.Background()
			admins := &aiImportAdmins{role: sample.role, status: sample.status}
			domain := domainservice.NewAIDomainService(domainservice.NewAdminDomainService(admins), do.AIAccessPolicy{AllowAllAdmins: true, PrivateKnowledgeAdminIDs: sample.owners}, 50)
			app := NewAIAppService(domain, infrarepo.NewAIStateFileRepo(t.TempDir()), aiTestKnowledge{}, nil, true, "gpt-6.1-sol", 50, infrarepo.NewAIImportFileRepo(directory))
			value := importedSample()
			value.AdminID = sample.id
			err := app.SyncImportedThread(ctx, value)
			if (err == nil) != sample.allowed {
				t.Fatalf("write permission mismatch: %v", err)
			}
			if !sample.allowed {
				if entries, _ := os.ReadDir(directory); len(entries) != 0 {
					t.Fatal("denied request touched import storage")
				}
				if _, err := app.ImportedThreads(ctx, sample.id); err == nil {
					t.Fatal("denied list accepted")
				}
				if _, err := app.ImportedThread(ctx, sample.id, value.ID, 0); err == nil {
					t.Fatal("denied detail accepted")
				}
				return
			}
			if err := app.SyncImportedThread(ctx, value); err != nil {
				t.Fatal(err)
			}
			values, err := app.ImportedThreads(ctx, 1)
			if err != nil || len(values) != 1 {
				t.Fatalf("idempotence: %v", err)
			}
			page, err := app.ImportedThread(ctx, 1, value.ID, 0)
			if err != nil || len(page.Messages) != 2 || strings.Contains(page.Messages[1].Content, "abcdefghijklmnopqr") {
				t.Fatal("unredacted import", err)
			}
			if _, err := app.Send(ctx, AISendCommand{AdminID: 1, ImportID: value.ID, ContextMode: consts.AIContextGeneral}); err == nil {
				t.Fatal("import leaked through general context")
			}
			admins.role = consts.AdminRoleNormal
			if _, err := app.ImportedThread(ctx, 1, value.ID, 0); err == nil {
				t.Fatal("downgraded owner read history")
			}
			admins.role = consts.AdminRoleSuper
			admins.status = 0
			if err := app.SyncImportedThread(ctx, value); err == nil {
				t.Fatal("disabled owner uploaded history")
			}
		})
	}
}

func TestAIImportRejectsForeignProjectAndToolContent(t *testing.T) {
	domain := domainservice.NewAIDomainService(domainservice.NewAdminDomainService(&aiImportAdmins{role: 0, status: 1}), do.AIAccessPolicy{AllowAllAdmins: true, PrivateKnowledgeAdminIDs: []uint64{1}}, 50)
	app := NewAIAppService(domain, nil, nil, nil, true, "gpt-6.1-sol", 50, infrarepo.NewAIImportFileRepo(t.TempDir()))
	for _, kind := range []string{"foreign project", "tool", "path", "duplicate"} {
		t.Run(kind, func(t *testing.T) {
			value := importedSample()
			switch kind {
			case "foreign project":
				value.Workspace = "yia-studio"
			case "tool":
				value.Messages[0].Role = "system"
			case "path":
				value.ID = "../../private-file"
			case "duplicate":
				value.Messages[1].ID = value.Messages[0].ID
			}
			if err := app.SyncImportedThread(context.Background(), value); err == nil {
				t.Fatal("invalid import accepted")
			}
		})
	}
}

func TestAIImportDoesNotOverwriteNewerSnapshot(t *testing.T) {
	directory := t.TempDir()
	r := infrarepo.NewAIImportFileRepo(directory)
	ctx := context.Background()
	value := importedSample()
	if err := r.Put(ctx, value); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "admin-1", value.ID+".json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	value.UpdatedAt = value.UpdatedAt.Add(-time.Hour)
	value.Title = "older snapshot"
	if err := r.Put(ctx, value); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("newer history overwritten")
	}
	if _, err := r.Get(ctx, 2, value.ID); err == nil {
		t.Fatal("different account read import")
	}
}
