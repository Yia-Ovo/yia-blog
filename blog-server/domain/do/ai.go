package do

import "time"

// AIAccessPolicy 分开控制模型使用资格与个人资料权限；资料访问仍要求现行主管理员角色。
type AIAccessPolicy struct {
	AllowedAdminIDs          []uint64
	PrivateKnowledgeAdminIDs []uint64
	AllowAllAdmins           bool
}

// AIAccess 是根据现行账号计算的能力，不接受 HTTP 客户端提交。
type AIAccess struct {
	AdminID          uint64
	PrivateKnowledge bool
}

// AIState 属于单个管理员，长期记忆与额度独立于浏览器和具体会话。
type AIState struct {
	AdminID          uint64
	PrivateKnowledge bool // 曾在私人资料模式使用过的记录不能因降级变为通用记录。
	Memory           string
	GeneralMemory    string
	Day              string
	Requests         int
	RecentRequests   []time.Time
	Conversations    []AIConversation
}

type AIConversation struct {
	OriginImportID string
	ContextMode    string
	ID             string
	Title          string
	UpdateAt       time.Time
	Messages       []AIMessage
}

// AIMessage 的 RequestID 由客户端生成以防提交响应丢失后重复扣额度。
type AIMessage struct {
	ID        string
	RequestID string
	Role      string
	Content   string
	Status    string
	CreateAt  time.Time
	Sources   []AISource
}

// AISource 标识被检索的资料版本；不向浏览器暴露磁盘绝对路径。
type AISource struct {
	Path      string
	Line      int
	UpdatedAt time.Time
	Excerpt   string
}

type AIModelMessage struct{ Role, Content string }
