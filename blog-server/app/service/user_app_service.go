/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-05-29 16:30:00
 * @LastEditors: Yia 1372594487@qq.com
 * @FilePath: \yia-blog\blog-server\app\service\user_app_service.go
 * @Description: 用户应用服务 - 业务编排、缓存处理、流程调度
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/utils/convert"
	"blog-server/common/utils/jwt"
	"blog-server/common/utils/logger"
	"blog-server/common/utils/password"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/domain/service"
	"context"
	"regexp"

	"go.uber.org/zap"
)

// UserAppService 应用服务：仅做流程编排 + 缓存处理
type UserAppService struct {
	userDomainService *service.UserDomainService // 领域服务（纯业务）
	userRedisRepo     repo.UserRedisRepo         // 缓存仓储（应用层编排）
}

// NewUserAppService 构造函数：依赖外部注入，无强耦合
func NewUserAppService(
	domainService *service.UserDomainService,
	redisRepo repo.UserRedisRepo,
) *UserAppService {
	return &UserAppService{
		userDomainService: domainService,
		userRedisRepo:     redisRepo,
	}
}

// Register 用户注册：调用领域服务
func (s *UserAppService) Register(ctx context.Context, req *dto.UserRegisterReq) error {
	if req == nil || req.NickName == "" || req.Password == "" {
		logger.Warn("用户注册参数为空")
		return common.ParamErr
	}

	// 1. 检查昵称是否存在
	_, err := s.userDomainService.FindUserByNickname(ctx, req.NickName)
	if err == nil {
		logger.Warn("用户昵称已存在", zap.String("nickname", req.NickName))
		return common.UserExistErr
	}

	// 2. 执行核心业务逻辑
	err = s.userDomainService.Register(ctx, req.NickName, req.Password)
	if err != nil {
		logger.Error("用户注册失败", zap.String("nickname", req.NickName), zap.Error(err))
		return err
	}

	logger.Info("用户注册成功", zap.String("nickname", req.NickName))
	return nil
}

// Login 用户登录：应用层处理 JWT + 缓存
func (s *UserAppService) Login(ctx context.Context, req *dto.UserLoginReq) (*dto.LoginUserDTO, string, error) {
	if req == nil || req.NickName == "" || req.Password == "" {
		logger.Warn("用户登录参数为空")
		return nil, "", common.LoginParamEmptyErr
	}

	// 1. 执行核心业务逻辑
	userDO, err := s.userDomainService.Login(ctx, req.NickName, req.Password)
	if err != nil {
		logger.Error("用户登录验证失败", zap.String("nickname", req.NickName), zap.Error(err))
		return nil, "", err
	}

	// 2. 应用层编排：清理旧缓存
	_ = s.userRedisRepo.DelUserFromRedis(ctx, userDO.ID)

	// 3. 应用层编排：生成 Token
	loginUser := convert.UserDOToLoginUserDTO(userDO)
	token, err := jwt.GenerateUserToken(loginUser)
	if err != nil {
		logger.Error("生成用户 Token 失败", zap.Uint64("user_id", userDO.ID), zap.Error(err))
		return nil, "", common.ServerErr
	}

	logger.Info("用户登录成功", zap.Uint64("user_id", userDO.ID), zap.String("nickname", req.NickName))
	return loginUser, token, nil
}

// Logout 用户登出：应用层处理 Redis 黑名单
func (s *UserAppService) Logout(ctx context.Context, userID uint64, token string) error {
	if token == "" {
		logger.Warn("用户登出 Token 为空")
		return common.TokenEmptyErr
	}

	// 1. 应用层编排：获取 Token 过期时间
	expire, _ := jwt.GetTokenRemainingTime(token)

	// 2. 应用层编排：加入黑名单
	_ = s.userRedisRepo.AddTokenToBlackList(ctx, token, expire)
	_ = s.userRedisRepo.ClearUserAllOnlineToken(ctx, userID)

	logger.Info("用户登出成功", zap.Uint64("user_id", userID))
	return nil
}

// GetUserInfo 获取用户信息：缓存优先
func (s *UserAppService) GetUserInfo(ctx context.Context, id uint64) (*dto.UserInfoData, error) {
	if id <= 0 {
		logger.Warn("获取用户信息参数非法")
		return nil, common.ParamErr
	}

	// 1. 优先读缓存
	userModel, err := s.userRedisRepo.GetUserFromRedis(ctx, id)
	if err == nil && userModel != nil {
		userDO := convert.UserModelToDO(userModel)
		return convert.UserDOToUserInfoDTO(userDO), nil
	}

	// 2. 缓存未命中，查询数据库
	logger.Info("用户信息缓存未命中，查询数据库", zap.Uint64("user_id", id))
	userDO, err := s.userDomainService.FindUserByID(ctx, id)
	if err != nil {
		// 这里只判断通用错误，不碰 gorm
		if err == common.UserNotFoundErr {
			logger.Warn("用户不存在", zap.Uint64("user_id", id))
			return nil, common.UserNotFoundErr
		}
		logger.Error("查询用户信息失败", zap.Uint64("user_id", id), zap.Error(err))
		return nil, common.ServerErr
	}

	// 3. 回写缓存
	_ = s.userRedisRepo.SetUserToRedis(ctx, id, convert.UserDOToModel(userDO))

	return convert.UserDOToUserInfoDTO(userDO), nil
}

// UpdatePWD 修改密码：应用层编排 + 清理缓存
func (s *UserAppService) UpdatePWD(ctx context.Context, id uint64, req *dto.UserUpdatePWDReq) error {
	if req.OldPassword == "" || req.NewPassword == "" {
		logger.Warn("修改密码参数为空")
		return common.UserPwdParamEmptyErr
	}
	if err := password.CheckPasswordFormat(req.NewPassword); err != nil {
		return common.UserNewPasswordInvalidErr
	}

	// 1. 执行核心业务逻辑
	err := s.userDomainService.UpdatePWD(ctx, id, req.NewPassword, req.OldPassword)
	if err != nil {
		logger.Error("修改密码失败", zap.Uint64("user_id", id), zap.Error(err))
		return err
	}

	// 2. 应用层编排：清理缓存
	_ = s.userRedisRepo.DelUserFromRedis(ctx, id)
	logger.Info("用户密码修改成功，缓存已清理", zap.Uint64("user_id", id))
	return nil
}

// UpdateUserInfo 更新用户信息：应用层编排 + 清理缓存
func (s *UserAppService) UpdateUserInfo(ctx context.Context, userID uint64, req *dto.UserUpdateInfoReq) (*dto.UserInfoData, error) {
	if userID <= 0 || req == nil {
		logger.Warn("更新用户信息参数非法")
		return nil, common.ParamErr
	}
	if req.Gender < 0 || req.Gender > 2 {
		return nil, common.ParamErr.WithMsg("性别不合法")
	}

	var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if req.Email != "" && !emailRegex.MatchString(req.Email) {
		return nil, common.ParamErr.WithMsg("邮箱格式错误")
	}

	// 1. 执行核心业务逻辑
	updateData := &do.UserDO{
		NickName:  req.NickName,
		Avatar:    req.Avatar,
		Gender:    req.Gender,
		Signature: req.Signature,
		Email:     req.Email,
	}
	err := s.userDomainService.UpdateUserInfo(ctx, userID, updateData)
	if err != nil {
		logger.Error("更新用户信息失败", zap.Uint64("user_id", userID), zap.Error(err))
		return nil, common.ServerErr
	}

	// 2. 应用层编排：清理缓存
	_ = s.userRedisRepo.DelUserFromRedis(ctx, userID)
	logger.Info("用户信息更新成功，缓存已清理", zap.Uint64("user_id", userID))

	// 3. 返回最新信息
	return s.GetUserInfo(ctx, userID)
}

// FindUserByNickname 根据昵称查询用户
func (s *UserAppService) FindUserByNickname(ctx context.Context, req *dto.UserFindInfoReq) (*dto.UserInfoData, error) {
	if req.NickName == "" {
		logger.Warn("查询用户昵称为空")
		return nil, common.ParamErr
	}

	userDO, err := s.userDomainService.FindUserByNickname(ctx, req.NickName)
	if err != nil {
		if err == common.UserNotFoundErr {
			logger.Warn("用户不存在", zap.String("nickname", req.NickName))
			return nil, common.UserNotFoundErr
		}
		logger.Error("查询用户失败", zap.String("nickname", req.NickName), zap.Error(err))
		return nil, common.ServerErr
	}

	return convert.UserDOToUserInfoDTO(userDO), nil
}
