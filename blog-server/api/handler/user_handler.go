/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 03:11:10
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 21:07:55
 * @FilePath: \yia-blog\blog-server\api\handler\user_handler.go
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
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserHandler 用户接口处理函数：只处理HTTP相关逻辑
type UserHandler struct {
	userAppSrv *service.UserAppService
}

func NewUserHandler(userAppSrv *service.UserAppService) *UserHandler {
	return &UserHandler{
		userAppSrv: userAppSrv,
	}
}

// Register 用户注册 POST /api/blog/v1/user/register
func (h *UserHandler) Register(c *gin.Context) {
	var req dto.UserRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.userAppSrv.Register(ctx, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}

// Login 用户登录 POST /api/blog/v1/user/login
func (h *UserHandler) Login(c *gin.Context) {
	var req dto.UserLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	user, token, err := h.userAppSrv.Login(ctx, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, gin.H{
		"user":  user,
		"token": token,
	}, common.OK)
}

// Logout 用户登出 POST /api/blog/v1/user/logout
func (h *UserHandler) Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		response.WriteResp(c, nil, common.ParamErr)
		return
	}

	userIdVal, exists := c.Get("user_id")
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	userId, ok := userIdVal.(uint64)
	if !ok {
		logger.Error("Logout user_id类型转换失败",
			zap.Any("user_id_value", userIdVal),
			zap.String("actual_type", fmt.Sprintf("%T", userIdVal)),
		)
		response.WriteResp(c, nil, common.ServerErr.WithMsg("用户ID类型断言失败"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.userAppSrv.Logout(ctx, userId, token)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, nil, common.OK)
}

// GetInfo 获取用户信息 GET /api/blog/v1/user/getInfo
func (h *UserHandler) GetInfo(c *gin.Context) {
	userIdVal, exists1 := c.Get("user_id")
	_, exists2 := c.Get(consts.CustomerUserKey)
	if !exists1 || !exists2 {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}

	userId, ok := userIdVal.(uint64)
	if !ok {
		logger.Error("GetInfo user_id类型转换失败",
			zap.Any("user_id_value", userIdVal),
			zap.String("actual_type", fmt.Sprintf("%T", userIdVal)),
		)
		response.WriteResp(c, nil, common.ServerErr.WithMsg("用户ID类型断言失败"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	userEntity, err := h.userAppSrv.GetUserInfo(ctx, userId)
	if err != nil {
		logger.Error("GetInfo 查询用户失败", zap.Uint64("user_id", userId), zap.Error(err))
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, userEntity, common.OK)
}

// UpdatePWD 修改密码 POST /api/blog/v1/user/updatePWD
func (h *UserHandler) UpdatePWD(c *gin.Context) {
	var req dto.UserUpdatePWDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	userIdVal, exists := c.Get("user_id")
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}

	userId, ok := userIdVal.(uint64)
	if !ok {
		logger.Error("UpdatePWD user_id类型转换失败",
			zap.Any("user_id_value", userIdVal),
			zap.String("actual_type", fmt.Sprintf("%T", userIdVal)),
		)
		response.WriteResp(c, nil, common.ServerErr.WithMsg("用户ID类型断言失败"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.userAppSrv.UpdatePWD(ctx, userId, &req)
	if err != nil {
		logger.Error("UpdatePWD 失败", zap.Uint64("user_id", userId), zap.Error(err))
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}
	response.WriteResp(c, nil, common.OK)
}

// UpdateUserInfo 更新用户信息 POST /api/blog/v1/user/updateUserInfo
func (h *UserHandler) UpdateUserInfo(c *gin.Context) {
	var req dto.UserUpdateInfoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	userIdVal, exists := c.Get("user_id")
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	userId, ok := userIdVal.(uint64)
	if !ok {
		logger.Error("UpdateUserInfo user_id类型转换失败",
			zap.Any("user_id_value", userIdVal),
			zap.String("actual_type", fmt.Sprintf("%T", userIdVal)),
		)
		response.WriteResp(c, nil, common.ServerErr.WithMsg("用户ID类型断言失败"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	res, err := h.userAppSrv.UpdateUserInfo(ctx, userId, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, res, common.OK)
}

// FindUserByNickname 根据昵称查询用户 GET /api/blog/v1/user/findUserByNickname
func (h *UserHandler) FindUserByNickname(c *gin.Context) {
	var req dto.UserFindInfoReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	res, err := h.userAppSrv.FindUserByNickname(ctx, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, res, common.OK)
}
