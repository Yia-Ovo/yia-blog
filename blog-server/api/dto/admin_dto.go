// AdminInfoData 管理员信息响应DTO
package dto

import "time"

type AdminInfoData struct {
	ID          uint64     `json:"id"`
	AdminName   string     `json:"admin_name"`
	Avatar      string     `json:"avatar"`
	Role        int        `json:"role"`   // 0-超级管理员 1-普通管理员
	Status      int        `json:"status"` // 1-正常 0-禁用
	Email       string     `json:"email"`
	Remark      string     `json:"remark"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreateAt    time.Time  `json:"create_at"`
}

// AdminFindInfoReq 管理员查询请求
type AdminFindInfoReq struct {
	AdminName string `form:"admin_name" binding:"required"`
}

type AdminLoginReq struct {
	AdminName string `form:"admin_name" binding:"required"`
	Password  string `form:"password" binding:"required"`
}
type AdminUpdatePWDReq struct {
	AdminName   string `form:"admin_name" binding:"required"`
	NewPassword string `form:"new_password" binding:"required"`
	OldPassword string `form:"old_password" binding:"required"`
}

type AdminUpdateInfoReq struct {
	ID        uint64 `form:"id" binding:"required"`
	AdminName string `form:"admin_name" binding:"required"`
	Avatar    string `form:"avatar"`
	Role      int    `form:"role"` // 0-超级管理员 1-普通管理员
	Email     string `form:"email"`
	Remark    string `form:"remark"`
}
