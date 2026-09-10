package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/domain/do"
	domainservice "blog-server/domain/service"
	"context"
)

// ArticleManagementQuery 来自 Handler 绑定的筛选参数，不承载当前操作者身份。
type ArticleManagementQuery struct {
	PageNum    int
	PageSize   int
	Title      string
	CategoryID *uint64
	Status     *int
	IsTop      *int
}

// ArticleManagementAppService 编排管理读取及响应映射，权限规则由 Domain 执行。
type ArticleManagementAppService struct {
	domain *domainservice.ArticleManagementDomainService
}

// NewArticleManagementAppService 构造管理读取应用服务。
func NewArticleManagementAppService(domain *domainservice.ArticleManagementDomainService) *ArticleManagementAppService {
	return &ArticleManagementAppService{domain: domain}
}

// ListMyArticles 读取本人文章列表。
func (s *ArticleManagementAppService) ListMyArticles(ctx context.Context, userID uint64, query *ArticleManagementQuery) (*dto.PageResult, error) {
	return s.list(ctx, userID, query, s.domain.ListMyArticles)
}

// ListAdminArticles 读取管理员文章列表。
func (s *ArticleManagementAppService) ListAdminArticles(ctx context.Context, adminID uint64, query *ArticleManagementQuery) (*dto.PageResult, error) {
	return s.list(ctx, adminID, query, s.domain.ListAdminArticles)
}

// GetMyArticle 读取本人文章详情。
func (s *ArticleManagementAppService) GetMyArticle(ctx context.Context, userID, articleID uint64) (*dto.ManagedArticle, error) {
	article, err := s.domain.GetMyArticle(ctx, userID, articleID)
	if err != nil {
		return nil, err
	}
	result := managedArticle(article, true)
	return &result, nil
}

// GetAdminArticle 读取管理员文章详情。
func (s *ArticleManagementAppService) GetAdminArticle(ctx context.Context, adminID, articleID uint64) (*dto.ManagedArticle, error) {
	article, err := s.domain.GetAdminArticle(ctx, adminID, articleID)
	if err != nil {
		return nil, err
	}
	result := managedArticle(article, true)
	return &result, nil
}

func (s *ArticleManagementAppService) list(ctx context.Context, actorID uint64, q *ArticleManagementQuery, load func(context.Context, uint64, *do.ArticleQuery) ([]do.ArticleDO, int64, error)) (*dto.PageResult, error) {
	if q == nil {
		return nil, common.ParamErr
	}
	query := &do.ArticleQuery{PageNum: q.PageNum, PageSize: q.PageSize, Title: q.Title, CategoryID: q.CategoryID, Status: q.Status, IsTop: q.IsTop}
	articles, total, err := load(ctx, actorID, query)
	if err != nil {
		return nil, err
	}
	list := make([]dto.ManagedArticle, 0, len(articles))
	for i := range articles {
		list = append(list, managedArticle(&articles[i], false))
	}
	return &dto.PageResult{Total: total, Pages: int((total + int64(query.PageSize) - 1) / int64(query.PageSize)), List: list, PageNum: query.PageNum, PageSize: query.PageSize}, nil
}

func managedArticle(article *do.ArticleDO, detail bool) dto.ManagedArticle {
	result := dto.ManagedArticle{
		ID: article.ID, UserID: article.UserID, AdminID: article.AdminID,
		Title: article.Title, Cover: article.Cover, Summary: article.Summary,
		CategoryID: article.CategoryID, TagIDs: article.TagIDs, Status: article.Status,
		Sort: article.Sort, IsTop: article.IsTop, CommentStatus: article.CommentStatus,
		PublishAt: article.PublishAt, CreateAt: article.CreateAt, UpdateAt: article.UpdateAt,
	}
	if detail {
		result.Content = article.Content
	}
	return result
}
