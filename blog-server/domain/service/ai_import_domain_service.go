package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"regexp"
	"strings"
	"unicode/utf8"
)

var aiImportedIDPattern = regexp.MustCompile(`^[a-z0-9-]{16,64}$`)

// ImportedContextMode 禁止通过通用聊天入口注入本地项目历史。
func (s *AIDomainService) ImportedContextMode(access do.AIAccess, mode string) error {
	if err := s.PrivateWorkspace(access); err != nil {
		return err
	}
	if mode != consts.AIContextProject {
		return common.ParamErr.WithMsg("继续本地历史时请开启项目资料模式。")
	}
	return nil
}

// ValidateImportedThread 将超管、资料所有者、项目范围和消息白名单作为所有入口的领域规则。
func (s *AIDomainService) ValidateImportedThread(value *do.AIImportedThread, access do.AIAccess, withMessages bool) error {
	if err := s.PrivateWorkspace(access); err != nil {
		return err
	}
	if value == nil || value.AdminID != access.AdminID {
		return common.PermissionErr
	}
	if !aiImportedIDPattern.MatchString(value.ID) || strings.TrimSpace(value.Title) == "" || !utf8.ValidString(value.Title) || utf8.RuneCountInString(value.Title) > 200 || value.UpdatedAt.IsZero() {
		return common.ParamErr
	}
	if value.Workspace != "yia-blog" && value.Workspace != "yia-project" {
		return common.ParamErr.WithMsg("仅允许同步 yia-blog / yia-project 的会话。")
	}
	if !withMessages {
		return nil
	}
	if len(value.Messages) == 0 || len(value.Messages) > consts.AIImportMaxMessages {
		return common.ParamErr
	}
	ids := make(map[string]bool, len(value.Messages))
	for _, message := range value.Messages {
		if message.Role != consts.AIRoleUser && message.Role != consts.AIRoleAssistant {
			return common.ParamErr
		}
		if message.ID == "" || len(message.ID) > 128 || ids[message.ID] || !utf8.ValidString(message.ID) || strings.TrimSpace(message.Content) == "" || !utf8.ValidString(message.Content) || len(message.Content) > 100000 {
			return common.ParamErr
		}
		ids[message.ID] = true
	}
	return nil
}
