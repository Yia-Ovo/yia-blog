/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 03:11:10
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-17 12:46:51
 * @FilePath: \yia-blog\blog-server\api\handler\user_handler.go
 * @Description: 用户Handler
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
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserHandler 用户接口处理函数：只处理HTTP相关逻辑
type UserHandler struct {
	// 注入应用服务实例
	userAppSrv *service.UserAppService
}

// 构造函数接收注入的AppService，不再自己创建
func NewUserHandler(userAppSrv *service.UserAppService) *UserHandler {
	return &UserHandler{
		userAppSrv: userAppSrv, // 由外部注入，Handler不关心AppService的依赖
	}
}

func (h *UserHandler) UpdatePWD(c *gin.Context) {
	var req dto.UserUpdatePWDReq
	// 1. 绑定请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}
	// 2. 从 Gin Context 中获取用户ID
	userIdVal, exists := c.Get("user_id")
	if !exists {
		logger.Warn("UpdatePWD Handler 未获取到user_id", zap.Bool("user_id_exists", exists))
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	// 3. ID断言
	userId, ok := userIdVal.(uint64)
	if !ok {
		logger.Error("UpdatePWD Handler user_id类型转换失败",
			zap.Any("user_id_value", userIdVal),
			zap.String("expected_type", "uint64"),
			zap.String("actual_type", fmt.Sprintf("%T", userIdVal)),
		)
		response.WriteResp(c, nil, common.ServerErr.WithMsg("用户ID类型断言失败"))
		return
	}
	// 4. 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	// 5. 调用应用层（透传ctx）
	err := h.userAppSrv.UpdatePWD(ctx, userId, &req)
	if err != nil {
		switch err {
		case common.UserNotFoundErr:
			logger.Error("UpdatePWD 查询用户不存在", zap.Uint64("user_id", userId), zap.Error(err))
		case common.UserOldPasswordErr:
			logger.Warn("UpdatePWD 旧密码验证失败", zap.Uint64("user_id", userId))
		case common.UserNewPasswordInvalidErr:
			logger.Warn("UpdatePWD 新密码格式不合法", zap.Uint64("user_id", userId))
		case common.UserPasswordSameError:
			logger.Warn("UpdatePWD 新密码与旧密码相同", zap.Uint64("user_id", userId))
		case common.PasswordEncryptError:
			logger.Error("UpdatePWD 密码加密失败", zap.Uint64("user_id", userId), zap.Error(err))
		case common.UpdatePasswordError:
			logger.Error("UpdatePWD 更新密码失败", zap.Uint64("user_id", userId), zap.Error(err))
		default:
			logger.Error("UpdatePWD 未知系统错误", zap.Uint64("user_id", userId), zap.Error(err))
		}
		response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		return
	}
	response.WriteResp(c, nil, common.OK)
}

// Register 注册接口处理函数：POST /api/blog/v1/user/register
func (h *UserHandler) Register(c *gin.Context) {
	// 1. 绑定请求参数
	var req dto.UserRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}
	// 2. 创建带超时的Contex
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	// 3. 调用应用层（透传ctx）
	err := h.userAppSrv.Register(ctx, &req)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	successData := gin.H{
		"message":  "注册成功",
		"nickname": req.NickName,
	}
	response.WriteResp(c, successData, common.OK)
}

func (h *UserHandler) FindUserByNickname(c *gin.Context) {
	var req dto.UserFindInfoReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}
	// 1. 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	// 2. 调用应用层（透传ctx）
	res, err := h.userAppSrv.FindUserByNickname(ctx, &req)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	successData := gin.H{
		"message": "查找成功",
		"data":    res,
	}
	response.WriteResp(c, successData, common.OK)
}

func (h *UserHandler) UpdateUserInfo(c *gin.Context) {
	var req dto.UserUpdateInfoReq
	if err := c.ShouldBind(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}
	// 1. 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	// 2. 调用应用层（透传ctx）
	res, err := h.userAppSrv.UpdateUserInfo(ctx, &req)
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

func (h *UserHandler) Login(c *gin.Context) {
	var req dto.UserLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr)
		return
	}
	// 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	// 调用应用层服务
	user, token, err := h.userAppSrv.Login(ctx, &req)
	if err != nil {
		e, ok := err.(common.Errno)
		if !ok {
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
			return
		}
		response.WriteResp(c, nil, e)
		return
	}

	respData := gin.H{
		"user":  user,
		"token": token,
	}
	response.WriteResp(c, respData, common.OK)
}

func (h *UserHandler) Logout(c *gin.Context) {
	// 1. 从请求头获取 Token
	token := c.GetHeader("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		logger.Warn("Logout Handler 未获取到Token")
		response.WriteResp(c, nil, common.ParamErr)
		return
	}
	// 2. 从中间件获取当前登录用户ID
	userIdVal, exists := c.Get("user_id")
	if !exists {
		logger.Warn("Logout Handler 未获取到user_id", zap.Bool("user_id_exists", exists))
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	userId, ok := userIdVal.(uint64)
	if !ok {
		logger.Error("Logout Handler user_id类型转换失败",
			zap.Any("user_id_value", userIdVal),
			zap.String("expected_type", "uint64"),
			zap.String("actual_type", fmt.Sprintf("%T", userIdVal)),
		)
		response.WriteResp(c, nil, common.ServerErr.WithMsg("用户ID类型断言失败"))
		return
	}
	// 3. 创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	// 4. 调用应用层
	err := h.userAppSrv.Logout(ctx, userId, token)
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

	response.WriteResp(c, nil, common.OK)
}

func (h *UserHandler) GetInfo(c *gin.Context) {
	// 1. 从 Gin Context 中获取用户ID
	userIdVal, exists1 := c.Get("user_id")
	if !exists1 {
		logger.Warn("GetInfo Handler 未获取到user_id", zap.Bool("user_id_exists", exists1))
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	userObj, exists2 := c.Get(consts.CustomerUserKey)
	if !exists2 {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}

	logger.Debug("GetInfo Handler 读取上下文",
		zap.Bool("user_id_exists", exists1),
		zap.Any("user_id_value", userIdVal),
		zap.Bool("user_obj_exists", exists2),
		zap.Any("user_obj_value", userObj),
	)

	userId, ok := userIdVal.(uint64)
	if !ok {
		logger.Error("GetInfo Handler user_id类型转换失败",
			zap.Any("user_id_value", userIdVal),
			zap.String("expected_type", "uint64"),
			zap.String("actual_type", fmt.Sprintf("%T", userIdVal)),
		)
		response.WriteResp(c, nil, common.ServerErr.WithMsg("用户ID类型断言失败"))
		return
	}
	// 2. 创建带超时的 Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// 3. 调用 App 层
	userEntity, err := h.userAppSrv.GetUserInfo(ctx, userId)
	if err != nil {
		switch err {
		case common.UserNotFoundErr:
			logger.Error("GetInfo 查询用户失败", zap.Uint64("user_id", userId), zap.Error(err))
		default:
			logger.Error("GetInfo 未知系统错误", zap.Uint64("user_id", userId), zap.Error(err))
		}
		response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		return
	}
	response.WriteResp(c, userEntity, common.OK)
}
