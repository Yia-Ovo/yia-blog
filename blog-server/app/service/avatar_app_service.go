package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"context"
	"errors"
)

// AvatarAppService 编排规范化、文件落盘、本人资料更新和缓存失效，不改变登录态。
type AvatarAppService struct {
	domain     *domainservice.AvatarDomainService
	files      repo.AvatarRepo
	adminCache repo.AdminRedisRepo
	userCache  repo.UserRedisRepo
}

func NewAvatarAppService(domain *domainservice.AvatarDomainService, files repo.AvatarRepo, adminCache repo.AdminRedisRepo, userCache repo.UserRedisRepo) *AvatarAppService {
	return &AvatarAppService{domain: domain, files: files, adminCache: adminCache, userCache: userCache}
}

func avatarStorageError() common.Errno {
	err := common.ServerErr
	err.Msg = "头像保存失败，请稍后重试"
	return err
}

func avatarCacheError() common.Errno {
	err := common.RedisErr
	err.Msg = "头像已保存，但资料缓存刷新失败，请稍后刷新页面"
	return err
}

// UploadAdmin 的 id 必须由调用入口从已验证身份取得，文件数据不携带目标账号。
func (s *AvatarAppService) UploadAdmin(ctx context.Context, id uint64, data []byte) (*dto.AvatarUploadResp, error) {
	image, err := s.domain.PrepareAdmin(ctx, id, data)
	if err != nil {
		return nil, err
	}
	name, err := s.files.Save(ctx, image)
	if err != nil {
		return nil, avatarStorageError()
	}
	if err := s.domain.SetAdminAvatar(ctx, id, name); err != nil {
		return nil, err
	}
	if err := s.adminCache.DelAdminFromRedis(ctx, id); err != nil {
		return nil, avatarCacheError()
	}
	return &dto.AvatarUploadResp{Avatar: consts.AvatarImageRoute + name}, nil
}

func (s *AvatarAppService) UploadUser(ctx context.Context, id uint64, data []byte) (*dto.AvatarUploadResp, error) {
	image, err := s.domain.PrepareUser(ctx, id, data)
	if err != nil {
		return nil, err
	}
	name, err := s.files.Save(ctx, image)
	if err != nil {
		return nil, avatarStorageError()
	}
	if err := s.domain.SetUserAvatar(ctx, id, name); err != nil {
		return nil, err
	}
	if err := s.userCache.DelUserFromRedis(ctx, id); err != nil {
		return nil, avatarCacheError()
	}
	return &dto.AvatarUploadResp{Avatar: consts.AvatarImageRoute + name}, nil
}

func (s *AvatarAppService) Image(ctx context.Context, name string) ([]byte, error) {
	notFound := common.ParamErr
	notFound.Msg = "头像图片不存在"
	if !domainservice.ValidAvatarName(name) {
		return nil, notFound
	}
	data, err := s.files.Read(ctx, name)
	if errors.Is(err, repo.ErrAvatarNotFound) {
		return nil, notFound
	}
	if err != nil {
		return nil, common.ServerErr
	}
	return data, nil
}
