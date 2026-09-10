package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"context"

	"go.uber.org/zap"
)

// CommentModerationQuery 来自管理页 HTTP 筛选参数，不含客户端可伪造的操作者字段。
type CommentModerationQuery struct {
	ArticleID uint64
	ParentID  *uint64
	Nickname  string
	Content   string
	Status    *int
	Page      int
	Size      int
	OrderBy   string
	OrderDir  string
}

// CommentReviewCommand 是 Handler 从审核 DTO 转换的用例输入，管理员 ID 单独来自鉴权。
type CommentReviewCommand struct {
	ID              uint64
	ExpectedContent string
	ExpectedStatus  int
	Status          int
}

// CommentModerationAppService 编排审核和公开计数缓存失效。
type CommentModerationAppService struct {
	domain *domainservice.CommentModerationDomainService
	cache  repo.CommentRedisRepo
}

// NewCommentModerationAppService 显式注入审核领域服务与缓存接口。
func NewCommentModerationAppService(domain *domainservice.CommentModerationDomainService, cache repo.CommentRedisRepo) *CommentModerationAppService {
	return &CommentModerationAppService{domain: domain, cache: cache}
}

// ListAdminComments 将内部评论转换为管理响应白名单。
func (s *CommentModerationAppService) ListAdminComments(ctx context.Context, adminID uint64, q *CommentModerationQuery) (*dto.PageResult, error) {
	if q == nil {
		return nil, common.ParamErr
	}
	query := &do.CommentQuery{ArticleID: q.ArticleID, ParentID: q.ParentID, Nickname: q.Nickname, Content: q.Content, Status: q.Status, Page: q.Page, Size: q.Size, OrderBy: q.OrderBy, OrderDir: q.OrderDir}
	comments, total, err := s.domain.ListAdminComments(ctx, adminID, query)
	if err != nil {
		return nil, err
	}
	list := make([]dto.ManagedComment, 0, len(comments))
	for _, comment := range comments {
		list = append(list, dto.ManagedComment{ID: comment.ID, ArticleID: comment.ArticleID, ParentID: comment.ParentID, Nickname: comment.Nickname, Avatar: comment.Avatar, Content: comment.Content, Status: comment.Status, Likes: comment.Likes, CreateAt: comment.CreateAt, UpdateAt: comment.UpdateAt})
	}
	return &dto.PageResult{Total: total, Pages: int((total + int64(query.Size) - 1) / int64(query.Size)), List: list, PageNum: query.Page, PageSize: query.Size}, nil
}

// ReviewComment 审核成功及幂等重试后均尝试清除公开计数缓存。
func (s *CommentModerationAppService) ReviewComment(ctx context.Context, adminID uint64, command *CommentReviewCommand) error {
	if command == nil {
		return common.ParamErr
	}
	articleID, err := s.domain.ReviewComment(ctx, adminID, &do.CommentReview{CommentID: command.ID, ExpectedContent: command.ExpectedContent, ExpectedStatus: command.ExpectedStatus, Status: command.Status})
	if err != nil {
		return err
	}
	if err := s.cache.DelArticleCommentCache(ctx, articleID); err != nil {
		logger.Warn("[APP] 评论审核已持久化，公开计数缓存失效失败", zap.Uint64("article_id", articleID), zap.Error(err))
	}
	return nil
}
