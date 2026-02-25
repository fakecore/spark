package system

import (
	"context"
	"fmt"
	"strconv"

	"spark/internal/biz/system"
	"spark/internal/data/dal/model"
	"spark/internal/data/dal/query"

	"spark/internal/data/do"
	clog "spark/pkg/clog"
	"spark/pkg/principal"
	"spark/pkg/utils"

	"gorm.io/gorm"
)

// Helper functions for string pointers
func strPtr(s string) *string {
	return &s
}

// Helper function to parse string to int64
func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

type systemUserRepo struct {
	db     *gorm.DB
	logger clog.Logger
	query  *query.Query
}

var _ system.SystemUserRepo = &systemUserRepo{}

func NewSystemUserRepo(db *gorm.DB, q *query.Query, logger clog.Logger) system.SystemUserRepo {
	return &systemUserRepo{
		db:     db,
		logger: logger,
		query:  q,
	}
}

func (r *systemUserRepo) FindByID(ctx context.Context, id int64) (*model.SysUser, error) {
	return r.query.SysUser.WithContext(ctx).Where(r.query.SysUser.ID.Eq(id)).First()
}

func (r *systemUserRepo) FindByName(ctx context.Context, name string) (*model.SysUser, error) {
	return r.query.SysUser.WithContext(ctx).Where(r.query.SysUser.Name.Eq(name)).First()
}

func (r *systemUserRepo) FindByMobile(ctx context.Context, mobile string) (*model.SysUser, error) {
	return r.query.SysUser.WithContext(ctx).Where(r.query.SysUser.Mobile.Eq(mobile)).First()
}

// Create 创建用户（只做数据库插入）
func (r *systemUserRepo) Create(ctx context.Context, user *model.SysUser) (*model.SysUser, error) {
	query := utils.GetQuery(ctx, r.db)
	userCtx := query.SysUser.WithContext(ctx)
	if err := userCtx.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

func (r *systemUserRepo) Update(ctx context.Context, user *do.SysUser) (*do.SysUser, error) {
	query := utils.GetQuery(ctx, r.db)
	userQuery := query.SysUser
	userQueryCtx := userQuery.WithContext(ctx)

	updatingMap := utils.StructToMap(user)
	_, err := userQueryCtx.Where(userQuery.ID.Eq(user.ID)).Updates(updatingMap)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *systemUserRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.query.SysUser.WithContext(ctx).Where(r.query.SysUser.ID.Eq(id)).Delete()
	return err
}

func (r *systemUserRepo) listDeptAndChildrenIDs(ctx context.Context, deptID int64) ([]int64, error) {
	depts, err := r.query.SysDept.WithContext(ctx).Where(
		r.query.SysDept.ID.Eq(deptID),
	).Or(
		r.query.SysDept.Ancestors.Like(fmt.Sprintf("%%,%d,%%", deptID)),
	).Or(
		r.query.SysDept.Ancestors.Like(fmt.Sprintf("%%,%d", deptID)),
	).Or(
		r.query.SysDept.Ancestors.Like(fmt.Sprintf("%d,%%", deptID)),
	).Find()
	if err != nil {
		return nil, fmt.Errorf("failed to get department IDs: %w", err)
	}

	deptIDs := make([]int64, 0, len(depts))
	for _, dept := range depts {
		if dept != nil {
			deptIDs = append(deptIDs, dept.ID)
		}
	}
	return deptIDs, nil
}

func (r *systemUserRepo) applyDataScope(ctx context.Context, userQuery query.ISysUserDo) (query.ISysUserDo, error) {
	p, ok := principal.FromContext(ctx)
	if !ok || p == nil || p.IsSuperAdmin {
		return userQuery, nil
	}

	switch p.Scope {
	case principal.ScopeAll:
		return userQuery, nil
	case principal.ScopeCustom:
		roleDepts, err := r.query.SysRoleDept.WithContext(ctx).Where(r.query.SysRoleDept.RoleID.Eq(p.RoleID)).Find()
		if err != nil {
			return nil, fmt.Errorf("failed to query custom data scope: %w", err)
		}
		if len(roleDepts) == 0 {
			return userQuery.Where(r.query.SysUser.ID.Eq(-1)), nil
		}

		deptIDs := make([]int64, 0, len(roleDepts))
		for _, item := range roleDepts {
			if item != nil {
				deptIDs = append(deptIDs, item.DeptID)
			}
		}
		return userQuery.Where(r.query.SysUser.DeptID.In(deptIDs...)), nil
	case principal.ScopeDept:
		if p.DeptID <= 0 {
			return userQuery.Where(r.query.SysUser.ID.Eq(-1)), nil
		}
		return userQuery.Where(r.query.SysUser.DeptID.Eq(p.DeptID)), nil
	case principal.ScopeDeptTree:
		if p.DeptID <= 0 {
			return userQuery.Where(r.query.SysUser.ID.Eq(-1)), nil
		}
		deptIDs, err := r.listDeptAndChildrenIDs(ctx, p.DeptID)
		if err != nil {
			return nil, err
		}
		if len(deptIDs) == 0 {
			return userQuery.Where(r.query.SysUser.ID.Eq(-1)), nil
		}
		return userQuery.Where(r.query.SysUser.DeptID.In(deptIDs...)), nil
	default:
		return userQuery.Where(r.query.SysUser.ID.Eq(p.UserID)), nil
	}
}

func (r *systemUserRepo) ListUser(ctx context.Context, current int32, pageSize int32, name *string, nickname *string, status *int32, deptId *int64) ([]*model.SysUser, int32, error) {
	userQuery := r.query.SysUser.WithContext(ctx)

	var err error
	userQuery, err = r.applyDataScope(ctx, userQuery)
	if err != nil {
		return nil, 0, err
	}

	// 应用过滤条件
	if name != nil && *name != "" {
		userQuery = userQuery.Where(r.query.SysUser.Name.Like("%" + *name + "%"))
	}
	if nickname != nil && *nickname != "" {
		userQuery = userQuery.Where(r.query.SysUser.Nickname.Like("%" + *nickname + "%"))
	}
	if status != nil {
		userQuery = userQuery.Where(r.query.SysUser.Status.Eq(*status))
	}
	// 处理部门过滤
	if deptId != nil && *deptId > 0 {
		deptIDs, err := r.listDeptAndChildrenIDs(ctx, *deptId)
		if err != nil {
			return nil, 0, err
		}

		if len(deptIDs) == 0 {
			userQuery = userQuery.Where(r.query.SysUser.ID.Eq(-1))
		} else {
			userQuery = userQuery.Where(r.query.SysUser.DeptID.In(deptIDs...))
		}
	}

	// 获取总数
	total, err := userQuery.Count()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// 执行分页查询获取用户基本信息
	users, err := userQuery.Scopes(utils.Paginate(current, pageSize)).Find()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}

	// 如果没有用户，直接返回空结果
	if len(users) == 0 {
		return users, int32(total), nil
	}

	return users, int32(total), nil
}
