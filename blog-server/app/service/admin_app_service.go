/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-29 16:00:00
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-08-12 10:28:50
 * @FilePath: \yia-blog\blog-server\app\service\admin_app_service.go
 * @Description: 管理员应用服务 - 业务编排、缓存处理、流程调度
 * Copyright (c) 2026 by Yia 2594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/jwt"
	"blog-server/common/utils/logger"
	"blog-server/common/utils/password"
	"blog-server/common/utils/ptr"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/domain/service"
	"context"
	"errors"
	"regexp"

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

// CreateAdmin 创建管理员
func (s *AdminAppService) CreateAdmin(
	ctx context.Context,
	req *dto.AdminCreateReq,
) (*dto.AdminInfoData, error) {
	if req == nil {
		logger.Warn("[APP] 创建管理员参数为空")
		return nil, common.ParamErr.WithMsg("参数不能为空")
	}
	if req.Role == nil {
		return nil, common.ParamErr.WithMsg("角色不能为空")
	}

	// 密码格式校验
	if msg := password.CheckPasswordFormat("", req.Password); msg != "" {
		logger.Warn("[APP] 密码格式错误", zap.String("reason", msg))
		return nil, common.UserNewPasswordInvalidErr.WithMsg(msg)
	}

	// 执行创建
	adminDO, err := s.adminDomainService.CreateAdmin(ctx, req.AdminName, req.Password, ptr.Val(req.Role))
	if err != nil {
		logger.Error("[APP] 创建管理员失败", zap.Error(err))
		return nil, err.(common.Errno).WithMsg("创建失败")
	}

	logger.Info("[APP] 创建管理员成功", zap.String("name", req.AdminName))
	return convert.AdminDOToAdminInfoData(adminDO), nil
}

// Login 管理员登录
func (s *AdminAppService) Login(ctx context.Context, req *dto.AdminLoginReq) (*dto.AdminLoginResp, error) {
	if req == nil {
		logger.Warn("[APP] 管理员登录参数为空")
		return nil, common.LoginParamEmptyErr.WithMsg("请求参数不能为空")
	}
	if req.AdminName == "" {
		logger.Warn("[APP] 登录账号为空")
		return nil, common.LoginParamEmptyErr.WithMsg("账号不能为空")
	}
	if req.Password == "" {
		logger.Warn("[APP] 登录密码为空")
		return nil, common.LoginParamEmptyErr.WithMsg("密码不能为空")
	}

	// 执行登录
	adminDO, err := s.adminDomainService.Login(ctx, req.AdminName, req.Password)
	if err != nil {
		logger.Error("[APP] 管理员登录验证失败", zap.Error(err))
		return nil, err.(common.Errno).WithMsg("登录失败，账号或密码不正确")
	}
	if adminDO == nil || adminDO.ID == 0 {
		logger.Error("[APP] 管理员登录成功但信息为空")
		return nil, common.LoginFailedErr.WithMsg("用户数据异常")
	}

	// 清理缓存
	_ = s.adminRedisRepo.DelAdminFromRedis(ctx, adminDO.ID)

	// 生成 Token
	loginDTO := convert.AdminDOToAdminBaseDTO(adminDO)
	token, err := jwt.GenerateAdminToken(&loginDTO)
	if err != nil {
		logger.Error("[APP] 生成管理员 Token 失败", zap.Error(err))
		return nil, common.ServerErr.WithMsg("生成令牌失败").WithErr(err)
	}

	// 写入在线缓存
	expireDur, errNo := jwt.GetTokenRemainingTime(token)
	if !errNo.IsOk() {
		logger.Warn("[APP] 获取管理员Token过期时间失败", zap.String("err", errNo.Error()))
		return nil, common.ServerErr.WithMsg("服务异常，登录失败")
	}

	err = s.adminRedisRepo.SetAdminOnlineToken(ctx, adminDO.ID, token, expireDur)
	if err != nil {
		logger.Warn("[APP] 写入管理员在线Token失败", zap.Uint64("admin_id", adminDO.ID), zap.Error(err))
		return nil, common.ServerErr.WithMsg("服务异常，登录失败").WithErr(err)
	}
	// 过期时间展示
	expireShow, showErr := jwt.GetTokenExpireAt(token)
	if !showErr.IsOk() {
		expireShow = ""
	}

	logger.Info("[APP] 管理员登录成功", zap.Uint64("id", adminDO.ID))
	return &dto.AdminLoginResp{
		Token:  token,
		Expire: expireShow,
		Data:   loginDTO,
	}, nil
}

// Logout 登出
func (s *AdminAppService) Logout(ctx context.Context, adminID uint64, token string) error {
	if token == "" {
		logger.Warn("[APP] 登出token为空")
		return common.TokenEmptyErr.WithMsg("令牌不能为空")
	}

	expire, errNo := jwt.GetTokenRemainingTime(token)
	if errNo.IsOk() {
		if err := s.adminRedisRepo.AddAdminTokenToBlackList(ctx, token, expire); err != nil {
			logger.Warn("[APP] 写入黑名单失败", zap.Uint64("admin_id", adminID), zap.Error(err))
			return common.ServerErr.WithMsg("登出失败").WithErr(err)
		}
	} else {
		logger.Warn("[APP] 获取Token剩余时间失败，跳过黑名单写入", zap.String("err", errNo.Error()))
	}

	if err := s.adminRedisRepo.ClearAdminAllOnlineToken(ctx, adminID); err != nil {
		logger.Warn("[APP] 清除管理员在线Token失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		return common.ServerErr.WithMsg("登出失败").WithErr(err)
	}

	logger.Info("[APP] 管理员登出成功", zap.Uint64("id", adminID))
	return nil
}

// GetAdminInfo 获取信息（缓存优先）
func (s *AdminAppService) GetAdminInfo(ctx context.Context, id uint64) (*dto.AdminInfoData, error) {
	if id <= 0 {
		logger.Warn("[APP] 管理员ID非法")
		return nil, common.ParamErr.WithMsg("ID非法")
	}

	// 查缓存
	model, err := s.adminRedisRepo.GetAdminFromRedis(ctx, id)
	if err == nil {
		if model == nil {
			logger.Warn("[APP] 缓存管理员不存在")
			return nil, common.AdminNotFoundErr.WithMsg("管理员不存在")
		}
		do := convert.AdminModelToDO(model)
		return convert.AdminDOToAdminInfoData(do), nil
	}

	// 查库
	logger.Info("[APP] 管理员缓存未命中，查询数据库")
	adminDO, err := s.adminDomainService.FindAdminByID(ctx, id)
	if err != nil {
		if errors.Is(err, common.AdminNotFoundErr) {
			_ = s.adminRedisRepo.SetAdminToRedis(ctx, id, nil)
			return nil, err
		}
		logger.Error("[APP] 查询管理员失败", zap.Error(err))
		return nil, err.(common.Errno).WithMsg("获取信息失败")
	}

	// 回写缓存
	_ = s.adminRedisRepo.SetAdminToRedis(ctx, id, convert.AdminDOToModel(adminDO))

	return convert.AdminDOToAdminInfoData(adminDO), nil
}

// UpdatePWD 修改密码
func (s *AdminAppService) UpdatePWD(ctx context.Context, id uint64, req *dto.AdminUpdatePWDReq) error {
	if req == nil {
		logger.Warn("[APP] 修改密码参数为空")
		return common.UserPwdParamEmptyErr.WithMsg("参数不能为空")
	}

	if msg := password.CheckPasswordFormat(req.OldPassword, req.NewPassword); msg != "" {
		logger.Warn("[APP] 新密码格式错误", zap.String("reason", msg))
		return common.UserNewPasswordInvalidErr.WithMsg(msg)
	}

	err := s.adminDomainService.UpdatePWD(ctx, id, req.NewPassword, req.OldPassword)
	if err != nil {
		logger.Error("[APP] 修改密码失败", zap.Error(err))
		return err.(common.Errno).WithMsg("修改失败，请检查原密码")
	}

	_ = s.adminRedisRepo.DelAdminFromRedis(ctx, id)

	// 清除在线Token
	if err := s.adminRedisRepo.ClearAdminAllOnlineToken(ctx, id); err != nil {
		logger.Warn("[APP] 清除管理员在线Token失败", zap.Uint64("admin_id", id), zap.Error(err))
	}

	logger.Info("[APP] 管理员密码修改成功", zap.Uint64("id", id))
	return nil
}

// UpdateAdminRole 变更管理员角色。
// operatorID 来自鉴权态（操作者），req.ID 是被操作对象——两个 id 不能混用。
// 权限规则全部在 Domain 校验，这里只做编排：调用领域服务 + 清理目标的缓存与在线令牌。
func (s *AdminAppService) UpdateAdminRole(ctx context.Context, operatorID uint64, req *dto.AdminUpdateRoleReq) error {
	if req == nil {
		logger.Warn("[APP] 修改管理员角色参数为空")
		return common.ParamErr.WithMsg("参数不能为空")
	}
	// 不能用 ptr.Val 兜底：nil 会被压成 0，而 0 正是超管角色，等于把"没传"变成"提升为超管"
	if req.Role == nil {
		logger.Warn("[APP] 修改管理员角色未指定角色值")
		return common.ParamErr.WithMsg("角色不能为空")
	}

	changed, err := s.adminDomainService.UpdateAdminRole(ctx, operatorID, req.ID, *req.Role)
	if err != nil {
		logger.Error("[APP] 修改管理员角色失败", zap.Error(err))
		return err.(common.Errno).WithMsg("修改失败")
	}
	if !changed {
		return nil
	}

	// 角色已变，清目标的信息缓存与在线令牌：Repo 侧已递增 token_version 使旧 token 失效，
	// 这里再清缓存，避免 GetAdminInfo 在 TTL 内继续返回旧 role。
	_ = s.adminRedisRepo.DelAdminFromRedis(ctx, req.ID)
	if err := s.adminRedisRepo.ClearAdminAllOnlineToken(ctx, req.ID); err != nil {
		logger.Warn("[APP] 清除管理员在线Token失败", zap.Uint64("admin_id", req.ID), zap.Error(err))
	}
	logger.Info("[APP] 管理员角色更新成功", zap.Uint64("operator_id", operatorID), zap.Uint64("admin_id", req.ID))
	return nil
}

// UpdateAdminInfo 更新信息
func (s *AdminAppService) UpdateAdminInfo(ctx context.Context, id uint64, req *dto.AdminUpdateInfoReq) (*dto.AdminInfoData, error) {
	if id <= 0 || req == nil {
		logger.Warn("[APP] 更新管理员参数非法")
		return nil, common.ParamErr.WithMsg("请求参数非法")
	}

	// 邮箱校验
	var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if req.Email != nil && *req.Email != "" && !emailRegex.MatchString(*req.Email) {
		return nil, common.ParamErr.WithMsg("邮箱格式不正确")
	}

	updateData := &do.AdminDO{
		AdminName: ptr.Val(req.AdminName),
		Avatar:    ptr.Val(req.Avatar),
		Email:     ptr.Val(req.Email),
		Remark:    ptr.Val(req.Remark),
	}

	err := s.adminDomainService.UpdateAdminInfo(ctx, id, updateData)
	if err != nil {
		logger.Error("[APP] 更新管理员信息失败", zap.Error(err), zap.Uint64("id", id))
		return nil, err.(common.Errno).WithMsg("更新失败")
	} else {
		_ = s.adminRedisRepo.DelAdminFromRedis(ctx, id)
		logger.Info("[APP] 管理员信息更新成功，缓存已清理", zap.Uint64("id", id))

	}
	return s.GetAdminInfo(ctx, id)
}

// FindAdminByName 按名称查询
func (s *AdminAppService) FindAdminByName(ctx context.Context, req *dto.AdminFindInfoReq) (*dto.AdminInfoData, error) {
	if req == nil || req.AdminName == "" {
		logger.Warn("[APP] 查询管理员参数为空")
		return nil, common.ParamErr.WithMsg("查询名称不能为空")
	}

	adminDO, err := s.adminDomainService.FindAdminByName(ctx, req.AdminName)
	if err != nil {
		logger.Error("[APP] 查询管理员失败", zap.Error(err))
		return nil, err.(common.Errno).WithMsg("查询异常")
	}

	return convert.AdminDOToAdminInfoData(adminDO), nil
}

// ListAdmin 列表
func (s *AdminAppService) ListAdmin(ctx context.Context, req *dto.AdminListReq) (*dto.AdminListResp, error) {
	doList, total, err := s.adminDomainService.ListAdmin(ctx, req.Page, req.PageSize)
	if err != nil {
		logger.Error("[APP] 获取管理员列表失败", zap.Error(err))
		return nil, err.(common.Errno).WithMsg("获取列表失败")
	}

	list := make([]dto.AdminInfoData, 0, len(doList))
	for _, item := range doList {
		list = append(list, *convert.AdminDOToAdminInfoData(item))
	}

	return &dto.AdminListResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// UpdateAdminStatus 更新状态
func (s *AdminAppService) UpdateAdminStatus(ctx context.Context, operatorId uint64, adminID uint64, status int) error {
	err := s.adminDomainService.UpdateAdminStatus(ctx, operatorId, adminID, status)
	if err != nil {
		logger.Error("[APP] 更新管理员状态失败", zap.Error(err))
		return err.(common.Errno).WithMsg("状态更新失败")
	}

	_ = s.adminRedisRepo.DelAdminFromRedis(ctx, adminID)
	if err := s.adminRedisRepo.ClearAdminAllOnlineToken(ctx, adminID); err != nil {
		logger.Warn("[APP] 清除管理员在线Token失败", zap.Uint64("admin_id", adminID), zap.Error(err))
	}
	logger.Info("[APP] 管理员状态更新成功", zap.Uint64("id", adminID))
	return nil
}

// ResetAdminPWD 重置密码
func (s *AdminAppService) ResetAdminPWD(ctx context.Context, operatorID uint64, adminID uint64, newPwd string) error {
	if newPwd == "" {
		return common.ParamErr.WithMsg("新密码不能为空")
	}
	err := s.adminDomainService.ResetAdminPWD(ctx, operatorID, adminID, newPwd)
	if err != nil {
		logger.Error("[APP] 重置密码失败", zap.Error(err))
		return err.(common.Errno).WithMsg("重置失败")
	}

	_ = s.adminRedisRepo.DelAdminFromRedis(ctx, adminID)
	if err := s.adminRedisRepo.ClearAdminAllOnlineToken(ctx, adminID); err != nil {
		logger.Warn("[APP] 清除管理员在线Token失败", zap.Uint64("admin_id", adminID), zap.Error(err))
	}
	logger.Info("[APP] 管理员密码重置成功", zap.Uint64("id", adminID))
	return nil
}

// 工具函数
