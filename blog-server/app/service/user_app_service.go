/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:58:44
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-03-13 23:51:28
 * @FilePath: \yia-blog\blog-server\app\service\user_app_service.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */

package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/service"
	"blog-server/infra/repo"
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

func (s *UserAppService) GetUserInfo(ctx context.Context, req *dto.UserGetInfoReq) (*dto.UserGetInfoRes, error) {
	// 1. 应用层参数校验（前置拦截非法参数，减少领域层压力）
	if req == nil {
		return nil, common.ParamErr.WithMsg("请求参数不能为空")
	}
	if req.ID <= 0 {
		return nil, common.ParamErr.WithMsg("用户ID不合法：" + strconv.Itoa(int(req.ID)))
	}
	// 2. 调用领域服务（核心逻辑交给领域层，应用层只做编排）
	// 注意：领域层也要传递ctx，控制数据库操作超时
	userDO, err := s.userDomainSrv.FindUserByID(ctx, req.ID)
	if err != nil {
		// 3. 错误分类处理
		switch err {
		// 匹配数据库无记录错误，封装为领域错误
		case gorm.ErrRecordNotFound:
			return nil, common.UserNotFoundErr.WithMsg("用户ID：" + strconv.Itoa(int(req.ID)) + " 不存在")
		// 匹配定义的数据库错误
		case common.DatabaseErr:
			return nil, common.DatabaseErr.WithErr(err)
		// 其他未知错误，封装为服务器错误
		default:
			return nil, common.ServerErr.WithErr(err)
		}
	}

	// 5. 组装DTO返回（领域对象 -> 传输对象）
	return &dto.UserGetInfoRes{
		Data: &dto.UserInfoData{
			ID:        userDO.ID,
			NickName:  userDO.NickName,
			Avatar:    userDO.Avatar,
			Gender:    userDO.Gender,
			Status:    userDO.Status,
			Signature: userDO.Signature,
			Email:     userDO.Email,
			CreateAt:  userDO.CreateAt,
		},
	}, nil

}

func (s *UserAppService) Login(req *dto.UserLoginReq) (*common.User, error) {
	// 1. 参数校验：使用标准化错误
	if req == nil {
		return nil, common.LoginParamEmptyErr.WithMsg("缺少请求参数")
	}
	if req.NickName == "" {
		return nil, common.LoginParamEmptyErr.WithMsg("用户名为空")
	}
	if req.Password == "" {
		return nil, common.LoginParamEmptyErr.WithMsg("密码为空")
	}
	// 2. 调用领域服务执行登录
	user, err := s.userDomainSrv.Login(req.NickName, req.Password)
	if err != nil {
		logger.Error("login failed", zap.Error(err))
		// 领域层可能返回 UserNotFoundErr/LoginPasswordErr 等，直接返回标准化错误
		return nil, err
	}

	return user, nil
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
			Gender:   userDO.Gender,
			Status:   userDO.Status,
			CreateAt: userDO.CreateAt,
		},
	}, nil
}

// 预编译邮箱正则（全局只编译一次，提升性能）
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (s *UserAppService) UpdateUserInfo(req *dto.UserUpdateInfoReq) (*dto.UserFindInfoRes, error) {
	// 1. 前置参数校验（核心：覆盖binding的不足+业务规则）
	if req == nil {
		return nil, common.ParamErr.WithMsg("更新用户信息请求参数不能为空")
	}

	// 校验ID：不仅非空，还要是正整数（覆盖binding:"required"的不足）
	if req.ID <= 0 {
		return nil, common.ParamErr.WithMsg("用户ID必须为正整数，当前值：" + strconv.FormatInt(req.ID, 10))
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
			return nil, common.UserNotFoundErr.WithMsg("用户ID：" + strconv.FormatInt(req.ID, 10))
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

	return &dto.UserFindInfoRes{
		Data: &dto.UserInfoData{
			ID:        userDO.ID,
			NickName:  userDO.NickName,
			Avatar:    userDO.Avatar,
			Status:    userDO.Status,
			CreateAt:  userDO.CreateAt,
			Gender:    userDO.Gender,
			Signature: userDO.Signature,
			Email:     userDO.Email,
		},
	}, nil
}
