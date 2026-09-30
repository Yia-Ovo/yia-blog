package service

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
	infrarepo "blog-server/infra/repo"
	"bytes"
	"context"
	"image"
	"image/png"
	"testing"
)

type avatarUserRepo struct {
	repo.UserMysqlRepo
	id     uint64
	update *do.UserInfoUpdate
}

func (r *avatarUserRepo) FindUserByID(_ context.Context, id uint64) (*model.UserModel, error) {
	return &model.UserModel{ID: id, Status: consts.UserStatusEnabled}, nil
}
func (r *avatarUserRepo) UpdateUserInfo(_ context.Context, id uint64, u *do.UserInfoUpdate) error {
	r.id = id
	r.update = u
	return nil
}

type avatarUserCache struct {
	repo.UserRedisRepo
	deleted []uint64
}

func (r *avatarUserCache) DelUserFromRedis(_ context.Context, id uint64) error {
	r.deleted = append(r.deleted, id)
	return nil
}

func TestAvatarUploadPersistsAndOnlyClearsOwnProfile(t *testing.T) {
	adminRepo := &adminFlowRepo{admins: map[uint64]*model.AdminModel{1: {ID: 1, Status: consts.AdminStatusEnabled}}}
	adminCache := &adminFlowCache{}
	userRepo := &avatarUserRepo{}
	userCache := &avatarUserCache{}
	domain := domainservice.NewAvatarDomainService(domainservice.NewAdminDomainService(adminRepo), domainservice.NewUserDomainService(userRepo))
	files := infrarepo.NewAvatarFileRepo(t.TempDir())
	s := NewAvatarAppService(domain, files, adminCache, userCache)
	var input bytes.Buffer
	if err := png.Encode(&input, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	admin, err := s.UploadAdmin(context.Background(), 1, input.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if admin.Avatar == "" || adminRepo.info == nil || adminRepo.info.Avatar == nil || *adminRepo.info.Avatar != admin.Avatar || adminRepo.info.AdminName != nil {
		t.Fatal("avatar not saved in profile")
	}
	if len(adminCache.deleted) != 1 || adminCache.deleted[0] != 1 || len(adminCache.cleared) != 0 || len(userCache.deleted) != 0 {
		t.Fatal("wrong cache/session invalidation")
	}
	user, err := s.UploadUser(context.Background(), 7, input.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if user.Avatar != admin.Avatar || userRepo.id != 7 || userRepo.update.Avatar == nil || *userRepo.update.Avatar != user.Avatar {
		t.Fatal("user avatar not saved")
	}
	if len(userCache.deleted) != 1 || userCache.deleted[0] != 7 || len(adminCache.deleted) != 1 {
		t.Fatal("cleared another identity")
	}
	data, err := s.Image(context.Background(), admin.Avatar[len(consts.AvatarImageRoute):])
	if err != nil || len(data) == 0 {
		t.Fatal("saved avatar unavailable")
	}
	adminRepo.writeErr = repo.ErrNoRowsAffected
	if _, err := s.UploadAdmin(context.Background(), 1, input.Bytes()); err == nil {
		t.Fatal("failed database update reported success")
	}
	if len(adminCache.deleted) != 1 {
		t.Fatal("cache cleared after failed update")
	}
}
