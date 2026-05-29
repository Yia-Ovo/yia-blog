/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-14 01:40:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 20:58:59
 * @FilePath: \yia-blog\blog-server\api\handler\article_handler.go
 * @Description: 文章接口处理函数：只处理HTTP相关逻辑
 * Copyright (c) 2026-2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package handler

import (
	"blog-server/api/dto"
	"blog-server/app/service"
	"blog-server/common"
	"blog-server/common/gin/response"
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ArticleHandler 文章接口处理函数：只处理HTTP相关逻辑
type ArticleHandler struct {
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
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.articleAppSrv.CreateArticle(ctx, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}

// ListArticle 分页查询文章 GET /api/blog/v1/article/listArticle
func (h *ArticleHandler) ListArticle(c *gin.Context) {
	var req dto.ArticlePageQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	result, err := h.articleAppSrv.ListArticleByPage(ctx, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, result, common.OK)
}

// GetArticleById 根据ID查询文章详情 GET /api/blog/v1/article/getArticleById
func (h *ArticleHandler) GetArticleById(c *gin.Context) {
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
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, data, common.OK)
}

// UpdateArticle 更新文章 POST /api/blog/v1/article/updateArticle
func (h *ArticleHandler) UpdateArticle(c *gin.Context) {
	var req dto.UpdateArticleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.articleAppSrv.UpdateArticle(ctx, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}

// DeleteArticle 删除文章 POST /api/blog/v1/article/deleteArticle
func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
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
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}
