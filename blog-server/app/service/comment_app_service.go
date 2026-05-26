/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-25 14:32:37
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-26 00:25:09
 * @FilePath: \yia-blog\blog-server\app\service\comment_app_service.go
 * @Description: 评论应用服务 - 业务编排、缓存处理、流程调度
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package service

import (
	"blog-server/common"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/domain/service"
	"context"

	"go.uber.org/zap"
)

// CommentAppService 应用服务：仅做流程编排 + 缓存处理
type CommentAppService struct {
	commentDomainService *service.CommentDomainService // 领域服务（纯业务）
	commentRedisRepo     repo.CommentRedisRepo         // 缓存仓储（应用层编排）
}

// NewCommentAppService 构造函数：依赖外部注入，无强耦合
func NewCommentAppService(
	domainService *service.CommentDomainService,
	redisRepo repo.CommentRedisRepo,
) *CommentAppService {
	return &CommentAppService{
		commentDomainService: domainService,
		commentRedisRepo:     redisRepo,
	}
}

// CreateComment 发布评论：调用领域服务 + 清理缓存
func (c *CommentAppService) CreateComment(ctx context.Context, commentDO *do.CommentDO) error {
	if commentDO == nil {
		logger.Warn("发布评论参数为空")
		return common.ParamErr
	}
	// 1. 执行核心业务逻辑
	err := c.commentDomainService.CreateComment(ctx, commentDO)
	if err != nil {
		return err
	}
	// 2. 应用层编排：清理缓存（保证数据一致性）
	_ = c.commentRedisRepo.DelArticleCommentCache(ctx, commentDO.ArticleID)
	logger.Info("创建评论，缓存已清理", zap.Uint64("article_id", commentDO.ArticleID))
	return nil
}

// ListCommentByPage 分页查询评论
func (c *CommentAppService) ListCommentByPage(ctx context.Context, query *do.CommentQuery) ([]do.CommentDO, int64, error) {
	if query == nil {
		logger.Warn("评论分页查询参数为空")
		return nil, 0, common.ParamErr
	}
	return c.commentDomainService.ListCommentByPage(ctx, query)
}

// GetCommentByID 根据ID获取评论详情
func (c *CommentAppService) GetCommentByID(ctx context.Context, commentID uint64) (*do.CommentDO, error) {
	return c.commentDomainService.GetCommentByID(ctx, commentID)
}

// UpdateComment 编辑评论：调用领域服务 + 清理缓存
func (c *CommentAppService) UpdateComment(ctx context.Context, commentDO *do.CommentDO) error {
	if commentDO == nil {
		logger.Warn("编辑评论参数为空")
		return common.ParamErr
	}
	// 1. 执行核心业务逻辑
	err := c.commentDomainService.UpdateComment(ctx, commentDO)
	if err != nil {
		return err
	}
	// 2. 应用层编排：清理缓存
	_ = c.commentRedisRepo.DelArticleCommentCache(ctx, commentDO.ArticleID)
	logger.Info("更新评论，缓存已清理", zap.Uint64("article_id", commentDO.ArticleID))
	return nil
}

// DeleteComment 删除评论：调用领域服务 + 清理缓存
func (c *CommentAppService) DeleteComment(ctx context.Context, commentID uint64) error {
	// 1. 先查询评论（获取文章ID）
	comment, err := c.commentDomainService.GetCommentByID(ctx, commentID)
	if err != nil {
		return err
	}
	// 2. 执行核心业务逻辑
	err = c.commentDomainService.DeleteComment(ctx, commentID)
	if err != nil {
		return err
	}
	// 3. 应用层编排：清理缓存
	_ = c.commentRedisRepo.DelArticleCommentCache(ctx, comment.ArticleID)
	logger.Info("删除评论，缓存已清理", zap.Uint64("article_id", comment.ArticleID))
	return nil
}

// LikeComment 评论点赞：应用层处理Redis防重 + 调用领域服务DB操作
func (c *CommentAppService) LikeComment(ctx context.Context, userID uint64, commentID uint64) error {
	// 参数校验
	if userID <= 0 || commentID <= 0 {
		logger.Warn("评论点赞参数非法", zap.Uint64("user_id", userID), zap.Uint64("comment_id", commentID))
		return common.ParamErr
	}

	// 1. 应用层编排：Redis校验重复点赞（防重逻辑）
	exist, err := c.commentRedisRepo.CheckUserLikeComment(ctx, userID, commentID)
	if err != nil {
		logger.Warn("检查点赞记录失败，降级执行", zap.Error(err))
	}
	if exist {
		return common.ErrCommentAlreadyLiked
	}

	// 2. 调用领域服务：执行数据库点赞（纯业务）
	err = c.commentDomainService.LikeComment(ctx, commentID)
	if err != nil {
		return err
	}

	// 3. 应用层编排：更新Redis缓存
	_ = c.commentRedisRepo.IncrCommentLikeCount(ctx, commentID)
	_ = c.commentRedisRepo.AddUserLikeRecord(ctx, userID, commentID)
	logger.Info("评论点赞成功", zap.Uint64("user_id", userID), zap.Uint64("comment_id", commentID))
	return nil
}

// DisLikeComment 取消点赞：应用层处理Redis + 调用领域服务DB操作
func (c *CommentAppService) DisLikeComment(ctx context.Context, userID uint64, commentID uint64) error {
	// 参数校验
	if userID <= 0 || commentID <= 0 {
		logger.Warn("取消点赞参数非法", zap.Uint64("user_id", userID), zap.Uint64("comment_id", commentID))
		return common.ParamErr
	}

	// 1. 应用层编排：Redis校验是否已点赞
	exist, err := c.commentRedisRepo.CheckUserLikeComment(ctx, userID, commentID)
	if err != nil {
		logger.Warn("检查点赞记录失败，降级执行", zap.Error(err))
	}
	if !exist {
		return common.ErrCommentNotLiked
	}

	// 2. 调用领域服务：执行数据库取消点赞
	err = c.commentDomainService.DisLikeComment(ctx, commentID)
	if err != nil {
		return err
	}

	// 3. 应用层编排：更新Redis缓存
	_ = c.commentRedisRepo.DecrCommentLikeCount(ctx, commentID)
	_ = c.commentRedisRepo.DelUserLikeRecord(ctx, userID, commentID)
	logger.Info("取消点赞成功", zap.Uint64("user_id", userID), zap.Uint64("comment_id", commentID))
	return nil
}

// GetArticleCommentCount 获取文章评论总数（Redis缓存优先，防击穿）
func (c *CommentAppService) GetArticleCommentCount(ctx context.Context, articleID uint64) (int, error) {
	if articleID <= 0 {
		logger.Warn("获取文章评论数失败：文章ID非法", zap.Uint64("article_id", articleID))
		return 0, common.ParamErr
	}

	// 1. 优先读缓存
	count, err := c.commentRedisRepo.GetArticleCommentCount(ctx, articleID)
	if err == nil {
		return count, nil
	}

	// 2. 缓存未命中，查询数据库
	logger.Info("文章评论数缓存未命中，查询数据库", zap.Uint64("article_id", articleID))
	query := &do.CommentQuery{ArticleID: articleID, Page: 1, Size: 1}
	_, total, err := c.commentDomainService.ListCommentByPage(ctx, query)
	if err != nil {
		logger.Error("查询文章评论总数失败", zap.Uint64("article_id", articleID), zap.Error(err))
		return 0, err
	}

	// 3. 回写缓存
	_ = c.commentRedisRepo.SetArticleCommentCount(ctx, articleID, int(total))
	return int(total), nil
}
