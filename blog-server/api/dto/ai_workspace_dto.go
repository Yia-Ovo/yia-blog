package dto

import "time"

// AICodeJobReq 只接受需求与资料模式，执行目录、命令和分支均由服务端维护。
type AICodeJobReq struct {
	ID          string `json:"id" binding:"required,max=64"`
	Prompt      string `json:"prompt" binding:"required,max=4000"`
	ContextMode string `json:"context_mode" binding:"omitempty,oneof=general project"`
}

// AIPublishJobReq 将审核绑定到当前差异摘要，避免旧页面发布已变化的内容。
type AIPublishJobReq struct {
	ID           string `json:"id" binding:"required,max=64"`
	ReviewDigest string `json:"review_digest" binding:"required,len=64"`
}

type AIWorkspace struct {
	Enabled    bool        `json:"enabled"`
	DailyLimit int         `json:"daily_limit"`
	Jobs       []AICodeJob `json:"jobs"`
}

// AICodeJob 不将执行租约、模型上下文或凭据返回浏览器。
type AICodeJob struct {
	ID           string    `json:"id"`
	Prompt       string    `json:"prompt"`
	ContextMode  string    `json:"context_mode"`
	Status       string    `json:"status"`
	Summary      string    `json:"summary"`
	Diff         string    `json:"diff,omitempty"`
	Checks       string    `json:"checks,omitempty"`
	Branch       string    `json:"branch"`
	URL          string    `json:"url,omitempty"`
	TestsPassed  bool      `json:"tests_passed"`
	ReviewDigest string    `json:"review_digest,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AIWorkerJob 仅通过独立凭据和回环入口交给服务器执行服务。
type AIWorkerJob struct {
	AdminID      uint64 `json:"admin_id"`
	ID           string `json:"id"`
	Prompt       string `json:"prompt"`
	Context      string `json:"context"`
	ContextMode  string `json:"context_mode"`
	Status       string `json:"status"`
	Lease        string `json:"lease"`
	Branch       string `json:"branch"`
	BaseRevision string `json:"base_revision"`
	Diff         string `json:"diff"`
}

type AIWorkerReportReq struct {
	AdminID      uint64 `json:"admin_id" binding:"required"`
	ID           string `json:"id" binding:"required,max=64"`
	Lease        string `json:"lease" binding:"required,max=64"`
	Phase        string `json:"phase" binding:"required,oneof=review completed failed"`
	Summary      string `json:"summary" binding:"max=4000"`
	Diff         string `json:"diff" binding:"max=100000"`
	Checks       string `json:"checks" binding:"max=12000"`
	BaseRevision string `json:"base_revision" binding:"max=40"`
	TestsPassed  bool   `json:"tests_passed"`
}
