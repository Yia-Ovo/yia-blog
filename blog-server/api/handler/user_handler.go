/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 03:11:10
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-02 03:11:22
 * @FilePath: \yia-blog\blog-server\api\handler\user_handler.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package handler

import (
	"blog-server/api/dto"
	"blog-server/app/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户接口处理函数：只处理HTTP相关逻辑
type UserHandler struct {
	// 注入应用服务实例
	userAppSrv *service.UserAppService
}

// 对外提供创建实例的方法，给路由调用
func NewUserHandler() *UserHandler {
	return &UserHandler{
		userAppSrv: service.NewUserAppService(),
	}
}

// Register 注册接口处理函数：POST /api/blog/v1/user/register
func (h *UserHandler) Register(c *gin.Context) {
	// 1. 定义请求DTO变量，用于绑定前端参数
	var req dto.UserRegisterReq
	// 2. 绑定并校验参数：JSON→结构体，校验失败直接返回400
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "参数错误：" + err.Error(),
		})
		return
	}
	// 3. 参数校验通过，调用应用层
	res, err := h.userAppSrv.Register(&req)
	if err != nil {
		// 4. 业务失败，返回应用层封装的错误响应
		c.JSON(http.StatusOK, res)
		return
	}
	// 5. 全部成功，返回成功响应
	c.JSON(http.StatusOK, res)
}
