/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 03:11:10
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-03-14 16:10:22
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
	"blog-server/common/utils/jwt"
	"blog-server/common/utils/logger"
	"context"
	"fmt"
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

// Register 注册接口处理函数：POST /api/blog/v1/user/register
func (h *UserHandler) Register(c *gin.Context) {
	// 1. 绑定请求参数
	var req dto.UserRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// 用统一响应工具，避免硬编码（保持格式一致）
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	// 2. 调用应用层（无gorm，无业务逻辑）
	err := h.userAppSrv.Register(&req)
	if err != nil {
		// 3. 业务失败：统一用gin_util.WriteResp封装响应
		// 断言错误为common.Errno类型，保证格式统一
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			// 兜底：未知错误封装为服务器错误
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	// 4. 成功响应,返回通用成功数据
	successData := gin.H{
		"message":  "注册成功",
		"nickname": req.NickName, // 可选：返回前端传的昵称，提升体验
	}
	response.WriteResp(c, successData, common.OK)

}

func (h *UserHandler) FindUserByNickname(c *gin.Context) {
	var req dto.UserFindInfoReq
	if err := c.ShouldBindQuery(&req); err != nil {
		// 用统一响应工具，避免硬编码（保持格式一致）
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	res, err := h.userAppSrv.FindUserByNickname(&req)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			// 兜底：未知错误封装为服务器错误
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	// 4. 成功响应,返回通用成功数据
	successData := gin.H{
		"message": "查找成功",
		"data":    res, // 可选：返回前端传的昵称，提升体验
	}
	response.WriteResp(c, successData, common.OK)
}

func (h *UserHandler) UpdateUserInfo(c *gin.Context) {
	var req dto.UserUpdateInfoReq
	if err := c.ShouldBind(&req); err != nil {
		// 用统一响应工具，避免硬编码（保持格式一致）
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}
	res, err := h.userAppSrv.UpdateUserInfo(&req)
	if err != nil {
		if errno, ok := err.(common.Errno); ok {
			response.WriteResp(c, nil, errno)
		} else {
			// 兜底：未知错误封装为服务器错误
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}
	successData := gin.H{
		"message": "更新成功",
		"data":    res, // 可选：返回前端传的昵称，提升体验
	}
	response.WriteResp(c, successData, common.OK)
}

func (h *UserHandler) Login(c *gin.Context) {
	var req dto.UserLoginReq
	if err := c.ShouldBind(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数错误："+err.Error()))
		return
	}

	user, err := h.userAppSrv.Login(&req)
	if err != nil {
		response.WriteResp(c, nil, common.AuthErr.WithMsg("账号或密码错误"))
		return
	}

	// 2. 生成包含User对象的JWT Token
	token, err := jwt.GenerateToken(user)
	if err != nil {
		response.WriteResp(c, nil, common.ServerErr.WithMsg("生成Token失败"))
		return
	}

	// 3. 返回Token和用户信息（前端存储Token，后续请求自动携带）
	response.WriteResp(c, gin.H{
		"token": token,
		"user":  user, // 可选：返回用户基本信息
	}, common.OK)
}

func (h *UserHandler) GetInfo(c *gin.Context) {

	// 从标准库context取，完全不受Gin ctx.Writer影响
	// userObj := c.Request.Context().Value(consts.CustomerUserKey)
	// if userObj == nil {
	// 	logger.Warn("GetInfo 未获取到用户信息")
	// 	c.JSON(200, gin.H{"code": 11009, "msg": "Not login"})
	// 	return
	// }

	// 强转成*common.User
	// user, ok := userObj.(*common.User)
	// if !ok {
	// 	logger.Error("用户信息类型错误")
	// 	c.JSON(200, common.AuthErr)
	// 	return
	// }
	// logger.Debug("GetInfo Handler 读取用户信息", zap.Any("user", userObj))

	// 1. 从 Gin Context 中获取用户ID（由 DefaultUserAuth 中间件存入）
	userIdVal, exists1 := c.Get("user_id")
	if !exists1 {
		logger.Warn("GetInfo Handler 未获取到user_id", zap.Bool("user_id_exists", exists1))
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	// 读取完整用户对象（保留）
	userObj, exists2 := c.Get(consts.CustomerUserKey)
	if !exists2 {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}

	// 调试日志（修复：打印原始值 userIdVal，而非转换后的 userId）
	logger.Debug("GetInfo Handler 读取上下文",
		zap.Bool("user_id_exists", exists1),
		zap.Any("user_id_value", userIdVal), // 原始值
		zap.Bool("user_obj_exists", exists2),
		zap.Any("user_obj_value", userObj),
	)

	userId, ok := userIdVal.(int64)
	if !ok {
		logger.Error("GetInfo Handler user_id类型转换失败",
			zap.Any("user_id_value", userId),
			zap.String("expected_type", "uint"),
			zap.String("actual_type", fmt.Sprintf("%T", userIdVal)),
		)
		response.WriteResp(c, nil, common.ServerErr.WithMsg("用户ID类型断言失败"))
		return
	}
	// 2. 创建带超时的 Context（控制整个请求链路的超时）
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel() // 必写：释放资源，避免内存泄漏

	// 3. 调用 App 层方法（Handler 层不直接操作领域/数据库）
	userEntity, err := h.userAppSrv.GetUserInfo(ctx, userId)
	if err != nil {
		// 根据错误类型返回对应响应
		switch err {
		case common.UserNotFoundErr:
			logger.Error("GetInfo 查询用户失败", zap.Int64("user_id", userId), zap.Error(err))
			response.WriteResp(c, nil, common.UserNotFoundErr)
		default:
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}

	// 4. 封装成功响应
	response.WriteResp(c, userEntity, common.OK)
}
