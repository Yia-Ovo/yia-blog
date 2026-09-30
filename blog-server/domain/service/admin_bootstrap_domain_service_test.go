package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"blog-server/common/consts"
	"blog-server/domain/repo"
	"blog-server/infra/model"

	"golang.org/x/crypto/bcrypt"
)

type bootstrapAdminStore struct {
	repo.AdminMysqlRepo
	existing bool
	queryErr error
	writeErr error
	written  *model.AdminModel
	calls    int
}

func (s *bootstrapAdminStore) WithExclusiveAdmin(_ context.Context, callback func(repo.AdminMysqlRepo, bool) error) error {
	s.calls++
	if s.queryErr != nil {
		return s.queryErr
	}
	return callback(s, s.existing)
}

func (*bootstrapAdminStore) FindAdminByName(context.Context, string) (*model.AdminModel, error) {
	return nil, nil
}

func (s *bootstrapAdminStore) CreateAdmin(_ context.Context, admin *model.AdminModel) error {
	s.written = admin
	return s.writeErr
}

func TestAdminBootstrapRefusesExistingRowsAndQueryFailure(t *testing.T) {
	queryErr := errors.New("query failed")
	for _, tc := range []struct {
		name     string
		existing bool
		queryErr error
		want     error
	}{
		{"existing including deleted", true, nil, ErrAdminBootstrapExists},
		{"query failure is not empty", false, queryErr, queryErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &bootstrapAdminStore{existing: tc.existing, queryErr: tc.queryErr}
			err := NewAdminBootstrapDomainService(store).Bootstrap(context.Background(), "owner", "Owner-example-73")
			if !errors.Is(err, tc.want) || store.written != nil {
				t.Fatalf("error=%v, write attempted=%v", err, store.written != nil)
			}
		})
	}
}

func TestAdminBootstrapUsesDomainPasswordRoleAndStatus(t *testing.T) {
	store := &bootstrapAdminStore{}
	password := "Owner-example-73"
	if err := NewAdminBootstrapDomainService(store).Bootstrap(context.Background(), "owner", password); err != nil {
		t.Fatal(err)
	}
	written := store.written
	if written == nil || written.Role != consts.AdminRoleSuper || written.Status != consts.AdminStatusEnabled || written.TokenVersion != 0 || written.Remark != "" {
		t.Fatal("initial administrator fields are incorrect")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(written.Password), []byte(password)); err != nil {
		t.Fatal("password was not hashed through the domain creation path")
	}
	store.existing = true
	store.written = nil
	if err := NewAdminBootstrapDomainService(store).Bootstrap(context.Background(), "other", password); !errors.Is(err, ErrAdminBootstrapExists) || store.written != nil {
		t.Fatal("a second initialization must not write or reset an account")
	}
}

func TestAdminBootstrapRejectsCredentialsBeforeAccessingStorage(t *testing.T) {
	for _, tc := range [][2]string{
		{"", "Owner-example-73"}, {" owner", "Owner-example-73"},
		{strings.Repeat("名", 17), "Owner-example-73"}, {"owner", "short"},
		{"owner", strings.Repeat("x", 21)}, {"owner", strings.Repeat("😀", 19)},
		{"owner", "Owner\nexample73"}, {"owner", "Owner-example\x00"},
	} {
		store := &bootstrapAdminStore{}
		err := NewAdminBootstrapDomainService(store).Bootstrap(context.Background(), tc[0], tc[1])
		if !errors.Is(err, ErrAdminBootstrapCredentials) || store.calls != 0 {
			t.Fatal("invalid credentials reached storage")
		}
	}
}

func TestAdminBootstrapDoesNotReportFailedWriteAsSuccess(t *testing.T) {
	store := &bootstrapAdminStore{writeErr: errors.New("insert failed")}
	if err := NewAdminBootstrapDomainService(store).Bootstrap(context.Background(), "owner", "Owner-example-73"); err == nil {
		t.Fatal("failed INSERT was reported as success")
	}
}
