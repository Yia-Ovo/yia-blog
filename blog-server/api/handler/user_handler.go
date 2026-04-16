/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 03:11:10
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-15 13:09:51
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
		// 用统一响应工具，避免硬编码（保持格式一致）
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}
	// 2. 从 Gin Context 中获取用户ID（由 DefaultUserAuth 中间件存入）
	userIdVal, exists := c.Get("user_id") // 获取用户id
	if !exists {
		logger.Warn("GetInfo Handler 未获取到user_id", zap.Bool("user_id_exists", exists))
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	// 3. ID断言
	userId, ok := userIdVal.(uint64)
	if !ok {
		logger.Error("GetInfo Handler user_id类型转换失败",
			zap.Any("user_id_value", userId),
			zap.String("expected_type", "uint"),
			zap.String("actual_type", fmt.Sprintf("%T", userIdVal)),
		)
		response.WriteResp(c, nil, common.ServerErr.WithMsg("用户ID类型断言失败"))
		return
	}
	// 4.创建带超时的Context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel() // 释放资源，避免内存泄漏
	// 5. 调用应用层
	err := h.userAppSrv.UpdatePWD(ctx, userId, &req)
	if err != nil {
		switch err {
		// 用户不存在
		case common.UserNotFoundErr:
			logger.Error("UpdatePWD 查询用户不存在", zap.Uint64("user_id", userId), zap.Error(err))
			response.WriteResp(c, nil, common.UserNotFoundErr)
		// 旧密码错误
		case common.UserOldPasswordErr:
			logger.Warn("UpdatePWD 旧密码验证失败", zap.Uint64("user_id", userId))
			response.WriteResp(c, nil, common.UserOldPasswordErr)
		// 新密码格式错误
		case common.UserNewPasswordInvalidErr:
			logger.Warn("UpdatePWD 新密码格式不合法", zap.Uint64("user_id", userId))
			response.WriteResp(c, nil, common.UserNewPasswordInvalidErr)
		// 新旧密码相同
		case common.UserPasswordSameError:
			logger.Warn("UpdatePWD 新密码与旧密码相同", zap.Uint64("user_id", userId))
			response.WriteResp(c, nil, common.UserPasswordSameError)
		// 密码加密失败
		case common.PasswordEncryptError:
			logger.Error("UpdatePWD 密码加密失败", zap.Uint64("user_id", userId), zap.Error(err))
			response.WriteResp(c, nil, common.PasswordEncryptError)
		// 更新密码失败
		case common.UpdatePasswordError:
			logger.Error("UpdatePWD 更新密码失败", zap.Uint64("user_id", userId), zap.Error(err))
			response.WriteResp(c, nil, common.UpdatePasswordError)
		// ❗ 只有未知/系统错误，才返回500
		default:
			logger.Error("UpdatePWD 未知系统错误", zap.Uint64("user_id", userId), zap.Error(err))
			response.WriteResp(c, nil, common.ServerErr)
		}
		return
	}
	// 6. 成功响应,返回通用成功数据
	response.WriteResp(c, nil, common.OK)

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
	// 1. 绑定请求参数
	var req dto.UserLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// 参数绑定失败：返回标准参数错误
		response.WriteResp(c, nil, common.ParamErr)
		return
	}
	// 2. 调用应用层服务
	user, token, err := h.userAppSrv.Login(c.Request.Context(), &req)
	if err != nil {
		// 断言错误为自定义标准错误码
		e, ok := err.(common.Errno)
		if !ok {
			// 非标准错误：返回服务器异常
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
			return
		}
		// 直接返回业务错误（用户名错误/密码错误/账号锁定等）
		response.WriteResp(c, nil, e)
		return
	}
	// 3. 登录成功：封装返回数据（用户信息 + Token）
	respData := gin.H{
		"user":  user,
		"token": token,
	}
	// 4. 用你封装的 WriteResp 返回统一响应体
	response.WriteResp(c, respData, common.OK)

}
func (h *UserHandler) Logout(c *gin.Context) {
	// 1. 从请求头获取 Token
	token := c.GetHeader("Authorization")
	// 兼容前端传 Bearer token 的情况
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		logger.Warn("Logout Handler 未获取到Token")
		response.WriteResp(c, nil, common.ParamErr)
		return
	}

	// 2. 从中间件获取当前登录用户ID
	userId, _ := c.Get("userId")
	// 3. 调用应用层：同时传 token + userId
	err := h.userAppSrv.Logout(c.Request.Context(), userId.(uint64), token)
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

	userId, ok := userIdVal.(uint64)
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
			logger.Error("GetInfo 查询用户失败", zap.Uint64("user_id", userId), zap.Error(err))
			response.WriteResp(c, nil, common.UserNotFoundErr)
		default:
			response.WriteResp(c, nil, common.ServerErr.WithErr(err))
		}
		return
	}
	// 4. 封装成功响应
	response.WriteResp(c, userEntity, common.OK)
}
