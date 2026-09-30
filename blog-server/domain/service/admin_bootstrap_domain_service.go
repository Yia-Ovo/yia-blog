package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"blog-server/common/consts"
	"blog-server/domain/repo"
)

// ErrAdminBootstrapExists 表示库中已有管理员；初始化工具不得重置任何原有账号。
var ErrAdminBootstrapExists = errors.New("管理员表已有记录（包括软删除账号），拒绝初始化")

// ErrAdminBootstrapCredentials 表示凭据不满足初始化和现有登录入口的共同约束。
var ErrAdminBootstrapCredentials = errors.New("账号须为 1–16 字符且无首尾空白；密码须为 12–20 字符、不超过 72 字节且不含换行")

// AdminBootstrapDomainService 只用于部署时创建首位超管，不注册 HTTP 入口。
type AdminBootstrapDomainService struct {
	repo repo.AdminBootstrapRepo
}

// NewAdminBootstrapDomainService 构造空库管理员初始化用例。
func NewAdminBootstrapDomainService(r repo.AdminBootstrapRepo) *AdminBootstrapDomainService {
	return &AdminBootstrapDomainService{repo: r}
}

// Bootstrap 在独占会话内校验空库条件，复用既有 CreateAdmin 的 bcrypt、角色和状态规则。
func (s *AdminBootstrapDomainService) Bootstrap(ctx context.Context, name, password string) error {
	if !utf8.ValidString(name) || strings.TrimSpace(name) != name || name == "" || utf8.RuneCountInString(name) > consts.BootstrapAdminNameMaxLength ||
		!utf8.ValidString(password) || utf8.RuneCountInString(password) < consts.BootstrapPasswordMinLength ||
		utf8.RuneCountInString(password) > consts.BootstrapPasswordMaxLength || len(password) > consts.BcryptPasswordMaxBytes ||
		strings.ContainsAny(name+password, "\r\n\x00") {
		return ErrAdminBootstrapCredentials
	}
	return s.repo.WithExclusiveAdmin(ctx, func(admins repo.AdminMysqlRepo, existing bool) error {
		if existing {
			return ErrAdminBootstrapExists
		}
		_, err := NewAdminDomainService(admins).CreateAdmin(ctx, name, password, consts.AdminRoleSuper)
		return err
	})
}
