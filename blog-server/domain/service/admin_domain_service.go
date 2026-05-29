/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-14 02:31:19
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-29 16:19:42
 * @FilePath: \yia-blog\blog-server\domain\service\admin_domain_service.go
 * @Description: 管理员领域服务（纯业务规则，无任何Redis/基础设施依赖）
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/common"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AdminDomainService 纯净领域服务：仅依赖MySQL仓储，仅处理业务规则
type AdminDomainService struct {
	adminMysqlRepo repo.AdminMysqlRepo
}

// NewAdminDomainService 构造函数：仅注入MySQL依赖
func NewAdminDomainService(adminMysqlRepo repo.AdminMysqlRepo) *AdminDomainService {
	return &AdminDomainService{
		adminMysqlRepo: adminMysqlRepo,
	}
}

// Login 管理员登录：仅业务校验 + 密码验证
func (a *AdminDomainService) Login(ctx context.Context, adminName string, password string) (*do.AdminDO, error) {
	// 参数校验
	if adminName == "" || password == "" {
		logger.Warn("管理员登录参数为空")
		return nil, common.LoginParamEmptyErr
	}

	// 查询数据库
	adminModel, err := a.adminMysqlRepo.FindAdminByName(ctx, adminName)
	if err != nil {
		logger.Error("查询管理员信息失败", zap.String("admin_name", adminName), zap.Error(err))
		return nil, common.DatabaseErr
	}
	if adminModel == nil {
		logger.Warn("管理员不存在", zap.String("admin_name", adminName))
		return nil, common.UserNotFoundErr
	}

	adminDO := convert.AdminModelToDO(adminModel)

	// 状态校验
	if adminDO.Status == 2 {
		logger.Warn("管理员账号已禁用", zap.Uint64("admin_id", adminDO.ID))
		return nil, common.UserAccountDisabledErr
	}

	// 密码校验
	if err := bcrypt.CompareHashAndPassword([]byte(adminDO.Password), []byte(password)); err != nil {
		logger.Warn("管理员密码错误", zap.Uint64("admin_id", adminDO.ID))
		return nil, common.LoginPasswordErr
	}

	logger.Info("管理员登录验证成功", zap.Uint64("admin_id", adminDO.ID))
	return adminDO, nil
}

// UpdatePWD 管理员修改密码：仅业务规则 + DB操作
func (a *AdminDomainService) UpdatePWD(ctx context.Context, adminId uint64, newPassword string, oldPassword string) error {
	// 查询管理员
	adminModel, err := a.adminMysqlRepo.FindAdminByID(ctx, adminId)
	if err != nil {
		logger.Error("查询管理员信息失败", zap.Uint64("admin_id", adminId), zap.Error(err))
		return common.DatabaseErr
	}
	if adminModel == nil {
		logger.Warn("管理员不存在", zap.Uint64("admin_id", adminId))
		return common.UserNotFoundErr
	}
	adminDO := convert.AdminModelToDO(adminModel)

	// 旧密码校验
	if err := bcrypt.CompareHashAndPassword([]byte(adminDO.Password), []byte(oldPassword)); err != nil {
		logger.Warn("旧密码验证失败", zap.Uint64("admin_id", adminId))
		return common.UserOldPasswordErr
	}

	// 新旧密码不能相同
	if newPassword == oldPassword {
		return common.UserPasswordSameError
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("密码加密失败", zap.Error(err))
		return common.PasswordEncryptError
	}

	// 更新数据库
	err = a.adminMysqlRepo.UpdatePWD(ctx, adminId, string(hashedPassword))
	if err != nil {
		logger.Error("更新密码失败", zap.Uint64("admin_id", adminId), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("管理员密码修改成功", zap.Uint64("admin_id", adminId))
	return nil
}

// UpdateAdminInfo 更新管理员信息：仅业务规则
func (a *AdminDomainService) UpdateAdminInfo(ctx context.Context, adminId uint64, updateData *do.AdminDO) error {
	// 查询原数据
	adminModel, err := a.adminMysqlRepo.FindAdminByID(ctx, adminId)
	if err != nil {
		logger.Error("查询管理员信息失败", zap.Uint64("admin_id", adminId), zap.Error(err))
		return common.DatabaseErr
	}
	if adminModel == nil {
		logger.Warn("管理员不存在", zap.Uint64("admin_id", adminId))
		return common.UserNotFoundErr
	}
	adminDO := convert.AdminModelToDO(adminModel)

	// 字段更新
	if updateData.AdminName != "" {
		adminDO.AdminName = updateData.AdminName
	}
	if updateData.Avatar != "" {
		adminDO.Avatar = updateData.Avatar
	}
	if updateData.Role != 0 {
		adminDO.Role = updateData.Role
	}
	if updateData.Email != "" {
		adminDO.Email = updateData.Email
	}
	if updateData.Remark != "" {
		adminDO.Remark = updateData.Remark
	}

	adminDO.UpdateAt = time.Now()
	newModel := convert.AdminDOToModel(adminDO)

	// 保存数据库
	err = a.adminMysqlRepo.UpdateAdminInfo(ctx, newModel)
	if err != nil {
		logger.Error("更新管理员信息失败", zap.Uint64("admin_id", adminId), zap.Error(err))
		return common.DatabaseErr
	}

	logger.Info("更新管理员信息成功", zap.Uint64("admin_id", adminId))
	return nil
}

// FindAdminByID 根据ID查询管理员
func (a *AdminDomainService) FindAdminByID(ctx context.Context, adminId uint64) (*do.AdminDO, error) {
	if adminId <= 0 {
		return nil, common.ParamErr
	}

	adminModel, err := a.adminMysqlRepo.FindAdminByID(ctx, adminId)
	if err != nil {
		logger.Error("查询管理员失败", zap.Uint64("admin_id", adminId), zap.Error(err))
		return nil, common.DatabaseErr
	}
	if adminModel == nil {
		return nil, common.UserNotFoundErr
	}

	return convert.AdminModelToDO(adminModel), nil
}

// FindAdminByName 根据用户名查询管理员
func (a *AdminDomainService) FindAdminByName(ctx context.Context, adminName string) (*do.AdminDO, error) {
	if adminName == "" {
		return nil, common.ParamErr
	}

	adminModel, err := a.adminMysqlRepo.FindAdminByName(ctx, adminName)
	if err != nil {
		logger.Error("查询管理员失败", zap.String("admin_name", adminName), zap.Error(err))
		return nil, common.DatabaseErr
	}
	if adminModel == nil {
		return nil, common.UserNotFoundErr
	}

	return convert.AdminModelToDO(adminModel), nil
}

// UpdateLastLoginInfo 更新最后登录时间
func (a *AdminDomainService) UpdateLastLoginInfo(ctx context.Context, adminId uint64, t *time.Time) error {
	err := a.adminMysqlRepo.UpdateLastLoginInfo(ctx, adminId, t)
	if err != nil {
		logger.Error("更新登录时间失败", zap.Uint64("admin_id", adminId), zap.Error(err))
		return common.DatabaseErr
	}
	return nil
}
