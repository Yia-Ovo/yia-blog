/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-14 01:40:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-14 01:32:19
 * @FilePath: \yia-blog\blog-server\api\handler\article_handler.go
 * @Description: 文章接口处理函数：只处理HTTP相关逻辑
 *
 * Copyright (c) 2026-2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package handler

import (
	"blog-server/api/dto"
	"blog-server/app/service"
	"blog-server/common"
	"blog-server/common/gin/response"
	"blog-server/common/utils/logger"
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ArticleHandler 文章接口处理函数：只处理HTTP相关逻辑
type ArticleHandler struct {
	// 注入文章应用服务实例
	articleAppSrv *service.ArticleAppService
}

// NewArticleHandler 构造函数，外部注入应用服务
func NewArticleHandler(articleAppSrv *service.ArticleAppService) *ArticleHandler {
	return &ArticleHandler{
		articleAppSrv: articleAppSrv,
	}
}

// CreateArticle 创建文章 POST /api/blog/v1/article/createArticle
func (h *ArticleHandler) CreateArticle(c *gin.Context) {
	var req dto.CreateArticleReq
	// 1. 绑定JSON请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	// 2. 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 3. 调用应用层
	err := h.articleAppSrv.CreateArticle(ctx, &req)
	if err != nil {
		// 错误类型断言
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			logger.Error("CreateArticle 未知系统错误", zap.String("article_title", req.Title), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	// 4. 成功响应
	response.WriteResp(c, gin.H{"message": "创建文章成功"}, common.OK)
}

// ListArticle 分页查询文章 GET /api/blog/v1/article/listArticle
func (h *ArticleHandler) ListArticle(c *gin.Context) {
	var req dto.ArticlePageQuery
	// 1. 绑定查询参数
	if err := c.ShouldBindQuery(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	// 2. 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 3. 调用应用层
	result, err := h.articleAppSrv.ListArticleByPage(ctx, &req)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			logger.Error("ListArticle 分页查询失败", zap.Int("page_num", req.PageNum), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	// 4. 成功响应（直接返回分页结果）
	response.WriteResp(c, result, common.OK)
}

// GetArticleById 根据ID查询文章详情 GET /api/blog/v1/article/getArticleById
func (h *ArticleHandler) GetArticleById(c *gin.Context) {
	// 从查询参数获取 ID（和分类模块风格一致）
	idStr := c.Query("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("文章ID格式非法"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	data, err := h.articleAppSrv.GetArticleById(ctx, id)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			logger.Error("GetArticleById 查询失败", zap.Uint64("article_id", id), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	response.WriteResp(c, data, common.OK)
}

// UpdateArticle 更新文章 POST /api/blog/v1/article/updateArticle
func (h *ArticleHandler) UpdateArticle(c *gin.Context) {
	var req dto.UpdateArticleReq
	// 1. 绑定JSON请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.articleAppSrv.UpdateArticle(ctx, &req)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			logger.Error("UpdateArticle 更新失败", zap.Uint64("article_id", req.ID), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	response.WriteResp(c, gin.H{"message": "更新文章成功"}, common.OK)
}

// DeleteArticle 删除文章 POST /api/blog/v1/article/deleteArticle
func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	// 从请求体获取 ID
	type Req struct {
		ID uint64 `json:"id" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("文章ID不能为空"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.articleAppSrv.DeleteArticle(ctx, req.ID)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			logger.Error("DeleteArticle 删除失败", zap.Uint64("article_id", req.ID), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	response.WriteResp(c, gin.H{"message": "删除文章成功"}, common.OK)
}
