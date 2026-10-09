/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 15:15:53
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 21:15:00
 * @FilePath: \yia-blog\blog-server\api\handler\category_handler.go
 * @Description: 分类接口处理函数：只处理HTTP相关逻辑
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
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

// CategoryHandler 分类接口处理函数：只处理HTTP相关逻辑
type CategoryHandler struct {
	categoryAppSrv *service.CategoryAppService
}

// NewCategoryHandler 构造函数，外部注入应用服务
func NewCategoryHandler(categoryAppSrv *service.CategoryAppService) *CategoryHandler {
	return &CategoryHandler{
		categoryAppSrv: categoryAppSrv,
	}
}

// CreateCategory 创建分类 POST /api/blog/v1/category/createCategory
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req dto.CreateCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithErr(err))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.categoryAppSrv.CreateCategory(ctx, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}

// ListCategory 分页查询分类 GET /api/blog/v1/category/listCategory
func (h *CategoryHandler) ListCategory(c *gin.Context) {
	var req dto.CategoryPageQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithErr(err))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	result, err := h.categoryAppSrv.ListCategoryByPage(ctx, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, result, common.OK)
}

// GetCategoryById 根据ID查询分类详情 GET /api/blog/v1/category/getCategoryById
func (h *CategoryHandler) GetCategoryById(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("分类ID格式非法"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	data, err := h.categoryAppSrv.GetCategoryById(ctx, id)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, data, common.OK)
}

// UpdateCategory 更新分类 POST /api/blog/v1/category/updateCategory
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	var req dto.UpdateCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithErr(err))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// Sort/Status 保持指针原样下传：nil 与 0 的区别一直保留到 Repo 组装更新列时才解释
	err := h.categoryAppSrv.UpdateCategory(ctx, req.ID, req.Name, req.Sort, req.Status)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}

// DeleteCategory 删除分类 POST /api/blog/v1/category/deleteCategory
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	type Req struct {
		ID uint64 `json:"id" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("分类ID不能为空"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.categoryAppSrv.DeleteCategory(ctx, req.ID)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}
