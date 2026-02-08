/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:58:44
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-09 00:52:40
 * @FilePath: \yia-blog\blog-server\app\service\user_app_service.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/domain/service"
	"blog-server/infra/repo"

	"gorm.io/gorm"
)

// UserAppService 用户应用服务：业务编排，串联领域层
type UserAppService struct {
	// 注入领域服务实例
	userDomainSrv *service.UserDomainService
}

// NewUserAppService 创建应用服务实例（初始化链路：DB → 仓库实现 → 领域服务 → 应用服务）
func NewUserAppService(db *gorm.DB) *UserAppService {
	// 1. 创建基础设施层的仓库实现（MySQL）
	userInfraRepo := repo.NewUserInfraRepo(db)
	// 2. 注入仓库实现到领域服务
	userDomainSrv := service.NewUserDomainService(userInfraRepo)
	// 3. 返回应用服务实例
	return &UserAppService{
		userDomainSrv: userDomainSrv,
	}
}

// Register 注册接口的应用层编排（仅编排，无核心业务逻辑）
func (s *UserAppService) Register(req *dto.UserRegisterReq) error {
	// 1. 先校验昵称是否已存在（调用领域服务）
	_, err := s.userDomainSrv.FindUserByNickname(req.NickName)
	if err == nil {
		// 昵称已存在，返回标准化错误
		return common.UserExistErr.WithMsg("昵称：" + req.NickName)
	}

	// 2. 调用领域服务执行注册
	if err := s.userDomainSrv.Register(req.NickName, req.Password); err != nil {
		// 数据库错误，封装为标准化错误
		return common.DatabaseErr.WithErr(err)
	}

	return nil
}

// FindUserByNickname 查询用户信息编排：调用领域层 → 封装响应DTO
func (s *UserAppService) FindUserByNickname(req *dto.UserFindInfoReq) (*dto.UserFindInfoRes, error) {
	// 1. 调用领域服务查询用户DO
	userDO, err := s.userDomainSrv.FindUserByNickname(req.NickName)
	if err != nil {
		// 2. 错误分类：映射为标准化 Errno 错误
		switch {
		// 场景1：用户不存在（匹配GORM无记录错误）
		case err == gorm.ErrRecordNotFound:
			return nil, common.UserNotFoundErr.WithMsg("昵称：" + req.NickName)
		// 场景2：数据库底层错误
		default:
			return nil, common.DatabaseErr.WithErr(err)
		}
	}

	// 3. 成功场景：转换领域对象(DO)到响应DTO，返回数据+nil（代表无错误）
	return &dto.UserFindInfoRes{
		Data: &dto.UserInfoData{
			ID:       userDO.ID,
			NickName: userDO.NickName,
			Avatar:   userDO.Avatar,
			Status:   userDO.Status,
			CreateAt: userDO.CreateAt,
		},
	}, nil
}
