package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"context"
	"errors"
	"time"
)

// AppearanceAppService 编排领域选图与响应映射，素材无需 MySQL 或 Redis。
type AppearanceAppService struct {
	domain *domainservice.AppearanceDomainService
	now    func() time.Time
}

func NewAppearanceAppService(domain *domainservice.AppearanceDomainService) *AppearanceAppService {
	return &AppearanceAppService{domain: domain, now: time.Now}
}

func (s *AppearanceAppService) Today(ctx context.Context) (*dto.DailyAppearanceResp, error) {
	now := s.now()
	today, err := s.domain.Today(ctx, now)
	if err != nil {
		return nil, common.ServerErr
	}
	return &dto.DailyAppearanceResp{
		Date: today.Date, ImageURL: consts.AppearanceImageRoute + today.ImageID,
		ServerTime: now, NextChangeAt: today.NextChangeAt,
	}, nil
}

func (s *AppearanceAppService) Image(ctx context.Context, id string) ([]byte, error) {
	data, err := s.domain.Image(ctx, id)
	if errors.Is(err, repo.ErrAppearanceImageNotFound) {
		return nil, common.ParamErr
	}
	if err != nil {
		return nil, common.ServerErr
	}
	return data, nil
}
