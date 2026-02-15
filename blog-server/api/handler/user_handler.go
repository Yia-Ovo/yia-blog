/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 03:11:10
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-16 03:43:56
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
	gin_util "blog-server/common/gin"

	"github.com/gin-gonic/gin"
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

// Register 注册接口处理函数：POST /api/blog/v1/user/register
func (h *UserHandler) Register(c *gin.Context) {
	// 1. 绑定请求参数
	var req dto.UserRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// 用统一响应工具，避免硬编码（保持格式一致）
		gin_util.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	// 2. 调用应用层（无gorm，无业务逻辑）
	err := h.userAppSrv.Register(&req)
	if err != nil {
		// 3. 业务失败：统一用gin_util.WriteResp封装响应
		// 断言错误为common.Errno类型，保证格式统一
		if errno, ok := err.(common.Errno); ok {
			gin_util.WriteResp(c, nil, errno)
		} else {
			// 兜底：未知错误封装为服务器错误
			gin_util.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	// 4. 成功响应,返回通用成功数据
	successData := gin.H{
		"message":  "注册成功",
		"nickname": req.NickName, // 可选：返回前端传的昵称，提升体验
	}
	gin_util.WriteResp(c, successData, common.OK)

}

func (h *UserHandler) FindUserByNickname(c *gin.Context) {
	var req dto.UserFindInfoReq
	if err := c.ShouldBind(&req); err != nil {
		// 用统一响应工具，避免硬编码（保持格式一致）
		gin_util.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	res, err := h.userAppSrv.FindUserByNickname(&req)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			gin_util.WriteResp(c, nil, errno)
		} else {
			// 兜底：未知错误封装为服务器错误
			gin_util.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	// 4. 成功响应,返回通用成功数据
	successData := gin.H{
		"message": "查找成功",
		"data":    res, // 可选：返回前端传的昵称，提升体验
	}
	gin_util.WriteResp(c, successData, common.OK)
}

func (h *UserHandler) UpdateUserInfo(c *gin.Context) {
	var req dto.UserUpdateInfoReq
	if err := c.ShouldBind(&req); err != nil {
		// 用统一响应工具，避免硬编码（保持格式一致）
		gin_util.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}
	res, err := h.userAppSrv.UpdateUserInfo(&req)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			gin_util.WriteResp(c, nil, errno)
		} else {
			// 兜底：未知错误封装为服务器错误
			gin_util.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}
	successData := gin.H{
		"message": "更新成功",
		"data":    res, // 可选：返回前端传的昵称，提升体验
	}
	gin_util.WriteResp(c, successData, common.OK)
}
