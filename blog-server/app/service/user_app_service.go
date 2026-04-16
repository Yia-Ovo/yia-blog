/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:58:44
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-16 09:55:43
 * @FilePath: \yia-blog\blog-server\app\service\user_app_service.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
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

// UserAppService 用户应用服务：业务编排，串联领域层
type UserAppService struct {
	// 注入领域服务实例
	userDomainSrv *service.UserDomainService
}

// NewUserAppService 创建应用服务实例（初始化链路：DB → 仓库实现 → 领域服务 → 应用服务）

// func NewUserAppService(db *gorm.DB, redisClient *redis.Client, conf *config.GlobalConfig) *UserAppService {
// 	// MySQL数据仓储
// 	userMysqlRepo := repo.NewUserMySQLRepo(db)
// 	// Redis缓存仓储
// 	userRedisRepo := repo.NewUserRedisRepo(redisClient, conf)

// 	// 注入领域服务
// 	userDomainSrv := service.NewUserDomainService(userMysqlRepo, userRedisRepo)

// 	return &UserAppService{
// 		userDomainSrv: userDomainSrv,
// 	}
// }

func NewUserAppService(
	// 全部传领域接口（IOC 从外面注入）
	userMysqlRepo repo.UserMysqlRepo,
	userRedisRepo repo.UserRedisRepo,
	conf *config.GlobalConfig,
) *UserAppService {
	// 注入领域服务
	userDomainSrv := service.NewUserDomainService(
		userMysqlRepo, // 传入接口
		userRedisRepo, // 传入接口
	)

	// 返回复用给 app service 使用
	return &UserAppService{
		userDomainSrv: userDomainSrv,
	}
}

func (s *UserAppService) UpdatePWD(ctx context.Context, id uint64, req *dto.UserUpdatePWDReq) error {
	if req.OldPassword == "" {
		return common.UserPwdParamEmptyErr.WithMsg("旧密码为空")
	}
	if req.NewPassword == "" {
		return common.UserPwdParamEmptyErr.WithMsg("新密码为空")
	}

	if err := password.CheckPasswordFormat(req.NewPassword); err != nil {
		return common.UserNewPasswordInvalidErr.WithMsg(err.Error())
	}

	err := s.userDomainSrv.UpdatePWD(ctx, id, req.NewPassword, req.OldPassword)
	if err != nil {
		logger.Warn("修改密码业务失败", zap.Uint64("userId", id), zap.String("err", err.Error()))
		return err
	}
	return nil
}

func (s *UserAppService) GetUserInfo(ctx context.Context, id uint64) (*dto.UserInfoData, error) {

	// 1. 调用领域服务（核心逻辑交给领域层，应用层只做编排）
	// 注意：领域层也要传递ctx，控制数据库操作超时
	userDO, err := s.userDomainSrv.FindUserByID(ctx, id)
	if err != nil {
		// 2. 错误分类处理
		switch err {
		// 匹配数据库无记录错误，封装为领域错误
		case gorm.ErrRecordNotFound:
			return nil, common.UserNotFoundErr.WithMsg("用户ID：" + strconv.Itoa(int(id)) + " 不存在")
		// 匹配定义的数据库错误
		case common.DatabaseErr:
			return nil, common.DatabaseErr.WithErr(err)
		// 其他未知错误，封装为服务器错误
		default:
			return nil, common.ServerErr.WithErr(err)
		}
	}

	// 3. 组装DTO返回（领域对象 -> 传输对象）
	data := &dto.UserInfoData{
		ID:        userDO.ID,
		NickName:  userDO.NickName,
		Avatar:    userDO.Avatar,
		Gender:    userDO.Gender,
		Status:    userDO.Status,
		Signature: userDO.Signature,
		Email:     userDO.Email,
		CreateAt:  userDO.CreateAt,
	}
	return data, nil

}

func (s *UserAppService) Login(ctx context.Context, req *dto.UserLoginReq) (*convert.User, string, error) {
	// 1. 参数校验：使用标准化错误
	if req == nil {
		return nil, "", common.LoginParamEmptyErr
	}
	if req.NickName == "" {
		return nil, "", common.LoginParamEmptyErr
	}
	if req.Password == "" {
		return nil, "", common.LoginParamEmptyErr
	}
	// 2. 调用领域服务(ctx + 核心参数)
	user, token, err := s.userDomainSrv.Login(ctx, req.NickName, req.Password)
	if err != nil {
		logger.Error("用户登录失败", zap.String("nickname", req.NickName), zap.Error(err))
		// 直接透传标准化错误码
		return nil, "", err
	}
	// 3. 登录成功，返回 用户信息+Token 给api层
	logger.Info("用户登录成功", zap.String("nickname", req.NickName), zap.Uint64("user_id", user.UserID))
	return user, token, nil
}

func (s *UserAppService) Logout(ctx context.Context, userId uint64, token string) error {
	// 1. 参数校验：使用标准化错误
	if token == "" {
		return common.TokenEmptyErr
	}
	// 2. 调用领域服务（失效当前登录Token）
	err := s.userDomainSrv.Logout(ctx, userId, token)
	if err != nil {
		logger.Error("用户登出失败",
			zap.Uint64("user_id", userId),
			zap.Error(err))
		return err
	}
	// 3. 登出成功，记录日志
	logger.Info("用户登出成功", zap.Uint64("user_id", userId))
	return nil
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
func (s *UserAppService) FindUserByNickname(req *dto.UserFindInfoReq) (*dto.UserInfoData, error) {
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
	data := &dto.UserInfoData{
		ID:       userDO.ID,
		NickName: userDO.NickName,
		Avatar:   userDO.Avatar,
		Gender:   userDO.Gender,
		Status:   userDO.Status,
		CreateAt: userDO.CreateAt,
	}

	return data, nil
}

// 预编译邮箱正则（全局只编译一次，提升性能）
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (s *UserAppService) UpdateUserInfo(req *dto.UserUpdateInfoReq) (*dto.UserInfoData, error) {
	// 1. 前置参数校验（核心：覆盖binding的不足+业务规则）
	if req == nil {
		return nil, common.ParamErr.WithMsg("更新用户信息请求参数不能为空")
	}

	// 校验ID：不仅非空，还要是正整数（覆盖binding:"required"的不足）
	if req.ID <= 0 {
		return nil, common.ParamErr.WithMsg("用户ID必须为正整数，当前值：" + strconv.FormatUint(req.ID, 10))
	}

	// 校验Gender：指针类型，非空时要校验值的合法性（比如性别只能是 0-未知 1-男 2-女）
	genderVal := req.Gender
	if genderVal < 0 || genderVal > 2 {
		return nil, common.ParamErr.WithMsg("性别值不合法，只能是0(未知)/1(男)/2(女)，当前值：" + strconv.Itoa(genderVal))
	}

	// 校验Email：非空时校验格式合法性
	if req.Email != "" && !emailRegex.MatchString(req.Email) {
		return nil, common.ParamErr.WithMsg("邮箱格式不合法，当前值：" + req.Email)
	}

	// 2. 原有逻辑：整合更新数据
	updateData := &do.UserDO{
		NickName:  req.NickName,
		Avatar:    req.Avatar,
		Gender:    req.Gender,
		Signature: req.Signature,
		Email:     req.Email,
	}

	// 3. 调用领域服务+错误处理
	err := s.userDomainSrv.UpdateUserInfo(req.ID, updateData)
	if err != nil {
		switch {
		case err == gorm.ErrRecordNotFound:
			return nil, common.UserNotFoundErr.WithMsg("用户ID：" + strconv.FormatUint(req.ID, 10))
		default:
			return nil, common.DatabaseErr.WithErr(err).WithMsg("更新用户信息失败")
		}
	}
	ctx := context.Background()
	// 4. 查询最新数据并返回
	userDO, err := s.userDomainSrv.FindUserByID(ctx, req.ID)
	if err != nil {
		return nil, common.DatabaseErr.WithErr(err).WithMsg("查询更新后用户信息失败")
	}

	data := &dto.UserInfoData{
		ID:          userDO.ID,
		NickName:    userDO.NickName,
		Avatar:      userDO.Avatar,
		Status:      userDO.Status,
		Gender:      userDO.Gender,
		Signature:   userDO.Signature,
		Email:       userDO.Email,
		CreateAt:    userDO.CreateAt,
		LastLoginAt: userDO.LastLoginAt,
	}

	return data, nil
}
