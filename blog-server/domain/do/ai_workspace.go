package do

import "time"

// AIWorkspaceState 保存超管的同步状态与代码任务，执行租约不返回浏览器。
type AIWorkspaceState struct {
	AdminID uint64
	Day     string
	Tasks   int
	Sync    AISyncStatus
	Jobs    []AICodeJob
}

type AISyncStatus struct {
	Phase, Revision, Message string
	UpdatedAt                time.Time
}

type AICodeJob struct {
	ID, Prompt, ContextMode, Status                           string
	Context, Summary, Diff, Checks, Branch, URL, BaseRevision string
	ReviewDigest                                              string
	TestsPassed                                               bool
	CreatedAt, UpdatedAt, LeaseUntil                          time.Time
	Lease                                                     string
}
