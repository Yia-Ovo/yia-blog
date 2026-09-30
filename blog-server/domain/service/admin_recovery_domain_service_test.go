package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"blog-server/common/consts"
	"blog-server/domain/repo"

	"golang.org/x/crypto/bcrypt"
)

type recoveryTestRepo struct {
	account     *repo.AdminRecoveryAccount
	lookupErr   error
	writeErr    error
	queries     int
	writes      int
	expected    repo.AdminRecoveryAccount
	replacement string
}

func (r *recoveryTestRepo) FindByName(context.Context, string) (*repo.AdminRecoveryAccount, error) {
	r.queries++
	return r.account, r.lookupErr
}

func (r *recoveryTestRepo) ReplacePassword(_ context.Context, expected *repo.AdminRecoveryAccount, hash string) error {
	r.writes++
	r.expected, r.replacement = *expected, hash
	return r.writeErr
}

func recoveryTarget() *repo.AdminRecoveryAccount {
	return &repo.AdminRecoveryAccount{
		ID: 7, Name: "super_admin", Role: consts.AdminRoleSuper, Status: consts.AdminStatusEnabled,
		PasswordHash: "original-hash", TokenVersion: 19,
	}
}

func TestAdminRecoveryHashesPasswordAndPreservesExpectedVersion(t *testing.T) {
	r := &recoveryTestRepo{account: recoveryTarget()}
	id, err := NewAdminRecoveryDomainService(r).Recover(context.Background(), "super_admin", "Recovery12345")
	if err != nil || id != 7 || r.writes != 1 {
		t.Fatalf("recovery id=%d writes=%d err=%v", id, r.writes, err)
	}
	if bcrypt.CompareHashAndPassword([]byte(r.replacement), []byte("Recovery12345")) != nil {
		t.Fatal("replacement is not a valid bcrypt hash for the supplied password")
	}
	if r.expected.ID != 7 || r.expected.Name != "super_admin" || r.expected.TokenVersion != 19 || r.expected.PasswordHash != "original-hash" {
		t.Fatal("recovery lost the account identity or expected credential version")
	}
	if r.account.Role != consts.AdminRoleSuper || r.account.Status != consts.AdminStatusEnabled || r.account.TokenVersion != 19 || r.account.PasswordHash != "original-hash" {
		t.Fatal("domain modified the loaded account instead of requesting a conditional write")
	}
}

func TestAdminRecoveryRejectsIneligibleTargetWithoutWriting(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*recoveryTestRepo)
	}{
		{"missing", func(r *recoveryTestRepo) { r.account = nil }},
		{"zero id", func(r *recoveryTestRepo) { r.account.ID = 0 }},
		{"normal admin", func(r *recoveryTestRepo) { r.account.Role = consts.AdminRoleNormal }},
		{"disabled", func(r *recoveryTestRepo) { r.account.Status = consts.AdminStatusDisabled }},
		{"unknown role", func(r *recoveryTestRepo) { r.account.Role = 8 }},
		{"different account", func(r *recoveryTestRepo) { r.account.Name = "another_admin" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &recoveryTestRepo{account: recoveryTarget()}
			tc.change(r)
			id, err := NewAdminRecoveryDomainService(r).Recover(context.Background(), "super_admin", "Recovery12345")
			if id != 0 || !errors.Is(err, ErrAdminRecoveryTarget) || r.writes != 0 {
				t.Fatalf("ineligible target: id=%d writes=%d err=%v", id, r.writes, err)
			}
		})
	}
}

func TestAdminRecoveryRejectsInvalidInputBeforeLookup(t *testing.T) {
	for _, tc := range []struct{ name, password string }{
		{"", "Recovery12345"}, {" super_admin", "Recovery12345"}, {"super_admin ", "Recovery12345"},
		{"super_admin\n", "Recovery12345"}, {"super_admin", ""}, {"super_admin", "abc12"},
		{"super_admin", "onlyletters"}, {"super_admin", "123456789012"},
		{"super_admin", "a1" + strings.Repeat("x", 19)}, {"super_admin", "Recovery123\n"},
		{"super_admin", "Recovery123\x00"}, {"super_admin", "Recovery12\xff"},
	} {
		r := &recoveryTestRepo{account: recoveryTarget()}
		id, err := NewAdminRecoveryDomainService(r).Recover(context.Background(), tc.name, tc.password)
		if id != 0 || !errors.Is(err, ErrAdminRecoveryCredentials) || r.writes != 0 || r.queries != 0 {
			t.Fatalf("invalid input reached lookup/write: id=%d queries=%d writes=%d err=%v", id, r.queries, r.writes, err)
		}
	}
}

func TestAdminRecoveryReportsLookupAndWriteFailures(t *testing.T) {
	lookupErr := errors.New("lookup unavailable")
	r := &recoveryTestRepo{account: recoveryTarget(), lookupErr: lookupErr}
	id, err := NewAdminRecoveryDomainService(r).Recover(context.Background(), "super_admin", "Recovery12345")
	if id != 0 || !errors.Is(err, lookupErr) || r.writes != 0 {
		t.Fatal("lookup failure reached the password write")
	}
	for _, writeErr := range []error{repo.ErrAdminRecoveryConflict, errors.New("write unavailable")} {
		r = &recoveryTestRepo{account: recoveryTarget(), writeErr: writeErr}
		id, err = NewAdminRecoveryDomainService(r).Recover(context.Background(), "super_admin", "Recovery12345")
		if id != 0 || !errors.Is(err, writeErr) || r.writes != 1 {
			t.Fatal("write failure was hidden or recovery retried implicitly")
		}
	}
}

func TestAdminRecoveryCheckTargetIsReadOnly(t *testing.T) {
	r := &recoveryTestRepo{account: recoveryTarget()}
	id, err := NewAdminRecoveryDomainService(r).CheckTarget(context.Background(), "super_admin")
	if err != nil || id != 7 || r.writes != 0 || r.queries != 1 {
		t.Fatalf("preflight id=%d queries=%d writes=%d err=%v", id, r.queries, r.writes, err)
	}
}

func TestAdminRecoveryCheckPasswordIsReadOnly(t *testing.T) {
	// 已有密码可能不满足当前新密码规则；核验应与登录一样只比较 bcrypt。
	const existingPassword = "legacy"
	hash, err := bcrypt.GenerateFromPassword([]byte(existingPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal("could not prepare test password hash")
	}
	for _, tc := range []struct {
		name     string
		password string
		match    bool
	}{
		{"matching legacy password", existingPassword, true},
		{"incorrect password", "Different123", false},
		{"empty password", "", false},
		{"leading whitespace differs", " " + existingPassword, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &recoveryTestRepo{account: recoveryTarget()}
			r.account.PasswordHash = string(hash)
			before := *r.account
			match, err := NewAdminRecoveryDomainService(r).CheckPassword(context.Background(), "super_admin", tc.password)
			if err != nil || match != tc.match || r.queries != 1 || r.writes != 0 {
				t.Fatalf("password check: match=%t queries=%d writes=%d err=%v", match, r.queries, r.writes, err)
			}
			if *r.account != before {
				t.Fatal("read-only password check modified account data")
			}
		})
	}
}

func TestAdminRecoveryCheckPasswordRejectsInvalidTargetsAndHash(t *testing.T) {
	lookupErr := errors.New("lookup unavailable")
	for _, tc := range []struct {
		name    string
		change  func(*recoveryTestRepo)
		wantErr error
	}{
		{"missing", func(r *recoveryTestRepo) { r.account = nil }, ErrAdminRecoveryTarget},
		{"normal admin", func(r *recoveryTestRepo) { r.account.Role = consts.AdminRoleNormal }, ErrAdminRecoveryTarget},
		{"disabled", func(r *recoveryTestRepo) { r.account.Status = consts.AdminStatusDisabled }, ErrAdminRecoveryTarget},
		{"lookup failure", func(r *recoveryTestRepo) { r.lookupErr = lookupErr }, lookupErr},
		{"invalid hash", func(r *recoveryTestRepo) { r.account.PasswordHash = "private-invalid-hash" }, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &recoveryTestRepo{account: recoveryTarget()}
			tc.change(r)
			match, err := NewAdminRecoveryDomainService(r).CheckPassword(context.Background(), "super_admin", "Sample123")
			if match || err == nil || r.queries != 1 || r.writes != 0 {
				t.Fatalf("invalid target/hash: match=%t queries=%d writes=%d err=%v", match, r.queries, r.writes, err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatal("target or lookup error was not preserved")
			}
			if tc.name == "invalid hash" && err.Error() != "无法核验密码，未修改账号" {
				t.Fatal("malformed hash did not produce the safe generic error")
			}
		})
	}
}
