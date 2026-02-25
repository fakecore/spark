package system

import (
	"context"
	"fmt"

	"spark/internal/data/do"
	clog "spark/pkg/clog"
)

type SystemPermissionUsecase struct {
	roleRepo   SystemRoleRepo
	menuRepo   SystemMenuRepo
	casbinRepo CasbinRuleRepo
	logger     clog.Logger
}

func NewSystemPermissionUsecase(
	roleRepo SystemRoleRepo,
	menuRepo SystemMenuRepo,
	casbinRepo CasbinRuleRepo,
	logger clog.Logger,
) *SystemPermissionUsecase {
	return &SystemPermissionUsecase{
		roleRepo:   roleRepo,
		menuRepo:   menuRepo,
		casbinRepo: casbinRepo,
		logger:     logger,
	}
}

// UpdateRoleMenus 更新角色的菜单权限
func (uc *SystemPermissionUsecase) UpdateRoleMenus(ctx context.Context, roleID int64, menuIDs []int64) error {
	// 1. 验证角色存在
	_, err := uc.roleRepo.Get(ctx, roleID)
	if err != nil {
		return fmt.Errorf("role not found: %w", err)
	}

	// 2. 验证菜单存在
	if len(menuIDs) > 0 {
		for _, menuID := range menuIDs {
			_, err := uc.menuRepo.Get(ctx, menuID)
			if err != nil {
				return fmt.Errorf("menu ID %d not found: %w", menuID, err)
			}
		}
	}

	// 3. 设置菜单权限
	return uc.casbinRepo.SetRoleMenus(ctx, roleID, menuIDs)
}

// UpdateRoleDataScope 更新角色的数据范围
func (uc *SystemPermissionUsecase) UpdateRoleDataScope(ctx context.Context, roleID int64, dataScope int32, dataScopeDeptIDs []int64) error {
	role := &do.SysRole{
		ID:               roleID,
		DataScope:        &dataScope,
		DataScopeDeptIDs: dataScopeDeptIDs,
	}
	return uc.roleRepo.Update(ctx, role)
}

// UpdateUserRoles 更新用户的角色分配
func (uc *SystemPermissionUsecase) UpdateUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	// 1. 验证所有角色存在
	if len(roleIDs) > 0 {
		for _, roleID := range roleIDs {
			_, err := uc.roleRepo.Get(ctx, roleID)
			if err != nil {
				return fmt.Errorf("role ID %d not found: %w", roleID, err)
			}
		}
	}

	// 2. 获取用户当前的角色ID列表
	currentRoleIDs, err := uc.casbinRepo.ListUserRoleIDs(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get current user roles: %w", err)
	}

	// 3. 计算需要添加和删除的角色
	currentMap := make(map[int64]bool)
	for _, id := range currentRoleIDs {
		currentMap[id] = true
	}

	newMap := make(map[int64]bool)
	for _, id := range roleIDs {
		newMap[id] = true
	}

	// 4. 删除不再需要的角色
	for _, roleID := range currentRoleIDs {
		if !newMap[roleID] {
			if err := uc.casbinRepo.RevokeUserRole(ctx, userID, roleID); err != nil {
				uc.logger.Warn(ctx, "failed to revoke user role", clog.Int64("user_id", userID), clog.Int64("role_id", roleID), clog.Err(err))
			}
		}
	}

	// 5. 添加新角色
	for _, roleID := range roleIDs {
		if !currentMap[roleID] {
			if err := uc.casbinRepo.AssignUserRole(ctx, userID, roleID); err != nil {
				return fmt.Errorf("failed to assign user role: %w", err)
			}
		}
	}

	return nil
}

// ListRoleMenus 获取角色的菜单ID列表
func (uc *SystemPermissionUsecase) ListRoleMenus(ctx context.Context, roleID int64) ([]int64, error) {
	return uc.casbinRepo.ListRoleMenuIDs(ctx, roleID)
}

// ListUserRoles 获取用户的角色ID列表
func (uc *SystemPermissionUsecase) ListUserRoles(ctx context.Context, userID int64) ([]int64, error) {
	return uc.casbinRepo.ListUserRoleIDs(ctx, userID)
}

// CheckAPIPermission 检查指定角色是否有权限访问指定API路径
func (uc *SystemPermissionUsecase) CheckAPIPermission(ctx context.Context, roleID int64, path string) (bool, error) {
	menuID, err := uc.menuRepo.GetMenuIDByPath(ctx, path)
	if err != nil {
		return false, fmt.Errorf("failed to get menu ID: %w", err)
	}

	// 获取角色的菜单ID列表
	menuIDs, err := uc.casbinRepo.ListRoleMenuIDs(ctx, roleID)
	if err != nil {
		return false, err
	}

	// 检查是否包含指定菜单
	for _, id := range menuIDs {
		if id == menuID {
			return true, nil
		}
	}

	return false, nil
}

// CheckMenuPermission 检查用户是否有权限访问指定菜单
func (uc *SystemPermissionUsecase) CheckMenuPermission(ctx context.Context, userID int64, menuID int64) (bool, error) {
	// 获取用户的菜单ID列表
	menuIDs, err := uc.casbinRepo.ListUserPermissionMenuIDs(ctx, userID)
	if err != nil {
		return false, err
	}

	// 检查是否包含指定菜单
	for _, id := range menuIDs {
		if id == menuID {
			return true, nil
		}
	}

	return false, nil
}

// ListUserPermissions 获取用户的所有权限列表
func (uc *SystemPermissionUsecase) ListUserPermissions(ctx context.Context, userID int64) (map[int64][]string, error) {
	menuIDs, err := uc.casbinRepo.ListUserPermissionMenuIDs(ctx, userID)
	if err != nil {
		return nil, err
	}

	permissionMap := make(map[int64][]string)
	for _, menuID := range menuIDs {
		// 默认授予GET权限
		permissionMap[menuID] = []string{"GET"}
	}

	return permissionMap, nil
}
