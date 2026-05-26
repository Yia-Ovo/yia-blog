/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-25 15:37:14
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-25 23:25:36
 * @FilePath: \yia-blog\blog-server\api\handler\comment_handler.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package handler

import (
	"blog-server/app/service"
	"blog-server/common"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CommentHandler struct {
	commentAppService *service.CommentAppService
}

func NewCommentHandler(commentAppService *service.CommentAppService) *CommentHandler {
	return &CommentHandler{
		commentAppService: commentAppService,
	}
}

// CreateComment 创建评论
func (h *CommentHandler) CreateComment(c *gin.Context) {
	var commentDO do.CommentDO
	if err := c.ShouldBindJSON(&commentDO); err != nil {
		logger.Warn("创建评论参数绑定失败", zap.Error(err))
		c.JSON(common.ParamErr.Code, common.ParamErr)
		return
	}

	err := h.commentAppService.CreateComment(context.Background(), &commentDO)
	if err != nil {
		logger.Error("创建评论失败", zap.Any("comment", commentDO), zap.Error(err))
		resp := common.ServerErr
		if e, ok := err.(common.Errno); ok {
			resp = e
		}
		c.JSON(resp.Code, resp)
		return
	}

	logger.Info("创建评论成功", zap.Uint64("article_id", commentDO.ArticleID))
	c.JSON(common.OK.Code, common.OK)
}

// ListCommentByPage 分页查询评论
func (h *CommentHandler) ListCommentByPage(c *gin.Context) {
	var query do.CommentQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.Warn("评论分页查询参数绑定失败", zap.Error(err))
		c.JSON(common.ParamErr.Code, common.ParamErr)
		return
	}

	list, total, err := h.commentAppService.ListCommentByPage(context.Background(), &query)
	if err != nil {
		logger.Error("分页查询评论失败", zap.Any("query", query), zap.Error(err))
		resp := common.ServerErr
		if e, ok := err.(common.Errno); ok {
			resp = e
		}
		c.JSON(resp.Code, resp)
		return
	}

	c.JSON(common.OK.Code, gin.H{
		"list":  list,
		"total": total,
	})
}

// GetCommentByID 根据ID查询评论
func (h *CommentHandler) GetCommentByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.Warn("评论ID格式非法", zap.String("id", idStr))
		c.JSON(common.ParamErr.Code, common.ParamErr)
		return
	}

	comment, err := h.commentAppService.GetCommentByID(context.Background(), id)
	if err != nil {
		logger.Error("根据ID查询评论失败", zap.Uint64("comment_id", id), zap.Error(err))
		resp := common.ServerErr
		if e, ok := err.(common.Errno); ok {
			resp = e
		}
		c.JSON(resp.Code, resp)
		return
	}

	c.JSON(common.OK.Code, comment)
}

// UpdateComment 更新评论
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	var commentDO do.CommentDO
	if err := c.ShouldBindJSON(&commentDO); err != nil {
		logger.Warn("更新评论参数绑定失败", zap.Error(err))
		c.JSON(common.ParamErr.Code, common.ParamErr)
		return
	}

	err := h.commentAppService.UpdateComment(context.Background(), &commentDO)
	if err != nil {
		logger.Error("更新评论失败", zap.Uint64("comment_id", commentDO.ID), zap.Error(err))
		resp := common.ServerErr
		if e, ok := err.(common.Errno); ok {
			resp = e
		}
		c.JSON(resp.Code, resp)
		return
	}

	logger.Info("更新评论成功", zap.Uint64("comment_id", commentDO.ID))
	c.JSON(common.OK.Code, common.OK)
}

// DeleteComment 删除评论
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.Warn("评论ID格式非法", zap.String("id", idStr))
		c.JSON(common.ParamErr.Code, common.ParamErr)
		return
	}

	err = h.commentAppService.DeleteComment(context.Background(), id)
	if err != nil {
		logger.Error("删除评论失败", zap.Uint64("comment_id", id), zap.Error(err))
		resp := common.ServerErr
		if e, ok := err.(common.Errno); ok {
			resp = e
		}
		c.JSON(resp.Code, resp)
		return
	}

	logger.Info("删除评论成功", zap.Uint64("comment_id", id))
	c.JSON(common.OK.Code, common.OK)
}

// LikeComment 评论点赞
func (h *CommentHandler) LikeComment(c *gin.Context) {
	idStr := c.Param("id")
	commentID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.Warn("评论ID格式非法", zap.String("id", idStr))
		c.JSON(common.ParamErr.Code, common.ParamErr)
		return
	}

	// 从上下文获取用户ID（根据你的项目实际获取方式调整）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(common.LoginNotLoginErr.Code, common.LoginNotLoginErr)
		return
	}

	err = h.commentAppService.LikeComment(context.Background(), userID.(uint64), commentID)
	if err != nil {
		logger.Error("评论点赞失败", zap.Uint64("comment_id", commentID), zap.Error(err))
		resp := common.ServerErr
		if e, ok := err.(common.Errno); ok {
			resp = e
		}
		c.JSON(resp.Code, resp)
		return
	}

	c.JSON(common.OK.Code, common.OK)
}

// DisLikeComment 取消评论点赞
func (h *CommentHandler) DisLikeComment(c *gin.Context) {
	idStr := c.Param("id")
	commentID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.Warn("评论ID格式非法", zap.String("id", idStr))
		c.JSON(common.ParamErr.Code, common.ParamErr)
		return
	}

	// 从上下文获取用户ID（根据你的项目实际获取方式调整）
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(common.LoginNotLoginErr.Code, common.LoginNotLoginErr)
		return
	}

	err = h.commentAppService.DisLikeComment(context.Background(), userID.(uint64), commentID)
	if err != nil {
		logger.Error("取消评论点赞失败", zap.Uint64("comment_id", commentID), zap.Error(err))
		resp := common.ServerErr
		if e, ok := err.(common.Errno); ok {
			resp = e
		}
		c.JSON(resp.Code, resp)
		return
	}

	c.JSON(common.OK.Code, common.OK)
}

// GetArticleCommentCount 获取文章评论总数
func (h *CommentHandler) GetArticleCommentCount(c *gin.Context) {
	articleIDStr := c.Query("article_id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		logger.Warn("文章ID格式非法", zap.String("article_id", articleIDStr))
		c.JSON(common.ParamErr.Code, common.ParamErr)
		return
	}

	count, err := h.commentAppService.GetArticleCommentCount(context.Background(), articleID)
	if err != nil {
		logger.Error("获取文章评论数失败", zap.Uint64("article_id", articleID), zap.Error(err))
		resp := common.ServerErr
		if e, ok := err.(common.Errno); ok {
			resp = e
		}
		c.JSON(resp.Code, resp)
		return
	}

	c.JSON(common.OK.Code, gin.H{
		"count": count,
	})
}
