package config

import (
	"blog-server/common/consts"
	"os"
	"path/filepath"
	"strings"
)

// AIConfig 的目录不属于网站静态目录；管理员白名单为空时禁止使用。
type AIConfig struct {
	Enabled                  bool     `yaml:"enabled"`
	BaseURL                  string   `yaml:"base_url"`
	APIKey                   string   `yaml:"api_key" json:"-"`
	Model                    string   `yaml:"model"`
	AllowedAdminIDs          []uint64 `yaml:"allowed_admin_ids"`
	AllowAllAdmins           bool     `yaml:"allow_all_admins"`            // true 时启用账号均可聊天，仍不能自动读取个人资料。
	PrivateKnowledgeAdminIDs []uint64 `yaml:"private_knowledge_admin_ids"` // 为空时不给任何账号资料权限。
	DataDir                  string   `yaml:"data_dir"`
	KnowledgeDir             string   `yaml:"knowledge_dir"`
	ImportsDir               string   `yaml:"imports_dir"`
	WorkspaceDir             string   `yaml:"workspace_dir"`
	WorkerEnabled            bool     `yaml:"worker_enabled"`
	WorkerKey                string   `yaml:"-" json:"-"`
	DailyLimit               int      `yaml:"daily_limit"`
}

// Effective 仅补充助手自身默认值；环境密钥优先于本地配置，禁用状态不会自动开启。
func (c AIConfig) Effective() AIConfig {
	c.WorkerKey = strings.TrimSpace(os.Getenv(consts.AIWorkerKeyEnv))
	if c.WorkspaceDir == "" && c.DataDir != "" {
		c.WorkspaceDir = filepath.Join(filepath.Dir(c.DataDir), "workspace")
	}
	if c.ImportsDir == "" && c.DataDir != "" {
		c.ImportsDir = filepath.Join(filepath.Dir(c.DataDir), "imports")
	}
	if key := strings.TrimSpace(os.Getenv(consts.AIKeyEnv)); key != "" {
		c.APIKey = key
	}
	if c.BaseURL == "" {
		c.BaseURL = "https://anyrouter.top/v1"
	}
	if c.Model == "" {
		c.Model = "gpt-6.1-sol"
	}
	if c.DailyLimit < 1 {
		c.DailyLimit = 50
	}
	if c.DailyLimit > 500 {
		c.DailyLimit = 500
	}
	return c
}
