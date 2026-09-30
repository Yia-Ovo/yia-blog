package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"blog-server/common/consts"
	"blog-server/common/utils/password"
	"blog-server/domain/repo"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrAdminRecoveryTarget 表示目标并非已有且启用的超级管理员。
	ErrAdminRecoveryTarget = errors.New("仅可恢复已有且启用的超级管理员")
	// ErrAdminRecoveryCredentials 表示输入无法安全用于现有管理员登录入口。
	ErrAdminRecoveryCredentials = errors.New("账号或新密码格式不合法")
)

// AdminRecoveryDomainService 仅供本地运维命令恢复已有超管，不注册 HTTP 入口。
type AdminRecoveryDomainService struct {
	repo repo.AdminRecoveryRepo
}

// NewAdminRecoveryDomainService 构造本地超管密码恢复用例。
func NewAdminRecoveryDomainService(r repo.AdminRecoveryRepo) *AdminRecoveryDomainService {
	return &AdminRecoveryDomainService{repo: r}
}

// CheckTarget 只读确认目标，不修改密码、登录版本或其他账号资料。
func (s *AdminRecoveryDomainService) CheckTarget(ctx context.Context, name string) (uint64, error) {
	account, err := s.findTarget(ctx, name)
	if err != nil {
		return 0, err
	}
	return account.ID, nil
}

// CheckPassword 只读核验已有超管密码，不套用新密码规则，也不修改账号或登录版本。
func (s *AdminRecoveryDomainService) CheckPassword(ctx context.Context, name, suppliedPassword string) (bool, error) {
	account, err := s.findTarget(ctx, name)
	if err != nil {
		return false, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(suppliedPassword))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	if err != nil {
		return false, errors.New("无法核验密码，未修改账号")
	}
	return true, nil
}

// Recover 用新 bcrypt 哈希替换原密码，持久化层同时使旧登录版本失效。
func (s *AdminRecoveryDomainService) Recover(ctx context.Context, name, newPassword string) (uint64, error) {
	if !utf8.ValidString(newPassword) || strings.ContainsAny(newPassword, "\r\n\x00") {
		return 0, ErrAdminRecoveryCredentials
	}
	if msg := password.CheckPasswordFormat("", newPassword); msg != "" {
		return 0, fmt.Errorf("%w：%s", ErrAdminRecoveryCredentials, msg)
	}
	account, err := s.findTarget(ctx, name)
	if err != nil {
		return 0, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, errors.New("新密码加密失败，未修改账号")
	}
	if err := s.repo.ReplacePassword(ctx, account, string(hash)); err != nil {
		return 0, err
	}
	return account.ID, nil
}

func (s *AdminRecoveryDomainService) findTarget(ctx context.Context, name string) (*repo.AdminRecoveryAccount, error) {
	// 不自动 TrimSpace，避免运维输入被静默改为另一个账号。
	if name == "" || strings.TrimSpace(name) != name || !utf8.ValidString(name) || strings.ContainsAny(name, "\r\n\x00") {
		return nil, ErrAdminRecoveryCredentials
	}
	account, err := s.repo.FindByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if account == nil || account.ID == 0 || account.Name != name || account.Role != consts.AdminRoleSuper || account.Status != consts.AdminStatusEnabled {
		return nil, ErrAdminRecoveryTarget
	}
	return account, nil
}
