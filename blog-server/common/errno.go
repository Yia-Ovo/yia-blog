/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-23 06:14:49
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-22 06:36:10
 * @FilePath: \yia-blog\blog-server\common\errno.go
 * @Description:
 *
 * Copyright (c) 2026 by Yia 1372594487@qq.com, All Rights Reserved.
 */
/*
 * @Author: Yia 1372594487@qq.com
 * @Date: 2026-01-23 06:14:49
 * @LastEditors: Yia 1372594487@qq.com
 * @LastEditTime: 2026-04-20 19:26:55
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
	UserNotFoundErr        = Errno{Code: 11001, Msg: "User Not Found"}
	UserExistErr           = Errno{Code: 11002, Msg: "User Already Exist"}
	UserNickNameTooLongErr = Errno{Code: 11011, Msg: "User nickname too long"}
	UserStatusInvalidErr   = Errno{Code: 11012, Msg: "User status invalid"}

	// 用户状态专属
	UserAccountDisabledErr = Errno{Code: 11019, Msg: "Account disabled"}         // 账号已禁用
	LoginInfoUpdateErr     = Errno{Code: 11020, Msg: "Update login info failed"} // 更新登录信息失败

	// 登录专属标准化错误
	LoginParamEmptyErr    = Errno{Code: 11003, Msg: "Login param empty"}  // 登录核心参数为空（用户名/密码）
	LoginPasswordErr      = Errno{Code: 11004, Msg: "Password error"}     // 密码错误
	LoginFreqLimitErr     = Errno{Code: 11005, Msg: "Login too frequent"} // 登录频率超限（防暴力破解）
	LoginAccountLockedErr = Errno{Code: 11006, Msg: "Account locked"}     // 账号被锁定（密码多次错误）
	LoginTokenExpiredErr  = Errno{Code: 11007, Msg: "Token expired"}      // 登录态token过期
	LoginTokenInvalidErr  = Errno{Code: 11008, Msg: "Token invalid"}      // 登录态token无效（伪造/篡改）
	LoginNotLoginErr      = Errno{Code: 11009, Msg: "Not login"}          // 未登录（需要登录才能操作）
	LoginCaptchaErr       = Errno{Code: 11010, Msg: "Captcha error"}      // 验证码错误/过期（可选）

	// 修改密码专属错误
	UserPwdParamEmptyErr      = Errno{Code: 11013, Msg: "Password param empty"}     // 修改密码参数为空
	UserOldPasswordErr        = Errno{Code: 11014, Msg: "Old password error"}       // 旧密码校验失败
	UserNewPasswordInvalidErr = Errno{Code: 11015, Msg: "New password invalid"}     // 新密码格式不合法
	UserPasswordSameError     = Errno{Code: 11016, Msg: "New password same as old"} // 新密码与旧密码相同
	PasswordEncryptError      = Errno{Code: 11017, Msg: "Password encrypt error"}   // 密码加密失败
	UpdatePasswordError       = Errno{Code: 11018, Msg: "Update password error"}    // 更新密码失败

	//登出相关错误
	LogoutSuccess    = Errno{Code: 200, Msg: "Logout success"}     // 登出成功
	TokenInvalid     = Errno{Code: 401, Msg: "Token invalid"}      // Token无效
	TokenExpired     = Errno{Code: 401, Msg: "Token expired"}      // Token过期
	TokenEmptyErr    = Errno{Code: 401, Msg: "Token empty"}        // Token为空
	TokenInBlacklist = Errno{Code: 401, Msg: "Token in blacklist"} // Token在黑名单中
	CacheNotExist    = Errno{Code: 401, Msg: "Cache not exist"}    // 缓存不存在

	//Redis错误
	RedisConnErr       = Errno{Code: 10001, Msg: "redis: connect failed"}     // 连接失败
	RedisCmdErr        = Errno{Code: 10002, Msg: "redis: command not found"}  // 命令不存在执行失败
	RedisTimeoutErr    = Errno{Code: 10003, Msg: "redis: command timeout"}    // 操作超时
	RedisUnmarshalErr  = Errno{Code: 10004, Msg: "redis: unmarshal failed"}   // 反序列化失败
	RedisMarshalErr    = Errno{Code: 10005, Msg: "redis: marshal failed"}     // 序列化失败
	RedisScanErr       = Errno{Code: 10006, Msg: "redis: scan failed"}        // 扫描键失败
	RedisDeleteErr     = Errno{Code: 10007, Msg: "redis: delete failed"}      // 删除键失败
	RedisTokenBlackErr = Errno{Code: 10008, Msg: "redis: token black failed"} // 校验黑名单失败

	// 分类相关错误
	CategoryNotFoundErr    = Errno{Code: 13001, Msg: "Category Not Found"}
	CategoryExistErr       = Errno{Code: 13002, Msg: "Category Already Exist"}
	ErrCategoryNameEmpty   = Errno{Code: 13003, Msg: "Category Name Empty"}
	ErrCategoryNameTooLong = Errno{Code: 13004, Msg: "Category Name Too Long"}
	ErrCategorySortInvalid = Errno{Code: 13005, Msg: "Category Sort Invalid"}

	// 文章相关错误
	ArticleNotFoundErr = Errno{Code: 12001, Msg: "Article Not Found"} // 文章不存在

	//

)
