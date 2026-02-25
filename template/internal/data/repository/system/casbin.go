package system

import (
	"context"
	"fmt"
	"slices"

	"spark/internal/biz/system"
	"spark/internal/data/dal/model"
	"spark/internal/data/dal/query"
	clog "spark/pkg/clog"
	"spark/pkg/utils"

	"gorm.io/gorm"
)

type casbinRuleRepo struct {
	db     *gorm.DB
	logger clog.Logger
	query  *query.Query
}

var _ system.CasbinRuleRepo = &casbinRuleRepo{}

func NewCasbinRuleRepo(db *gorm.DB, q *query.Query, logger clog.Logger) system.CasbinRuleRepo {
	return &casbinRuleRepo{
		db:     db,
		logger: logger,
		query:  q,
	}
}

func (r *casbinRuleRepo) getRoleByID(ctx context.Context, roleID int64) (*model.SysRole, error) {
	query := utils.GetQuery(ctx, r.db)
	return query.SysRole.WithContext(ctx).Where(query.SysRole.ID.Eq(roleID)).First()
}

func (r *casbinRuleRepo) getMenuPathsByIDs(ctx context.Context, menuIDs []int64) (map[int64]string, error) {
	if len(menuIDs) == 0 {
		return map[int64]string{}, nil
	}

	query := utils.GetQuery(ctx, r.db)
	menus, err := query.SysMenu.WithContext(ctx).Where(query.SysMenu.ID.In(menuIDs...)).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query menu paths: %w", err)
	}

	result := make(map[int64]string, len(menus))
	for _, menu := range menus {
		if menu != nil && menu.Path != "" {
			result[menu.ID] = menu.Path
		}
	}
	return result, nil
}

func (r *casbinRuleRepo) getMenuIDsByPaths(ctx context.Context, paths []string) ([]int64, error) {
	if len(paths) == 0 {
		return []int64{}, nil
	}

	query := utils.GetQuery(ctx, r.db)
	menus, err := query.SysMenu.WithContext(ctx).Where(query.SysMenu.Path.In(paths...)).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query menus by path: %w", err)
	}

	menuIDs := make([]int64, 0, len(menus))
	for _, menu := range menus {
		if menu != nil {
			menuIDs = append(menuIDs, menu.ID)
		}
	}
	return menuIDs, nil
}

// AssignUserRole implements system.CasbinRuleRepo.
func (r *casbinRuleRepo) AssignUserRole(ctx context.Context, userID, roleID int64) error {
	role, err := r.getRoleByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to load role: %w", err)
	}

	rule := &model.CasbinRule{
		Ptype: strPtr("g"),
		V0:    strPtr(fmt.Sprintf("%d", userID)),
		V1:    strPtr(role.Code),
	}

	err = r.query.CasbinRule.WithContext(ctx).Create(rule)
	if err != nil {
		return fmt.Errorf("failed to create casbin g rule: %w", err)
	}
	return nil
}

// RevokeUserRole implements system.CasbinRuleRepo.
func (r *casbinRuleRepo) RevokeUserRole(ctx context.Context, userID, roleID int64) error {
	role, err := r.getRoleByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to load role: %w", err)
	}

	result, err := r.query.CasbinRule.WithContext(ctx).
		Where(r.query.CasbinRule.Ptype.Eq("g")).
		Where(r.query.CasbinRule.V0.Eq(fmt.Sprintf("%d", userID))).
		Where(r.query.CasbinRule.V1.Eq(role.Code)).
		Delete()

	if err != nil {
		return fmt.Errorf("failed to delete casbin g rule: %w", err)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no role assignment found for user %d and role %d", userID, roleID)
	}

	return nil
}

// DeleteUserRoles implements system.CasbinRuleRepo.
func (r *casbinRuleRepo) DeleteUserRoles(ctx context.Context, userID int64) error {
	_, err := r.query.CasbinRule.WithContext(ctx).
		Where(r.query.CasbinRule.Ptype.Eq("g")).
		Where(r.query.CasbinRule.V0.Eq(fmt.Sprintf("%d", userID))).
		Delete()

	if err != nil {
		return fmt.Errorf("failed to delete user roles: %w", err)
	}
	return nil
}

// ListUserRoleIDs implements system.CasbinRuleRepo.
func (r *casbinRuleRepo) ListUserRoleIDs(ctx context.Context, userID int64) ([]int64, error) {
	rules, err := r.query.CasbinRule.WithContext(ctx).
		Where(r.query.CasbinRule.Ptype.Eq("g")).
		Where(r.query.CasbinRule.V0.Eq(fmt.Sprintf("%d", userID))).
		Find()

	if err != nil {
		return nil, fmt.Errorf("failed to query casbin rules: %w", err)
	}

	roleCodes := make([]string, 0, len(rules))
	for _, rule := range rules {
		if rule.V1 != nil && *rule.V1 != "" {
			roleCodes = append(roleCodes, *rule.V1)
		}
	}

	if len(roleCodes) == 0 {
		return []int64{}, nil
	}

	roles, err := r.query.SysRole.WithContext(ctx).Where(r.query.SysRole.Code.In(roleCodes...)).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to query roles by code: %w", err)
	}

	roleIDs := make([]int64, 0, len(roles))
	for _, role := range roles {
		if role != nil && !slices.Contains(roleIDs, role.ID) {
			roleIDs = append(roleIDs, role.ID)
		}
	}
	return roleIDs, nil
}

// SetRoleMenus implements system.CasbinRuleRepo.
func (r *casbinRuleRepo) SetRoleMenus(ctx context.Context, roleID int64, menuIDs []int64) error {
	query := utils.GetQuery(ctx, r.db)

	role, err := query.SysRole.WithContext(ctx).Where(query.SysRole.ID.Eq(roleID)).First()
	if err != nil {
		return fmt.Errorf("failed to load role: %w", err)
	}

	menuPaths := map[int64]string{}
	if len(menuIDs) > 0 {
		menus, err := query.SysMenu.WithContext(ctx).Where(query.SysMenu.ID.In(menuIDs...)).Find()
		if err != nil {
			return fmt.Errorf("failed to query menu paths: %w", err)
		}
		for _, menu := range menus {
			if menu != nil && menu.Path != "" {
				menuPaths[menu.ID] = menu.Path
			}
		}
	}

	tx, ok := utils.GetTx(ctx)
	ownsTx := false
	if !ok || tx == nil {
		tx = r.db.WithContext(ctx).Begin()
		if tx.Error != nil {
			return fmt.Errorf("failed to begin transaction: %w", tx.Error)
		}
		ownsTx = true
		ctx = utils.WithTx(ctx, tx)
	}

	defer func() {
		if rec := recover(); rec != nil && ownsTx {
			tx.Rollback()
		}
	}()

	if err := tx.Where("ptype = 'p' AND v0 = ?", role.Code).Delete(&model.CasbinRule{}).Error; err != nil {
		if ownsTx {
			tx.Rollback()
		}
		return fmt.Errorf("failed to delete existing permissions: %w", err)
	}

	for _, menuID := range menuIDs {
		menuPath, ok := menuPaths[menuID]
		if !ok {
			continue
		}
		rule := &model.CasbinRule{
			Ptype: strPtr("p"),
			V0:    strPtr(role.Code),
			V1:    strPtr(menuPath),
		}
		if err := tx.Create(rule).Error; err != nil {
			if ownsTx {
				tx.Rollback()
			}
			return fmt.Errorf("failed to create permission rule: %w", err)
		}
	}

	if ownsTx {
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
}

func (r *casbinRuleRepo) ListRoleMenuIDs(ctx context.Context, roleID int64) ([]int64, error) {
	role, err := r.getRoleByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to load role: %w", err)
	}

	query := utils.GetQuery(ctx, r.db)
	rules, err := query.CasbinRule.WithContext(ctx).
		Where(query.CasbinRule.Ptype.Eq("p")).
		Where(query.CasbinRule.V0.Eq(role.Code)).
		Find()

	if err != nil {
		return nil, fmt.Errorf("failed to query casbin rules: %w", err)
	}

	paths := make([]string, 0, len(rules))
	for _, rule := range rules {
		if rule.V1 != nil && *rule.V1 != "" {
			paths = append(paths, *rule.V1)
		}
	}

	return r.getMenuIDsByPaths(ctx, paths)
}

// DeleteRolePermissions implements system.CasbinRuleRepo.
func (r *casbinRuleRepo) DeleteRolePermissions(ctx context.Context, roleID int64) error {
	role, err := r.getRoleByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to load role: %w", err)
	}

	query := utils.GetQuery(ctx, r.db)
	_, err = query.CasbinRule.WithContext(ctx).
		Where(query.CasbinRule.Ptype.Eq("p")).
		Where(query.CasbinRule.V0.Eq(role.Code)).
		Delete()

	if err != nil {
		return fmt.Errorf("failed to delete role permissions: %w", err)
	}
	return nil
}

// ListUserPermissionMenuIDs implements system.CasbinRuleRepo.
func (r *casbinRuleRepo) ListUserPermissionMenuIDs(ctx context.Context, userID int64) ([]int64, error) {
	var menuPaths []string
	err := utils.GetDB(ctx, r.db).Raw(`
		SELECT DISTINCT cr_p.v1 FROM casbin_rule cr_g
		JOIN casbin_rule cr_p ON cr_g.v1 = cr_p.v0
		WHERE cr_g.ptype = 'g'
		AND cr_p.ptype = 'p'
		AND cr_g.v0 = ?
		AND cr_p.v1 != ''
	`, fmt.Sprintf("%d", userID)).Scan(&menuPaths).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query user permissions: %w", err)
	}

	return r.getMenuIDsByPaths(ctx, menuPaths)
}
