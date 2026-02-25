package system

import (
	"context"

	pb "spark/api/system/v1"
	"spark/internal/biz/system"
	"spark/internal/data/dal/model"
	"spark/internal/data/do"
	"spark/pkg/viewer"
)

type MenuService struct {
	pb.UnimplementedMenuServer
	menuUc *system.SystemMenuUsecase
}

func NewMenuService(menuUc *system.SystemMenuUsecase) *MenuService {
	return &MenuService{menuUc: menuUc}
}

func (s *MenuService) CreateMenu(ctx context.Context, req *pb.CreateMenuRequest) (*pb.CreateMenuReply, error) {
	userView := viewer.MustGetUserViewFromContext(ctx)
	user := userView.GetUser()

	menu := &model.SysMenu{
		Pid:        req.Pid,
		Name:       req.Name,
		Title:      req.Title,
		Icon:       req.Icon,
		Condition:  req.Condition,
		Remark:     req.Remark,
		MenuType:   req.MenuType,
		Weight:     req.Weight,
		IsShow:     req.IsShow,
		Path:       req.Path,
		Component:  req.Component,
		IsLink:     req.IsLink,
		ModuleType: req.ModuleType,
		ModelID:    req.ModelId,
		IsIframe:   req.IsIframe,
		IsCached:   req.IsCached,
		Redirect:   req.Redirect,
		IsAffix:    req.IsAffix,
		LinkURL:    req.LinkUrl,
		BaseModelNoSoftDelete: model.BaseModelNoSoftDelete{
			CreatedBy: &user.ID,
			UpdatedBy: &user.ID,
		},
	}

	err := s.menuUc.Create(ctx, menu)
	if err != nil {
		return nil, err
	}

	return &pb.CreateMenuReply{}, nil
}

func (s *MenuService) UpdateMenu(ctx context.Context, req *pb.UpdateMenuRequest) (*pb.UpdateMenuReply, error) {
	menu := &do.SysMenu{
		ID:         req.Id,
		Name:       req.Name,
		Title:      req.Title,
		Icon:       req.Icon,
		Condition:  req.Condition,
		Remark:     req.Remark,
		MenuType:   req.MenuType,
		Weight:     req.Weight,
		IsShow:     req.IsShow,
		Path:       req.Path,
		Component:  req.Component,
		IsLink:     req.IsLink,
		ModuleType: req.ModuleType,
		ModelID:    req.ModelId,
		IsIframe:   req.IsIframe,
		IsCached:   req.IsCached,
		Redirect:   req.Redirect,
		IsAffix:    req.IsAffix,
		LinkUrl:    req.LinkUrl,
	}

	err := s.menuUc.Update(ctx, menu)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateMenuReply{}, nil
}

func (s *MenuService) DeleteMenu(ctx context.Context, req *pb.DeleteMenuRequest) (*pb.DeleteMenuReply, error) {
	err := s.menuUc.Delete(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteMenuReply{}, nil
}

func (s *MenuService) GetMenu(ctx context.Context, req *pb.GetMenuRequest) (*pb.GetMenuReply, error) {
	menuInfo, err := s.menuUc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	menu := convertMenuInfoToPb(menuInfo)

	return &pb.GetMenuReply{
		Menu: menu,
	}, nil
}

func (s *MenuService) ListMenu(ctx context.Context, req *pb.ListMenuRequest) (*pb.ListMenuReply, error) {
	menus, total, err := s.menuUc.List(ctx, req.CurrentPage, req.PageSize, req.Name, req.IsShow)
	if err != nil {
		return nil, err
	}

	menuList := make([]*pb.MenuInfo, 0, len(menus))
	for _, menuInfo := range menus {
		menu := convertMenuInfoToPb(menuInfo)
		menuList = append(menuList, menu)
	}

	return &pb.ListMenuReply{
		Items: menuList,
		Total: total,
	}, nil
}

func (s *MenuService) ListAllMenu(ctx context.Context, req *pb.ListAllMenuRequest) (*pb.ListAllMenuReply, error) {
	menus, err := s.menuUc.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	menuList := make([]*pb.MenuInfo, 0, len(menus))
	for _, menuInfo := range menus {
		menu := convertMenuInfoToPb(menuInfo)
		menuList = append(menuList, menu)
	}

	return &pb.ListAllMenuReply{
		Items: menuList,
	}, nil
}

// convertMenuInfoToPb converts a do.MenuInfo to a pb.MenuInfo
func convertMenuInfoToPb(menuInfo *do.MenuInfo) *pb.MenuInfo {
	menu := &pb.MenuInfo{
		Id:         menuInfo.ID,
		Pid:        menuInfo.Pid,
		Name:       menuInfo.Name,
		Title:      menuInfo.Title,
		Icon:       menuInfo.Icon,
		Condition:  menuInfo.Condition,
		Remark:     menuInfo.Remark,
		MenuType:   menuInfo.MenuType,
		Weight:     menuInfo.Weight,
		IsShow:     menuInfo.IsShow,
		Path:       menuInfo.Path,
		Component:  menuInfo.Component,
		IsLink:     menuInfo.IsLink,
		ModuleType: menuInfo.ModuleType,
		ModelId:    menuInfo.ModelID,
		IsIframe:   menuInfo.IsIframe,
		IsCached:   menuInfo.IsCached,
		Redirect:   menuInfo.Redirect,
		IsAffix:    menuInfo.IsAffix,
		LinkUrl:    menuInfo.LinkURL,
		CreatedAt:  menuInfo.CreatedAt,
		UpdatedAt:  menuInfo.UpdatedAt,
	}

	// Handle nullable fields
	if menuInfo.CreatedBy != nil {
		menu.CreatedBy = *menuInfo.CreatedBy
	}
	if menuInfo.UpdatedBy != nil {
		menu.UpdatedBy = *menuInfo.UpdatedBy
	}

	// Convert children recursively
	if len(menuInfo.Children) > 0 {
		children := make([]*pb.MenuInfo, 0, len(menuInfo.Children))
		for _, child := range menuInfo.Children {
			children = append(children, convertMenuInfoToPb(child))
		}
		menu.Children = children
	}

	return menu
}
