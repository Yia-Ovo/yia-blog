/*
 * @Author: Yia 2594487
 * @Date: 2026-05-14 02:00:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-05-17 10:17:10
 * @Description: 管理员领域服务
 */
package service

import (
	"blog-server/common"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/jwt"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AdminDomainService 管理员领域服务：封装核心业务逻辑
type AdminDomainService struct {
	adminMysqlRepo repo.AdminMysqlRepo
	adminRedisRepo repo.AdminRedisRepo
}

func NewAdminDomainService(adminMysqlRepo repo.AdminMysqlRepo, adminRedisRepo repo.AdminRedisRepo) *AdminDomainService {
	return &AdminDomainService{
		adminMysqlRepo: adminMysqlRepo,
		adminRedisRepo: adminRedisRepo,
	}
}

// UpdatePWD 管理员修改密码
func (a *AdminDomainService) UpdatePWD(ctx context.Context, adminId uint64, newPassword string, oldPassword string) error {
	// 走缓存查询
	adminDO, err := a.FindAdminByID(ctx, adminId)
	if err != nil {
		logger.Warn("通过管理员ID查询用户失败", zap.Error(err), zap.Uint64("adminId", adminId))
		return err
	}
	if adminDO == nil {
		logger.Warn("管理员不存在", zap.Uint64("adminId", adminId))
		return common.UserNotFoundErr
	}

	// 旧密码校验
	if err := bcrypt.CompareHashAndPassword([]byte(adminDO.Password), []byte(oldPassword)); err != nil {
		logger.Warn("管理员旧密码验证失败", zap.Uint64("adminId", adminId))
		return common.UserOldPasswordErr
	}

	// 新旧密码不能相同
	if newPassword == oldPassword {
		return common.UserPasswordSameError
	}

	// 加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return common.PasswordEncryptError
	}

	// 更新数据库
	if err := a.adminMysqlRepo.UpdatePWD(ctx, adminId, string(hashedPassword)); err != nil {
		return common.UpdatePasswordError
	}

	// 删除缓存（关键！）
	_ = a.adminRedisRepo.DelAdminFromRedis(ctx, adminId)

	logger.Info("管理员密码修改成功", zap.Uint64("adminId", adminId))
	return nil
}

// Login 管理员登录
func (a *AdminDomainService) Login(ctx context.Context, adminName string, password string) (*convert.Admin, string, error) {
	// ===================== 修复：统一走带缓存的方法 =====================
	adminDO, err := a.FindAdminByName(ctx, adminName)
	if err != nil {
		if errors.Is(err, common.UserNotFoundErr) {
			return nil, "", common.LoginPasswordErr
		}
		logger.Error("管理员查询失败", zap.Error(err), zap.String("adminName", adminName))
		return nil, "", common.DatabaseErr.WithErr(err)
	}

	// 状态校验
	if adminDO.Status == 2 {
		return nil, "", common.UserAccountDisabledErr
	}

	// 密码校验
	if err = bcrypt.CompareHashAndPassword([]byte(adminDO.Password), []byte(password)); err != nil {
		logger.Warn("管理员密码校验失败", zap.Uint64("admin_id", adminDO.ID))
		return nil, "", common.LoginPasswordErr
	}

	// 更新登录时间
	now := time.Now()
	updateErr := a.adminMysqlRepo.UpdateLastLoginInfo(ctx, adminDO.ID, &now)
	if updateErr == nil {
		_ = a.adminRedisRepo.DelAdminFromRedis(ctx, adminDO.ID)
	}

	// 生成 token
	commonAdmin := convert.AdminDOToAdmin(adminDO)
	token, err := jwt.GenerateAdminToken(commonAdmin)
	if err != nil {
		logger.Error("生成管理员Token失败", zap.Error(err))
		return nil, "", common.ServerErr.WithErr(err)
	}

	return commonAdmin, token, nil
}

// Logout 登出
func (a *AdminDomainService) Logout(ctx context.Context, adminId uint64, token string) error {
	expireDuration, err := jwt.GetTokenRemainingTime(token)
	if err != nil {
		return err
	}

	_ = a.adminRedisRepo.AddAdminTokenToBlackList(ctx, token, expireDuration)
	_ = a.adminRedisRepo.ClearAdminAllOnlineToken(ctx, adminId)

	logger.Info("管理员登出成功", zap.Uint64("adminId", adminId))
	return nil
}

// FindAdminByID 根据ID查询 → 先走缓存（你写的正确）
func (a *AdminDomainService) FindAdminByID(ctx context.Context, adminId uint64) (*do.AdminDO, error) {
	// 1. 查缓存
	adminModel, err := a.adminRedisRepo.GetAdminFromRedis(ctx, adminId)
	if err == nil {
		if adminModel == nil {
			return nil, common.UserNotFoundErr
		}
		return convert.AdminModelToDO(adminModel), nil
	}

	// 2. 降级查库
	adminModel, err = a.adminMysqlRepo.FindAdminByID(ctx, adminId)
	if err != nil {
		_ = a.adminRedisRepo.SetAdminToRedis(ctx, adminId, nil)
		return nil, err
	}

	// 3. 写缓存
	_ = a.adminRedisRepo.SetAdminToRedis(ctx, adminId, adminModel)
	return convert.AdminModelToDO(adminModel), nil
}

// UpdateAdminInfo 更新信息
func (a *AdminDomainService) UpdateAdminInfo(ctx context.Context, adminId uint64, updateData *do.AdminDO) error {
	// 走缓存查询
	adminDO, err := a.FindAdminByID(ctx, adminId)
	if err != nil {
		return err
	}

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

	// 更新数据库
	adminModel := convert.AdminDOToModel(adminDO)
	err = a.adminMysqlRepo.UpdateAdminInfo(ctx, adminModel)
	if err != nil {
		return err
	}

	// 删除缓存（关键）
	_ = a.adminRedisRepo.DelAdminFromRedis(ctx, adminId)

	return nil
}

// ===================== 【重点修复】FindAdminByName 增加完整缓存逻辑 =====================
func (a *AdminDomainService) FindAdminByName(ctx context.Context, adminName string) (*do.AdminDO, error) {
	// 1. 先从 Redis 根据用户名查（你必须实现这个 repo 方法）
	adminModel, err := a.adminRedisRepo.GetAdminByNameFromRedis(ctx, adminName)
	if err == nil {
		if adminModel == nil {
			return nil, common.UserNotFoundErr
		}
		return convert.AdminModelToDO(adminModel), nil
	}

	// 2. Redis 异常/不存在 → 查询数据库
	adminModel, err = a.adminMysqlRepo.FindAdminByName(ctx, adminName)
	if err != nil {
		// 缓存空值，防止穿透
		_ = a.adminRedisRepo.SetAdminByNameToRedis(ctx, adminName, nil)
		return nil, err
	}

	// 3. 写入缓存
	_ = a.adminRedisRepo.SetAdminByNameToRedis(ctx, adminName, adminModel)
	return convert.AdminModelToDO(adminModel), nil
}
