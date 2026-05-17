/*
 * @Author: Yia 1372594487@qq.com>
 * @Date: 2026-05-14 03:30:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-17 09:06:59
 * @FilePath: \yia-blog\blog-server\app\service\admin_app_service.go
 * @Description: 管理员应用服务：业务编排，串联领域层（
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com>, All Rights Reserved.
 */

package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/logger"
	"blog-server/common/utils/password.go"
	"blog-server/config"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/domain/service"
	"context"
	"regexp"
	"strconv"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AdminAppService 管理员应用服务：业务编排，串联领域层
type AdminAppService struct {
	// 注入领域服务实例
	adminDomainSrv *service.AdminDomainService
}

// NewAdminAppService 创建应用服务实例（IOC 外部注入）
func NewAdminAppService(
	// 全部传领域接口（IOC 从外面注入）
	adminMysqlRepo repo.AdminMysqlRepo,
	adminRedisRepo repo.AdminRedisRepo,
	conf *config.GlobalConfig,
) *AdminAppService {
	// 注入领域服务
	adminDomainSrv := service.NewAdminDomainService(
		adminMysqlRepo,
		adminRedisRepo,
	)

	return &AdminAppService{
		adminDomainSrv: adminDomainSrv,
	}
}

// UpdatePWD 管理员修改密码 应用层编排
func (s *AdminAppService) UpdatePWD(ctx context.Context, id uint64, req *dto.AdminUpdatePWDReq) error {
	if req.OldPassword == "" {
		return common.UserPwdParamEmptyErr.WithMsg("旧密码为空")
	}
	if req.NewPassword == "" {
		return common.UserPwdParamEmptyErr.WithMsg("新密码为空")
	}

	if err := password.CheckPasswordFormat(req.NewPassword); err != nil {
		return common.UserNewPasswordInvalidErr.WithMsg(err.Error())
	}

	err := s.adminDomainSrv.UpdatePWD(ctx, id, req.NewPassword, req.OldPassword)
	if err != nil {
		logger.Warn("管理员修改密码业务失败", zap.Uint64("adminId", id), zap.String("err", err.Error()))
		return err
	}
	return nil
}

// GetAdminInfo 获取管理员信息 应用层编排
func (s *AdminAppService) GetAdminInfo(ctx context.Context, id uint64) (*dto.AdminInfoData, error) {
	// 1. 调用领域服务
	adminDO, err := s.adminDomainSrv.FindAdminByID(ctx, id)
	if err != nil {
		// 2. 错误分类处理
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, common.UserNotFoundErr.WithMsg("管理员ID：" + strconv.Itoa(int(id)) + " 不存在")
		case common.DatabaseErr:
			return nil, common.DatabaseErr.WithErr(err)
		default:
			return nil, common.ServerErr.WithErr(err)
		}
	}

	// 3. 组装DTO返回
	data := &dto.AdminInfoData{
		ID:          adminDO.ID,
		AdminName:   adminDO.AdminName,
		Avatar:      adminDO.Avatar,
		Role:        adminDO.Role,
		Status:      adminDO.Status,
		Email:       adminDO.Email,
		Remark:      adminDO.Remark,
		LastLoginAt: adminDO.LastLoginAt,
		CreateAt:    adminDO.CreateAt,
	}
	return data, nil
}

// Login 管理员登录 应用层编排
func (s *AdminAppService) Login(ctx context.Context, req *dto.AdminLoginReq) (*convert.Admin, string, error) {
	// 1. 参数校验
	if req == nil {
		return nil, "", common.LoginParamEmptyErr
	}
	if req.AdminName == "" {
		return nil, "", common.LoginParamEmptyErr
	}
	if req.Password == "" {
		return nil, "", common.LoginParamEmptyErr
	}
	// 2. 调用领域服务
	admin, token, err := s.adminDomainSrv.Login(ctx, req.AdminName, req.Password)
	if err != nil {
		logger.Error("管理员登录失败", zap.String("adminName", req.AdminName), zap.Error(err))
		return nil, "", err
	}
	// 3. 登录成功返回
	logger.Info("管理员登录成功", zap.String("adminName", req.AdminName), zap.Uint64("admin_id", admin.ID))
	return admin, token, nil
}

// Logout 管理员登出 应用层编排
func (s *AdminAppService) Logout(ctx context.Context, adminId uint64, token string) error {
	// 1. 参数校验
	if token == "" {
		return common.TokenEmptyErr
	}
	// 2. 调用领域服务
	err := s.adminDomainSrv.Logout(ctx, adminId, token)
	if err != nil {
		logger.Error("管理员登出失败",
			zap.Uint64("admin_id", adminId),
			zap.Error(err))
		return err
	}
	logger.Info("管理员登出成功", zap.Uint64("admin_id", adminId))
	return nil
}

// FindAdminByName 根据账号查询管理员 应用层编排
func (s *AdminAppService) FindAdminByName(ctx context.Context, req *dto.AdminFindInfoReq) (*dto.AdminInfoData, error) {
	// 1. 调用领域服务
	adminDO, err := s.adminDomainSrv.FindAdminByName(ctx, req.AdminName)
	if err != nil {
		// 2. 错误分类
		switch {
		case err == gorm.ErrRecordNotFound:
			return nil, common.UserNotFoundErr.WithMsg("管理员账号：" + req.AdminName)
		default:
			return nil, common.DatabaseErr.WithErr(err)
		}
	}

	// 3. 转换DTO
	data := &dto.AdminInfoData{
		ID:        adminDO.ID,
		AdminName: adminDO.AdminName,
		Avatar:    adminDO.Avatar,
		Role:      adminDO.Role,
		Status:    adminDO.Status,
		CreateAt:  adminDO.CreateAt,
	}

	return data, nil
}

// 邮箱正则（全局预编译）
var adminEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// UpdateAdminInfo 更新管理员信息 应用层编排
func (s *AdminAppService) UpdateAdminInfo(ctx context.Context, req *dto.AdminUpdateInfoReq) (*dto.AdminInfoData, error) {
	// 1. 前置参数校验
	if req == nil {
		return nil, common.ParamErr.WithMsg("更新管理员信息请求参数不能为空")
	}

	if req.ID <= 0 {
		return nil, common.ParamErr.WithMsg("管理员ID必须为正整数，当前值：" + strconv.FormatUint(req.ID, 10))
	}

	// 校验角色：0-超级管理员 1-普通管理员
	if req.Role < 0 || req.Role > 1 {
		return nil, common.ParamErr.WithMsg("角色值不合法，只能是0(超级管理员)/1(普通管理员)，当前值：" + strconv.Itoa(req.Role))
	}

	// 校验邮箱格式
	if req.Email != "" && !adminEmailRegex.MatchString(req.Email) {
		return nil, common.ParamErr.WithMsg("邮箱格式不合法，当前值：" + req.Email)
	}

	// 2. 组装更新数据
	updateData := &do.AdminDO{
		AdminName: req.AdminName,
		Avatar:    req.Avatar,
		Role:      req.Role,
		Email:     req.Email,
		Remark:    req.Remark,
	}

	// 3. 调用领域服务
	err := s.adminDomainSrv.UpdateAdminInfo(ctx, req.ID, updateData)
	if err != nil {
		switch {
		case err == gorm.ErrRecordNotFound:
			return nil, common.UserNotFoundErr.WithMsg("管理员ID：" + strconv.FormatUint(req.ID, 10))
		default:
			return nil, common.DatabaseErr.WithErr(err).WithMsg("更新管理员信息失败")
		}
	}

	// 4. 查询最新数据返回（使用传入的ctx）
	adminDO, err := s.adminDomainSrv.FindAdminByID(ctx, req.ID)
	if err != nil {
		return nil, common.DatabaseErr.WithErr(err).WithMsg("查询更新后管理员信息失败")
	}

	data := &dto.AdminInfoData{
		ID:          adminDO.ID,
		AdminName:   adminDO.AdminName,
		Avatar:      adminDO.Avatar,
		Role:        adminDO.Role,
		Status:      adminDO.Status,
		Email:       adminDO.Email,
		Remark:      adminDO.Remark,
		LastLoginAt: adminDO.LastLoginAt,
		CreateAt:    adminDO.CreateAt,
	}

	return data, nil
}
