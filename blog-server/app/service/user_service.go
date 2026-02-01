/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:58:44
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-02 03:00:39
 * @FilePath: \yia-blog\blog-server\app\service\user_service.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package service

import (
	"blog-server/api/dto"
	"blog-server/domain/service"
)

// UserAppService 用户应用服务：业务编排，串联领域层
type UserAppService struct {
	// 注入领域服务实例
	domainUserSrv *service.UserService
}

// 对外提供创建实例的方法，给handler调用
func NewUserAppService() *UserAppService {
	return &UserAppService{
		domainUserSrv: service.NewUserService(),
	}
}

// Register 注册编排：调用领域层 → 封装响应DTO
func (s *UserAppService) Register(req *dto.UserRegisterReq) (*dto.UserRegisterRes, error) {
	// 调用领域层核心逻辑（只传昵称和密码，其他由领域层处理）
	err := s.domainUserSrv.Register(req.NickName, req.Password)
	if err != nil {
		// 失败：封装错误响应
		return &dto.UserRegisterRes{
			Code: 500,
			Msg:  "注册失败：" + err.Error(),
		}, err
	}
	// 成功：封装成功响应
	return &dto.UserRegisterRes{
		Code: 200,
		Msg:  "注册成功",
	}, nil
}
