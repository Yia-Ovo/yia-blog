package dto

import "time"

// AISendReq 不接受 admin_id、角色、额度、模型和磁盘路径，这些由后端维护。
type AISendReq struct {
	ConversationID string `json:"conversation_id" binding:"required,max=64"`
	RequestID      string `json:"request_id" binding:"required,max=64"`
	Content        string `json:"content" binding:"required,max=4000"`
	ContextMode    string `json:"context_mode" binding:"omitempty,oneof=general project"`
	ImportID       string `json:"import_id" binding:"omitempty,max=64"`
}

type AIConversationReq struct {
	ID string `json:"id" form:"id" binding:"required,max=64"`
}
type AIMemoryReq struct {
	Memory      string `json:"memory" binding:"max=4000"`
	ContextMode string `json:"context_mode" binding:"omitempty,oneof=general project"`
}
type AISource struct {
	Excerpt   string    `json:"excerpt"`
	Path      string    `json:"path"`
	Line      int       `json:"line"`
	UpdatedAt time.Time `json:"updated_at"`
}
type AIMessage struct {
	ID       string     `json:"id"`
	Role     string     `json:"role"`
	Content  string     `json:"content"`
	Status   string     `json:"status"`
	CreateAt time.Time  `json:"create_at"`
	Sources  []AISource `json:"sources"`
}
type AIConversation struct {
	OriginImportID string      `json:"origin_import_id,omitempty"`
	ContextMode    string      `json:"context_mode"`
	ID             string      `json:"id"`
	Title          string      `json:"title"`
	UpdateAt       time.Time   `json:"update_at"`
	Messages       []AIMessage `json:"messages,omitempty"`
}

type AIImportReq struct {
	ID     string `form:"id" binding:"required,max=64"`
	Before int    `form:"before" binding:"min=0"`
}

type AIImportedThread struct {
	ID            string      `json:"id"`
	Title         string      `json:"title"`
	UpdatedAt     time.Time   `json:"updated_at"`
	Messages      []AIMessage `json:"messages,omitempty"`
	TotalMessages int         `json:"total_messages"`
	NextBefore    int         `json:"next_before"`
}

// AISyncImportReq 仅接受项目标识和用户/助手文字，不接受账号、工具输出、角色或磁盘目录。
type AISyncImportReq struct {
	ID        string                `json:"id" binding:"required,max=64"`
	Title     string                `json:"title" binding:"required,max=200"`
	Workspace string                `json:"workspace" binding:"required,oneof=yia-blog yia-project"`
	UpdatedAt time.Time             `json:"updated_at" binding:"required"`
	Messages  []AISyncImportMessage `json:"messages" binding:"required,min=1,max=20000,dive"`
}

type AISyncImportMessage struct {
	ID       string    `json:"id" binding:"required,max=128"`
	Role     string    `json:"role" binding:"required,oneof=user assistant"`
	Content  string    `json:"content" binding:"required,max=100000"`
	CreateAt time.Time `json:"create_at"`
}
type AIOverview struct {
	Model                   string           `json:"model"`
	Memory                  string           `json:"memory"`
	ProjectMemory           string           `json:"project_memory,omitempty"`
	DailyLimit              int              `json:"daily_limit"`
	Used                    int              `json:"used"`
	KnowledgeDocuments      int              `json:"knowledge_documents"`
	PrivateKnowledgeEnabled bool             `json:"private_knowledge_enabled"` // 服务端能力结果，仅用于界面说明。
	Conversations           []AIConversation `json:"conversations"`
}
