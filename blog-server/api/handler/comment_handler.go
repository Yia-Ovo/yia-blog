/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-25 15:37:14
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-16 19:42:01
 * @FilePath: \yia-blog\blog-server\api\handler\comment_handler.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package handler

import (
	"blog-server/api/dto"
	"blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/common/gin/response"
	"blog-server/common/utils/logger"
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
	var req dto.CommentCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("[API] 创建评论参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg("评论参数格式不正确"))
		return
	}

	userValue, exists := c.Get(consts.CustomerUserKey)
	user, ok := userValue.(*dto.UserBaseDTO)
	if !exists || !ok || user == nil || user.ID == 0 {
		logger.Warn("[API] 创建评论失败：未获取到有效登录用户",
			zap.Uint64("article_id", req.ArticleID),
			zap.Bool("context_key_exists", exists),
			zap.Bool("user_type_valid", ok),
		)
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("登录状态已失效，请重新登录后发表评论"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	command := &service.CommentCreateCommand{
		UserID:    user.ID,
		ArticleID: req.ArticleID,
		ParentID:  req.ParentID,
		Nickname:  user.NickName,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Content:   req.Content,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
	err := h.commentAppService.CreateComment(ctx, command)
	if err != nil {
		logger.Error("[API] 创建评论失败",
			zap.Uint64("user_id", user.ID),
			zap.Uint64("article_id", req.ArticleID),
			zap.Error(err),
		)
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	logger.Info("[API] 创建评论成功",
		zap.Uint64("user_id", user.ID),
		zap.Uint64("article_id", req.ArticleID),
	)
	response.WriteResp(c, nil, common.OK)
}

// ListCommentByPage 分页查询评论 GET /api/blog/v1/comment/listCommentByPage
func (h *CommentHandler) ListCommentByPage(c *gin.Context) {
	var req dto.CommentPageQueryReq
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Warn("[API] 评论分页查询参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg("评论查询参数格式不正确"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	query := &service.CommentListQuery{
		ArticleID: req.ArticleID,
		ParentID:  req.ParentID,
		Status:    req.Status,
		AdminID:   req.AdminID,
		Nickname:  req.Nickname,
		Content:   req.Content,
		Page:      req.Page,
		Size:      req.Size,
		OrderBy:   req.OrderBy,
		OrderDir:  req.OrderDir,
	}
	list, total, err := h.commentAppService.ListCommentByPage(ctx, query)
	if err != nil {
		logger.Error("[API] 分页查询评论失败", zap.Any("query", req), zap.Error(err))
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
	var req dto.CommentUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("[API] 更新评论参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg("评论参数格式不正确"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	operatorIDValue, exists := c.Get(consts.UserIDKey)
	operatorID, ok := operatorIDValue.(uint64)
	if !exists || !ok || operatorID == 0 {
		logger.Warn("[API] 更新评论失败：未获取到有效登录用户",
			zap.Uint64("comment_id", req.ID),
			zap.Bool("context_key_exists", exists),
			zap.Bool("user_id_type_valid", ok),
		)
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("登录状态已失效，请重新登录后修改评论"))
		return
	}
	err := h.commentAppService.UpdateComment(ctx, operatorID, req.ID, req.Content)
	if err != nil {
		logger.Error("[API] 更新评论失败", zap.Uint64("comment_id", req.ID), zap.Error(err))
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	logger.Info("[API] 更新评论成功", zap.Uint64("comment_id", req.ID))
	response.WriteResp(c, nil, common.OK)
}

// DeleteComment 删除评论 POST /api/blog/v1/comment/deleteComment
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	var req dto.CommentDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("[API] 删除评论参数格式非法", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg("评论参数格式不正确"))
		return
	}

	userIDValue, exists := c.Get(consts.UserIDKey)
	userID, ok := userIDValue.(uint64)
	if !exists || !ok || userID == 0 {
		logger.Warn("[API] 删除评论失败：未获取到有效登录用户",
			zap.Uint64("comment_id", req.ID),
			zap.Bool("context_key_exists", exists),
			zap.Bool("user_id_type_valid", ok),
		)
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("登录状态已失效，请重新登录后删除评论"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.commentAppService.DeleteComment(ctx, userID, req.ID)
	if err != nil {
		logger.Error("[API] 删除评论失败", zap.Uint64("comment_id", req.ID), zap.Error(err))
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	logger.Info("[API] 删除评论成功", zap.Uint64("comment_id", req.ID))
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

	userIDValue, exists := c.Get(consts.UserIDKey)
	userID, ok := userIDValue.(uint64)
	if !exists || !ok || userID == 0 {
		logger.Warn("[API] 评论点赞失败：未获取到有效登录用户",
			zap.Uint64("comment_id", req.ID),
			zap.Bool("context_key_exists", exists),
			zap.Bool("user_id_type_valid", ok),
		)
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("登录状态已失效，请重新登录后点赞"))
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

	userIDValue, exists := c.Get(consts.UserIDKey)
	userID, ok := userIDValue.(uint64)
	if !exists || !ok || userID == 0 {
		logger.Warn("[API] 取消评论点赞失败：未获取到有效登录用户",
			zap.Uint64("comment_id", req.ID),
			zap.Bool("context_key_exists", exists),
			zap.Bool("user_id_type_valid", ok),
		)
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("登录状态已失效，请重新登录后取消点赞"))
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
