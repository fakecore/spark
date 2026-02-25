package system

import (
	"context"
	"sort"

	"spark/internal/data/dal/model"
	"spark/internal/data/do"
)

type SystemMenuRepo interface {
	Get(ctx context.Context, id int64) (*model.SysMenu, error)
	ListByIDs(ctx context.Context, ids []int64) ([]*model.SysMenu, error)
	List(ctx context.Context, current int32, pageSize int32, name *string, isShow *int32, listAll bool) ([]*model.SysMenu, int64, error)
	Create(ctx context.Context, menu *model.SysMenu) error
	Update(ctx context.Context, menu *do.SysMenu) error
	Delete(ctx context.Context, id int64) error
	GetMenuIDByPath(ctx context.Context, path string) (int64, error)
}

type SystemMenuUsecase struct {
	repo SystemMenuRepo
}

func NewSystemMenuUsecase(repo SystemMenuRepo) *SystemMenuUsecase {
	return &SystemMenuUsecase{repo: repo}
}

func (uc *SystemMenuUsecase) Get(ctx context.Context, id int64) (*do.MenuInfo, error) {
	menu, err := uc.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &do.MenuInfo{SysMenu: *menu}, nil
}

func (uc *SystemMenuUsecase) ListAll(ctx context.Context) ([]*do.MenuInfo, error) {
	menus, _, err := uc.repo.List(ctx, 1, 9999, nil, nil, true)
	if err != nil {
		return nil, err
	}
	menuInfos := make([]*do.MenuInfo, len(menus))
	for i, menu := range menus {
		menuInfos[i] = &do.MenuInfo{SysMenu: *menu}
	}
	return menuInfos, nil
}

func (uc *SystemMenuUsecase) List(ctx context.Context, current int32, pageSize int32, name *string, isShow *int32) ([]*do.MenuInfo, int64, error) {
	menus, total, err := uc.repo.List(ctx, current, pageSize, name, isShow, false)
	if err != nil {
		return nil, 0, err
	}
	menuInfos := make([]*do.MenuInfo, len(menus))
	for i, menu := range menus {
		menuInfos[i] = &do.MenuInfo{SysMenu: *menu}
	}
	return uc.buildMenuTree(menuInfos), total, nil
}

func (uc *SystemMenuUsecase) Create(ctx context.Context, menu *model.SysMenu) error {
	return uc.repo.Create(ctx, menu)
}

func (uc *SystemMenuUsecase) Update(ctx context.Context, menu *do.SysMenu) error {
	return uc.repo.Update(ctx, menu)
}

func (uc *SystemMenuUsecase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

// buildMenuTree 将平面的菜单列表构建成树形结构
func (uc *SystemMenuUsecase) buildMenuTree(menus []*do.MenuInfo) []*do.MenuInfo {
	// 创建一个map用于快速查找菜单
	menuMap := make(map[int64]*do.MenuInfo)
	for _, menu := range menus {
		menuMap[menu.ID] = menu
		// 初始化 Children 切片
		menu.Children = make([]*do.MenuInfo, 0)
	}

	var roots []*do.MenuInfo
	// 构建树形结构
	for _, menu := range menus {
		if parent, exists := menuMap[menu.Pid]; exists {
			parent.Children = append(parent.Children, menu)
		} else {
			// 如果找不到父节点，则作为根节点
			roots = append(roots, menu)
		}
	}

	// 递归排序所有层级的菜单
	var sortMenus func([]*do.MenuInfo)
	sortMenus = func(items []*do.MenuInfo) {
		// 按权重排序，权重大的排在前面
		sort.Slice(items, func(i, j int) bool {
			return items[i].Weight > items[j].Weight
		})
		// 递归排序子菜单
		for _, item := range items {
			if len(item.Children) > 0 {
				sortMenus(item.Children)
			}
		}
	}

	sortMenus(roots)
	return roots
}

func (uc *SystemMenuUsecase) GetMenuIDByPath(ctx context.Context, path string) (int64, error) {
	return uc.repo.GetMenuIDByPath(ctx, path)
}
