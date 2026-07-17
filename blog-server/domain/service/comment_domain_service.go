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
	if article == nil {
		logger.Warn("[DOMAIN] 创建评论失败：文章不存在", zap.Uint64("article_id", commentDO.ArticleID))
		return common.ArticleNotFoundErr
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
