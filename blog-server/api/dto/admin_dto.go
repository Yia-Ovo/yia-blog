/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-17 08:01:48
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 14:17:31
 * @FilePath: \yia-blog\blog-server\api\dto\admin_dto.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package dto

import "time"

// ------------------------------ 请求 DTO ------------------------------
type AdminLoginReq struct {
	AdminName string `json:"admin_name" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

type AdminUpdatePWDReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type AdminUpdateInfoReq struct {
	AdminName string `json:"admin_name"`
	Avatar    string `json:"avatar"`
	Role      int    `json:"role"` // 0-超级管理员 1-普通管理员
	Email     string `json:"email"`
	Remark    string `json:"remark"`
}

type AdminFindInfoReq struct {
	AdminName string `form:"admin_name" binding:"required"`
}

// ------------------------------ 响应 DTO ------------------------------

// AdminInfoData 管理员详情（返回前端用）
type AdminInfoData struct {
	ID          uint64     `json:"id"`
	AdminName   string     `json:"admin_name"`
	Avatar      string     `json:"avatar"`
	Role        int        `json:"role"`
	Status      int        `json:"status"`
	Email       string     `json:"email"`
	Remark      string     `json:"remark"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreateAt    time.Time  `json:"create_at"`
}

// AdminLoginDTO 专门给 JWT 生成 Token 使用（精简结构）
type AdminLoginDTO struct {
	ID          uint64     `json:"id"`
	AdminName   string     `json:"admin_name"`
	Role        int        `json:"role"`
	Status      int        `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}
