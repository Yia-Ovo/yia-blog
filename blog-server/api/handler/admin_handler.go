/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-17 08:01:56
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-17 09:00:00
 * @FilePath: \yia-blog\blog-server\api\handler\admin_handler.go
 * @Description: 管理员Handler（全链路ctx上下文规范修复）
 *
 * Copyright (c) 2026 Yia 1372594487@qq.com, All Rights Reserved.
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
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminHandler 管理员接口处理函数：只处理HTTP相关逻辑
type AdminHandler struct {
	// 注入应用服务实例
	adminAppSrv *service.AdminAppService
}

// 构造函数接收注入的AppService，不再自己创建
func NewAdminHandler(adminAppSrv *service.AdminAppService) *AdminHandler {
	return &AdminHandler{
		adminAppSrv: adminAppSrv,
	}
}

// UpdatePWD 管理员修改密码
func (h *AdminHandler) UpdatePWD(c *gin.Context) {
	var req dto.AdminUpdatePWDReq
	// 1. 绑定请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}
	// 2. 从 Gin Context 中获取管理员ID（由 AdminAuth 中间件存入）
	adminIdVal, exists := c.Get("admin_id")
	if !exists {
		logger.Warn("UpdatePWD Handler 未获取到admin_id", zap.Bool("admin_id_exists", exists))
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	// 3. ID断言
	adminId, ok := adminIdVal.(uint64)
	if !ok {
		logger.Error("UpdatePWD Handler admin_id类型转换失败",
			zap.Any("admin_id_value", adminIdVal), // 修复日志变量错误
			zap.String("expected_type", "uint64"),
			zap.String("actual_type", fmt.Sprintf("%T", adminIdVal)),
		)
		response.WriteResp(c, nil, common.ServerErr.WithMsg("管理员ID类型断言失败"))
		return
	}
	// 4.创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	// 5. 调用应用层
	err := h.adminAppSrv.UpdatePWD(ctx, adminId, &req)
	if err != nil {
		switch err {
		case common.UserNotFoundErr:
			logger.Error("UpdatePWD 查询管理员不存在", zap.Uint64("admin_id", adminId), zap.Error(err))
		case common.UserOldPasswordErr:
			logger.Warn("UpdatePWD 旧密码验证失败", zap.Uint64("admin_id", adminId))
		case common.UserNewPasswordInvalidErr:
			logger.Warn("UpdatePWD 新密码格式不合法", zap.Uint64("admin_id", adminId))
		case common.UserPasswordSameError:
			logger.Warn("UpdatePWD 新密码与旧密码相同", zap.Uint64("admin_id", adminId))
		case common.PasswordEncryptError:
			logger.Error("UpdatePWD 密码加密失败", zap.Uint64("admin_id", adminId), zap.Error(err))
		case common.UpdatePasswordError:
			logger.Error("UpdatePWD 更新密码失败", zap.Uint64("admin_id", adminId), zap.Error(err))
		default:
			logger.Error("UpdatePWD 未知系统错误", zap.Uint64("admin_id", adminId), zap.Error(err))
		}
		response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		return
	}
	// 6. 成功响应
	response.WriteResp(c, nil, common.OK)
}

// Login 管理员登录接口
func (h *AdminHandler) Login(c *gin.Context) {
	// 1. 绑定请求参数
	var req dto.AdminLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr)
		return
	}
	// 统一创建超时ctx（规范对齐）
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	// 2. 调用应用层服务
	admin, token, err := h.adminAppSrv.Login(ctx, &req)
	if err != nil {
		e, ok := err.(common.Errno)
		if !ok {
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
			return
		}
		response.WriteResp(c, nil, e)
		return
	}
	// 3. 登录成功：封装返回数据
	respData := gin.H{
		"admin": admin,
		"token": token,
	}
	response.WriteResp(c, respData, common.OK)
}

// Logout 管理员登出接口
func (h *AdminHandler) Logout(c *gin.Context) {
	// 1. 从请求头获取 Token
	token := c.GetHeader("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		logger.Warn("Logout Handler 未获取到Token")
		response.WriteResp(c, nil, common.ParamErr)
		return
	}

	// 2. 从中间件获取当前登录管理员ID（修复：增加断言判断，防止panic）
	adminIdVal, exists := c.Get("admin_id")
	if !exists {
		logger.Warn("Logout Handler 未获取到admin_id", zap.Bool("admin_id_exists", exists))
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	adminId, ok := adminIdVal.(uint64)
	if !ok {
		logger.Error("Logout Handler admin_id类型转换失败",
			zap.Any("admin_id_value", adminIdVal),
			zap.String("expected_type", "uint64"),
			zap.String("actual_type", fmt.Sprintf("%T", adminIdVal)),
		)
		response.WriteResp(c, nil, common.ServerErr.WithMsg("管理员ID类型断言失败"))
		return
	}

	// 统一创建超时ctx（规范对齐）
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	// 3. 调用应用层
	err := h.adminAppSrv.Logout(ctx, adminId, token)
	if err != nil {
		e, ok := err.(common.Errno)
		if !ok {
			logger.Error("Logout Handler 未知系统错误", zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
			return
		}
		response.WriteResp(c, nil, e)
		return
	}

	// 登出成功
	response.WriteResp(c, nil, common.OK)
}

// GetInfo 获取管理员信息
func (h *AdminHandler) GetInfo(c *gin.Context) {
	// 1. 从 Gin Context 中获取管理员ID
	adminIdVal, exists1 := c.Get("admin_id")
	if !exists1 {
		logger.Warn("GetInfo Handler 未获取到admin_id", zap.Bool("admin_id_exists", exists1))
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	// 读取完整管理员对象
	adminObj, exists2 := c.Get(consts.CustomerAdminKey)
	if !exists2 {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}

	// 调试日志
	logger.Debug("GetInfo Handler 读取上下文",
		zap.Bool("admin_id_exists", exists1),
		zap.Any("admin_id_value", adminIdVal),
		zap.Bool("admin_obj_exists", exists2),
		zap.Any("admin_obj_value", adminObj),
	)

	adminId, ok := adminIdVal.(uint64)
	if !ok {
		logger.Error("GetInfo Handler admin_id类型转换失败",
			zap.Any("admin_id_value", adminIdVal), // 修复日志变量错误
			zap.String("expected_type", "uint64"),
			zap.String("actual_type", fmt.Sprintf("%T", adminIdVal)),
		)
		response.WriteResp(c, nil, common.ServerErr.WithMsg("管理员ID类型断言失败"))
		return
	}
	// 2. 创建带超时的 Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 3. 调用 App 层
	adminEntity, err := h.adminAppSrv.GetAdminInfo(ctx, adminId)
	if err != nil {
		switch err {
		case common.UserNotFoundErr:
			logger.Error("GetInfo 查询管理员失败", zap.Uint64("admin_id", adminId), zap.Error(err))
		default:
			logger.Error("GetInfo 未知系统错误", zap.Uint64("admin_id", adminId), zap.Error(err))
		}
		response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		return
	}
	// 4. 成功响应
	response.WriteResp(c, adminEntity, common.OK)
}

// UpdateAdminInfo 更新管理员信息
func (h *AdminHandler) UpdateAdminInfo(c *gin.Context) {
	var req dto.AdminUpdateInfoReq
	if err := c.ShouldBind(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}
	// 1. 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	// 2. 调用应用层（传递ctx，修复规范问题）
	res, err := h.adminAppSrv.UpdateAdminInfo(ctx, &req)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}
	successData := gin.H{
		"message": "更新成功",
		"data":    res,
	}
	response.WriteResp(c, successData, common.OK)
}
