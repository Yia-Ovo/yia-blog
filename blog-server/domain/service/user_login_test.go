package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/repo"
	"blog-server/infra/model"

	"golang.org/x/crypto/bcrypt"
)

type loginUserRepo struct {
	repo.UserMysqlRepo
	user               *model.UserModel
	queryErr, auditErr error
	loginAt            *time.Time
	writes             int
}

func (r *loginUserRepo) FindUserByNickname(context.Context, string) (*model.UserModel, error) {
	return r.user, r.queryErr
}
func (r *loginUserRepo) FindUserByID(context.Context, uint64) (*model.UserModel, error) {
	return r.user, r.queryErr
}
func (r *loginUserRepo) UpdateLastLoginInfo(_ context.Context, _ uint64, at *time.Time) error {
	r.writes++
	r.loginAt = at
	return r.auditErr
}

func TestUserLoginErrorSemantics(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name     string
		user     *model.UserModel
		queryErr error
		want     common.Errno
	}{
		{"missing", nil, nil, common.LoginPasswordErr},
		{"empty-record", &model.UserModel{}, nil, common.LoginPasswordErr},
		{"query-failure", nil, errors.New("unavailable"), common.UserQueryErr},
		{"wrong-password", &model.UserModel{ID: 7, Status: consts.UserStatusEnabled, Password: string(hash)}, nil, common.LoginPasswordErr},
		{"disabled", &model.UserModel{ID: 7, Status: consts.UserStatusDisabled}, nil, common.UserAccountDisabledErr},
		{"locked", &model.UserModel{ID: 7, Status: consts.UserStatusLocked}, nil, common.LoginAccountLockedErr},
		{"invalid-status", &model.UserModel{ID: 7, Status: 99}, nil, common.UserAccountDisabledErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &loginUserRepo{user: tt.user, queryErr: tt.queryErr}
			user, err := NewUserDomainService(r).Login(context.Background(), "user", "wrong-password")
			if user != nil || !errors.Is(err, tt.want) || r.writes != 0 {
				t.Fatalf("unexpected login result: user=%+v err=%v writes=%d", user, err, r.writes)
			}
		})
	}
}

func TestUserLoginRecordsCurrentTime(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, auditErr := range []error{nil, errors.New("audit unavailable")} {
		r := &loginUserRepo{user: &model.UserModel{ID: 7, Status: consts.UserStatusEnabled, Password: string(hash), LastLoginAt: &old}, auditErr: auditErr}
		before := time.Now()
		user, err := NewUserDomainService(r).Login(context.Background(), "user", "correct-password")
		after := time.Now()
		if err != nil || user == nil || r.writes != 1 || r.loginAt == nil || user.LastLoginAt == nil || !user.LastLoginAt.Equal(*r.loginAt) || r.loginAt.Before(before) || r.loginAt.After(after) {
			t.Fatalf("login time not current/consistent: user=%+v at=%v err=%v", user, r.loginAt, err)
		}
	}
}

func TestUserQueriesReportMissing(t *testing.T) {
	s := NewUserDomainService(&loginUserRepo{})
	if _, err := s.FindUserByID(context.Background(), 7); !errors.Is(err, common.UserNotFoundErr) {
		t.Fatalf("ID lookup: %v", err)
	}
	if _, err := s.FindUserByNickname(context.Background(), "missing"); !errors.Is(err, common.UserNotFoundErr) {
		t.Fatalf("nickname lookup: %v", err)
	}
}
