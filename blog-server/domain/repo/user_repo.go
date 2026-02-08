/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-02-02 02:21:26
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-09 00:37:59
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
}
