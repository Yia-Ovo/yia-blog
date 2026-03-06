/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-23 06:14:49
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-02-19 04:15:58
 * @FilePath: \yia-blog\blog-server\common\errno.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
package common

type Errno struct {
	Code   int
	Msg    string // 对外展示的友好提示
	ErrMsg string // 内部详细错误信息
}

func (err Errno) Error() string {
	return err.Msg
}

func (err Errno) WithMsg(msg string) Errno {
	err.Msg = err.Msg + "," + msg
	return err
}

func (err Errno) WithErr(rawErr error) Errno {
	var msg string
	if rawErr != nil {
		msg = rawErr.Error()
	}
	err.ErrMsg = err.Msg + "," + msg
	return err
}

func (err Errno) IsOk() bool {
	return err.Code == 200
}

var (
	// 通用基础错误
	OK            = Errno{Code: 200, Msg: "OK"}
	ServerErr     = Errno{Code: 500, Msg: "Internal Server Error"}
	ParamErr      = Errno{Code: 400, Msg: "Param Error"}
	AuthErr       = Errno{Code: 401, Msg: "Auth Error"}
	PermissionErr = Errno{Code: 403, Msg: "Permission Error"}

	// 中间件/存储层错误
	DatabaseErr = Errno{Code: 10000, Msg: "Database Error"}
	RedisErr    = Errno{Code: 10001, Msg: "Redis Error"}

	// 用户基础错误
	UserNotFoundErr = Errno{Code: 11001, Msg: "User Not Found"}
	UserExistErr    = Errno{Code: 11002, Msg: "User Already Exist"}

	// 登录专属标准化错误（新增）
	LoginParamEmptyErr    = Errno{Code: 11003, Msg: "Login param empty"}  // 登录核心参数为空（用户名/密码）
	LoginPasswordErr      = Errno{Code: 11004, Msg: "Password error"}     // 密码错误
	LoginFreqLimitErr     = Errno{Code: 11005, Msg: "Login too frequent"} // 登录频率超限（防暴力破解）
	LoginAccountLockedErr = Errno{Code: 11006, Msg: "Account locked"}     // 账号被锁定（密码多次错误）
	LoginTokenExpiredErr  = Errno{Code: 11007, Msg: "Token expired"}      // 登录态token过期
	LoginTokenInvalidErr  = Errno{Code: 11008, Msg: "Token invalid"}      // 登录态token无效（伪造/篡改）
	LoginNotLoginErr      = Errno{Code: 11009, Msg: "Not login"}          // 未登录（需要登录才能操作）
	LoginCaptchaErr       = Errno{Code: 11010, Msg: "Captcha error"}      // 验证码错误/过期（可选）
)
