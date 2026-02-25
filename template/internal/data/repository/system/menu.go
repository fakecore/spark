package system

import (
	"context"

	v1 "spark/api/common/v1"
	"spark/internal/biz/system"
	"spark/internal/data/dal/model"
	"spark/internal/data/dal/query"

	"spark/internal/data/do"
	clog "spark/pkg/clog"
	"spark/pkg/utils"
	"spark/pkg/viewer"
)

type systemMenuRepo struct {
	logger clog.Logger
	query  *query.Query
}

// NewSystemMenuRepo .
func NewSystemMenuRepo(q *query.Query, logger clog.Logger) system.SystemMenuRepo {
	return &systemMenuRepo{
		logger: logger,
		query:  q,
	}
}

func (r *systemMenuRepo) Get(ctx context.Context, id int64) (*model.SysMenu, error) {
	menuQuery := r.query.SysMenu
	menu, err := menuQuery.WithContext(ctx).Where(menuQuery.ID.Eq(id)).First()
	if err != nil {
		return nil, v1.ErrorSystemMenuError("get menu failed: %v", err)
	}
	return menu, nil
}

func (r *systemMenuRepo) ListByIDs(ctx context.Context, ids []int64) ([]*model.SysMenu, error) {
	if len(ids) == 0 {
		return []*model.SysMenu{}, nil
	}

	menuQuery := r.query.SysMenu
	menus, err := menuQuery.WithContext(ctx).Where(menuQuery.ID.In(ids...)).Find()
	if err != nil {
		return nil, v1.ErrorSystemMenuError("list menus by ids failed: %v", err)
	}
	return menus, nil
}

func (r *systemMenuRepo) List(ctx context.Context, current int32, pageSize int32, name *string, isShow *int32, listAll bool) ([]*model.SysMenu, int64, error) {
	menuQuery := r.query.SysMenu
	menuDo := menuQuery.WithContext(ctx)

	if name != nil && *name != "" {
		menuDo = menuDo.Where(menuQuery.Name.Like("%" + *name + "%"))
	}
	if isShow != nil {
		menuDo = menuDo.Where(menuQuery.IsShow.Eq(*isShow))
	}
	total, err := menuDo.Count()
	if err != nil {
		return nil, 0, v1.ErrorSystemMenuError("list menu failed: %v", err)
	}

	var menus []*model.SysMenu
	if listAll {
		menus, err = menuDo.Find()
	} else {
		menus, err = menuDo.Scopes(utils.Paginate(current, pageSize)).Find()
	}
	if err != nil {
		return nil, 0, v1.ErrorSystemMenuError("list menu failed: %v", err)
	}
	return menus, total, nil
}

func (r *systemMenuRepo) Create(ctx context.Context, menu *model.SysMenu) error {
	menuQuery := r.query.SysMenu
	return menuQuery.WithContext(ctx).Create(menu)
}

func (r *systemMenuRepo) Update(ctx context.Context, menu *do.SysMenu) error {
	menuQuery := r.query.SysMenu

	updateData := map[string]interface{}{
		"updated_by": viewer.MustGetUserViewFromContext(ctx).GetUser().ID,
	}

	// 只更新非空字段
	if menu.Pid != nil {
		updateData["pid"] = menu.Pid
	}
	if menu.Name != nil {
		updateData["name"] = menu.Name
	}
	if menu.Title != nil {
		updateData["title"] = menu.Title
	}
	if menu.Icon != nil {
		updateData["icon"] = menu.Icon
	}
	if menu.Condition != nil {
		updateData["condition"] = menu.Condition
	}
	if menu.Remark != nil {
		updateData["remark"] = menu.Remark
	}
	if menu.MenuType != nil {
		updateData["menu_type"] = menu.MenuType
	}
	if menu.Weight != nil {
		updateData["weight"] = menu.Weight
	}
	if menu.IsShow != nil {
		updateData["is_show"] = menu.IsShow
	}
	if menu.Path != nil {
		updateData["path"] = menu.Path
	}
	if menu.Component != nil {
		updateData["component"] = menu.Component
	}
	if menu.IsLink != nil {
		updateData["is_link"] = menu.IsLink
	}
	if menu.ModuleType != nil {
		updateData["module_type"] = menu.ModuleType
	}
	if menu.ModelID != nil {
		updateData["model_id"] = menu.ModelID
	}
	if menu.IsIframe != nil {
		updateData["is_iframe"] = menu.IsIframe
	}
	if menu.IsCached != nil {
		updateData["is_cached"] = menu.IsCached
	}
	if menu.Redirect != nil {
		updateData["redirect"] = menu.Redirect
	}
	if menu.IsAffix != nil {
		updateData["is_affix"] = menu.IsAffix
	}
	if menu.LinkUrl != nil {
		updateData["link_url"] = menu.LinkUrl
	}

	_, err := menuQuery.WithContext(ctx).Where(menuQuery.ID.Eq(menu.ID)).Updates(updateData)
	return err
}

func (r *systemMenuRepo) Delete(ctx context.Context, id int64) error {
	menuQuery := r.query.SysMenu

	// 查询所有需要删除的菜单ID（包括子菜单）
	var menuIds []int64
	err := menuQuery.WithContext(ctx).
		Or(menuQuery.ID.Eq(id)).
		Or(menuQuery.Pid.Eq(id)).
		Pluck(menuQuery.ID, &menuIds)
	if err != nil {
		return v1.ErrorSystemMenuError("query menu ids failed: %v", err)
	}

	// 如果没有找到任何菜单，返回错误
	if len(menuIds) == 0 {
		return v1.ErrorSystemMenuError("menu not found")
	}

	// 批量删除所有相关菜单
	_, err = menuQuery.WithContext(ctx).Where(
		menuQuery.ID.In(menuIds...),
	).Delete()
	if err != nil {
		return v1.ErrorSystemMenuError("delete menu failed: %v", err)
	}
	return nil
}

func (r *systemMenuRepo) GetMenuIDByPath(ctx context.Context, path string) (int64, error) {
	menuQuery := r.query.SysMenu
	menu, err := menuQuery.WithContext(ctx).Where(menuQuery.Path.Eq(path)).First()
	if err != nil {
		return 0, v1.ErrorSystemMenuError("menu not found")
	}
	return menu.ID, nil
}
