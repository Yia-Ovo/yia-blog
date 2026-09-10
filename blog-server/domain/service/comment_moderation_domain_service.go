package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"

	"go.uber.org/zap"
)

// CommentModerationDomainService 承载管理员资格、审核状态流转和陈旧审核保护。
type CommentModerationDomainService struct {
	comments *CommentDomainService
	admins   *AdminDomainService
	reviews  repo.CommentReviewRepo
}

// NewCommentModerationDomainService 注入内部评论读取、管理员身份与审核持久化接口。
func NewCommentModerationDomainService(comments *CommentDomainService, admins *AdminDomainService, reviews repo.CommentReviewRepo) *CommentModerationDomainService {
	return &CommentModerationDomainService{comments: comments, admins: admins, reviews: reviews}
}

// ListAdminComments 允许管理员查看待审/通过/拒绝评论，始终排除软删除记录。
func (s *CommentModerationDomainService) ListAdminComments(ctx context.Context, adminID uint64, query *do.CommentQuery) ([]do.CommentDO, int64, error) {
	if query == nil || (query.Status != nil && !validReviewStatus(*query.Status)) {
		return nil, 0, common.ParamErr
	}
	if err := ensureContentAdmin(ctx, s.admins, adminID); err != nil {
		return nil, 0, err
	}
	query.WithDeleted = false
	query.PublishedArticleOnly = false
	return s.comments.ListCommentByPage(ctx, query)
}

// ReviewComment 只审核管理员看到的正文；允许通过、拒绝及撤下已通过内容。
// 返回文章 ID 给 App 失效公开计数缓存，不在领域层访问 Redis。
func (s *CommentModerationDomainService) ReviewComment(ctx context.Context, adminID uint64, review *do.CommentReview) (uint64, error) {
	if review == nil || review.CommentID == 0 || !validReviewStatus(review.ExpectedStatus) ||
		(review.Status != consts.CommentStatusPassed && review.Status != consts.CommentStatusRejected) {
		return 0, common.ParamErr
	}
	if err := ensureContentAdmin(ctx, s.admins, adminID); err != nil {
		return 0, err
	}
	comment, err := s.comments.GetCommentByID(ctx, review.CommentID)
	if err != nil {
		return 0, err
	}
	if comment.Content != review.ExpectedContent {
		return 0, staleCommentReview()
	}
	// 同一正文已经处于目标状态时，响应丢失后的重试可安全视为成功。
	if comment.Status == review.Status {
		return comment.ArticleID, nil
	}
	if comment.Status != review.ExpectedStatus {
		return 0, staleCommentReview()
	}
	rows, err := s.reviews.ReviewComment(ctx, review)
	if err != nil {
		logger.Error("[DOMAIN] 评论审核持久化失败", zap.Uint64("comment_id", review.CommentID), zap.Error(err))
		return 0, common.DatabaseErr
	}
	if rows == 0 {
		current, err := s.comments.GetCommentByID(ctx, review.CommentID)
		if err != nil {
			return 0, err
		}
		if current.Content == review.ExpectedContent && current.Status == review.Status {
			return current.ArticleID, nil
		}
		return 0, staleCommentReview()
	}
	logger.Info("[DOMAIN] 评论审核完成", zap.Uint64("admin_id", adminID), zap.Uint64("comment_id", review.CommentID), zap.Int("status", review.Status))
	return comment.ArticleID, nil
}

func validReviewStatus(status int) bool {
	return status == consts.CommentStatusPending || status == consts.CommentStatusPassed || status == consts.CommentStatusRejected
}

func staleCommentReview() common.Errno {
	return common.ParamErr.WithMsg("评论内容或状态已变化，请刷新后重新审核")
}
