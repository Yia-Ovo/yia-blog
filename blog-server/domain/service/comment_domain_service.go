/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-07 05:30:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-26 00:03:26
 * @FilePath: \yia-blog\blog-server\domain\service\comment_domain_service.go
 * @Description: 评论领域服务（纯业务规则，无任何Redis/基础设施依赖）
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/common"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"

	"go.uber.org/zap"
)

// CommentDomainService 纯净领域服务：仅依赖MySQL仓储，仅处理业务规则
type CommentDomainService struct {
	commentMysqlRepo repo.CommentMysqlRepo
}

// NewCommentDomainService 构造函数：仅注入MySQL依赖
func NewCommentDomainService(commentMysqlRepo repo.CommentMysqlRepo) *CommentDomainService {
	return &CommentDomainService{
		commentMysqlRepo: commentMysqlRepo,
	}
}

// CreateComment 创建评论：仅业务校验 + 数据库操作
func (c *CommentDomainService) CreateComment(ctx context.Context, commentDO *do.CommentDO) error {
	// 业务规则校验
	if commentDO.ArticleID <= 0 {
		logger.Warn("创建评论失败：文章ID非法", zap.Uint64("article_id", commentDO.ArticleID))
		return common.ParamErr
	}
	if commentDO.Nickname == "" {
		logger.Warn("创建评论失败：昵称不能为空", zap.Uint64("article_id", commentDO.ArticleID))
		return common.ErrCommentNicknameEmpty
	}
	if commentDO.Content == "" {
		logger.Warn("创建评论失败：评论内容不能为空", zap.Uint64("article_id", commentDO.ArticleID))
		return common.ErrCommentContentEmpty
	}

	// DO转Model
	commentModel := convert.CommentDOToModel(commentDO)

	// 调用MySQL仓储
	err := c.commentMysqlRepo.CreateComment(ctx, commentModel)
	if err != nil {
		logger.Error("创建评论数据库操作失败", zap.Uint64("article_id", commentDO.ArticleID), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("创建评论成功", zap.Uint64("article_id", commentDO.ArticleID))
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
		logger.Warn("根据ID查询评论失败：ID非法", zap.Uint64("comment_id", commentID))
		return nil, common.ParamErr
	}

	// 查询数据库
	commentModel, err := c.commentMysqlRepo.GetCommentByID(ctx, commentID)
	if err != nil {
		logger.Error("根据ID查询评论失败", zap.Uint64("comment_id", commentID), zap.Error(err))
		return nil, common.DatabaseErr
	}

	// 数据不存在
	if commentModel == nil {
		logger.Warn("评论不存在", zap.Uint64("comment_id", commentID))
		return nil, common.CommentNotFoundErr
	}

	logger.Info("根据ID查询评论成功", zap.Uint64("comment_id", commentID))
	return convert.CommentModelToDO(commentModel), nil
}

// UpdateComment 更新评论：仅业务校验 + 数据库操作
func (c *CommentDomainService) UpdateComment(ctx context.Context, commentDO *do.CommentDO) error {
	// 业务规则校验
	if commentDO.ID <= 0 {
		logger.Warn("更新评论失败：ID非法", zap.Uint64("comment_id", commentDO.ID))
		return common.ParamErr
	}
	if commentDO.Content == "" {
		logger.Warn("更新评论失败：内容不能为空", zap.Uint64("comment_id", commentDO.ID))
		return common.ErrCommentContentEmpty
	}

	// DO转Model
	commentModel := convert.CommentDOToModel(commentDO)

	// 更新数据库
	err := c.commentMysqlRepo.UpdateComment(ctx, commentModel)
	if err != nil {
		logger.Error("更新评论数据库操作失败", zap.Uint64("comment_id", commentDO.ID), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("更新评论成功", zap.Uint64("comment_id", commentDO.ID))
	return nil
}

// DeleteComment 删除评论(软删除)：仅参数校验 + 数据库操作
func (c *CommentDomainService) DeleteComment(ctx context.Context, commentID uint64) error {
	// 参数校验
	if commentID <= 0 {
		logger.Warn("删除评论失败：ID非法", zap.Uint64("comment_id", commentID))
		return common.ParamErr
	}

	// 删除数据库数据（软删除）
	err := c.commentMysqlRepo.DeleteComment(ctx, commentID)
	if err != nil {
		logger.Error("删除评论数据库操作失败", zap.Uint64("comment_id", commentID), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("删除评论成功", zap.Uint64("comment_id", commentID))
	return nil
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
