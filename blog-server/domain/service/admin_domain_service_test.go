package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/repo"
	"blog-server/infra/model"

	"golang.org/x/crypto/bcrypt"
)

type adminSecurityRepo struct {
	repo.AdminMysqlRepo
	admins       map[uint64]*model.AdminModel
	lookupErrors map[uint64]error
	nameResult   *model.AdminModel
	nameErr      error
	writeErr     error
	writes       int
	writtenID    uint64
	role         int
	status       int
	ownPassword  string
	resetHash    string
}

func (r *adminSecurityRepo) FindAdminByID(_ context.Context, id uint64) (*model.AdminModel, error) {
	return r.admins[id], r.lookupErrors[id]
}

func (r *adminSecurityRepo) FindAdminByName(_ context.Context, _ string) (*model.AdminModel, error) {
	return r.nameResult, r.nameErr
}

func (r *adminSecurityRepo) UpdateAdminStatus(_ context.Context, id uint64, status int) error {
	r.writes++
	r.writtenID, r.status = id, status
	return r.writeErr
}

func (r *adminSecurityRepo) ChangeAdminRole(_ context.Context, id uint64, role int) error {
	r.writes++
	r.writtenID, r.role = id, role
	return r.writeErr
}

func (r *adminSecurityRepo) ResetAdminPWDByID(_ context.Context, id uint64, hash string) error {
	r.writes++
	r.writtenID, r.resetHash = id, hash
	return r.writeErr
}

func (r *adminSecurityRepo) UpdatePWD(_ context.Context, id uint64, hash string) error {
	r.writes++
	r.writtenID, r.ownPassword = id, hash
	return r.writeErr
}

func adminPair() *adminSecurityRepo {
	return &adminSecurityRepo{admins: map[uint64]*model.AdminModel{
		1: {ID: 1, Role: consts.AdminRoleSuper, Status: consts.AdminStatusEnabled},
		2: {ID: 2, Role: consts.AdminRoleNormal, Status: consts.AdminStatusEnabled},
	}}
}

func assertAdminErrCode(t *testing.T, err error, code int) {
	t.Helper()
	if code == 0 {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	var errno common.Errno
	if !errors.As(err, &errno) || errno.Code != code {
		t.Fatalf("error = %v, want code %d", err, code)
	}
}

func TestAdminManagementPermissions(t *testing.T) {
	actions := []struct {
		name string
		run  func(*AdminDomainService, uint64, uint64) error
	}{
		{"status", func(s *AdminDomainService, operator, target uint64) error {
			return s.UpdateAdminStatus(context.Background(), operator, target, consts.AdminStatusDisabled)
		}},
		{"password", func(s *AdminDomainService, operator, target uint64) error {
			return s.ResetAdminPWD(context.Background(), operator, target, "Reset123")
		}},
		{"role", func(s *AdminDomainService, operator, target uint64) error {
			_, err := s.UpdateAdminRole(context.Background(), operator, target, consts.AdminRoleSuper)
			return err
		}},
	}
	cases := []struct {
		name   string
		change func(*adminSecurityRepo)
		target uint64
		code   int
	}{
		{"normal target allowed", func(*adminSecurityRepo) {}, 2, 0},
		{"super target rejected", func(r *adminSecurityRepo) { r.admins[2].Role = consts.AdminRoleSuper }, 2, common.PermissionDeniedErr.Code},
		{"unknown target role rejected", func(r *adminSecurityRepo) { r.admins[2].Role = 9 }, 2, common.PermissionDeniedErr.Code},
		{"normal operator rejected", func(r *adminSecurityRepo) { r.admins[1].Role = consts.AdminRoleNormal }, 2, common.PermissionDeniedErr.Code},
		{"disabled operator rejected", func(r *adminSecurityRepo) { r.admins[1].Status = consts.AdminStatusDisabled }, 2, common.PermissionDeniedErr.Code},
		{"missing operator", func(r *adminSecurityRepo) { delete(r.admins, 1) }, 2, common.PermissionDeniedErr.Code},
		{"missing target", func(r *adminSecurityRepo) { delete(r.admins, 2) }, 2, common.AdminNotFoundErr.Code},
		{"operator query failure", func(r *adminSecurityRepo) { r.lookupErrors = map[uint64]error{1: errors.New("db unavailable")} }, 2, common.AdminQueryErr.Code},
		{"target query failure", func(r *adminSecurityRepo) { r.lookupErrors = map[uint64]error{2: errors.New("db unavailable")} }, 2, common.AdminQueryErr.Code},
		{"self rejected", func(*adminSecurityRepo) {}, 1, common.PermissionDeniedErr.Code},
	}
	for _, action := range actions {
		for _, tc := range cases {
			t.Run(action.name+"/"+tc.name, func(t *testing.T) {
				r := adminPair()
				tc.change(r)
				code := tc.code
				if action.name == "role" && tc.target == 1 {
					code = common.AdminChangeSelfRoleErr.Code
				}
				err := action.run(NewAdminDomainService(r), 1, tc.target)
				assertAdminErrCode(t, err, code)
				if code == 0 {
					if r.writes != 1 || r.writtenID != 2 {
						t.Fatalf("writes = %d, target = %d", r.writes, r.writtenID)
					}
				} else if r.writes != 0 {
					t.Fatal("rejected management request reached a write")
				}
			})
		}
	}
}

func TestAdminStatusRejectsInvalidValues(t *testing.T) {
	for _, status := range []int{-1, 2} {
		err := NewAdminDomainService(nil).UpdateAdminStatus(context.Background(), 1, 2, status)
		assertAdminErrCode(t, err, common.ParamErr.Code)
	}
}

func TestAdminRoleUnchangedDoesNotWrite(t *testing.T) {
	r := adminPair()
	changed, err := NewAdminDomainService(r).UpdateAdminRole(context.Background(), 1, 2, consts.AdminRoleNormal)
	assertAdminErrCode(t, err, 0)
	if changed || r.writes != 0 {
		t.Fatalf("unchanged role produced changed=%v, writes=%d", changed, r.writes)
	}
	_, err = NewAdminDomainService(r).UpdateAdminRole(context.Background(), 1, 2, 3)
	assertAdminErrCode(t, err, common.AdminRoleInvalidErr.Code)
	if r.writes != 0 {
		t.Fatal("invalid role reached a write")
	}
}

func TestAdminConditionalWritesReportConflict(t *testing.T) {
	for _, action := range []string{"status", "password", "role"} {
		t.Run(action, func(t *testing.T) {
			r := adminPair()
			r.writeErr = fmt.Errorf("conditional update: %w", repo.ErrNoRowsAffected)
			s := NewAdminDomainService(r)
			var err error
			switch action {
			case "status":
				err = s.UpdateAdminStatus(context.Background(), 1, 2, consts.AdminStatusDisabled)
			case "password":
				err = s.ResetAdminPWD(context.Background(), 1, 2, "Reset123")
			case "role":
				var changed bool
				changed, err = s.UpdateAdminRole(context.Background(), 1, 2, consts.AdminRoleSuper)
				if changed {
					t.Fatal("conflicting role update reported a change")
				}
			}
			assertAdminErrCode(t, err, common.AdminConcurrentModifyErr.Code)
		})
	}
}

func TestSuperAdminCanChangeOwnPassword(t *testing.T) {
	r := adminPair()
	hash, err := bcrypt.GenerateFromPassword([]byte("Old12345"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	r.admins[1].Password = string(hash)
	err = NewAdminDomainService(r).UpdatePWD(context.Background(), 1, "New12345", "Old12345")
	assertAdminErrCode(t, err, 0)
	if r.writtenID != 1 || r.resetHash != "" || bcrypt.CompareHashAndPassword([]byte(r.ownPassword), []byte("New12345")) != nil {
		t.Fatal("own-password change used the management reset path or stored an invalid hash")
	}
}

func TestAdminLookupAndLoginNotFound(t *testing.T) {
	s := NewAdminDomainService(&adminSecurityRepo{})
	_, err := s.FindAdminByID(context.Background(), 99)
	assertAdminErrCode(t, err, common.AdminNotFoundErr.Code)
	_, err = s.FindAdminByName(context.Background(), "missing")
	assertAdminErrCode(t, err, common.AdminNotFoundErr.Code)
	_, err = s.Login(context.Background(), "missing", "Password123")
	assertAdminErrCode(t, err, common.LoginPasswordErr.Code)

	s = NewAdminDomainService(&adminSecurityRepo{nameErr: errors.New("db unavailable")})
	_, err = s.CreateAdmin(context.Background(), "newadmin", "Password123", consts.AdminRoleNormal)
	assertAdminErrCode(t, err, common.AdminQueryErr.Code)
	_, err = s.Login(context.Background(), "existing", "Password123")
	assertAdminErrCode(t, err, common.AdminQueryErr.Code)
}
