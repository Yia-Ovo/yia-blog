package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

var aiIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,64}$`)

// AIDomainService 只依赖管理员领域服务；磁盘和模型调用由 App 编排。
type AIDomainService struct {
	admins     *AdminDomainService
	policy     do.AIAccessPolicy
	dailyLimit int
}

func NewAIDomainService(admins *AdminDomainService, policy do.AIAccessPolicy, dailyLimit int) *AIDomainService {
	policy.AllowedAdminIDs = slices.Clone(policy.AllowedAdminIDs)
	policy.PrivateKnowledgeAdminIDs = slices.Clone(policy.PrivateKnowledgeAdminIDs)
	return &AIDomainService{admins: admins, policy: policy, dailyLimit: dailyLimit}
}

// Authorize 每个入口都核对现行管理员状态和配置白名单，不能由前端角色替代。
func (s *AIDomainService) Authorize(ctx context.Context, adminID uint64) error {
	_, err := s.Access(ctx, adminID)
	return err
}

// Access 根据现行账号区分通用聊天和私人资料，不依赖客户端传来的角色或开关。
func (s *AIDomainService) Access(ctx context.Context, adminID uint64) (do.AIAccess, error) {
	access := do.AIAccess{AdminID: adminID}
	if adminID == 0 {
		return access, common.LoginNotLoginErr
	}
	if !s.policy.AllowAllAdmins && !slices.Contains(s.policy.AllowedAdminIDs, adminID) {
		return access, common.PermissionErr.WithMsg("此管理员未开通 AI 助手。")
	}
	admin, err := s.admins.FindAdminByID(ctx, adminID)
	if err != nil {
		return access, err
	}
	if admin == nil || admin.ID != adminID || admin.Status != consts.AdminStatusEnabled {
		return access, common.PermissionErr
	}
	if admin.Role != consts.AdminRoleSuper && admin.Role != consts.AdminRoleNormal {
		return access, common.PermissionErr
	}
	access.PrivateKnowledge = admin.Role == consts.AdminRoleSuper && slices.Contains(s.policy.PrivateKnowledgeAdminIDs, adminID)
	return access, nil
}

// AuthorizeState 私有历史和记忆不能在管理员降级或移出资料名单后重新读取。
func (s *AIDomainService) AuthorizeState(state *do.AIState, access do.AIAccess) error {
	if state.AdminID != access.AdminID || (state.PrivateKnowledge && !access.PrivateKnowledge) {
		return common.PermissionErr.WithMsg("无权读取此账号的私有 AI 记录。")
	}
	if access.PrivateKnowledge {
		state.PrivateKnowledge = true
	}
	return nil
}

func (s *AIDomainService) Conversation(state *do.AIState, adminID uint64, id string) (*do.AIConversation, error) {
	if state.AdminID != adminID {
		return nil, common.PermissionErr
	}
	for i := range state.Conversations {
		if state.Conversations[i].ID == id {
			return &state.Conversations[i], nil
		}
	}
	return nil, common.ParamErr.WithMsg("对话不存在，请重新选择。")
}

// PrepareTurn 对同一 requestID 返回原任务；失败任务也不能隐式重新调用模型。
func (s *AIDomainService) PrepareTurn(state *do.AIState, adminID uint64, conversationID, requestID, content string, now time.Time) (*do.AIConversation, bool, error) {
	return s.prepareTurn(state, adminID, conversationID, requestID, content, now, consts.AIContextGeneral)
}

func (s *AIDomainService) PrepareContextTurn(state *do.AIState, access do.AIAccess, conversationID, requestID, content string, now time.Time, requestedMode string) (*do.AIConversation, bool, error) {
	mode, err := s.ContextMode(access, requestedMode)
	if err != nil {
		return nil, false, err
	}
	return s.prepareTurn(state, access.AdminID, conversationID, requestID, content, now, mode)
}

func (s *AIDomainService) prepareTurn(state *do.AIState, adminID uint64, conversationID, requestID, content string, now time.Time, mode string) (*do.AIConversation, bool, error) {
	if state.AdminID != adminID {
		return nil, false, common.PermissionErr
	}
	if mode != consts.AIContextGeneral && mode != consts.AIContextProject {
		return nil, false, common.ParamErr
	}
	if existing, err := s.Conversation(state, adminID, conversationID); err == nil {
		currentMode := existing.ContextMode
		if currentMode == "" {
			currentMode = consts.AIContextGeneral
		}
		if currentMode != mode {
			return nil, false, common.ParamErr.WithMsg("切换资料模式时请新建对话，避免沿用私人历史。")
		}
	}
	content = strings.TrimSpace(content)
	if !aiIDPattern.MatchString(conversationID) || !aiIDPattern.MatchString(requestID) || content == "" || !utf8.ValidString(content) || utf8.RuneCountInString(content) > consts.AIMessageMaxRunes {
		return nil, false, common.ParamErr.WithMsg("问题不能为空且须在 4000 字以内。")
	}
	s.ExpirePending(state, now)
	for i := range state.Conversations {
		for _, message := range state.Conversations[i].Messages {
			if message.RequestID == requestID && message.Role == consts.AIRoleUser {
				if state.Conversations[i].ID != conversationID || message.Content != content {
					return nil, false, common.ParamErr.WithMsg("重复请求内容不一致。")
				}
				return &state.Conversations[i], false, nil
			}
			if message.Status == consts.AIStatusPending {
				return nil, false, common.Errno{Code: 429, Msg: "已有问题正在回答，请稍候。"}
			}
		}
	}
	day := now.In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format("2006-01-02")
	if state.Day != day {
		state.Day = day
		state.Requests = 0
	}
	if s.dailyLimit < 1 || state.Requests >= s.dailyLimit {
		return nil, false, common.Errno{Code: 429, Msg: "今日 AI 使用次数已达上限。"}
	}
	state.RecentRequests = slices.DeleteFunc(state.RecentRequests, func(t time.Time) bool { return !t.After(now.Add(-time.Minute)) })
	if len(state.RecentRequests) >= 5 {
		return nil, false, common.Errno{Code: 429, Msg: "问题发送过于频繁，请稍后再试。"}
	}
	conversation, _ := s.Conversation(state, adminID, conversationID)
	if conversation == nil {
		if len(state.Conversations) >= consts.AIMaxConversations {
			return nil, false, common.ParamErr.WithMsg("对话数量已达上限，请先删除旧对话。")
		}
		title := []rune(content)
		if len(title) > 24 {
			title = title[:24]
		}
		state.Conversations = append(state.Conversations, do.AIConversation{ID: conversationID, ContextMode: mode, Title: string(title), Messages: []do.AIMessage{}})
		conversation = &state.Conversations[len(state.Conversations)-1]
	}
	if len(conversation.Messages) >= consts.AIMaxMessages {
		return nil, false, common.ParamErr.WithMsg("此对话已达上限，请新建对话。")
	}
	conversation.Messages = append(conversation.Messages,
		do.AIMessage{ID: requestID + "-u", RequestID: requestID, Role: consts.AIRoleUser, Content: content, Status: consts.AIStatusCompleted, CreateAt: now},
		do.AIMessage{ID: requestID + "-a", RequestID: requestID, Role: consts.AIRoleAssistant, Status: consts.AIStatusPending, CreateAt: now})
	conversation.UpdateAt = now
	state.Requests++
	state.RecentRequests = append(state.RecentRequests, now)
	return conversation, true, nil
}

// FinishTurn 只结束尚在租期内的任务；重复或延迟结果不能改写已有结论。
func (s *AIDomainService) FinishTurn(state *do.AIState, adminID uint64, conversationID, requestID, answer string, failed bool, sources []do.AISource, now time.Time) error {
	c, err := s.Conversation(state, adminID, conversationID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(answer) == "" || !utf8.ValidString(answer) || utf8.RuneCountInString(answer) > consts.AIAnswerMaxRunes {
		return common.ParamErr.WithMsg("AI 回答为空或超过长度限制。")
	}
	s.ExpirePending(state, now)
	for i := range c.Messages {
		m := &c.Messages[i]
		if m.RequestID != requestID || m.Role != consts.AIRoleAssistant {
			continue
		}
		if m.Status != consts.AIStatusPending {
			return nil
		}
		m.Status = consts.AIStatusCompleted
		if failed {
			m.Status = consts.AIStatusFailed
		}
		m.Content = answer
		m.Sources = slices.Clone(sources)
		c.UpdateAt = now
		return nil
	}
	return common.ParamErr.WithMsg("回答对应的请求不存在。")
}

// ExpirePending 将进程重启或超时遗留的任务标记失败，避免永久占用会话。
func (s *AIDomainService) ExpirePending(state *do.AIState, now time.Time) {
	for i := range state.Conversations {
		for j := range state.Conversations[i].Messages {
			m := &state.Conversations[i].Messages[j]
			if m.Status == consts.AIStatusPending && !m.CreateAt.Add(consts.AILeaseDuration).After(now) {
				m.Status = consts.AIStatusFailed
				m.Content = "回答中断，请重新发送问题。本次尝试已计入额度。"
			}
		}
	}
}

func (s *AIDomainService) SetMemory(state *do.AIState, adminID uint64, memory string) error {
	if state.AdminID != adminID {
		return common.PermissionErr
	}
	if !utf8.ValidString(memory) || utf8.RuneCountInString(memory) > consts.AIMemoryMaxRunes {
		return common.ParamErr.WithMsg("长期记忆须在 4000 字以内。")
	}
	state.Memory = strings.TrimSpace(memory)
	return nil
}

// PrivateWorkspace 限制同步、导入和执行服务只能由资料所有者的现行超管身份访问。
func (s *AIDomainService) PrivateWorkspace(access do.AIAccess) error {
	if access.AdminID == 0 || !access.PrivateKnowledge {
		return common.PermissionErr
	}
	return nil
}

func (s *AIDomainService) ImportedPage(value *do.AIImportedThread, access do.AIAccess, before int) ([]do.AIMessage, int, error) {
	if err := s.ValidateImportedThread(value, access, true); err != nil {
		return nil, 0, err
	}
	end := before
	if end == 0 {
		end = len(value.Messages)
	}
	if end < 0 || end > len(value.Messages) {
		return nil, 0, common.ParamErr
	}
	start := end - 50
	if start < 0 {
		start = 0
	}
	return value.Messages[start:end], start, nil
}

func (s *AIDomainService) SeedImportedConversation(state *do.AIState, access do.AIAccess, id string, imported *do.AIImportedThread) error {
	if state.AdminID != access.AdminID {
		return common.PermissionErr
	}
	if err := s.ValidateImportedThread(imported, access, true); err != nil {
		return err
	}
	if !aiIDPattern.MatchString(id) {
		return common.ParamErr
	}
	if existing, err := s.Conversation(state, access.AdminID, id); err == nil {
		if existing.OriginImportID != imported.ID || existing.ContextMode != consts.AIContextProject {
			return common.ParamErr.WithMsg("导入不同历史时请新建对话。")
		}
		return nil
	}
	if len(state.Conversations) >= consts.AIMaxConversations {
		return common.ParamErr.WithMsg("对话数量已达上限。")
	}
	messages := []do.AIMessage{}
	start := len(imported.Messages) - 20
	if start < 0 {
		start = 0
	}
	for index, message := range imported.Messages[start:] {
		if message.Role != consts.AIRoleUser && message.Role != consts.AIRoleAssistant {
			return common.ParamErr
		}
		if !utf8.ValidString(message.Content) {
			return common.ParamErr
		}
		content := []rune(message.Content)
		if len(content) > 4000 {
			content = content[:4000]
		}
		messages = append(messages, do.AIMessage{ID: fmt.Sprintf("codex-%s-%d", imported.ID, index), Role: message.Role, Content: string(content), Status: consts.AIStatusCompleted, CreateAt: message.CreateAt})
	}
	state.PrivateKnowledge = true
	state.Conversations = append(state.Conversations, do.AIConversation{ID: id, Title: imported.Title, ContextMode: consts.AIContextProject, OriginImportID: imported.ID, UpdateAt: imported.UpdatedAt, Messages: messages})
	return nil
}

// ContextMode 只允许具备私人资料权限的账号主动开启项目上下文。
func (s *AIDomainService) ContextMode(access do.AIAccess, mode string) (string, error) {
	if mode == "" {
		mode = consts.AIContextGeneral
	}
	if mode != consts.AIContextGeneral && mode != consts.AIContextProject {
		return "", common.ParamErr
	}
	if mode == consts.AIContextProject && !access.PrivateKnowledge {
		return "", common.PermissionErr
	}
	return mode, nil
}

// ContextMemory 将通用和项目背景分开，私人记忆不能作为通用聊天输入。
func (s *AIDomainService) ContextMemory(state *do.AIState, mode string) string {
	if mode == consts.AIContextProject {
		return state.Memory
	}
	return state.GeneralMemory
}

func (s *AIDomainService) SetContextMemory(state *do.AIState, access do.AIAccess, mode, memory string) error {
	mode, err := s.ContextMode(access, mode)
	if err != nil {
		return err
	}
	if state.AdminID != access.AdminID {
		return common.PermissionErr
	}
	if !utf8.ValidString(memory) || utf8.RuneCountInString(memory) > consts.AIMemoryMaxRunes {
		return common.ParamErr.WithMsg("长期记忆须在 4000 字以内。")
	}
	if mode == consts.AIContextProject {
		state.Memory = strings.TrimSpace(memory)
	} else {
		state.GeneralMemory = strings.TrimSpace(memory)
	}
	return nil
}

func (s *AIDomainService) DeleteConversation(state *do.AIState, adminID uint64, id string) error {
	c, err := s.Conversation(state, adminID, id)
	if err != nil {
		return err
	}
	for _, m := range c.Messages {
		if m.Status == consts.AIStatusPending {
			return common.ParamErr.WithMsg("请等待回答结束后再删除。")
		}
	}
	state.Conversations = slices.DeleteFunc(state.Conversations, func(c do.AIConversation) bool { return c.ID == id })
	return nil
}
