/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-07 05:30:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-16 19:37:20
 * @FilePath: \yia-blog\blog-server\domain\service\comment_domain_service.go
 * @Description: 评论领域服务（纯业务规则，无任何Redis/基础设施依赖）
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"
)

const maxCommentContentLength = 1000

// CommentDomainService 纯净领域服务：仅依赖MySQL仓储，仅处理业务规则
type CommentDomainService struct {
	commentMysqlRepo repo.CommentMysqlRepo
	articleMysqlRepo repo.ArticleMysqlRepo
}

// NewCommentDomainService 构造函数：注入评论和文章MySQL仓储
func NewCommentDomainService(commentMysqlRepo repo.CommentMysqlRepo, articleMysqlRepo repo.ArticleMysqlRepo) *CommentDomainService {
	return &CommentDomainService{
		commentMysqlRepo: commentMysqlRepo,
		articleMysqlRepo: articleMysqlRepo,
	}
}

// CreateComment 创建评论：仅业务校验 + 数据库操作
func (c *CommentDomainService) CreateComment(ctx context.Context, commentDO *do.CommentDO) error {
	if commentDO == nil {
		return common.ParamErr
	}
	if commentDO.UserID == 0 {
		logger.Warn("[DOMAIN] 创建评论失败：用户ID非法")
		return common.LoginNotLoginErr
	}

	// 业务规则校验
	if commentDO.ArticleID <= 0 {
		logger.Warn("[DOMAIN] 创建评论失败：文章ID非法", zap.Uint64("article_id", commentDO.ArticleID))
		return common.ParamErr
	}
	if commentDO.Nickname == "" {
		logger.Warn("[DOMAIN] 创建评论失败：昵称不能为空", zap.Uint64("article_id", commentDO.ArticleID))
		return common.ErrCommentNicknameEmpty
	}
	if err := validateCommentContent(commentDO.Content); err != nil {
		return err
	}

	article, err := c.articleMysqlRepo.GetArticleById(ctx, commentDO.ArticleID)
	if err != nil {
		logger.Error("[DOMAIN] 创建评论失败：查询文章失败", zap.Uint64("article_id", commentDO.ArticleID), zap.Error(err))
		return common.DatabaseErr
	}
	if article == nil || article.Status != consts.ArticleStatusPublished {
		logger.Warn("[DOMAIN] 创建评论失败：文章不存在", zap.Uint64("article_id", commentDO.ArticleID))
		return common.ArticleNotFoundErr
	}
	if article.CommentStatus != consts.ArticleCommentsEnabled {
		return common.PermissionDeniedErr.WithMsg("文章已关闭评论")
	}

	if commentDO.ParentID != 0 {
		parent, err := c.commentMysqlRepo.GetCommentByID(ctx, commentDO.ParentID)
		if err != nil {
			logger.Error("[DOMAIN] 创建评论失败：查询父评论失败", zap.Uint64("parent_id", commentDO.ParentID), zap.Error(err))
			return common.DatabaseErr
		}
		if parent == nil {
			logger.Warn("[DOMAIN] 创建评论失败：父评论不存在", zap.Uint64("parent_id", commentDO.ParentID))
			return common.CommentNotFoundErr
		}
		if parent.ArticleID != commentDO.ArticleID {
			logger.Warn("[DOMAIN] 创建评论失败：父评论不属于当前文章",
				zap.Uint64("article_id", commentDO.ArticleID),
				zap.Uint64("parent_id", commentDO.ParentID),
				zap.Uint64("parent_article_id", parent.ArticleID),
			)
			return common.ParamErr
		}
	}

	// 创建用户评论时，作者类型和安全字段始终由领域规则决定。
	commentDO.AdminID = 0
	commentDO.Status = consts.CommentStatusPending
	commentDO.Likes = 0
	commentDO.CreateAt = time.Now()
	commentDO.UpdateAt = commentDO.CreateAt

	// DO转Model
	commentModel := convert.CommentDOToModel(commentDO)

	// 调用MySQL仓储
	err = c.commentMysqlRepo.CreateComment(ctx, commentModel)
	if err != nil {
		logger.Error("[DOMAIN] 创建评论数据库操作失败", zap.Uint64("article_id", commentDO.ArticleID), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("[DOMAIN] 创建评论成功", zap.Uint64("article_id", commentDO.ArticleID))
	return nil
}

// ListCommentByPage 评论分页查询：仅参数校验 + 数据库查询
// 该方法信任调用方传入的 Status 等筛选条件，只允许管理端用例使用；
// 公开入口必须走 ListPublicComments，由领域强制可见范围。
func (c *CommentDomainService) ListCommentByPage(ctx context.Context, query *do.CommentQuery) ([]do.CommentDO, int64, error) {
	// 分页参数合法性校验
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Size <= 0 || query.Size > 50 {
		query.Size = 10
	}

	// 查询数据库
	list, total, err := c.commentMysqlRepo.ListCommentByPage(ctx, query)
	if err != nil {
		logger.Error("分页查询评论失败", zap.Any("query", query), zap.Error(err))
		return nil, 0, common.DatabaseErr
	}

	logger.Info("分页查询评论成功", zap.Int64("total", total))
	return convert.CommentModelListToDOList(list), total, nil
}

// GetCommentByID 根据ID查询评论：仅参数校验 + 数据库查询
// 该方法不做可见性判断，会返回任意审核状态的评论，只允许内部流程和管理端用例使用；
// 公开入口必须走 GetPublicComment。
func (c *CommentDomainService) GetCommentByID(ctx context.Context, commentID uint64) (*do.CommentDO, error) {
	// 参数校验
	if commentID <= 0 {
		logger.Warn("[DOMAIN] 根据ID查询评论失败：ID非法", zap.Uint64("comment_id", commentID))
		return nil, common.ParamErr
	}

	// 查询数据库
	commentModel, err := c.commentMysqlRepo.GetCommentByID(ctx, commentID)
	if err != nil {
		logger.Error("[DOMAIN] 根据ID查询评论失败", zap.Uint64("comment_id", commentID), zap.Error(err))
		return nil, common.DatabaseErr
	}

	// 数据不存在
	if commentModel == nil {
		logger.Warn("[DOMAIN] 评论不存在", zap.Uint64("comment_id", commentID))
		return nil, common.CommentNotFoundErr
	}

	logger.Info("[DOMAIN] 根据ID查询评论成功", zap.Uint64("comment_id", commentID))
	return convert.CommentModelToDO(commentModel), nil
}

// EnsureArticlePublic 校验文章当前对访客可见；缓存命中也必须先通过该领域规则。
func (c *CommentDomainService) EnsureArticlePublic(ctx context.Context, articleID uint64) error {
	if articleID == 0 {
		return common.ParamErr
	}
	article, err := c.articleMysqlRepo.GetArticleById(ctx, articleID)
	if err != nil {
		return common.DatabaseErr
	}
	if article == nil || article.Status != consts.ArticleStatusPublished {
		return common.ArticleNotFoundErr
	}
	return nil
}

// ListPublicComments 公开评论分页只允许已发布文章下的已通过评论。
// 调用方传入的审核状态和软删除条件一律被覆盖，因此无论访客怎么构造筛选参数，
// 都无法翻出待审核或已拒绝的评论；这条规则对任何入口（HTTP、任务、未来 RPC）都成立。
func (c *CommentDomainService) ListPublicComments(ctx context.Context, query *do.CommentQuery) ([]do.CommentDO, int64, error) {
	if query == nil {
		logger.Warn("[DOMAIN] 公开评论分页查询失败：查询条件为空")
		return nil, 0, common.ParamErr
	}
	if query.ArticleID != 0 {
		if err := c.EnsureArticlePublic(ctx, query.ArticleID); err != nil {
			return nil, 0, err
		}
	}

	// 服务端钉死可见范围，忽略调用方传入值
	passed := consts.CommentStatusPassed
	query.Status = &passed
	query.WithDeleted = false
	// ArticleID=0 时仍需通过关联文章筛选，不能靠单个 ID 的预检替代 SQL 范围。
	query.PublishedArticleOnly = true

	return c.ListCommentByPage(ctx, query)
}

// GetPublicComment 公开评论详情：只返回审核通过的评论。
// 未通过审核的评论按"不存在"处理并复用 CommentNotFoundErr，
// 避免访客通过错误码差异探测某条评论是否存在、是否被拒绝。
func (c *CommentDomainService) GetPublicComment(ctx context.Context, commentID uint64) (*do.CommentDO, error) {
	comment, err := c.GetCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}

	if comment.Status != consts.CommentStatusPassed {
		logger.Warn("[DOMAIN] 公开查询命中未通过审核的评论，按不存在处理",
			zap.Uint64("comment_id", commentID),
			zap.Int("status", comment.Status),
		)
		return nil, common.CommentNotFoundErr
	}

	if err := c.EnsureArticlePublic(ctx, comment.ArticleID); err != nil {
		if errors.Is(err, common.ArticleNotFoundErr) || errors.Is(err, common.ParamErr) {
			return nil, common.CommentNotFoundErr
		}
		return nil, err
	}
	return comment, nil
}

// UpdateComment 更新评论：仅业务校验 + 数据库操作
func (c *CommentDomainService) UpdateComment(ctx context.Context, operatorID uint64, commentID uint64, content string) (uint64, error) {
	if operatorID == 0 || commentID == 0 {
		return 0, common.ParamErr
	}
	if err := validateCommentContent(content); err != nil {
		return 0, err
	}
	comment, err := c.GetCommentByID(ctx, commentID)
	if err != nil {
		return 0, err
	}

	if comment.UserID != operatorID {
		return 0, common.PermissionDeniedErr.WithMsg("无权修改该评论")
	}
	if comment.Content == content {
		return comment.ArticleID, nil
	}

	rows, err := c.commentMysqlRepo.UpdateCommentByUser(
		ctx,
		commentID,
		operatorID,
		content,
		consts.CommentStatusPending,
	)
	if err != nil {
		logger.Error("[DOMAIN] 更新评论失败", zap.Error(err))
		return 0, common.DatabaseErr
	}
	if rows == 0 {
		return 0, common.CommentNotFoundErr
	}

	return comment.ArticleID, nil
}

func validateCommentContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return common.ErrCommentContentEmpty
	}
	if utf8.RuneCountInString(content) > maxCommentContentLength {
		return common.ParamErr
	}
	return nil
}

// DeleteComment 删除评论(软删除)：仅参数校验 + 数据库操作
func (c *CommentDomainService) DeleteComment(
	ctx context.Context,
	operatorID uint64,
	commentID uint64,
) (uint64, error) {
	if operatorID == 0 || commentID == 0 {
		return 0, common.ParamErr
	}

	comment, err := c.GetCommentByID(ctx, commentID)
	if err != nil {
		return 0, err
	}

	if comment.UserID != operatorID {
		return 0, common.PermissionDeniedErr.WithMsg("无权删除该评论")
	}

	rows, err := c.commentMysqlRepo.DeleteCommentByUser(
		ctx,
		commentID,
		operatorID,
	)
	if err != nil {
		logger.Error("[DOMAIN] 删除评论失败", zap.Error(err))
		return 0, common.DatabaseErr
	}
	if rows == 0 {
		return 0, common.CommentNotFoundErr
	}

	return comment.ArticleID, nil
}

// LikeComment 评论点赞：仅业务校验（评论存在）+ 数据库点赞
// Redis 相关逻辑全部移至应用服务
func (c *CommentDomainService) LikeComment(ctx context.Context, commentID uint64) error {
	// 校验评论是否存在
	comment, err := c.GetCommentByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return common.CommentNotFoundErr
	}

	// 数据库点赞+1
	err = c.commentMysqlRepo.LikeComment(ctx, commentID)
	if err != nil {
		logger.Error("评论点赞数据库操作失败", zap.Uint64("comment_id", commentID), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("评论点赞数据库操作成功", zap.Uint64("comment_id", commentID))
	return nil
}

// DisLikeComment 取消评论点赞：仅数据库取消点赞
// Redis 相关逻辑全部移至应用服务
func (c *CommentDomainService) DisLikeComment(ctx context.Context, commentID uint64) error {
	// 校验评论是否存在
	comment, err := c.GetCommentByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return common.CommentNotFoundErr
	}

	// 数据库点赞-1
	err = c.commentMysqlRepo.DisLikeComment(ctx, commentID)
	if err != nil {
		logger.Error("取消点赞数据库操作失败", zap.Uint64("comment_id", commentID), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("取消评论点赞数据库操作成功", zap.Uint64("comment_id", commentID))
	return nil
}
