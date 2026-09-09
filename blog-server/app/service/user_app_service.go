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
	if req == nil {
		logger.Warn("[APP] 用户注册参数为空")
		return common.ParamErr.WithMsg("注册参数不能为空")
	}

	// 密码格式校验
	if msg := password.CheckPasswordFormat("", req.Password); msg != "" {
		logger.Warn("[APP] 注册密码格式错误", zap.String("reason", msg))
		return common.UserNewPasswordInvalidErr.WithMsg(msg)
	}

	// 检查昵称是否存在
	_, err := s.userDomainService.FindUserByNickname(ctx, req.NickName)
	if err == nil {
		logger.Warn("[APP] 用户昵称已存在", zap.String("nickname", req.NickName))
		return common.UserExistErr.WithMsg("该昵称已被占用")
	}

	// 执行注册
	err = s.userDomainService.Register(ctx, req.NickName, req.Password)
	if err != nil {
		logger.Error("[APP] 用户注册失败", zap.String("nickname", req.NickName), zap.Error(err))
		return err.(common.Errno).WithMsg("注册失败，请稍后重试")
	}

	logger.Info("[APP] 用户注册成功", zap.String("nickname", req.NickName))
	return nil
}

// Login 用户登录：应用层处理 JWT + 缓存
func (s *UserAppService) Login(ctx context.Context, req *dto.UserLoginReq) (*dto.LoginResp, error) {
	if req == nil {
		logger.Warn("[APP] 用户登录参数为空")
		return nil, common.LoginParamEmptyErr.WithMsg("请求参数不能为空")
	}

	if req.NickName == "" {
		logger.Warn("[APP] 登录账号为空")
		return nil, common.LoginParamEmptyErr.WithMsg("账号不能为空")
	}
	if req.Password == "" {
		logger.Warn("[APP] 登录密码为空")
		return nil, common.LoginParamEmptyErr.WithMsg("密码不能为空")
	}

	// 执行登录逻辑
	userDO, err := s.userDomainService.Login(ctx, req.NickName, req.Password)
	if err != nil {
		logger.Error("[APP] 用户登录验证失败", zap.String("nickname", req.NickName), zap.Error(err))
		return nil, err.(common.Errno).WithMsg("登录失败，账号或密码不正确")
	}

	if userDO == nil || userDO.ID == 0 {
		logger.Error("[APP] 用户登录成功但信息为空", zap.String("nickname", req.NickName))
		return nil, common.LoginFailedErr.WithMsg("用户数据异常，登录失败")
	}

	// 清理缓存
	_ = s.userRedisRepo.DelUserFromRedis(ctx, userDO.ID)

	// 生成 Token
	loginUserDTO := convert.UserDOToBaseDTO(userDO)
	token, err := jwt.GenerateUserToken(&loginUserDTO)
	if err != nil {
		logger.Error("[APP] 生成用户 Token 失败", zap.Uint64("user_id", userDO.ID), zap.Error(err))
		return nil, common.ServerErr.WithMsg("服务异常，生成令牌失败").WithErr(err)
	}

	expireDur, errNo := jwt.GetTokenRemainingTime(token)
	if !errNo.IsOk() {
		logger.Warn("[APP] 获取Token过期时间失败", zap.String("err", errNo.Error()))
		return nil, common.ServerErr.WithMsg("服务异常，登录失败")
	}

	err = s.userRedisRepo.SetOnlineToken(ctx, userDO.ID, token, expireDur)
	if err != nil {
		logger.Warn("[APP] 写入在线Token失败", zap.Uint64("user_id", userDO.ID), zap.Error(err))
		return nil, common.ServerErr.WithMsg("服务异常，登录失败").WithErr(err)
	}

	expireShow, showErr := jwt.GetTokenExpireAt(token)
	if !showErr.IsOk() {
		logger.Warn("[APP] 获取前端过期展示时间失败", zap.String("err", showErr.Error()))
		expireShow = ""
	}
	logger.Info("[APP] 用户登录成功", zap.Uint64("user_id", userDO.ID), zap.String("nickname", req.NickName))
	return &dto.LoginResp{
		Token:  token,
		Expire: expireShow,
		Data:   loginUserDTO,
	}, nil
}

// Logout 用户登出
func (s *UserAppService) Logout(ctx context.Context, userID uint64, token string) error {
	if token == "" {
		logger.Warn("[APP] 用户登出 Token 为空")
		return common.TokenEmptyErr.WithMsg("令牌不能为空")
	}

	// 获取token的剩余有效期时间
	expire, errNo := jwt.GetTokenRemainingTime(token)
	if errNo.IsOk() {
		if err := s.userRedisRepo.AddTokenToBlackList(ctx, token, expire); err != nil {
			logger.Warn("[APP] 写入黑名单失败", zap.Uint64("user_id", userID), zap.Error(err))
			return common.ServerErr.WithMsg("登出失败").WithErr(err)
		}
	} else {
		logger.Warn("[APP] 获取Token剩余时间失败，跳过黑名单写入", zap.String("err", errNo.Error()))
	}

	if err := s.userRedisRepo.ClearUserAllOnlineToken(ctx, userID); err != nil {
		logger.Warn("[APP] 清除用户在线Token失败", zap.Uint64("user_id", userID), zap.Error(err))
		return common.ServerErr.WithMsg("登出失败").WithErr(err)
	}

	logger.Info("[APP] 用户登出成功", zap.Uint64("user_id", userID))
	return nil
}

// GetUserInfo 获取用户信息：缓存优先
func (s *UserAppService) GetUserInfo(ctx context.Context, id uint64) (*dto.UserInfoData, error) {
	if id <= 0 {
		logger.Warn("[APP] 获取用户信息参数非法")
		return nil, common.ParamErr.WithMsg("用户ID参数非法")
	}

	// 1. 查缓存
	userModel, err := s.userRedisRepo.GetUserFromRedis(ctx, id)

	if err == nil {
		// 命中【空值缓存】→ 用户不存在
		if userModel == nil {
			logger.Warn("[APP] 缓存用户不存在", zap.Uint64("user_id", id))
			return nil, common.UserNotFoundErr.WithMsg("用户不存在")
		}

		// 正常缓存
		userDO := convert.UserModelToDO(userModel)
		return convert.UserDOToUserInfoData(userDO), nil
	}

	// 2. 缓存未命中/异常 → 查数据库
	logger.Info("[APP] 用户信息缓存未命中，查询数据库", zap.Uint64("user_id", id))
	userDO, err := s.userDomainService.FindUserByID(ctx, id)
	if err != nil {
		logger.Error("[APP] 查询用户信息失败", zap.Uint64("user_id", id), zap.Error(err))
		return nil, err.(common.Errno).WithMsg("获取用户信息失败")
	}

	// 3. 回写缓存（正常用户/空值都会写）
	_ = s.userRedisRepo.SetUserToRedis(ctx, id, convert.UserDOToModel(userDO))

	return convert.UserDOToUserInfoData(userDO), nil
}

// UpdatePWD 修改密码
func (s *UserAppService) UpdatePWD(ctx context.Context, id uint64, req *dto.UserUpdatePWDReq) error {
	if req == nil {
		logger.Warn("[APP] 修改密码参数为空")
		return common.UserPwdParamEmptyErr.WithMsg("修改密码参数不能为空")
	}

	// 密码格式校验
	if msg := password.CheckPasswordFormat(req.OldPassword, req.NewPassword); msg != "" {
		logger.Warn("[APP] 新密码格式错误", zap.String("reason", msg))
		return common.UserNewPasswordInvalidErr.WithMsg(msg)
	}

	// 修改密码
	err := s.userDomainService.UpdatePWD(ctx, id, req.NewPassword, req.OldPassword)
	if err != nil {
		logger.Error("[APP] 修改密码失败", zap.Uint64("user_id", id), zap.Error(err))
		return err.(common.Errno).WithMsg("修改密码失败，请检查原密码")
	}

	// 清理缓存
	_ = s.userRedisRepo.DelUserFromRedis(ctx, id)

	// 清理在线Token
	if err := s.userRedisRepo.ClearUserAllOnlineToken(ctx, id); err != nil {
		logger.Warn("[APP] 清除用户在线Token失败", zap.Uint64("user_id", id), zap.Error(err))
	}

	logger.Info("[APP] 用户密码修改成功", zap.Uint64("user_id", id))
	return nil
}

// UpdateUserInfo 更新用户信息
func (s *UserAppService) UpdateUserInfo(ctx context.Context, userID uint64, req *dto.UserUpdateInfoReq) (*dto.UserInfoData, error) {
	if userID <= 0 || req == nil {
		logger.Warn("[APP] 更新用户信息参数非法")
		return nil, common.ParamErr.WithMsg("请求参数非法")
	}

	updateData := &do.UserInfoUpdate{
		NickName:  req.NickName,
		Avatar:    req.Avatar,
		Gender:    req.Gender,
		Signature: req.Signature,
		Email:     req.Email,
	}

	err := s.userDomainService.UpdateUserInfo(ctx, userID, updateData)
	if err != nil {
		logger.Error("[APP] 更新用户信息失败", zap.Uint64("user_id", userID), zap.Error(err))
		return nil, err
	}

	_ = s.userRedisRepo.DelUserFromRedis(ctx, userID)
	logger.Info("[APP] 用户信息更新成功，缓存已清理", zap.Uint64("user_id", userID))

	return s.GetUserInfo(ctx, userID)
}

// FindUserByNickname 根据昵称查询公开资料，完整资料由鉴权后的 GetUserInfo 返回。
func (s *UserAppService) FindUserByNickname(ctx context.Context, req *dto.UserFindInfoReq) (*dto.PublicUserInfo, error) {
	if req == nil {
		logger.Warn("[APP] 查询用户参数为空")
		return nil, common.ParamErr.WithMsg("查询参数不能为空")
	}

	if req.NickName == "" {
		logger.Warn("[APP] 查询用户昵称为空")
		return nil, common.ParamErr.WithMsg("查询昵称不能为空")
	}

	userDO, err := s.userDomainService.FindUserByNickname(ctx, req.NickName)
	if err != nil {
		logger.Error("[APP] 查询用户失败", zap.String("nickname", req.NickName), zap.Error(err))
		return nil, err.(common.Errno).WithMsg("用户信息查询异常")
	}

	return convert.UserDOToPublicUserInfo(userDO), nil
}
