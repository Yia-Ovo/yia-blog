/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-26 07:44:28
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-05 18:50:06
 * @FilePath: \yia-blog\blog-server\api\dto\user_dto.go
 * @Description: 用户请求&出参DTO
 */
package dto

import "time"

// ================= 入参Req ===================
// 注册
type UserRegisterReq struct {
	NickName string `json:"nick_name" binding:"required,min=1,max=16"`
	Password string `json:"password" binding:"required,min=6,max=20"`
}

// 登录
type UserLoginReq struct {
	NickName string `json:"nick_name" binding:"required,min=1,max=16"`
	Password string `json:"password" binding:"required,min=6,max=20"`
}

// 修改密码
type UserUpdatePWDReq struct {
	OldPassword string `json:"old_password" binding:"required,min=6,max=20"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=20"`
}

// UserUpdateInfoReq 只接受本人可编辑资料；nil 不更新，空串可清空头像/签名/邮箱，gender=0 可重置。
// status/password/token_version 不接受客户端提交，由独立领域用例维护。
type UserUpdateInfoReq struct {
	NickName  *string `json:"nick_name,omitempty" binding:"omitempty,min=1,max=16"`
	Avatar    *string `json:"avatar,omitempty"`
	Gender    *int    `json:"gender,omitempty"`
	Signature *string `json:"signature,omitempty"`
	Email     *string `json:"email,omitempty"`
}

// 根据昵称查询用户
type UserFindInfoReq struct {
	NickName string `form:"nick_name" query:"nick_name" binding:"required,min=1,max=16"`
}

// ================= 出参 ===================
// PublicUserInfo 公开展示白名单，不嵌入包含邮箱和账号控制字段的本人资料。
type PublicUserInfo struct {
	ID        uint64 `json:"id"`
	NickName  string `json:"nick_name"`
	Avatar    string `json:"avatar"`
	Signature string `json:"signature"`
}

// UserBaseDTO：基础用户字段【被详情、登录信息共用，核心去重】
type UserBaseDTO struct {
	ID           uint64 `json:"id"`
	NickName     string `json:"nick_name"`
	Avatar       string `json:"avatar"`
	Status       int    `json:"status"`
	TokenVersion uint64 `json:"-"`
	Gender       int    `json:"gender"`
	Signature    string `json:"signature"`
	Email        string `json:"email"`
}

// UserInfoData 用户完整详情（嵌入基础DTO，补充扩展字段，不用重复写字段）
type UserInfoData struct {
	UserBaseDTO
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreateAt    time.Time  `json:"create_at"`
	UpdateAt    time.Time  `json:"update_at"`
}

// LoginResp 登录返回结构体（token+用户基础信息，替代原LoginUserDTO）
type LoginResp struct {
	Token  string      `json:"token"`
	Expire string      `json:"expire"`
	Data   UserBaseDTO `json:"data"`
}
