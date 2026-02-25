package system

import (
	"context"
)

// CasbinRuleRepo 定义 casbin_rule 表的操作接口
// 负责管理用户-角色关系(g规则)和角色-菜单权限(p规则)
type CasbinRuleRepo interface {
	// AssignUserRole 为用户分配角色 (创建 g 规则)
	AssignUserRole(ctx context.Context, userID, roleID int64) error

	// RevokeUserRole 撤销用户的某个角色 (删除 g 规则)
	RevokeUserRole(ctx context.Context, userID, roleID int64) error

	// DeleteUserRoles 删除用户的所有角色关联
	DeleteUserRoles(ctx context.Context, userID int64) error

	// ListUserRoleIDs 获取用户的所有角色ID列表
	ListUserRoleIDs(ctx context.Context, userID int64) ([]int64, error)

	// SetRoleMenus 设置角色的菜单权限 (替换式更新)
	SetRoleMenus(ctx context.Context, roleID int64, menuIDs []int64) error

	// ListRoleMenuIDs 获取角色的所有菜单ID列表
	ListRoleMenuIDs(ctx context.Context, roleID int64) ([]int64, error)

	// DeleteRolePermissions 删除角色的所有权限(p规则)
	DeleteRolePermissions(ctx context.Context, roleID int64) error

	// ListUserPermissionMenuIDs 获取用户有权限的所有菜单ID列表
	ListUserPermissionMenuIDs(ctx context.Context, userID int64) ([]int64, error)
}

// CasbinEnforcer 定义 Casbin 权限检查接口
type CasbinEnforcer interface {
	// Enforce 执行权限检查
	Enforce(ctx context.Context, userID string, resource string, action string) (bool, error)

	// LoadPolicy 重新加载策略
	LoadPolicy() error
}
