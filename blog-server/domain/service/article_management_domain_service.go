package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"context"
)

// ArticleManagementDomainService 提供受身份与归属约束的私有文章读取用例。
// 复用内部读取和身份领域服务，不接触 HTTP、Redis 或持久化模型。
type ArticleManagementDomainService struct {
	articles *ArticleDomainService
	users    *UserDomainService
	admins   *AdminDomainService
}

// NewArticleManagementDomainService 显式装配私有读取需要的领域服务。
func NewArticleManagementDomainService(articles *ArticleDomainService, users *UserDomainService, admins *AdminDomainService) *ArticleManagementDomainService {
	return &ArticleManagementDomainService{articles: articles, users: users, admins: admins}
}

// ListMyArticles 仅查询当前用户作者的文章；客户端不能指定其他作者扩大范围。
func (s *ArticleManagementDomainService) ListMyArticles(ctx context.Context, userID uint64, query *do.ArticleQuery) ([]do.ArticleDO, int64, error) {
	if err := validateManagedArticleQuery(query); err != nil {
		return nil, 0, err
	}
	if err := s.ensureUser(ctx, userID); err != nil {
		return nil, 0, err
	}
	query.UserID = &userID
	return s.articles.ListArticleByPage(ctx, query)
}

// GetMyArticle 对其他作者的文章返回不存在，不泄露其私有内容及存在性。
func (s *ArticleManagementDomainService) GetMyArticle(ctx context.Context, userID, articleID uint64) (*do.ArticleDO, error) {
	if err := s.ensureUser(ctx, userID); err != nil {
		return nil, err
	}
	article, err := s.articles.GetArticleById(ctx, articleID)
	if err != nil {
		return nil, err
	}
	if article.UserID != userID || article.AdminID != 0 {
		return nil, common.ArticleNotFoundErr
	}
	return article, nil
}

// ListAdminArticles 允许当前有效管理员查看全部未删除文章，包括草稿和下架文章。
func (s *ArticleManagementDomainService) ListAdminArticles(ctx context.Context, adminID uint64, query *do.ArticleQuery) ([]do.ArticleDO, int64, error) {
	if err := validateManagedArticleQuery(query); err != nil {
		return nil, 0, err
	}
	if err := ensureContentAdmin(ctx, s.admins, adminID); err != nil {
		return nil, 0, err
	}
	query.UserID = nil
	return s.articles.ListArticleByPage(ctx, query)
}

// GetAdminArticle 管理详情仍须在领域入口验证当前管理员资格。
func (s *ArticleManagementDomainService) GetAdminArticle(ctx context.Context, adminID, articleID uint64) (*do.ArticleDO, error) {
	if err := ensureContentAdmin(ctx, s.admins, adminID); err != nil {
		return nil, err
	}
	return s.articles.GetArticleById(ctx, articleID)
}

func (s *ArticleManagementDomainService) ensureUser(ctx context.Context, userID uint64) error {
	if userID == 0 {
		return common.LoginNotLoginErr
	}
	user, err := s.users.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil || user.ID != userID {
		return common.AuthErr
	}
	if user.Status != consts.UserStatusEnabled {
		return common.UserAccountDisabledErr
	}
	return nil
}

func validateManagedArticleQuery(query *do.ArticleQuery) error {
	if query == nil {
		return common.ParamErr
	}
	if query.Status != nil && (*query.Status < consts.ArticleStatusDraft || *query.Status > consts.ArticleStatusOffline) {
		return common.ParamErr
	}
	if query.IsTop != nil && *query.IsTop != consts.ArticleTopDisabled && *query.IsTop != consts.ArticleTopEnabled {
		return common.ParamErr
	}
	return nil
}
