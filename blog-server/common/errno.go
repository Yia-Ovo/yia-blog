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
	err.ErrMsg = msg
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

	// ======================
	// 用户错误 11000
	// ======================
	UserNotFoundErr        = Errno{Code: 11001, Msg: "User Not Found"}
	UserExistErr           = Errno{Code: 11002, Msg: "User Already Exist"}
	UserNickNameTooLongErr = Errno{Code: 11011, Msg: "User nickname too long"}
	UserStatusInvalidErr   = Errno{Code: 11012, Msg: "User status invalid"}
	UserCreateErr          = Errno{Code: 11021, Msg: "Create user failed"}
	UserUpdateErr          = Errno{Code: 11022, Msg: "Update user info failed"}
	UserQueryErr           = Errno{Code: 11023, Msg: "Query user failed"}
	UserLoginInfoUpdateErr = Errno{Code: 11024, Msg: "Update login info failed"}
	UserAlreadyExistErr    = Errno{Code: 11025, Msg: "User already exist"}

	// 用户状态
	UserAccountDisabledErr = Errno{Code: 11019, Msg: "Account disabled"}
	LoginInfoUpdateErr     = Errno{Code: 11020, Msg: "Update login info failed"}

	// 登录错误
	LoginParamEmptyErr    = Errno{Code: 11003, Msg: "Login param empty"}
	LoginPasswordErr      = Errno{Code: 11004, Msg: "Password error"}
	LoginFreqLimitErr     = Errno{Code: 11005, Msg: "Login too frequent"}
	LoginAccountLockedErr = Errno{Code: 11006, Msg: "Account locked"}
	LoginTokenExpiredErr  = Errno{Code: 11007, Msg: "Token expired"}
	LoginTokenInvalidErr  = Errno{Code: 11008, Msg: "Token invalid"}
	LoginNotLoginErr      = Errno{Code: 11009, Msg: "Not login"}
	LoginCaptchaErr       = Errno{Code: 11010, Msg: "Captcha error"}
	LoginFailedErr        = Errno{Code: 11011, Msg: "Login Success but info empty"}

	// 密码错误
	UserPwdParamEmptyErr      = Errno{Code: 11013, Msg: "Password param empty"}
	UserOldPasswordErr        = Errno{Code: 11014, Msg: "Old password error"}
	UserNewPasswordInvalidErr = Errno{Code: 11015, Msg: "New password invalid"}
	UserPasswordSameError     = Errno{Code: 11016, Msg: "New password same as old"}
	PasswordEncryptError      = Errno{Code: 11017, Msg: "Password encrypt error"}
	UpdatePasswordError       = Errno{Code: 11018, Msg: "Update password error"}

	// ======================
	// 【新增】管理员错误 16000
	// ======================
	AdminNotFoundErr        = Errno{Code: 16001, Msg: "Admin Not Found"}
	AdminAlreadyExistErr    = Errno{Code: 16002, Msg: "Admin already exist"}
	AdminCreateErr          = Errno{Code: 16003, Msg: "Create admin failed"}
	AdminUpdateErr          = Errno{Code: 16004, Msg: "Update admin info failed"}
	AdminQueryErr           = Errno{Code: 16005, Msg: "Query admin failed"}
	AdminAccountDisabledErr = Errno{Code: 16006, Msg: "Admin account disabled"}

	// ======================
	// 其他业务错误
	// ======================
	// 登出
	LogoutSuccess    = Errno{Code: 200, Msg: "Logout success"}
	TokenInvalid     = Errno{Code: 401, Msg: "Token invalid"}
	TokenExpired     = Errno{Code: 401, Msg: "Token expired"}
	TokenEmptyErr    = Errno{Code: 401, Msg: "Token empty"}
	TokenInBlacklist = Errno{Code: 401, Msg: "Token in blacklist"}
	CacheNotExist    = Errno{Code: 401, Msg: "Cache not exist"}
	GetTokenTimeErr  = Errno{Code: 401, Msg: "Get token time err"}

	// Redis
	RedisConnErr       = Errno{Code: 10001, Msg: "redis: connect failed"}
	RedisCmdErr        = Errno{Code: 10002, Msg: "redis: command not found"}
	RedisTimeoutErr    = Errno{Code: 10003, Msg: "redis: command timeout"}
	RedisUnmarshalErr  = Errno{Code: 10004, Msg: "redis: unmarshal failed"}
	RedisMarshalErr    = Errno{Code: 10005, Msg: "redis: marshal failed"}
	RedisScanErr       = Errno{Code: 10006, Msg: "redis: scan failed"}
	RedisDeleteErr     = Errno{Code: 10007, Msg: "redis: delete failed"}
	RedisTokenBlackErr = Errno{Code: 10008, Msg: "redis: token black failed"}

	// 分类
	CategoryNotFoundErr    = Errno{Code: 13001, Msg: "Category Not Found"}
	CategoryExistErr       = Errno{Code: 13002, Msg: "Category Already Exist"}
	ErrCategoryNameEmpty   = Errno{Code: 13003, Msg: "Category Name Empty"}
	ErrCategoryNameTooLong = Errno{Code: 13004, Msg: "Category Name Too Long"}
	ErrCategorySortInvalid = Errno{Code: 13005, Msg: "Category Sort Invalid"}

	// 文章
	ArticleNotFoundErr     = Errno{Code: 12001, Msg: "Article Not Found"}
	ErrArticleTitleEmpty   = Errno{Code: 12002, Msg: "Article Title Empty"}
	ErrArticleTitleTooLong = Errno{Code: 12003, Msg: "Article Title Too Long"}
	ErrArticleSortInvalid  = Errno{Code: 12004, Msg: "Article Sort Invalid"}
	ErrArticleContentEmpty = Errno{Code: 12005, Msg: "Article Content Empty"}
	ArticleExistErr        = Errno{Code: 12006, Msg: "Article Already Exist"}

	// 评论
	CommentNotFoundErr      = Errno{Code: 14001, Msg: "Comment Not Found"}
	ErrCommentNicknameEmpty = Errno{Code: 14002, Msg: "Comment Nickname Empty"}
	ErrCommentContentEmpty  = Errno{Code: 14003, Msg: "Comment Content Empty"}
	ErrCommentAlreadyLiked  = Errno{Code: 14004, Msg: "Comment Already Liked"}
	ErrCommentNotLiked      = Errno{Code: 14005, Msg: "Comment Not Liked"}
	ErrCommentStatusInvalid = Errno{Code: 14006, Msg: "Comment Status Invalid"}

	// 权限
	PermissionDeniedErr = Errno{Code: 15001, Msg: "Permission Denied"}
)
