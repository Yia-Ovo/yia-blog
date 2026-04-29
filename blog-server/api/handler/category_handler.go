/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 15:15:53
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-30 01:19:30
 * @FilePath: \yia-blog\blog-server\api\handler\category_handler.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
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

// CategoryHandler 分类接口处理函数：只处理HTTP相关逻辑
type CategoryHandler struct {
	// 注入分类应用服务实例
	categoryAppSrv *service.CategoryAppService
}

// NewCategoryHandler 构造函数，外部注入应用服务
func NewCategoryHandler(categoryAppSrv *service.CategoryAppService) *CategoryHandler {
	return &CategoryHandler{
		categoryAppSrv: categoryAppSrv,
	}
}

// CreateCategory 创建分类 POST /api/blog/v1/category
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req dto.CreateCategoryReq
	// 1. 绑定JSON请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	// 2. 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 3. 调用应用层
	err := h.categoryAppSrv.CreateCategory(ctx, &req)
	if err != nil {
		// 错误类型断言
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			logger.Error("CreateCategory 未知系统错误", zap.String("category_name", req.Name), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	// 4. 成功响应
	response.WriteResp(c, gin.H{"message": "创建分类成功"}, common.OK)
}

// ListCategory 分页查询分类 GET /api/blog/v1/category/list
func (h *CategoryHandler) ListCategory(c *gin.Context) {
	var req dto.CategoryPageQuery
	// 1. 绑定查询参数
	if err := c.ShouldBindQuery(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	// 2. 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 3. 调用应用层
	result, err := h.categoryAppSrv.ListCategoryByPage(ctx, &req)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			logger.Error("ListCategory 分页查询失败", zap.Int("page_num", req.PageNum), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	// 4. 成功响应（直接返回分页结果）
	response.WriteResp(c, result, common.OK)
}

// GetCategoryById 根据ID查询分类详情 GET /api/blog/v1/category/:id
func (h *CategoryHandler) GetCategoryById(c *gin.Context) {
	// 从查询参数获取 ID（和用户模块风格一致）
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
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			logger.Error("GetCategoryById 查询失败", zap.Uint64("category_id", id), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	response.WriteResp(c, data, common.OK)
}

// UpdateCategory 更新分类 PUT /api/blog/v1/category/:id
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	// 定义请求体（包含 ID，和用户模块一致）
	type Req struct {
		ID     uint64 `json:"id" binding:"required"`
		Name   string `json:"name" binding:"required,max=64"`
		Sort   int    `json:"sort"`
		Status int    `json:"status"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.categoryAppSrv.UpdateCategory(ctx, req.ID, req.Name, req.Sort, req.Status)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			logger.Error("UpdateCategory 更新失败", zap.Uint64("category_id", req.ID), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	response.WriteResp(c, gin.H{"message": "更新分类成功"}, common.OK)
}

// DeleteCategory 删除分类 DELETE /api/blog/v1/category/:id
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	// 从请求体获取 ID
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
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			logger.Error("DeleteCategory 删除失败", zap.Uint64("category_id", req.ID), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	response.WriteResp(c, gin.H{"message": "删除分类成功"}, common.OK)
}
