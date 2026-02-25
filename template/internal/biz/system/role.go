package system

import (
	"context"
	"fmt"

	"spark/internal/data/dal/model"
	"spark/internal/data/do"
	clog "spark/pkg/clog"
)

// SystemRoleRepo 定义角色数据访问接口
// 只负责 sys_role 表的 CRUD 操作
type SystemRoleRepo interface {
	Get(ctx context.Context, id int64) (*model.SysRole, error)
	ListByIDs(ctx context.Context, ids []int64) ([]*model.SysRole, error)
	List(ctx context.Context, pageSize, current int32, roleName *string, status *int32) ([]*model.SysRole, int32, error)
	Create(ctx context.Context, role *model.SysRole) error
	Update(ctx context.Context, role *do.SysRole) error
	UpSert(ctx context.Context, role *model.SysRole) error
	Delete(ctx context.Context, id int64) error
}

// SystemRoleMenuRepo 定义角色需要的菜单数据访问接口
type SystemRoleMenuRepo interface {
	Get(ctx context.Context, id int64) (*model.SysMenu, error)
	ListByIDs(ctx context.Context, ids []int64) ([]*model.SysMenu, error)
	List(ctx context.Context, current int32, pageSize int32, name *string, isShow *int32, listAll bool) ([]*model.SysMenu, int64, error)
}

type SystemRoleUsecase struct {
	repo       SystemRoleRepo
	casbinRepo CasbinRuleRepo
	menuRepo   SystemRoleMenuRepo
	logger     clog.Logger
}

func NewSystemRoleUsecase(
	repo SystemRoleRepo,
	casbinRepo CasbinRuleRepo,
	menuRepo SystemRoleMenuRepo,
	logger clog.Logger,
) *SystemRoleUsecase {
	return &SystemRoleUsecase{
		repo:       repo,
		casbinRepo: casbinRepo,
		menuRepo:   menuRepo,
		logger:     logger,
	}
}

// Get 获取角色详情
func (uc *SystemRoleUsecase) Get(ctx context.Context, id int64) (*model.SysRole, error) {
	return uc.repo.Get(ctx, id)
}

// List 获取角色列表
func (uc *SystemRoleUsecase) List(ctx context.Context, pageSize, current int32, roleName *string, status *int32) ([]*model.SysRole, int32, error) {
	return uc.repo.List(ctx, pageSize, current, roleName, status)
}

// Create 创建角色
func (uc *SystemRoleUsecase) Create(ctx context.Context, role *model.SysRole) error {
	return uc.repo.Create(ctx, role)
}

// CreateWithMenu 创建角色并分配菜单权限
func (uc *SystemRoleUsecase) CreateWithMenu(ctx context.Context, role *model.SysRole, menuIDs []int64) error {
	// 1. 验证菜单存在
	if len(menuIDs) > 0 {
		for _, menuID := range menuIDs {
			_, err := uc.menuRepo.Get(ctx, menuID)
			if err != nil {
				return fmt.Errorf("menu ID %d not found: %w", menuID, err)
			}
		}
	}

	// 2. 创建角色
	if err := uc.repo.Create(ctx, role); err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	// 3. 分配菜单权限
	if len(menuIDs) > 0 {
		if err := uc.casbinRepo.SetRoleMenus(ctx, role.ID, menuIDs); err != nil {
			return fmt.Errorf("failed to set role menus: %w", err)
		}
	}

	return nil
}

// Update 更新角色
func (uc *SystemRoleUsecase) Update(ctx context.Context, role *do.SysRole) error {
	// 如果包含菜单ID，则更新菜单权限
	if len(role.MenuIds) > 0 {
		// 验证菜单存在
		for _, menuID := range role.MenuIds {
			_, err := uc.menuRepo.Get(ctx, menuID)
			if err != nil {
				return fmt.Errorf("menu ID %d not found: %w", menuID, err)
			}
		}

		// 更新菜单权限
		if err := uc.casbinRepo.SetRoleMenus(ctx, role.ID, role.MenuIds); err != nil {
			return fmt.Errorf("failed to set role menus: %w", err)
		}
	}

	// 更新角色基本信息
	return uc.repo.Update(ctx, role)
}

// Delete 删除角色
func (uc *SystemRoleUsecase) Delete(ctx context.Context, id int64) error {
	// 1. 删除角色的权限规则
	if err := uc.casbinRepo.DeleteRolePermissions(ctx, id); err != nil {
		uc.logger.Warn(ctx, "failed to delete role permissions", clog.Int64("role_id", id), clog.Err(err))
	}

	// 2. 删除角色记录
	return uc.repo.Delete(ctx, id)
}

// UpSert 插入或更新角色
func (uc *SystemRoleUsecase) UpSert(ctx context.Context, role *model.SysRole) error {
	return uc.repo.UpSert(ctx, role)
}

// GetMenuIdsByRoleId 获取角色的菜单ID列表
func (uc *SystemRoleUsecase) GetMenuIdsByRoleId(ctx context.Context, roleID int64) ([]int64, error) {
	return uc.casbinRepo.ListRoleMenuIDs(ctx, roleID)
}

// ListRolePermission 获取角色的权限规则列表
func (uc *SystemRoleUsecase) ListRolePermission(ctx context.Context, roleID int64) ([]*model.CasbinRule, error) {
	role, err := uc.repo.Get(ctx, roleID)
	if err != nil {
		return nil, err
	}

	menus, err := uc.ListRolePermissionMenus(ctx, roleID)
	if err != nil {
		return nil, err
	}

	rules := make([]*model.CasbinRule, 0, len(menus))
	for _, menu := range menus {
		if menu == nil || menu.Path == "" {
			continue
		}
		v0 := role.Code
		v1 := menu.Path
		rule := &model.CasbinRule{
			Ptype: strPtr("p"),
			V0:    &v0,
			V1:    &v1,
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// ListRolePermissionMenus 获取角色有权限的菜单列表
func (uc *SystemRoleUsecase) ListRolePermissionMenus(ctx context.Context, roleID int64) ([]*model.SysMenu, error) {
	// 获取角色的菜单ID列表
	menuIDs, err := uc.casbinRepo.ListRoleMenuIDs(ctx, roleID)
	if err != nil {
		return nil, err
	}

	if len(menuIDs) == 0 {
		return []*model.SysMenu{}, nil
	}

	menus, err := uc.menuRepo.ListByIDs(ctx, menuIDs)
	if err != nil {
		return nil, err
	}

	return menus, nil
}

// UpdateRoleMenus 更新角色的菜单权限
func (uc *SystemRoleUsecase) UpdateRoleMenus(ctx context.Context, roleID int64, menuIDs []int64) error {
	// 1. 验证角色存在
	_, err := uc.repo.Get(ctx, roleID)
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

// ListUserPermission 获取用户的权限菜单ID列表
func (uc *SystemRoleUsecase) ListUserPermission(ctx context.Context, userID int64) ([]int64, error) {
	return uc.casbinRepo.ListUserPermissionMenuIDs(ctx, userID)
}

// ListUserPermissionMenus 获取用户有权限的菜单列表
func (uc *SystemRoleUsecase) ListUserPermissionMenus(ctx context.Context, userID int64) ([]*model.SysMenu, error) {
	// 获取用户的菜单ID列表
	menuIDs, err := uc.casbinRepo.ListUserPermissionMenuIDs(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(menuIDs) == 0 {
		return []*model.SysMenu{}, nil
	}

	menus, err := uc.menuRepo.ListByIDs(ctx, menuIDs)
	if err != nil {
		return nil, err
	}

	return menus, nil
}

// UpdateRoleDataScope 更新角色的数据范围
func (uc *SystemRoleUsecase) UpdateRoleDataScope(ctx context.Context, roleID int64, dataScope int32, dataScopeDeptIDs []int64) error {
	role := &do.SysRole{
		ID:               roleID,
		DataScope:        &dataScope,
		DataScopeDeptIDs: dataScopeDeptIDs,
	}
	return uc.repo.Update(ctx, role)
}

// CheckPermission 检查角色是否有权限访问指定菜单
func (uc *SystemRoleUsecase) CheckPermission(ctx context.Context, roleID, menuID int64) (bool, error) {
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

// ListUserRoles 获取用户的角色ID列表
func (uc *SystemRoleUsecase) ListUserRoles(ctx context.Context, userID int64) ([]int64, error) {
	return uc.casbinRepo.ListUserRoleIDs(ctx, userID)
}

// UpdateUserRoles 更新用户的角色分配
func (uc *SystemRoleUsecase) UpdateUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	// 1. 验证所有角色存在
	if len(roleIDs) > 0 {
		for _, roleID := range roleIDs {
			_, err := uc.repo.Get(ctx, roleID)
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

// strPtr 辅助函数：返回字符串指针
func strPtr(s string) *string {
	return &s
}

// IsAdmin 检查角色是否为管理员（通过角色属性而非硬编码ID）
func (uc *SystemRoleUsecase) IsAdmin(ctx context.Context, roleID int64) (bool, error) {
	role, err := uc.repo.Get(ctx, roleID)
	if err != nil {
		return false, err
	}
	return role.IsAdmin == 1, nil
}

// IsUserAdmin 检查用户是否为管理员（检查用户的所有角色）
func (uc *SystemRoleUsecase) IsUserAdmin(ctx context.Context, userID int64) (bool, error) {
	roleIDs, err := uc.casbinRepo.ListUserRoleIDs(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, roleID := range roleIDs {
		isAdmin, err := uc.IsAdmin(ctx, roleID)
		if err != nil {
			continue
		}
		if isAdmin {
			return true, nil
		}
	}

	return false, nil
}

// CheckAPIPermission 检查指定角色是否有权限访问指定API路径
func (uc *SystemRoleUsecase) CheckAPIPermission(ctx context.Context, roleID int64, path string) (bool, error) {
	// 获取该角色拥有的所有菜单
	menus, err := uc.ListRolePermissionMenus(ctx, roleID)
	if err != nil {
		return false, err
	}

	// 遍历菜单，检查是否有匹配的路径
	for _, menu := range menus {
		if menu.Path == path {
			return true, nil
		}
	}

	return false, nil
}
