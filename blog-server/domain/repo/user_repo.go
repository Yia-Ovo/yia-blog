/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:21:26
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-16 02:29:47
 * @FilePath: \yia-blog\blog-server\domain\repo\user_repo.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package repo

import "blog-server/domain/do"

type UserRepo interface {
	CreateUser(userDO *do.UserDO) error
	FindUserByNickname(nickName string) (*do.UserDO, error)
	FindUserByID(id int64) (*do.UserDO, error)
	UpdateUserInfo(userDO *do.UserDO) error
}
