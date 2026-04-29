/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-17 15:15:53
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-29 07:59:49
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
	// 1. 获取路径参数ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("分类ID格式非法"))
		return
	}

	// 2. 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 3. 调用应用层
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

	// 4. 成功响应
	response.WriteResp(c, data, common.OK)
}

// UpdateCategory 更新分类 PUT /api/blog/v1/category/:id
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	// 1. 获取路径参数ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("分类ID格式非法"))
		return
	}

	// 2. 绑定请求体参数
	var req dto.CreateCategoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	// 3. 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 4. 调用应用层
	err = h.categoryAppSrv.UpdateCategory(ctx, id, req.Name, req.Sort, req.Status)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			logger.Error("UpdateCategory 更新失败", zap.Uint64("category_id", id), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	// 5. 成功响应
	response.WriteResp(c, gin.H{"message": "更新分类成功"}, common.OK)
}

// DeleteCategory 删除分类 DELETE /api/blog/v1/category/:id
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	// 1. 获取路径参数ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("分类ID格式非法"))
		return
	}

	// 2. 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 3. 调用应用层
	err = h.categoryAppSrv.DeleteCategory(ctx, id)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			logger.Error("DeleteCategory 删除失败", zap.Uint64("category_id", id), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	// 4. 成功响应
	response.WriteResp(c, gin.H{"message": "删除分类成功"}, common.OK)
}
