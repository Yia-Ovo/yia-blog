/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-06-06 11:30:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-12 15:41:37
 * @FilePath: \yia-blog\blog-server\api\handler\admin_handler.go
 * @Description: 管理员 API 控制器层
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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AdminHandler struct {
	adminAppSrv *service.AdminAppService
}

func NewAdminHandler(adminAppSrv *service.AdminAppService) *AdminHandler {
	return &AdminHandler{
		adminAppSrv: adminAppSrv,
	}
}

// CreateAdmin 管理员注册 POST /api/blog/v1/admin/create
func (h *AdminHandler) CreateAdmin(c *gin.Context) {
	var req dto.AdminCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("[HANDLER] 注册参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg("请求参数格式错误，请检查后重试"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	data, err := h.adminAppSrv.CreateAdmin(ctx, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, data, common.OK)
}

// Login 管理员登录 POST /api/blog/v1/admin/login
func (h *AdminHandler) Login(c *gin.Context) {
	var req dto.AdminLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("[HANDLER] 管理员登录参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg("登录参数有误，请检查输入内容"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	loginResp, err := h.adminAppSrv.Login(ctx, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, loginResp, common.OK)
}

// GetInfo 获取管理员信息 GET /api/blog/v1/admin/getInfo
func (h *AdminHandler) GetInfo(c *gin.Context) {
	adminIdVal, exists := c.Get("admin_id")
	if !exists {
		logger.Warn("[HANDLER] 未获取到登录管理员ID")
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("请先登录管理员账号"))
		return
	}
	adminId, ok := adminIdVal.(uint64)
	if !ok {
		logger.Error("[HANDLER] 管理员ID类型转换失败")
		response.WriteResp(c, nil, common.ServerErr.WithMsg("管理员ID类型错误"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	adminInfo, err := h.adminAppSrv.GetAdminInfo(ctx, adminId)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}
	response.WriteResp(c, adminInfo, common.OK)
}

// UpdatePWD 管理员修改密码 POST /api/blog/v1/admin/updatePWD
func (h *AdminHandler) UpdatePWD(c *gin.Context) {
	var req dto.AdminUpdatePWDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("[HANDLER] 修改密码参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	adminIdVal, exists := c.Get("admin_id")
	if !exists {
		logger.Warn("[HANDLER] 修改密码未获取到登录管理员ID")
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("登录状态已失效，请重新登录"))
		return
	}
	adminId, ok := adminIdVal.(uint64)
	if !ok {
		logger.Error("[HANDLER] 管理员ID类型转换失败")
		response.WriteResp(c, nil, common.ServerErr.WithMsg("管理员ID类型错误"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.adminAppSrv.UpdatePWD(ctx, adminId, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}
	response.WriteResp(c, nil, common.OK)
}

// UpdateAdminInfo 更新管理员信息 POST /api/blog/v1/admin/updateAdminInfo
func (h *AdminHandler) UpdateAdminInfo(c *gin.Context) {
	var req dto.AdminUpdateInfoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("[HANDLER] 更新管理员信息参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg("请求参数错误，请核对后提交"))
		return
	}

	adminIdVal, exists := c.Get("admin_id")
	if !exists {
		logger.Warn("[HANDLER] 更新信息未获取到登录管理员ID")
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("请先完成管理员登录"))
		return
	}
	adminId, ok := adminIdVal.(uint64)
	if !ok {
		logger.Error("[HANDLER] 管理员ID类型转换失败")
		response.WriteResp(c, nil, common.ServerErr.WithMsg("管理员ID类型错误"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	res, err := h.adminAppSrv.UpdateAdminInfo(ctx, adminId, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, res, common.OK)
}

// Logout 管理员登出 POST /api/blog/v1/admin/logout
func (h *AdminHandler) Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		logger.Warn("[HANDLER] 管理员登出token为空")
		response.WriteResp(c, nil, common.ParamErr.WithMsg("身份令牌不能为空"))
		return
	}

	adminIdVal, exists := c.Get("admin_id")
	if !exists {
		logger.Warn("[HANDLER] 登出未获取到登录管理员ID")
		response.WriteResp(c, nil, common.LoginNotLoginErr.WithMsg("当前未处于登录状态"))
		return
	}
	adminId, ok := adminIdVal.(uint64)
	if !ok {
		logger.Error("[HANDLER] 管理员ID类型转换失败")
		response.WriteResp(c, nil, common.ServerErr.WithMsg("管理员ID类型错误"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.adminAppSrv.Logout(ctx, adminId, token)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}
	response.WriteResp(c, nil, common.OK)
}

// ListAdmin 分页查询管理员列表 GET /api/blog/v1/admin/list
func (h *AdminHandler) ListAdmin(c *gin.Context) {
	var req dto.AdminListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Warn("[HANDLER] 列表查询参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg("分页查询参数格式错误"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	data, err := h.adminAppSrv.ListAdmin(ctx, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}
	response.WriteResp(c, data, common.OK)
}

// UpdateAdminStatus 修改账号状态 PATCH /api/blog/v1/admin/status/:id
func (h *AdminHandler) UpdateAdminStatus(c *gin.Context) {
	adminID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || adminID <= 0 {
		logger.Warn("[HANDLER] 管理员ID参数非法")
		response.WriteResp(c, nil, common.ParamErr.WithMsg("管理员ID不合法"))
		return
	}

	var req dto.AdminStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("[HANDLER] 状态参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg("账号状态参数有误"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err = h.adminAppSrv.UpdateAdminStatus(ctx, adminID, req.Status)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}
	response.WriteResp(c, nil, common.OK)
}

// ResetAdminPWD 重置管理员密码 POST /api/blog/v1/admin/resetPWD/:id
func (h *AdminHandler) ResetAdminPWD(c *gin.Context) {
	adminID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || adminID <= 0 {
		logger.Warn("[HANDLER] 管理员ID参数非法")
		response.WriteResp(c, nil, common.ParamErr.WithMsg("管理员ID不合法"))
		return
	}

	var req dto.AdminResetPWDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("[HANDLER] 重置密码参数绑定失败", zap.Error(err))
		response.WriteResp(c, nil, common.ParamErr.WithMsg("重置密码参数校验失败"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err = h.adminAppSrv.ResetAdminPWD(ctx, adminID, req.NewPassword)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}
	response.WriteResp(c, nil, common.OK)
}
