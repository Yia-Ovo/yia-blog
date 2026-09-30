package consts

import "time"

const (
	// 初始化密码更严格，但上限仍兼容现有登录 DTO 的 20 字符限制。
	BootstrapAdminNameMaxLength = 16
	BootstrapPasswordMinLength  = 12
	BootstrapPasswordMaxLength  = 20
	BcryptPasswordMaxBytes      = 72
	BootstrapTimeout            = 30 * time.Second
	BootstrapLockWaitSeconds    = 10
	BootstrapUnlockTimeout      = 3 * time.Second
	BootstrapLockPrefix         = "yia-blog/bootstrap/"
)
