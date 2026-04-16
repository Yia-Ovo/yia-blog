package convert

// AdminUser 管理员信息（适配管理员认证）
type AdminUser struct {
	AdminID uint64 `json:"admin_id"` // 管理员ID
	Name    string `json:"name"`     // 管理员姓名
	RoleID  int64  `json:"role_id"`  // 角色ID（RBAC权限用）
}
