/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-14 01:40:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-07-09 08:34:49
 * @FilePath: \yia-blog\blog-server\api\handler\article_handler.go
 * @Description: 文章接口处理函数：只处理HTTP相关逻辑
 * Copyright (c) 2026-2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package handler

import (
	"blog-server/api/dto"
	"blog-server/app/service"
	"blog-server/common"
	"blog-server/common/consts"
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
	// 取用户ID
	userIDVal, exists := c.Get(consts.UserIDKey)
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("请先登录"))
		return
	}

	userID, ok := userIDVal.(uint64)
	if !ok || userID == 0 {
		response.WriteResp(c, nil, common.AuthErr.WithMsg("用户身份异常"))
		return
	}

	var req dto.CreateArticleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.articleAppSrv.CreateArticle(ctx, userID, &req)
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

// UpdateArticle 更新文章（普通用户，仅本人） POST /api/blog/v1/article/updateArticle
func (h *ArticleHandler) UpdateArticle(c *gin.Context) {
	// 取用户ID
	userIDVal, exists := c.Get(consts.UserIDKey)
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("请先登录"))
		return
	}
	userID, ok := userIDVal.(uint64)
	if !ok || userID == 0 {
		response.WriteResp(c, nil, common.AuthErr.WithMsg("用户身份异常"))
		return
	}

	var req dto.UpdateArticleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.articleAppSrv.UpdateArticle(ctx, userID, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}

// DeleteArticle 删除文章（普通用户，仅本人） POST /api/blog/v1/article/deleteArticle
func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	// 取用户ID
	userIDVal, exists := c.Get(consts.UserIDKey)
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("请先登录"))
		return
	}
	userID, ok := userIDVal.(uint64)
	if !ok || userID == 0 {
		response.WriteResp(c, nil, common.AuthErr.WithMsg("用户身份异常"))
		return
	}

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

	err := h.articleAppSrv.DeleteArticle(ctx, userID, req.ID)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}

// AdminCreateArticle 创建文章（管理员） POST /api/blog/v1/article/adminCreateArticle
func (h *ArticleHandler) AdminCreateArticle(c *gin.Context) {
	// 取管理员ID
	adminIDVal, exists := c.Get(consts.AdminIDKey)
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("管理员请先登录"))
		return
	}
	adminID, ok := adminIDVal.(uint64)
	if !ok || adminID == 0 {
		response.WriteResp(c, nil, common.AuthErr.WithMsg("管理员身份异常"))
		return
	}

	var req dto.CreateArticleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.articleAppSrv.AdminCreateArticle(ctx, adminID, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}

// AdminUpdateArticle 更新文章（管理员，任意文章） POST /api/blog/v1/article/adminUpdateArticle
func (h *ArticleHandler) AdminUpdateArticle(c *gin.Context) {
	// 取管理员ID
	adminIDVal, exists := c.Get(consts.AdminIDKey)
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("管理员请先登录"))
		return
	}
	adminID, ok := adminIDVal.(uint64)
	if !ok || adminID == 0 {
		response.WriteResp(c, nil, common.AuthErr.WithMsg("管理员身份异常"))
		return
	}

	var req dto.UpdateArticleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.articleAppSrv.AdminUpdateArticle(ctx, adminID, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}

// AdminDeleteArticle 删除文章（管理员，任意文章） POST /api/blog/v1/article/adminDeleteArticle
func (h *ArticleHandler) AdminDeleteArticle(c *gin.Context) {
	// 取管理员ID
	adminIDVal, exists := c.Get(consts.AdminIDKey)
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("管理员请先登录"))
		return
	}
	adminID, ok := adminIDVal.(uint64)
	if !ok || adminID == 0 {
		response.WriteResp(c, nil, common.AuthErr.WithMsg("管理员身份异常"))
		return
	}

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

	err := h.articleAppSrv.AdminDeleteArticle(ctx, adminID, req.ID)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}
