/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-26 07:44:28
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-10 16:15:43
 * @FilePath: \yia-blog\blog-server\api\dto\admin_dto.go
 * @Description: 管理员请求&出参DTO
 */
package dto

import "time"

// ================= 入参Req ===================
// 管理员创建
type AdminCreateReq struct {
	AdminName string `json:"admin_name" binding:"required,min=1,max=16"`
	Password  string `json:"password" binding:"required,min=6,max=20"`
	Role      int    `json:"role" binding:"required,oneof=0 1"`
}

// 管理员登录
type AdminLoginReq struct {
	AdminName string `json:"admin_name" binding:"required,min=1,max=16"`
	Password  string `json:"password" binding:"required,min=6,max=20"`
}

// 修改密码
type AdminUpdatePWDReq struct {
	OldPassword string `json:"old_password" binding:"required,min=6,max=20"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=20"`
}

// 修改基础信息
type AdminUpdateInfoReq struct {
	AdminName *string `json:"admin_name,omitempty" binding:"omitempty,min=1,max=16"`
	Avatar    *string `json:"avatar,omitempty"`
	Role      *int    `json:"role,omitempty" binding:"omitempty,oneof=0 1"`
	Email     *string `json:"email,omitempty"`
	Remark    *string `json:"remark,omitempty"`
}

// 根据名称查询
type AdminFindInfoReq struct {
	AdminName string `form:"admin_name" query:"admin_name" binding:"required,min=1,max=16"`
}

// 列表请求
type AdminListReq struct {
	Page     int `form:"page" default:"1"`
	PageSize int `form:"page_size" default:"10"`
}

// 修改管理员状态请求
type AdminStatusReq struct {
	Status int `json:"status" binding:"required,oneof=0 1"`
}

// 重置管理员密码请求
type AdminResetPWDReq struct {
	NewPassword string `json:"new_password" binding:"required,min=6,max=20"`
}

// ================= 出参 ===================
// AdminBaseDTO：基础管理员字段【被详情、登录信息共用】
type AdminBaseDTO struct {
	ID           uint64 `json:"id"`
	AdminName    string `json:"admin_name"`
	Avatar       string `json:"avatar"`
	Role         int    `json:"role"`
	Status       int    `json:"status"`
	TokenVersion uint64 `json:"-"`
	Email        string `json:"email"`
	Remark       string `json:"remark"`
}

// AdminInfoData 管理员完整详情
type AdminInfoData struct {
	AdminBaseDTO
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreateAt    time.Time  `json:"create_at"`
	UpdateAt    time.Time  `json:"update_at"`
}

// AdminLoginResp 登录返回结构体（和 user 完全一致）
type AdminLoginResp struct {
	Token  string       `json:"token"`
	Expire string       `json:"expire"`
	Data   AdminBaseDTO `json:"data"`
}

// 列表返回
type AdminListResp struct {
	List     []AdminInfoData `json:"list"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}
