package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"context"
	"errors"
)

// ArticleImageAppService 编排图片校验和保存，不写文章、账号资料或登录态缓存。
type ArticleImageAppService struct {
	domain *domainservice.ArticleImageDomainService
	files  repo.ArticleImageRepo
}

func NewArticleImageAppService(domain *domainservice.ArticleImageDomainService, files repo.ArticleImageRepo) *ArticleImageAppService {
	return &ArticleImageAppService{domain: domain, files: files}
}

func (s *ArticleImageAppService) UploadAdmin(ctx context.Context, id uint64, data []byte) (*dto.ArticleImageUploadResp, error) {
	image, err := s.domain.PrepareAdmin(ctx, id, data)
	if err != nil {
		return nil, err
	}
	return s.save(ctx, image)
}

func (s *ArticleImageAppService) UploadUser(ctx context.Context, id uint64, data []byte) (*dto.ArticleImageUploadResp, error) {
	image, err := s.domain.PrepareUser(ctx, id, data)
	if err != nil {
		return nil, err
	}
	return s.save(ctx, image)
}

func (s *ArticleImageAppService) save(ctx context.Context, image *do.ArticleImage) (*dto.ArticleImageUploadResp, error) {
	name, err := s.files.Save(ctx, image.Data)
	if err != nil {
		failed := common.ServerErr
		failed.Msg = "文章图片保存失败，请稍后重试"
		return nil, failed
	}
	return &dto.ArticleImageUploadResp{URL: consts.ArticleImageRoute + name, Width: image.Width, Height: image.Height, ContentType: image.ContentType}, nil
}

func (s *ArticleImageAppService) Image(ctx context.Context, name string) (*do.ArticleImage, error) {
	notFound := common.ParamErr
	notFound.Msg = "文章图片不存在"
	if !domainservice.ValidArticleImageName(name) {
		return nil, notFound
	}
	image, err := s.files.Read(ctx, name)
	if errors.Is(err, repo.ErrArticleImageNotFound) {
		return nil, notFound
	}
	if err != nil {
		return nil, common.ServerErr
	}
	return image, nil
}
