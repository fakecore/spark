package system

import (
	"context"
	"fmt"
	"slices"

	"spark/internal/biz/system"
	"spark/internal/common/constants"
	"spark/internal/data/dal/model"
	"spark/internal/data/dal/query"

	"spark/internal/data/do"
	clog "spark/pkg/clog"
	"spark/pkg/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type systemRoleRepo struct {
	db     *gorm.DB
	logger clog.Logger
	query  *query.Query
}

var _ system.SystemRoleRepo = &systemRoleRepo{}

func NewSystemRoleRepo(db *gorm.DB, q *query.Query, logger clog.Logger) system.SystemRoleRepo {
	return &systemRoleRepo{
		db:     db,
		logger: logger,
		query:  q,
	}
}

// UpSert implements system.SystemRoleRepo.
func (s *systemRoleRepo) UpSert(ctx context.Context, role *model.SysRole) error {
	roleQuery := s.query.SysRole.WithContext(ctx)
	err := roleQuery.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "sort", "name", "code", "data_scope", "remark"}),
	}).Create(role)
	if err != nil {
		return fmt.Errorf("upsert failed: %w", err)
	}
	return nil
}

// Create implements system.SystemRoleRepo.
func (s *systemRoleRepo) Create(ctx context.Context, role *model.SysRole) error {
	if role.ID != 0 {
		return fmt.Errorf("invalid ID %d", role.ID)
	}
	roleQuery := s.query.SysRole.WithContext(ctx)

	err := roleQuery.Create(role)
	if err != nil {
		return fmt.Errorf("create failed: %w", err)
	}
	return nil
}

func (s *systemRoleRepo) ListByIDs(ctx context.Context, ids []int64) ([]*model.SysRole, error) {
	if len(ids) == 0 {
		return []*model.SysRole{}, nil
	}

	roles, err := s.query.SysRole.WithContext(ctx).Where(s.query.SysRole.ID.In(ids...)).Find()
	if err != nil {
		return nil, fmt.Errorf("query roles by ids failed: %w", err)
	}

	ordered := make([]*model.SysRole, 0, len(ids))
	for _, id := range ids {
		for _, role := range roles {
			if role != nil && role.ID == id && !slices.Contains(ordered, role) {
				ordered = append(ordered, role)
				break
			}
		}
	}
	return ordered, nil
}

// Delete implements system.SystemRoleRepo.
func (s *systemRoleRepo) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid ID %d", id)
	}

	result, err := s.query.SysRole.WithContext(ctx).Where(s.query.SysRole.ID.Eq(id)).Delete()
	if err != nil {
		return fmt.Errorf("delete role failed: %w", err)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: role with ID %d", gorm.ErrRecordNotFound, id)
	}

	return nil
}

// Update implements system.SystemRoleRepo.
func (s *systemRoleRepo) Update(ctx context.Context, role *do.SysRole) error {
	if role.ID == 0 {
		return fmt.Errorf("invalid ID %d", role.ID)
	}

	tx, ok := utils.GetTx(ctx)
	ownsTx := false
	if !ok || tx == nil {
		tx = s.db.WithContext(ctx).Begin()
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

	// 只更新角色基本字段
	updatingMap := utils.StructToMap(role, utils.IgnoreFieldsWithDefault(), utils.IgnoreFields("MenuIds", "DataScopeDeptIDs"))
	if len(updatingMap) > 0 {
		if err := tx.Model(&model.SysRole{}).Where("id = ?", role.ID).Updates(updatingMap).Error; err != nil {
			if ownsTx {
				tx.Rollback()
			}
			return fmt.Errorf("failed to update role info: %w", err)
		}
	}

	if role.DataScope != nil || role.DataScopeDeptIDs != nil {
		if err := tx.Where("role_id = ?", role.ID).Delete(&model.SysRoleDept{}).Error; err != nil {
			if ownsTx {
				tx.Rollback()
			}
			return fmt.Errorf("failed to clear role dept scope: %w", err)
		}

		if role.DataScope != nil && *role.DataScope == constants.DataScope_Custom {
			for _, deptID := range role.DataScopeDeptIDs {
				scope := &model.SysRoleDept{
					RoleID: role.ID,
					DeptID: deptID,
				}
				if err := tx.Create(scope).Error; err != nil {
					if ownsTx {
						tx.Rollback()
					}
					return fmt.Errorf("failed to persist role dept scope: %w", err)
				}
			}
		}
	}

	if ownsTx {
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("failed to commit role update: %w", err)
		}
	}

	return nil
}

// Get implements system.SystemRoleRepo.
func (s *systemRoleRepo) Get(ctx context.Context, id int64) (*model.SysRole, error) {
	role, err := s.query.SysRole.WithContext(ctx).Where(s.query.SysRole.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("%w: role with ID %d", err, id)
		}
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return role, nil
}

// List implements system.SystemRoleRepo.
func (s *systemRoleRepo) List(ctx context.Context, pageSize int32, current int32, roleName *string, status *int32) ([]*model.SysRole, int32, error) {
	roleQuery := s.query.SysRole.WithContext(ctx)
	if roleName != nil {
		roleQuery = roleQuery.Where(s.query.SysRole.Name.Like("%" + *roleName + "%"))
	}
	if status != nil {
		roleQuery = roleQuery.Where(s.query.SysRole.Status.Eq(*status))
	}

	var total int64
	result, err := roleQuery.Count()
	if err != nil {
		return nil, 0, fmt.Errorf("count failed: %w", err)
	}
	total = result

	roles, err := roleQuery.Scopes(utils.Paginate(current, pageSize)).Find()
	if err != nil {
		return nil, 0, fmt.Errorf("query failed: %w", err)
	}
	return roles, int32(total), nil
}
