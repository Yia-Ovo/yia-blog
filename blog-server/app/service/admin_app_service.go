/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-29 16:00:00
 * @LastEditors: Yia 1372594487@qq.com
 * @FilePath: \yia-blog\blog-server\app\service\admin_app_service.go
 * @Description: 管理员应用服务 - 业务编排、缓存处理、流程调度
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/utils/jwt"
	"blog-server/common/utils/logger"
	"blog-server/common/utils/password"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/domain/service"
	"context"
	"regexp"
	"time"

	"go.uber.org/zap"
)

// AdminAppService 应用服务：仅做流程编排 + 缓存处理
type AdminAppService struct {
	adminDomainService *service.AdminDomainService // 领域服务（纯业务）
	adminRedisRepo     repo.AdminRedisRepo         // 缓存仓储（应用层编排）
}

// NewAdminAppService 构造函数：依赖外部注入，无强耦合
func NewAdminAppService(
	domainService *service.AdminDomainService,
	redisRepo repo.AdminRedisRepo,
) *AdminAppService {
	return &AdminAppService{
		adminDomainService: domainService,
		adminRedisRepo:     redisRepo,
	}
}

// Login 管理员登录：应用层处理 JWT + 登录时间更新
func (s *AdminAppService) Login(ctx context.Context, req *dto.AdminLoginReq) (*dto.AdminLoginDTO, string, error) {
	if req == nil || req.AdminName == "" || req.Password == "" {
		logger.Warn("管理员登录参数为空")
		return nil, "", common.LoginParamEmptyErr
	}

	// 1. 执行核心业务逻辑
	adminDO, err := s.adminDomainService.Login(ctx, req.AdminName, req.Password)
	if err != nil {
		logger.Error("管理员登录验证失败", zap.Error(err))
		return nil, "", err
	}

	// 2. 应用层编排：更新登录时间
	now := time.Now()
	_ = s.adminDomainService.UpdateLastLoginInfo(ctx, adminDO.ID, &now)

	// 3. 应用层编排：生成 JWT
	loginDTO := &dto.AdminLoginDTO{
		ID:          adminDO.ID,
		AdminName:   adminDO.AdminName,
		Role:        adminDO.Role,
		Status:      adminDO.Status,
		LastLoginAt: adminDO.LastLoginAt,
	}
	token, err := jwt.GenerateAdminToken(loginDTO)
	if err != nil {
		logger.Error("生成管理员 Token 失败", zap.Error(err))
		return nil, "", common.ServerErr
	}

	logger.Info("管理员登录成功", zap.Uint64("admin_id", adminDO.ID))
	return loginDTO, token, nil
}

// Logout 管理员登出：应用层处理 Redis 黑名单
func (s *AdminAppService) Logout(ctx context.Context, adminID uint64, token string) error {
	if token == "" {
		logger.Warn("管理员登出 Token 为空")
		return common.TokenEmptyErr
	}

	// 1. 应用层编排：获取 Token 过期时间
	expire, errNo := jwt.GetTokenRemainingTime(token)
	if !errNo.IsOk() {
		logger.Warn("获取 Token 过期时间失败", zap.Uint64("admin_id", adminID))
		return errNo
	}

	// 2. 应用层编排：加入黑名单
	_ = s.adminRedisRepo.AddAdminTokenToBlackList(ctx, token, expire)
	_ = s.adminRedisRepo.ClearAdminAllOnlineToken(ctx, adminID)

	logger.Info("管理员登出成功", zap.Uint64("admin_id", adminID))
	return nil
}

// GetAdminInfo 获取管理员信息
func (s *AdminAppService) GetAdminInfo(ctx context.Context, id uint64) (*dto.AdminInfoData, error) {
	adminDO, err := s.adminDomainService.FindAdminByID(ctx, id)
	if err != nil {
		if err == common.UserNotFoundErr {
			logger.Warn("管理员不存在", zap.Uint64("admin_id", id))
			return nil, common.UserNotFoundErr
		}
		logger.Error("查询管理员信息失败", zap.Uint64("admin_id", id), zap.Error(err))
		return nil, common.ServerErr
	}

	return &dto.AdminInfoData{
		ID:          adminDO.ID,
		AdminName:   adminDO.AdminName,
		Avatar:      adminDO.Avatar,
		Role:        adminDO.Role,
		Status:      adminDO.Status,
		Email:       adminDO.Email,
		Remark:      adminDO.Remark,
		LastLoginAt: adminDO.LastLoginAt,
		CreateAt:    adminDO.CreateAt,
	}, nil
}

// UpdatePWD 修改密码：应用层编排 + 清理缓存
func (s *AdminAppService) UpdatePWD(ctx context.Context, id uint64, req *dto.AdminUpdatePWDReq) error {
	if req.OldPassword == "" {
		logger.Warn("修改密码旧密码为空")
		return common.UserPwdParamEmptyErr.WithMsg("旧密码为空")
	}
	if req.NewPassword == "" {
		logger.Warn("修改密码新密码为空")
		return common.UserPwdParamEmptyErr.WithMsg("新密码为空")
	}
	if err := password.CheckPasswordFormat(req.NewPassword); err != nil {
		return common.UserNewPasswordInvalidErr.WithMsg(err.Error())
	}

	// 1. 执行核心业务逻辑
	err := s.adminDomainService.UpdatePWD(ctx, id, req.NewPassword, req.OldPassword)
	if err != nil {
		logger.Error("修改密码失败", zap.Uint64("admin_id", id), zap.Error(err))
		return err
	}

	// 2. 应用层编排：清理缓存
	_ = s.adminRedisRepo.DelAdminFromRedis(ctx, id)
	logger.Info("管理员密码修改成功，缓存已清理", zap.Uint64("admin_id", id))
	return nil
}

// UpdateAdminInfo 更新管理员信息：应用层编排 + 清理缓存
func (s *AdminAppService) UpdateAdminInfo(ctx context.Context, id uint64, req *dto.AdminUpdateInfoReq) (*dto.AdminInfoData, error) {
	if req.AdminName == "" {
		return nil, common.ParamErr.WithMsg("管理员名称不能为空")
	}
	if req.Role < 0 || req.Role > 1 {
		return nil, common.ParamErr.WithMsg("角色只能是 0 或 1")
	}

	var adminEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if req.Email != "" && !adminEmailRegex.MatchString(req.Email) {
		return nil, common.ParamErr.WithMsg("邮箱格式错误")
	}

	updateData := &do.AdminDO{
		AdminName: req.AdminName,
		Avatar:    req.Avatar,
		Role:      req.Role,
		Email:     req.Email,
		Remark:    req.Remark,
	}

	// 1. 执行核心业务逻辑
	err := s.adminDomainService.UpdateAdminInfo(ctx, id, updateData)
	if err != nil {
		logger.Error("更新管理员信息失败", zap.Uint64("admin_id", id), zap.Error(err))
		return nil, common.ServerErr
	}

	// 2. 应用层编排：清理缓存
	_ = s.adminRedisRepo.DelAdminFromRedis(ctx, id)

	newAdmin, _ := s.adminDomainService.FindAdminByID(ctx, id)
	return &dto.AdminInfoData{
		ID:          newAdmin.ID,
		AdminName:   newAdmin.AdminName,
		Avatar:      newAdmin.Avatar,
		Role:        newAdmin.Role,
		Status:      newAdmin.Status,
		Email:       newAdmin.Email,
		Remark:      newAdmin.Remark,
		LastLoginAt: newAdmin.LastLoginAt,
		CreateAt:    newAdmin.CreateAt,
	}, nil
}

// FindAdminByName 根据用户名查询管理员
func (s *AdminAppService) FindAdminByName(ctx context.Context, req *dto.AdminFindInfoReq) (*dto.AdminInfoData, error) {
	adminDO, err := s.adminDomainService.FindAdminByName(ctx, req.AdminName)
	if err != nil {
		if err == common.UserNotFoundErr {
			logger.Warn("管理员不存在", zap.String("admin_name", req.AdminName))
			return nil, common.UserNotFoundErr
		}
		logger.Error("查询管理员失败", zap.String("admin_name", req.AdminName), zap.Error(err))
		return nil, common.ServerErr
	}

	return &dto.AdminInfoData{
		ID:        adminDO.ID,
		AdminName: adminDO.AdminName,
		Avatar:    adminDO.Avatar,
		Role:      adminDO.Role,
		Status:    adminDO.Status,
		CreateAt:  adminDO.CreateAt,
	}, nil
}
