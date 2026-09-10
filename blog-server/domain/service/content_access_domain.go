package service

import (
	"blog-server/common"
	"blog-server/common/consts"
	"context"
)

// ensureContentAdmin 读取当前账号状态；内容管理允许启用的普通管理员和超管。
// 不能仅信任 HTTP 中间件，否则其他入口调用领域用例时会绕过权限规则。
func ensureContentAdmin(ctx context.Context, admins *AdminDomainService, adminID uint64) error {
	if adminID == 0 {
		return common.LoginNotLoginErr
	}
	admin, err := admins.FindAdminByID(ctx, adminID)
	if err != nil {
		return err
	}
	if admin == nil || admin.ID != adminID {
		return common.AuthErr
	}
	if admin.Status != consts.AdminStatusEnabled {
		return common.AdminAccountDisabledErr
	}
	if admin.Role != consts.AdminRoleSuper && admin.Role != consts.AdminRoleNormal {
		return common.PermissionDeniedErr
	}
	return nil
}
