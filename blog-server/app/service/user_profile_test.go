package service

import (
	"context"
	"testing"

	"blog-server/api/dto"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"blog-server/infra/model"
)

type profileUserRepo struct {
	repo.UserMysqlRepo
	user   model.UserModel
	update *do.UserInfoUpdate
}

func (r *profileUserRepo) FindUserByID(context.Context, uint64) (*model.UserModel, error) {
	copy := r.user
	return &copy, nil
}

func (r *profileUserRepo) UpdateUserInfo(_ context.Context, id uint64, update *do.UserInfoUpdate) error {
	if id != r.user.ID {
		return repo.ErrNoRowsAffected
	}
	r.update = update
	// 固定交错：读取后账号被管理操作禁用，资料更新不得恢复旧状态。
	r.user.Status = consts.UserStatusDisabled
	if update.NickName != nil {
		r.user.NickName = *update.NickName
	}
	if update.Avatar != nil {
		r.user.Avatar = *update.Avatar
	}
	if update.Gender != nil {
		r.user.Gender = *update.Gender
	}
	if update.Email != nil {
		r.user.Email = *update.Email
	}
	if update.Signature != nil {
		r.user.Signature = *update.Signature
	}
	return nil
}

type profileUserCache struct {
	repo.UserRedisRepo
	deletes int
}

func (r *profileUserCache) GetUserFromRedis(context.Context, uint64) (*model.UserModel, error) {
	return nil, repo.ErrCacheMiss
}
func (r *profileUserCache) SetUserToRedis(context.Context, uint64, *model.UserModel) error {
	return nil
}
func (r *profileUserCache) DelUserFromRedis(context.Context, uint64) error { r.deletes++; return nil }

func TestUserProfilePreservesOptionalFieldsAndAccountStatus(t *testing.T) {
	name, empty, unknown := "new-name", "", 0
	for _, tt := range []struct {
		name       string
		req        *dto.UserUpdateInfoReq
		wantGender int
		wantEmail  string
	}{
		{"omitted", &dto.UserUpdateInfoReq{NickName: &name}, 2, "old@example.com"},
		{"explicit-zero", &dto.UserUpdateInfoReq{Gender: &unknown, Email: &empty, Avatar: &empty, Signature: &empty}, 0, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &profileUserRepo{user: model.UserModel{ID: 7, NickName: "old-name", Gender: 2, Email: "old@example.com", Avatar: "old.png", Signature: "old", Status: consts.UserStatusEnabled}}
			cache := &profileUserCache{}
			app := NewUserAppService(domainservice.NewUserDomainService(r), cache)
			got, err := app.UpdateUserInfo(context.Background(), 7, tt.req)
			if err != nil || got == nil {
				t.Fatalf("profile update failed: got=%+v err=%v", got, err)
			}
			if got.Gender != tt.wantGender || got.Email != tt.wantEmail || got.Status != consts.UserStatusDisabled || cache.deletes != 1 {
				t.Fatalf("partial update/state lost: got=%+v deletes=%d", got, cache.deletes)
			}
			if r.update.Gender != tt.req.Gender || r.update.Email != tt.req.Email {
				t.Fatal("optional fields were flattened")
			}
			if tt.req.Avatar != nil && (got.Avatar != "" || got.Signature != "") {
				t.Fatal("explicit empty values did not clear profile")
			}
		})
	}
}
