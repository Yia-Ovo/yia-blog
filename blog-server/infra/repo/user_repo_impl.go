/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-01 04:59:06
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-02 05:52:18
 * @FilePath: \yia-blog\blog-server\infra\repo\user_repo_impl.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import (
	"blog-server/common/utils/logger"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	"blog-server/infra/model"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type userRepoImpl struct {
	db *gorm.DB
}

// 私有结构体，实现domain/repo中的UserRepo接口
func NewUserRepoImpl(db *gorm.DB) repo.UserRepo {
	return &userRepoImpl{db: db}
}

// 初始化用户仓库：给domain/repo的全局变量赋值，让上层能调用
func InitUserRepo(db *gorm.DB) {
	repo.UserRepoIns = NewUserRepoImpl(db)
}

// 实现UserRepo接口的Create方法：创建用户（DO转Model+写入数据库）
func (r *userRepoImpl) CreateUser(userDO *do.UserDO) error {
	// 领域对象DO转换成数据库实体Model
	userModel := &model.UserModel{
		NickName: userDO.NickName,
		Password: userDO.Password,
		Avatar:   userDO.Avatar,
		Status:   userDO.Status,
	}
	// Create inserts value, returning the inserted data's primary key in value's id
	// 把userModel这个结构体实例的字段值，插入到对应的数据库表中（GORM 会根据model.UserModel的结构体标签 / 名称映射到对应的表，
	// GORM 会在插入成功后，自动把数据库生成的主键 ID、创建时间（gorm.Model 的 CreatedAt） 等字段回写到这个指针对象中
	return r.db.Debug().Create(userModel).Error
}

func (r *userRepoImpl) FindUserByNickName(nickName string) (*do.UserDO, error) {
	var userModel model.UserModel
	// 查询一条记录，匹配第一个找到的字段
	if err := r.db.Debug().Where("nick_name = ?", nickName).First(&userModel).Error; err != nil {
		logger.Error("func FindByNickName error", zap.Error(err))
		return nil, err
	}
	// 把数据库实体Model转换成领域对象DO
	userDO := &do.UserDO{
		ID:        int64(userModel.ID),
		NickName:  userModel.NickName,
		Password:  userModel.Password,
		Avatar:    userModel.Avatar,
		Status:    userModel.Status,
		CreateAt:  userModel.CreateAt,
		UpdateAt:  userModel.UpdateAt,
		IsDeleted: userModel.DeletedAt.Valid,
	}

	// 最后登录时间：Model中是指针，需判断是否为nil
	if userModel.LastLoginAt != nil {
		loginAt := time.Time(*userModel.LastLoginAt)
		userDO.LastLoginAt = &loginAt
	}
	return userDO, nil
}
