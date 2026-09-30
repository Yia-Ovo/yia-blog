package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"
	"sort"
	"time"
)

// AppearanceDomainService 维护按北京时间每日轮换、同日全站一致的规则。
type AppearanceDomainService struct{ repo repo.AppearanceRepo }

func NewAppearanceDomainService(repo repo.AppearanceRepo) *AppearanceDomainService {
	return &AppearanceDomainService{repo: repo}
}

// Today 按固定排序循环素材；不依赖进程启动时间或客户端时钟。
func (s *AppearanceDomainService) Today(ctx context.Context, now time.Time) (*do.DailyAppearanceDO, error) {
	images, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, common.ServerErr
	}
	images = append([]string(nil), images...)
	sort.Strings(images)
	local := now.In(time.FixedZone(consts.AppearanceTimezone, consts.AppearanceUTCOffset))
	year, month, day := local.Date()
	civilDay := time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix() / consts.AppearanceDaySeconds
	index := (civilDay%int64(len(images)) + int64(len(images))) % int64(len(images))
	midnight := time.Date(year, month, day, 0, 0, 0, 0, local.Location())
	return &do.DailyAppearanceDO{
		ImageID: images[index], Date: local.Format(time.DateOnly), NextChangeAt: midnight.AddDate(0, 0, 1),
	}, nil
}

// Image 允许读取素材目录中的历史图片，跨日后旧页面仍可完成加载。
func (s *AppearanceDomainService) Image(ctx context.Context, id string) ([]byte, error) {
	return s.repo.Read(ctx, id)
}
