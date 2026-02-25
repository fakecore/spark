package system

import (
	"context"
	"fmt"

	v1 "spark/api/common/v1"
	"spark/internal/biz/system"
	"spark/internal/data/dal/model"
	"spark/internal/data/dal/query"

	"spark/internal/data/do"
	clog "spark/pkg/clog"
	"spark/pkg/utils"
)

type systemDeptRepo struct {
	logger clog.Logger
	query  *query.Query
}

// NewSystemDeptRepo .
func NewSystemDeptRepo(q *query.Query, logger clog.Logger) system.DeptRepo {
	return &systemDeptRepo{
		logger: logger,
		query:  q,
	}
}

func (r *systemDeptRepo) Create(ctx context.Context, dept *model.SysDept) error {
	deptQuery := r.query.SysDept

	// 检查同级是否有同名部门
	exists, err := deptQuery.WithContext(ctx).Where(
		deptQuery.ParentID.Eq(dept.ParentID),
		deptQuery.DeptName.Eq(dept.DeptName),
	).First()

	if err == nil && exists != nil {
		return v1.ErrorSystemDeptError("部门名称已存在")
	}

	// 如果是非顶级部门，检查父部门是否存在
	if dept.ParentID != 0 {
		parent, err := deptQuery.WithContext(ctx).Where(
			deptQuery.ID.Eq(dept.ParentID),
		).First()
		if err != nil || parent == nil {
			return v1.ErrorSystemDeptError("父部门不存在")
		}

		// 使用辅助函数构建ancestors
		ancestors, err := r.buildDeptAncestors(ctx, dept.ParentID)
		if err != nil {
			return v1.ErrorSystemDeptError("构建部门ancestors失败: %v", err)
		}
		dept.Ancestors = ancestors
	} else {
		// 根部门ancestors为空
		dept.Ancestors = ""
	}

	return deptQuery.WithContext(ctx).Create(dept)
}

func (r *systemDeptRepo) Update(ctx context.Context, dept *do.SysDept) error {
	deptQuery := r.query.SysDept

	// 检查更新的部门是否存在
	oldDept, err := deptQuery.WithContext(ctx).Where(
		deptQuery.ID.Eq(dept.ID),
	).First()
	if err != nil {
		return v1.ErrorSystemDeptError("部门不存在")
	}

	// 如果更改了父部门，检查新父部门是否存在并重新生成祖级列表
	if dept.ParentID != nil && *dept.ParentID != oldDept.ParentID {
		if *dept.ParentID != 0 {
			parent, err := deptQuery.WithContext(ctx).Where(
				deptQuery.ID.Eq(*dept.ParentID),
			).First()
			if err != nil || parent == nil {
				return v1.ErrorSystemDeptError("父部门不存在")
			}

			// 确保不会形成循环引用
			if *dept.ParentID == dept.ID {
				return v1.ErrorSystemDeptError("上级部门不能是自己")
			}

			// 使用辅助函数构建ancestors
			ancestors, err := r.buildDeptAncestors(ctx, *dept.ParentID)
			if err != nil {
				return v1.ErrorSystemDeptError("构建部门ancestors失败: %v", err)
			}
			dept.Ancestors = &ancestors
		} else {
			// 根部门ancestors为空
			defaultA := ""
			dept.Ancestors = &defaultA
		}
	}

	// 如果更改了名称，检查同级是否有同名部门
	if dept.DeptName != nil && *dept.DeptName != oldDept.DeptName {
		pid := oldDept.ParentID
		if dept.ParentID != nil {
			pid = *dept.ParentID
		}
		exists, err := deptQuery.WithContext(ctx).Where(
			deptQuery.ParentID.Eq(pid),
			deptQuery.DeptName.Eq(*dept.DeptName),
			deptQuery.ID.Neq(dept.ID), // 排除自身
		).First()
		if err == nil && exists != nil {
			return v1.ErrorSystemDeptError("部门名称已存在")
		}
	}

	// 构建更新数据
	updateData := make(map[string]interface{})
	if dept.ParentID != nil {
		updateData["parent_id"] = dept.ParentID
	}
	if dept.Ancestors != nil {
		updateData["ancestors"] = dept.Ancestors
	}
	if dept.DeptName != nil {
		updateData["dept_name"] = dept.DeptName
	}
	if dept.OrderNum != nil {
		updateData["order_num"] = dept.OrderNum
	}
	if dept.Leader != nil {
		updateData["leader"] = dept.Leader
	}
	if dept.Phone != nil {
		updateData["phone"] = dept.Phone
	}
	if dept.Email != nil {
		updateData["email"] = dept.Email
	}
	if dept.Status != nil {
		updateData["status"] = dept.Status
	}

	_, err = deptQuery.WithContext(ctx).Where(deptQuery.ID.Eq(dept.ID)).Updates(updateData)
	if err != nil {
		return v1.ErrorSystemDeptError("更新部门失败: %v", err)
	}

	// 如果更改了父部门，需要更新所有子部门的祖级列表
	if dept.ParentID != nil && *dept.ParentID != oldDept.ParentID {
		// 使用辅助函数递归更新所有子部门的ancestors（传递当前部门的新ancestors）
		if err := r.updateChildrenDeptAncestors(ctx, dept.ID, *dept.Ancestors); err != nil {
			return v1.ErrorSystemDeptError("更新子部门祖级列表失败: %v", err)
		}
	}

	return nil
}

func (r *systemDeptRepo) Delete(ctx context.Context, id int64) error {
	deptQuery := r.query.SysDept

	// 检查是否有子部门
	count, err := deptQuery.WithContext(ctx).Where(
		deptQuery.ParentID.Eq(id),
	).Count()
	if err != nil {
		return v1.ErrorSystemDeptError("检查子部门失败: %v", err)
	}
	if count > 0 {
		return v1.ErrorSystemDeptError("存在子部门，不能删除")
	}

	// 检查是否有用户关联
	userQuery := r.query.SysUser
	userCount, err := userQuery.WithContext(ctx).Where(
		userQuery.DeptID.Eq(id),
	).Count()
	if err != nil {
		return v1.ErrorSystemDeptError("检查部门用户失败: %v", err)
	}
	if userCount > 0 {
		return v1.ErrorSystemDeptError("部门存在用户，不能删除")
	}

	// 软删除部门
	_, err = deptQuery.WithContext(ctx).Where(
		deptQuery.ID.Eq(id),
	).Delete()
	if err != nil {
		return v1.ErrorSystemDeptError("删除部门失败: %v", err)
	}

	return nil
}

func (r *systemDeptRepo) Get(ctx context.Context, id int64) (*model.SysDept, error) {
	deptQuery := r.query.SysDept
	dept, err := deptQuery.WithContext(ctx).Where(deptQuery.ID.Eq(id)).First()
	if err != nil {
		return nil, v1.ErrorSystemDeptError("部门不存在")
	}
	return dept, nil
}

func (r *systemDeptRepo) List(ctx context.Context, current int32, pageSize int32, deptName *string, status *int32) ([]*model.SysDept, int64, error) {
	deptQuery := r.query.SysDept
	query := deptQuery.WithContext(ctx)

	if deptName != nil && *deptName != "" {
		query = query.Where(deptQuery.DeptName.Like("%" + *deptName + "%"))
	}
	if status != nil {
		query = query.Where(deptQuery.Status.Eq(*status))
	}
	total, err := query.Count()
	if err != nil {
		return nil, 0, v1.ErrorSystemDeptError("查询部门列表失败: %v", err)
	}

	// 按照父ID和排序号排序
	query = query.Order(deptQuery.ParentID, deptQuery.Sort.Desc())

	depts, err := query.Scopes(utils.Paginate(current, pageSize)).Find()
	if err != nil {
		return nil, 0, v1.ErrorSystemDeptError("查询部门列表失败: %v", err)
	}

	return depts, total, nil
}

// buildDeptAncestors 构建部门的ancestors字符串（格式：root为空，第二层为rootid，第三层为rootid,第二层id）
func (r *systemDeptRepo) buildDeptAncestors(ctx context.Context, parentID int64) (string, error) {
	if parentID == 0 {
		return "", nil // 根部门没有祖先
	}

	// 验证父部门ID的有效性
	if parentID < 0 {
		return "", fmt.Errorf("invalid parent ID: %d", parentID)
	}

	deptQuery := r.query.SysDept
	parent, err := deptQuery.WithContext(ctx).Where(deptQuery.ID.Eq(parentID)).First()
	if err != nil {
		return "", err
	}

	if parent.Ancestors == "" {
		// 父部门是根部门，当前部门ancestors = "rootid,"
		return fmt.Sprintf("%d,", parentID), nil
	} else {
		// 父部门有ancestors，当前部门ancestors = "parent的ancestors+parentID,"
		return fmt.Sprintf("%s%d,", parent.Ancestors, parentID), nil
	}
}

// updateChildrenDeptAncestors 更新所有子部门的ancestors字段（当父部门移动时）
func (r *systemDeptRepo) updateChildrenDeptAncestors(ctx context.Context, deptID int64, newAncestors string) error {
	deptQuery := r.query.SysDept

	// 获取所有直接子部门
	children, err := deptQuery.WithContext(ctx).Where(deptQuery.ParentID.Eq(deptID)).Find()
	if err != nil {
		return err
	}

	for _, child := range children {
		// 计算子部门的新ancestors（使用带尾部逗号格式）
		var childAncestors string
		if newAncestors == "" {
			// 父部门是根部门，子部门ancestors = "父部门ID,"
			childAncestors = fmt.Sprintf("%d,", deptID)
		} else {
			// 父部门有ancestors，子部门ancestors = "父部门的ancestors+父部门ID,"
			childAncestors = fmt.Sprintf("%s%d,", newAncestors, deptID)
		}

		// 更新子部门的ancestors
		updateData := map[string]interface{}{
			"ancestors": childAncestors,
		}

		_, err = deptQuery.WithContext(ctx).Where(deptQuery.ID.Eq(child.ID)).Updates(updateData)
		if err != nil {
			return err
		}

		// 递归更新孙部门
		err = r.updateChildrenDeptAncestors(ctx, child.ID, childAncestors)
		if err != nil {
			return err
		}
	}

	return nil
}
