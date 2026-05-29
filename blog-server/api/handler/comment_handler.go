/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-25 15:37:14
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 21:06:05
 * @FilePath: \yia-blog\blog-server\api\handler\comment_handler.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package handler

import (
	"blog-server/app/service"
	"blog-server/common"
	"blog-server/common/gin/response"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"context"
	"strconv"
	"time"

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

// CreateComment 创建评论 POST /api/blog/v1/comment/createComment
func (h *CommentHandler) CreateComment(c *gin.Context) {
	var commentDO do.CommentDO
	if err := c.ShouldBindJSON(&commentDO); err != nil {
		logger.Warn("创建评论参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg(err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.commentAppService.CreateComment(ctx, &commentDO)
	if err != nil {
		logger.Error("创建评论失败", zap.Any("comment", commentDO), zap.Error(err))
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	logger.Info("创建评论成功", zap.Uint64("article_id", commentDO.ArticleID))
	response.WriteResp(c, nil, common.OK)
}

// ListCommentByPage 分页查询评论 GET /api/blog/v1/comment/listCommentByPage
func (h *CommentHandler) ListCommentByPage(c *gin.Context) {
	var query do.CommentQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		logger.Warn("评论分页查询参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg(err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	list, total, err := h.commentAppService.ListCommentByPage(ctx, &query)
	if err != nil {
		logger.Error("分页查询评论失败", zap.Any("query", query), zap.Error(err))
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, gin.H{
		"list":  list,
		"total": total,
	}, common.OK)
}

// GetCommentByID 根据ID查询评论 GET /api/blog/v1/comment/getCommentByID
func (h *CommentHandler) GetCommentByID(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		logger.Warn("评论ID格式非法", zap.String("id", idStr))
		response.WriteResp(c, nil, common.ParamErr)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	comment, err := h.commentAppService.GetCommentByID(ctx, id)
	if err != nil {
		logger.Error("根据ID查询评论失败", zap.Uint64("comment_id", id), zap.Error(err))
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, comment, common.OK)
}

// UpdateComment 更新评论 POST /api/blog/v1/comment/updateComment
func (h *CommentHandler) UpdateComment(c *gin.Context) {
	var commentDO do.CommentDO
	if err := c.ShouldBindJSON(&commentDO); err != nil {
		logger.Warn("更新评论参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg(err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.commentAppService.UpdateComment(ctx, &commentDO)
	if err != nil {
		logger.Error("更新评论失败", zap.Uint64("comment_id", commentDO.ID), zap.Error(err))
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	logger.Info("更新评论成功", zap.Uint64("comment_id", commentDO.ID))
	response.WriteResp(c, nil, common.OK)
}

// DeleteComment 删除评论 POST /api/blog/v1/comment/deleteComment
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	type Req struct {
		ID uint64 `json:"id" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("评论ID格式非法", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.commentAppService.DeleteComment(ctx, req.ID)
	if err != nil {
		logger.Error("删除评论失败", zap.Uint64("comment_id", req.ID), zap.Error(err))
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	logger.Info("删除评论成功", zap.Uint64("comment_id", req.ID))
	response.WriteResp(c, nil, common.OK)
}

// LikeComment 评论点赞 POST /api/blog/v1/comment/likeComment
func (h *CommentHandler) LikeComment(c *gin.Context) {
	type Req struct {
		ID uint64 `json:"id" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("评论ID格式非法", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr)
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	userID, ok := userIDVal.(uint64)
	if !ok {
		logger.Error("LikeComment user_id类型转换失败", zap.Any("user_id", userIDVal))
		response.WriteResp(c, nil, common.ServerErr.WithMsg("用户ID类型异常"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.commentAppService.LikeComment(ctx, userID, req.ID)
	if err != nil {
		logger.Error("评论点赞失败", zap.Uint64("comment_id", req.ID), zap.Error(err))
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}

// DisLikeComment 取消评论点赞 POST /api/blog/v1/comment/disLikeComment
func (h *CommentHandler) DisLikeComment(c *gin.Context) {
	type Req struct {
		ID uint64 `json:"id" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("评论ID格式非法", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr)
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	userID, ok := userIDVal.(uint64)
	if !ok {
		logger.Error("DisLikeComment user_id类型转换失败", zap.Any("user_id", userIDVal))
		response.WriteResp(c, nil, common.ServerErr.WithMsg("用户ID类型异常"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.commentAppService.DisLikeComment(ctx, userID, req.ID)
	if err != nil {
		logger.Error("取消点赞失败", zap.Uint64("comment_id", req.ID), zap.Error(err))
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}

// GetArticleCommentCount 获取文章评论总数 GET /api/blog/v1/comment/getArticleCommentCount
func (h *CommentHandler) GetArticleCommentCount(c *gin.Context) {
	articleIDStr := c.Query("article_id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		logger.Warn("文章ID格式非法", zap.String("article_id", articleIDStr))
		response.WriteResp(c, nil, common.ParamErr)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	count, err := h.commentAppService.GetArticleCommentCount(ctx, articleID)
	if err != nil {
		logger.Error("获取文章评论数失败", zap.Uint64("article_id", articleID), zap.Error(err))
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, gin.H{"count": count}, common.OK)
}
