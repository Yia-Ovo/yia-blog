package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"bytes"
	"context"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"math"
	"regexp"
)

var avatarNamePattern = regexp.MustCompile(`^[a-f0-9]{64}\.png$`)

// AvatarDomainService 保证只有启用账号可以修改自己的头像，图片进入存储前统一裁切编码。
type AvatarDomainService struct {
	admin *AdminDomainService
	user  *UserDomainService
}

func NewAvatarDomainService(admin *AdminDomainService, user *UserDomainService) *AvatarDomainService {
	return &AvatarDomainService{admin: admin, user: user}
}

func avatarParam(message string) common.Errno {
	err := common.ParamErr
	err.Msg = message
	return err
}

// ValidAvatarName 校验公开图片标识；名称不能包含任何路径片段。
func ValidAvatarName(name string) bool { return avatarNamePattern.MatchString(name) }

func (s *AvatarDomainService) checkAdmin(ctx context.Context, id uint64) error {
	if id == 0 {
		return common.LoginNotLoginErr
	}
	admin, err := s.admin.FindAdminByID(ctx, id)
	if err != nil {
		return err
	}
	if admin == nil {
		return common.AdminNotFoundErr
	}
	if admin.Status != consts.AdminStatusEnabled {
		return common.PermissionErr
	}
	return nil
}

func (s *AvatarDomainService) checkUser(ctx context.Context, id uint64) error {
	if id == 0 {
		return common.LoginNotLoginErr
	}
	user, err := s.user.FindUserByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return common.UserNotFoundErr
	}
	if user.Status != consts.UserStatusEnabled || user.IsDeleted {
		return common.PermissionErr
	}
	return nil
}

func (s *AvatarDomainService) PrepareAdmin(ctx context.Context, id uint64, data []byte) ([]byte, error) {
	if err := s.checkAdmin(ctx, id); err != nil {
		return nil, err
	}
	return NormalizeAvatar(ctx, data)
}

func (s *AvatarDomainService) PrepareUser(ctx context.Context, id uint64, data []byte) ([]byte, error) {
	if err := s.checkUser(ctx, id); err != nil {
		return nil, err
	}
	return NormalizeAvatar(ctx, data)
}

// SetAdminAvatar 只更新当前身份的头像，不能携带其他目标或资料字段。
func (s *AvatarDomainService) SetAdminAvatar(ctx context.Context, id uint64, name string) error {
	if !ValidAvatarName(name) {
		return avatarParam("头像地址无效")
	}
	if err := s.checkAdmin(ctx, id); err != nil {
		return err
	}
	url := consts.AvatarImageRoute + name
	return s.admin.UpdateAdminInfo(ctx, id, &do.AdminInfoUpdate{Avatar: &url})
}

func (s *AvatarDomainService) SetUserAvatar(ctx context.Context, id uint64, name string) error {
	if !ValidAvatarName(name) {
		return avatarParam("头像地址无效")
	}
	if err := s.checkUser(ctx, id); err != nil {
		return err
	}
	url := consts.AvatarImageRoute + name
	return s.user.UpdateUserInfo(ctx, id, &do.UserInfoUpdate{Avatar: &url})
}

// NormalizeAvatar 先检查解码尺寸，再中心裁切并缩小到最多 256 像素；重新编码会移除元信息。
func NormalizeAvatar(ctx context.Context, data []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > consts.AvatarMaxUploadBytes {
		return nil, avatarParam("请选择不超过 5 MB 的图片")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		return nil, avatarParam("只支持有效的 JPG 或 PNG 图片")
	}
	if config.Width < 1 || config.Height < 1 || config.Width > consts.AvatarMaxDimension || config.Height > consts.AvatarMaxDimension || int64(config.Width)*int64(config.Height) > consts.AvatarMaxPixels {
		return nil, avatarParam("图片边长不能超过 4096 像素，总像素不能超过 400 万")
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, avatarParam("图片已损坏，请选择其他图片")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b := source.Bounds()
	side := min(b.Dx(), b.Dy())
	target := min(side, consts.AvatarOutputDimension)
	output := image.NewNRGBA(image.Rect(0, 0, target, target))
	x0, y0 := b.Min.X+(b.Dx()-side)/2, b.Min.Y+(b.Dy()-side)/2
	// 双线性取样保留透明通道；小图只裁切，不放大。
	for y := 0; y < target; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fy := max(0, (float64(y)+0.5)*float64(side)/float64(target)-0.5)
		ya := int(math.Floor(fy))
		yb := min(ya+1, side-1)
		wy := fy - float64(ya)
		for x := 0; x < target; x++ {
			fx := max(0, (float64(x)+0.5)*float64(side)/float64(target)-0.5)
			xa := int(math.Floor(fx))
			xb := min(xa+1, side-1)
			wx := fx - float64(xa)
			points := [4]color.Color{source.At(x0+xa, y0+ya), source.At(x0+xb, y0+ya), source.At(x0+xa, y0+yb), source.At(x0+xb, y0+yb)}
			weights := [4]float64{(1 - wx) * (1 - wy), wx * (1 - wy), (1 - wx) * wy, wx * wy}
			var r, g, b, a float64
			for i, p := range points {
				pr, pg, pb, pa := p.RGBA()
				r += float64(pr) * weights[i]
				g += float64(pg) * weights[i]
				b += float64(pb) * weights[i]
				a += float64(pa) * weights[i]
			}
			pixel := color.NRGBA{}
			if a > 0 {
				pixel = color.NRGBA{R: uint8(math.Min(255, r*255/a) + 0.5), G: uint8(math.Min(255, g*255/a) + 0.5), B: uint8(math.Min(255, b*255/a) + 0.5), A: uint8(a/257 + 0.5)}
			}
			output.SetNRGBA(x, y, pixel)
		}
	}
	var result bytes.Buffer
	if err := png.Encode(&result, output); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}
