/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-26 07:44:28
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 13:47:12
 * @FilePath: \yia-blog\blog-server\api\dto\user_dto.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package dto

import "time"

// ------------------------------ 请求 DTO（入参） ------------------------------
type UserRegisterReq struct {
	NickName string `json:"nick_name" binding:"required,min=1,max=16"`
	Password string `json:"password" binding:"required,min=6,max=20"`
}

type UserLoginReq struct {
	NickName string `json:"nick_name" binding:"required,min=1,max=16"`
	Password string `json:"password" binding:"required,min=6,max=20"`
}

type UserUpdatePWDReq struct {
	OldPassword string `json:"old_password" binding:"required,min=6,max=20"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=20"`
}

type UserUpdateInfoReq struct {
	NickName  string `json:"nick_name"`
	Avatar    string `json:"avatar"`
	Gender    int    `json:"gender"`
	Signature string `json:"signature"`
	Email     string `json:"email"`
}

type UserFindInfoReq struct {
	NickName string `form:"nick_name" query:"nick_name" binding:"required,min=1,max=16"`
}

// ------------------------------ 响应 DTO（出参） ------------------------------

// UserInfoData 标准用户详情（所有接口共用）
type UserInfoData struct {
	ID          uint64     `json:"id"`
	NickName    string     `json:"nick_name"`
	Avatar      string     `json:"avatar"`
	Status      int        `json:"status"`
	Gender      int        `json:"gender"`
	Signature   string     `json:"signature"`
	Email       string     `json:"email"`
	CreateAt    time.Time  `json:"create_at"`
	UpdateAt    time.Time  `json:"update_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

// LoginUserDTO 专门给 JWT 生成 Token 使用
type LoginUserDTO struct {
	UserID      uint64     `json:"user_id"`
	NickName    string     `json:"nick_name"`
	Status      int        `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// ------------------------------ 通用统一响应（全局复用，删除冗余结构体） ------------------------------
type BaseResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// DataResponse 带 data 的通用返回
type DataResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}
