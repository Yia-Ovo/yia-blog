package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"regexp"
)

var articleImageNamePattern = regexp.MustCompile(`^[a-f0-9]{64}\.(png|jpg)$`)

// ArticleImageDomainService 保证启用账号才能上传，并在落盘前去除元信息、限制尺寸。
type ArticleImageDomainService struct {
	admin *AdminDomainService
	user  *UserDomainService
}

func NewArticleImageDomainService(admin *AdminDomainService, user *UserDomainService) *ArticleImageDomainService {
	return &ArticleImageDomainService{admin: admin, user: user}
}

func articleImageParam(message string) common.Errno {
	err := common.ParamErr
	err.Msg = message
	return err
}

// ValidArticleImageName 只允许内容哈希及规范化扩展名，不接受任何路径片段。
func ValidArticleImageName(name string) bool { return articleImageNamePattern.MatchString(name) }

// PrepareAdmin 的 id 由可信入口提供，图片没有目标账号或文章字段。
func (s *ArticleImageDomainService) PrepareAdmin(ctx context.Context, id uint64, data []byte) (*do.ArticleImage, error) {
	if id == 0 {
		return nil, common.LoginNotLoginErr
	}
	admin, err := s.admin.FindAdminByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if admin == nil {
		return nil, common.AdminNotFoundErr
	}
	if admin.Status != consts.AdminStatusEnabled {
		return nil, common.PermissionErr
	}
	return NormalizeArticleImage(ctx, data)
}

func (s *ArticleImageDomainService) PrepareUser(ctx context.Context, id uint64, data []byte) (*do.ArticleImage, error) {
	if id == 0 {
		return nil, common.LoginNotLoginErr
	}
	user, err := s.user.FindUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, common.UserNotFoundErr
	}
	if user.Status != consts.UserStatusEnabled || user.IsDeleted {
		return nil, common.PermissionErr
	}
	return NormalizeArticleImage(ctx, data)
}

// NormalizeArticleImage 保留纵横比及 PNG 透明度，只缩小不放大；重编码移除 EXIF 等元信息。
func NormalizeArticleImage(ctx context.Context, data []byte) (*do.ArticleImage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > consts.ArticleImageMaxUploadBytes {
		return nil, articleImageParam("请选择不超过 5 MB 的图片")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		return nil, articleImageParam("只支持有效的 JPG 或 PNG 图片")
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > consts.ArticleImageMaxDimension || cfg.Height > consts.ArticleImageMaxDimension || int64(cfg.Width)*int64(cfg.Height) > consts.ArticleImageMaxPixels {
		return nil, articleImageParam("图片边长不能超过 8192 像素，总像素不能超过 400 万")
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, articleImageParam("图片已损坏，请选择其他图片")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	width, height := cfg.Width, cfg.Height
	if longest := max(width, height); longest > consts.ArticleImageOutputDimension {
		scale := float64(consts.ArticleImageOutputDimension) / float64(longest)
		width = max(1, int(math.Round(float64(width)*scale)))
		height = max(1, int(math.Round(float64(height)*scale)))
		source, err = resizeArticleImage(ctx, source, width, height)
		if err != nil {
			return nil, err
		}
	}
	var result bytes.Buffer
	contentType := "image/png"
	if format == "jpeg" {
		contentType = "image/jpeg"
		err = jpeg.Encode(&result, source, &jpeg.Options{Quality: consts.ArticleImageJPEGQuality})
	} else {
		err = png.Encode(&result, source)
	}
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if result.Len() > consts.ArticleImageMaxStoredBytes {
		return nil, articleImageParam("图片处理后过大，请选择其他图片")
	}
	return &do.ArticleImage{Data: result.Bytes(), ContentType: contentType, Width: width, Height: height}, nil
}

// resizeArticleImage 对预乘透明度做双线性采样，避免半透明边缘出现黑边。
func resizeArticleImage(ctx context.Context, source image.Image, width, height int) (image.Image, error) {
	bounds := source.Bounds()
	output := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fy := max(0, (float64(y)+0.5)*float64(bounds.Dy())/float64(height)-0.5)
		ya := int(math.Floor(fy))
		yb := min(ya+1, bounds.Dy()-1)
		wy := fy - float64(ya)
		for x := 0; x < width; x++ {
			fx := max(0, (float64(x)+0.5)*float64(bounds.Dx())/float64(width)-0.5)
			xa := int(math.Floor(fx))
			xb := min(xa+1, bounds.Dx()-1)
			wx := fx - float64(xa)
			points := [4]color.Color{source.At(bounds.Min.X+xa, bounds.Min.Y+ya), source.At(bounds.Min.X+xb, bounds.Min.Y+ya), source.At(bounds.Min.X+xa, bounds.Min.Y+yb), source.At(bounds.Min.X+xb, bounds.Min.Y+yb)}
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
	return output, nil
}
