/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-04-14 18:30:10
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-06-03 14:53:43
 * @FilePath: \yia-blog\blog-server\common\utils\convert\user_convert.go
 * @Description: 用户三层实体转换 Model<->DO<->DTO
 */
package convert

import (
	"blog-server/api/dto"
	"blog-server/domain/do"
	"blog-server/infra/model"
)

// Model ↔ DO 数据库模型和领域对象互转
func UserModelToDO(m *model.UserModel) *do.UserDO {
	if m == nil || m.ID == 0 {
		return nil
	}
	return &do.UserDO{
		ID:           m.ID,
		NickName:     m.NickName,
		Password:     m.Password,
		Status:       m.Status,
		TokenVersion: m.TokenVersion,
		LastLoginAt:  m.LastLoginAt,
		Avatar:       m.Avatar,
		Gender:       m.Gender,
		Signature:    m.Signature,
		Email:        m.Email,
		CreateAt:     m.CreateAt,
		UpdateAt:     m.UpdateAt,
	}
}

func UserDOToModel(d *do.UserDO) *model.UserModel {
	if d == nil {
		return &model.UserModel{}
	}
	return &model.UserModel{
		ID:           d.ID,
		NickName:     d.NickName,
		Password:     d.Password,
		Status:       d.Status,
		TokenVersion: d.TokenVersion,
		LastLoginAt:  d.LastLoginAt,
		Avatar:       d.Avatar,
		Gender:       d.Gender,
		Signature:    d.Signature,
		Email:        d.Email,
		CreateAt:     d.CreateAt,
		UpdateAt:     d.UpdateAt,
	}
}

// UserDOToBaseDTO 列表/简易查询：只返回基础信息
func UserDOToBaseDTO(do *do.UserDO) dto.UserBaseDTO {
	if do == nil {
		return dto.UserBaseDTO{}
	}
	return dto.UserBaseDTO{
		ID:           do.ID,
		NickName:     do.NickName,
		Avatar:       do.Avatar,
		Status:       do.Status,
		TokenVersion: do.TokenVersion,
		Gender:       do.Gender,
		Signature:    do.Signature,
		Email:        do.Email,
	}
}

// UserDOToUserInfoData 用户详情全量信息
func UserDOToUserInfoData(do *do.UserDO) *dto.UserInfoData {
	if do == nil {
		return nil
	}
	return &dto.UserInfoData{
		UserBaseDTO: UserDOToBaseDTO(do),
		LastLoginAt: do.LastLoginAt,
		CreateAt:    do.CreateAt,
		UpdateAt:    do.UpdateAt,
	}
}

// UserDOToLoginResp 登录专用：拼接token + 用户基础信息
func UserDOToLoginResp(do *do.UserDO, token string) *dto.LoginResp {
	if do == nil {
		return nil
	}
	return &dto.LoginResp{
		Token: token,
		Data:  UserDOToBaseDTO(do),
	}
}
