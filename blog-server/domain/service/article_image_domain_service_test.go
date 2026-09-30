package service

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/infra/model"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestArticleImageNormalizationLimits(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty": nil, "svg": []byte("<svg/>"), "gif": []byte("GIF89a"), "fake-png": []byte("\x89PNG\r\n\x1a\ninvalid"),
		"bytes": make([]byte, consts.ArticleImageMaxUploadBytes+1), "width": oversizedPNGHeader(t, 8193, 1),
		"height": oversizedPNGHeader(t, 1, 8193), "pixels": oversizedPNGHeader(t, 2001, 2000), "truncated": avatarTestPNG(t, 5, 5)[:40],
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NormalizeArticleImage(context.Background(), data); err == nil {
				t.Fatal("invalid image accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NormalizeArticleImage(ctx, avatarTestPNG(t, 1, 1)); err == nil {
		t.Fatal("canceled input accepted")
	}
}

func TestArticleImagePreservesRatioTransparencyAndStripsMetadata(t *testing.T) {
	for _, size := range []image.Point{{30, 10}, {2400, 1200}, {1200, 2400}, {8192, 1}} {
		source := image.NewNRGBA(image.Rect(0, 0, size.X, size.Y))
		for y := 0; y < size.Y; y++ {
			for x := 0; x < size.X; x++ {
				source.SetNRGBA(x, y, color.NRGBA{R: 200, G: 60, B: 20, A: 128})
			}
		}
		var input bytes.Buffer
		if err := png.Encode(&input, source); err != nil {
			t.Fatal(err)
		}
		input.WriteString("private-metadata-marker")
		got, err := NormalizeArticleImage(context.Background(), input.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		if got.ContentType != "image/png" || bytes.Contains(got.Data, []byte("private-metadata-marker")) {
			t.Fatal("format or metadata changed")
		}
		wantWidth, wantHeight := size.X, size.Y
		if size.X > 1920 {
			wantWidth = 1920
			wantHeight = max(1, size.Y*1920/size.X)
		} else if size.Y > 1920 {
			wantHeight = 1920
			wantWidth = max(1, size.X*1920/size.Y)
		}
		if got.Width != wantWidth || got.Height != wantHeight {
			t.Fatalf("got %dx%d, want %dx%d", got.Width, got.Height, wantWidth, wantHeight)
		}
		img, err := png.Decode(bytes.NewReader(got.Data))
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range []image.Point{{0, 0}, {got.Width / 2, got.Height / 2}, {got.Width - 1, got.Height - 1}} {
			pixel := color.NRGBAModel.Convert(img.At(p.X, p.Y)).(color.NRGBA)
			if pixel.A != 128 || pixel.R < 199 || pixel.G < 59 || pixel.B < 19 {
				t.Fatalf("transparent pixel changed: %+v", pixel)
			}
		}
	}
}

func TestArticleImageJPEGReencodedAndMetadataStripped(t *testing.T) {
	var input bytes.Buffer
	if err := jpeg.Encode(&input, image.NewGray(image.Rect(0, 0, 60, 20)), nil); err != nil {
		t.Fatal(err)
	}
	// APP1 元信息位于合法 JPEG 标记之间，重编码必须移除。
	data := input.Bytes()
	metadata := []byte("Exif\x00\x00private-location-marker")
	segment := append([]byte{0xff, 0xe1, byte((len(metadata) + 2) >> 8), byte(len(metadata) + 2)}, metadata...)
	withMetadata := append(append(append([]byte{}, data[:2]...), segment...), data[2:]...)
	got, err := NormalizeArticleImage(context.Background(), withMetadata)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != 60 || got.Height != 20 || got.ContentType != "image/jpeg" || bytes.Contains(got.Data, []byte("private-location-marker")) {
		t.Fatal("JPEG format/ratio/metadata incorrect")
	}
	if _, err := jpeg.Decode(bytes.NewReader(got.Data)); err != nil {
		t.Fatal(err)
	}
}

func TestArticleImageDomainRequiresCurrentEnabledIdentity(t *testing.T) {
	admins := adminPair()
	users := &avatarDomainUserRepo{user: &model.UserModel{ID: 7, Status: consts.UserStatusEnabled}}
	s := NewArticleImageDomainService(NewAdminDomainService(admins), NewUserDomainService(users))
	data := avatarTestPNG(t, 20, 10)
	for _, prepare := range []func(context.Context, uint64, []byte) (*do.ArticleImage, error){s.PrepareAdmin, s.PrepareUser} {
		if _, err := prepare(context.Background(), 0, data); err == nil {
			t.Fatal("missing identity accepted")
		}
	}
	if _, err := s.PrepareAdmin(context.Background(), 1, data); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PrepareUser(context.Background(), 7, data); err != nil {
		t.Fatal(err)
	}
	if admins.info != nil || users.update != nil {
		t.Fatal("upload updated identity")
	}
	admins.admins[1].Status = consts.AdminStatusDisabled
	users.user.Status = consts.UserStatusDisabled
	if _, err := s.PrepareAdmin(context.Background(), 1, data); err == nil {
		t.Fatal("disabled admin accepted")
	}
	if _, err := s.PrepareUser(context.Background(), 7, data); err == nil {
		t.Fatal("disabled user accepted")
	}
	users.user.Status = consts.UserStatusEnabled
	users.user.DeletedAt.Valid = true
	if _, err := s.PrepareUser(context.Background(), 7, data); err == nil {
		t.Fatal("deleted user accepted")
	}
	users.user = nil
	if _, err := s.PrepareUser(context.Background(), 7, data); err == nil {
		t.Fatal("missing user accepted")
	}
	if _, err := s.PrepareAdmin(context.Background(), 999, data); err == nil {
		t.Fatal("missing admin accepted")
	}
}
