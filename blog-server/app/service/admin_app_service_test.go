package service

import (
	"context"
	"errors"
	"testing"

	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainsvc "blog-server/domain/service"
	"blog-server/infra/model"
)

type adminFlowRepo struct {
	repo.AdminMysqlRepo
	admins   map[uint64]*model.AdminModel
	writeErr error
	writes   int
	info     *do.AdminInfoUpdate
}

func (r *adminFlowRepo) FindAdminByID(_ context.Context, id uint64) (*model.AdminModel, error) {
	return r.admins[id], nil
}

func (r *adminFlowRepo) ChangeAdminRole(_ context.Context, _ uint64, _ int) error {
	r.writes++
	return r.writeErr
}

func (r *adminFlowRepo) UpdateAdminInfo(_ context.Context, _ uint64, info *do.AdminInfoUpdate) error {
	r.writes++
	r.info = info
	return r.writeErr
}

type adminFlowCache struct {
	repo.AdminRedisRepo
	deleted []uint64
	cleared []uint64
	setIDs  []uint64
	cached  *model.AdminModel
}

func (r *adminFlowCache) GetAdminFromRedis(_ context.Context, _ uint64) (*model.AdminModel, error) {
	return nil, repo.ErrCacheMiss
}

func (r *adminFlowCache) DelAdminFromRedis(_ context.Context, id uint64) error {
	r.deleted = append(r.deleted, id)
	return nil
}

func (r *adminFlowCache) ClearAdminAllOnlineToken(_ context.Context, id uint64) error {
	r.cleared = append(r.cleared, id)
	return nil
}

func (r *adminFlowCache) SetAdminToRedis(_ context.Context, id uint64, admin *model.AdminModel) error {
	r.setIDs = append(r.setIDs, id)
	r.cached = admin
	return nil
}

func TestAdminRoleClearsOnlyChangedTarget(t *testing.T) {
	for _, tc := range []struct {
		name string
		role int
		err  error
		want bool
	}{
		{"promote", consts.AdminRoleSuper, nil, true},
		{"unchanged", consts.AdminRoleNormal, nil, false},
		{"conflict", consts.AdminRoleSuper, repo.ErrNoRowsAffected, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &adminFlowRepo{admins: map[uint64]*model.AdminModel{
				1: {ID: 1, Role: consts.AdminRoleSuper, Status: consts.AdminStatusEnabled},
				2: {ID: 2, Role: consts.AdminRoleNormal, Status: consts.AdminStatusEnabled},
			}, writeErr: tc.err}
			cache := &adminFlowCache{}
			s := NewAdminAppService(domainsvc.NewAdminDomainService(r), cache)
			err := s.UpdateAdminRole(context.Background(), 1, &dto.AdminUpdateRoleReq{ID: 2, Role: &tc.role})
			if (err != nil) != (tc.err != nil) {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.want {
				if len(cache.deleted) != 1 || cache.deleted[0] != 2 || len(cache.cleared) != 1 || cache.cleared[0] != 2 {
					t.Fatalf("wrong cache targets: deleted=%v, cleared=%v", cache.deleted, cache.cleared)
				}
			} else if len(cache.deleted) != 0 || len(cache.cleared) != 0 {
				t.Fatal("failed or unchanged role request invalidated a session")
			}
		})
	}
}

func TestAdminRoleOmittedIsNotPromotion(t *testing.T) {
	s := NewAdminAppService(nil, nil)
	err := s.UpdateAdminRole(context.Background(), 1, &dto.AdminUpdateRoleReq{ID: 2})
	var errno common.Errno
	if !errors.As(err, &errno) || errno.Code != common.ParamErr.Code {
		t.Fatalf("omitted role must fail before domain: %v", err)
	}
}

func TestAdminInfoDoesNotFlattenPointersOrClearSessions(t *testing.T) {
	empty := ""
	r := &adminFlowRepo{admins: map[uint64]*model.AdminModel{
		1: {ID: 1, Role: consts.AdminRoleSuper, Status: consts.AdminStatusEnabled},
	}}
	cache := &adminFlowCache{}
	s := NewAdminAppService(domainsvc.NewAdminDomainService(r), cache)
	_, err := s.UpdateAdminInfo(context.Background(), 1, &dto.AdminUpdateInfoReq{Email: &empty, Avatar: &empty, Remark: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if r.info == nil || r.info.AdminName != nil || r.info.Email == nil || *r.info.Email != "" ||
		r.info.Avatar == nil || r.info.Remark == nil {
		t.Fatalf("submitted pointer state was lost: %+v", r.info)
	}
	if len(cache.deleted) != 1 || cache.deleted[0] != 1 || len(cache.cleared) != 0 {
		t.Fatalf("profile cache/session invalidation: deleted=%v, cleared=%v", cache.deleted, cache.cleared)
	}
}

func TestAdminInfoMissingUsesEmptyCache(t *testing.T) {
	cache := &adminFlowCache{}
	s := NewAdminAppService(domainsvc.NewAdminDomainService(&adminFlowRepo{}), cache)
	info, err := s.GetAdminInfo(context.Background(), 99)
	if !errors.Is(err, common.AdminNotFoundErr) || info != nil {
		t.Fatalf("missing admin = %+v, %v", info, err)
	}
	if len(cache.setIDs) != 1 || cache.setIDs[0] != 99 || cache.cached != nil {
		t.Fatal("missing admin did not write the existing empty-cache placeholder")
	}
}
