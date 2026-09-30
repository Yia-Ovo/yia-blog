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
	"errors"
	"image"
	"image/png"
	"strings"
	"testing"
)

type articleImageFailingFiles struct{ repo.ArticleImageRepo }

func (articleImageFailingFiles) Save(context.Context, []byte) (string, error) {
	return "", errors.New("disk unavailable")
}

func TestArticleImageUploadIndependentIdentitiesAndNoProfileMutation(t *testing.T) {
	admins := &adminFlowRepo{admins: map[uint64]*model.AdminModel{1: {ID: 1, Status: consts.AdminStatusEnabled}}}
	users := &avatarUserRepo{}
	domain := domainservice.NewArticleImageDomainService(domainservice.NewAdminDomainService(admins), domainservice.NewUserDomainService(users))
	files := infrarepo.NewArticleImageFileRepo(t.TempDir())
	s := NewArticleImageAppService(domain, files)
	var input bytes.Buffer
	if err := png.Encode(&input, image.NewNRGBA(image.Rect(0, 0, 30, 10))); err != nil {
		t.Fatal(err)
	}
	admin, err := s.UploadAdmin(context.Background(), 1, input.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	user, err := s.UploadUser(context.Background(), 7, input.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if admin.URL != user.URL || admin.Width != 30 || admin.Height != 10 || admin.ContentType != "image/png" || !strings.HasPrefix(admin.URL, consts.ArticleImageRoute) {
		t.Fatalf("invalid result: %+v / %+v", admin, user)
	}
	if admins.info != nil || users.update != nil {
		t.Fatal("image upload changed account")
	}
	got, err := s.Image(context.Background(), strings.TrimPrefix(admin.URL, consts.ArticleImageRoute))
	if err != nil || got.Width != 30 || got.Height != 10 || got.ContentType != "image/png" {
		t.Fatalf("stored image unavailable: %v", err)
	}
	if _, err := s.Image(context.Background(), "../outside.png"); err == nil {
		t.Fatal("path traversal accepted")
	}
	s.files = articleImageFailingFiles{}
	if _, err := s.UploadUser(context.Background(), 7, input.Bytes()); err == nil {
		t.Fatal("storage failure reported success")
	}
}

type articleImageNeverFiles struct{ repo.ArticleImageRepo }

func (articleImageNeverFiles) Save(context.Context, []byte) (string, error) {
	panic("rejected identity reached storage")
}
func (articleImageNeverFiles) Read(context.Context, string) (*do.ArticleImage, error) {
	panic("unexpected read")
}

func TestArticleImageRejectedIdentityNeverWrites(t *testing.T) {
	admins := &adminFlowRepo{admins: map[uint64]*model.AdminModel{1: {ID: 1, Status: consts.AdminStatusDisabled}}}
	domain := domainservice.NewArticleImageDomainService(domainservice.NewAdminDomainService(admins), domainservice.NewUserDomainService(&avatarUserRepo{}))
	s := NewArticleImageAppService(domain, articleImageNeverFiles{})
	if _, err := s.UploadAdmin(context.Background(), 1, []byte("bad")); err == nil {
		t.Fatal("disabled admin accepted")
	}
	if _, err := s.UploadUser(context.Background(), 0, []byte("bad")); err == nil {
		t.Fatal("missing user accepted")
	}
}
