package service

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func avatarTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func oversizedPNGHeader(t *testing.T, w, h uint32) []byte {
	data := avatarTestPNG(t, 1, 1)
	binary.BigEndian.PutUint32(data[16:20], w)
	binary.BigEndian.PutUint32(data[20:24], h)
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	return data
}

func TestAvatarNormalizationRejectsInvalidAndOversized(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty": nil, "svg": []byte(`<svg/>`), "gif": []byte("GIF89a"), "fake-png": []byte("\x89PNG\r\n\x1a\nnot a png"),
		"bytes": make([]byte, consts.AvatarMaxUploadBytes+1), "dimension": oversizedPNGHeader(t, 4097, 1), "pixels": oversizedPNGHeader(t, 2001, 2000),
		"truncated": avatarTestPNG(t, 5, 5)[:40],
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NormalizeAvatar(context.Background(), data); err == nil {
				t.Fatal("invalid upload accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NormalizeAvatar(ctx, avatarTestPNG(t, 5, 5)); err == nil {
		t.Fatal("canceled request accepted")
	}
}

func TestAvatarNormalizationCropsAndStripsTrailingMetadata(t *testing.T) {
	for _, size := range []int{30, 360} {
		source := image.NewNRGBA(image.Rect(0, 0, size*3, size))
		for y := 0; y < size; y++ {
			for x := 0; x < size*3; x++ {
				c := color.NRGBA{R: 255, A: 255}
				if x >= size && x < 2*size {
					c = color.NRGBA{G: 255, A: 255}
				}
				source.SetNRGBA(x, y, c)
			}
		}
		var input bytes.Buffer
		if err := png.Encode(&input, source); err != nil {
			t.Fatal(err)
		}
		input.WriteString("private-metadata")
		data, err := NormalizeAvatar(context.Background(), input.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte("private-metadata")) {
			t.Fatal("metadata retained")
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		want := min(size, 256)
		if img.Bounds().Dx() != want || img.Bounds().Dy() != want {
			t.Fatal(img.Bounds())
		}
		for _, point := range []image.Point{{0, 0}, {want / 2, want / 2}, {want - 1, want - 1}} {
			r, g, b, a := img.At(point.X, point.Y).RGBA()
			if r != 0 || g != 65535 || b != 0 || a != 65535 {
				t.Fatal("wrong center crop")
			}
		}
	}
	var jpg bytes.Buffer
	if err := jpeg.Encode(&jpg, image.NewGray(image.Rect(0, 0, 10, 20)), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeAvatar(context.Background(), jpg.Bytes()); err != nil {
		t.Fatalf("valid JPEG rejected: %v", err)
	}
}

type avatarDomainUserRepo struct {
	repo.UserMysqlRepo
	user   *model.UserModel
	id     uint64
	update *do.UserInfoUpdate
}

func (r *avatarDomainUserRepo) FindUserByID(context.Context, uint64) (*model.UserModel, error) {
	return r.user, nil
}
func (r *avatarDomainUserRepo) UpdateUserInfo(_ context.Context, id uint64, u *do.UserInfoUpdate) error {
	r.id = id
	r.update = u
	return nil
}

func TestAvatarDomainUpdatesOnlyCurrentEnabledAccount(t *testing.T) {
	name := strings.Repeat("a", 64) + ".png"
	adminRepo := adminPair()
	userRepo := &avatarDomainUserRepo{user: &model.UserModel{ID: 7, Status: consts.UserStatusEnabled}}
	s := NewAvatarDomainService(NewAdminDomainService(adminRepo), NewUserDomainService(userRepo))
	if err := s.SetAdminAvatar(context.Background(), 1, name); err != nil {
		t.Fatal(err)
	}
	if adminRepo.writtenID != 1 || adminRepo.info.Avatar == nil || adminRepo.info.AdminName != nil || adminRepo.info.Email != nil || adminRepo.info.Remark != nil {
		t.Fatal("admin update contains unrelated fields")
	}
	if err := s.SetUserAvatar(context.Background(), 7, name); err != nil {
		t.Fatal(err)
	}
	if userRepo.id != 7 || userRepo.update.Avatar == nil || userRepo.update.NickName != nil || userRepo.update.Gender != nil || userRepo.update.Email != nil || userRepo.update.Signature != nil {
		t.Fatal("user update contains unrelated fields")
	}
	adminRepo.admins[1].Status = consts.AdminStatusDisabled
	userRepo.user.Status = consts.UserStatusDisabled
	if err := s.SetAdminAvatar(context.Background(), 1, name); err == nil {
		t.Fatal("disabled admin accepted")
	}
	if err := s.SetUserAvatar(context.Background(), 7, name); err == nil {
		t.Fatal("disabled user accepted")
	}
	if _, err := s.PrepareAdmin(context.Background(), 0, []byte("bad")); err == nil {
		t.Fatal("missing identity accepted")
	}
	if err := s.SetUserAvatar(context.Background(), 7, "../outside.png"); err == nil {
		t.Fatal("invalid name accepted")
	}
}
