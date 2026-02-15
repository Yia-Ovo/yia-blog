/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-26 07:44:28
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-16 03:03:55
 * @FilePath: \yia-blog\blog-server\api\dto\user_dto.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package dto

import "time"

// UserRegisterReq 注册请求DTO：前端必须传这些参数，带Gin自动校验规则
type UserRegisterReq struct {
	NickName string `json:"nick_name" binding:"required,min=1,max=16"` // 昵称：必填，2-16位
	Password string `json:"password" binding:"required,min=6"`         // 密码：必填，至少6位
}

// UserRegisterRes 注册响应DTO：后端固定返回这个格式给前端
type UserRegisterRes struct {
	Code int    `json:"code"` // 状态码：200成功，500失败，400参数错误
	Msg  string `json:"msg"`  // 提示信息：成功/失败原因
}

// 仅定义前端需要的字段，无敏感信息、无领域层依赖
type UserInfoData struct {
	ID        int64     `json:"id"`        // 用户ID
	NickName  string    `json:"nick_name"` // 昵称
	Avatar    string    `json:"avatar"`    // 头像
	Status    int       `json:"status"`    // 用户状态（仅返回前端需要的字段）
	Gender    *int      `json:"gender"`    // 性别
	Signature string    `json:"signature"` // 个性签名
	Email     string    `json:"email"`     // 邮箱
	CreateAt  time.Time `json:"create_at"` // 创建时间
}

// UserFindInfoReq 查询用户信息请求DTO：前端必须传这些参数，带Gin自动校验规则
type UserFindInfoReq struct {
	NickName string `form:"nick_name" query:"nick_name" binding:"required,min=1,max=16"`
}
type UserFindInfoRes struct {
	Code int           `json:"code"` // 状态码：200成功，500失败，400参数错误
	Msg  string        `json:"msg"`  // 提示信息：成功/失败原因
	Data *UserInfoData `json:"data"` // 用户信息
}

type UserUpdateInfoReq struct {
	ID        int64  `json:"user_id" binding:"required"` // 用户ID
	NickName  string `json:"nick_name"`                  // 昵称
	Avatar    string `json:"avatar" `                    // 头像
	Gender    *int   `json:"gender"`                     // 性别
	Signature string `json:"signature"`                  // 个性签名
	Email     string `json:"email"`                      // 邮箱
}

type UserUpdateInfoRes struct {
	Code int    `json:"code"` // 状态码：200成功，500失败，400参数错误
	Msg  string `json:"msg"`  // 提示信息：成功/失败原因
}
