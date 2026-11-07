package do

import "time"

// AIImportedThread 是站主授权同步的 Codex 文字记录，不包含工具输出或系统指令。
type AIImportedThread struct {
	AdminID              uint64
	ID, Title, Workspace string
	UpdatedAt            time.Time
	Messages             []AIMessage
}
