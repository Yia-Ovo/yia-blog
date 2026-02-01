/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-26 07:44:28
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-02 03:07:18
 * @FilePath: \yia-blog\blog-server\api\dto\user_dto.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package dto

// UserRegisterReq 注册请求DTO：前端必须传这些参数，带Gin自动校验规则
type UserRegisterReq struct {
	NickName string `json:"nick_name" binding:"required,min=2,max=16"` // 昵称：必填，2-16位
	Password string `json:"password" binding:"required,min=6"`         // 密码：必填，至少6位
}

// UserRegisterRes 注册响应DTO：后端固定返回这个格式给前端
type UserRegisterRes struct {
	Code int    `json:"code"` // 状态码：200成功，500失败，400参数错误
	Msg  string `json:"msg"`  // 提示信息：成功/失败原因
}
