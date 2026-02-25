package system

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	commonv1 "spark/api/common/v1"
	pb "spark/api/system/v1"
	"spark/internal/biz/system"
	"spark/internal/common/constants"
	"spark/internal/data/dal/model"
	"spark/internal/data/do"
	"spark/pkg/utils"
	"spark/pkg/viewer"
)

type RoleService struct {
	pb.UnimplementedRoleServer
	roleUc *system.SystemRoleUsecase
}

var _ pb.RoleServer = &RoleService{}

func NewRoleService(roleUc *system.SystemRoleUsecase) *RoleService {
	return &RoleService{roleUc: roleUc}
}

func (s *RoleService) ensureRoleWritable(ctx context.Context, roleID int64) error {
	role, err := s.roleUc.Get(ctx, roleID)
	if err != nil {
		return err
	}
	if constants.IsRoleReadonly(role.Code) {
		return commonv1.ErrorSystemRoleError("role %s is managed by system and cannot be modified", role.Code)
	}
	return nil
}

func generateRoleCode(name string) string {
	base := strings.ToLower(strings.TrimSpace(name))
	base = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(base, "_")
	base = strings.Trim(base, "_")
	if base == "" {
		base = "role"
	}
	return fmt.Sprintf("%s_%d", base, time.Now().UnixMilli())
}

func (s *RoleService) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (*pb.CreateRoleReply, error) {
	userView := viewer.MustGetUserViewFromContext(ctx)
	user := userView.GetUser()
	role := &model.SysRole{
		Status:    utils.GetDataOrDefault(req.Status, constants.RoleStatus_Active),
		Sort:      utils.GetDataOrDefault(req.Sort, 0),
		Name:      req.Name,
		Code:      generateRoleCode(req.Name),
		DataScope: utils.GetDataOrDefault(req.DataScope, constants.DataScope_Person),
		Remark:    req.Remark,
		BaseModelNoSoftDelete: model.BaseModelNoSoftDelete{
			CreatedBy: &user.ID,
			UpdatedBy: &user.ID,
		},
	}

	// 如果请求中包含菜单ID，则使用CreateWithMenu
	if len(req.MenuIds) > 0 {
		err := s.roleUc.CreateWithMenu(ctx, role, req.MenuIds)
		if err != nil {
			return nil, err
		}
	} else {
		err := s.roleUc.Create(ctx, role)
		if err != nil {
			return nil, err
		}
	}

	return &pb.CreateRoleReply{}, nil
}

func (s *RoleService) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (*pb.UpdateRoleReply, error) {
	userView := viewer.MustGetUserViewFromContext(ctx)
	user := userView.GetUser()
	if err := s.ensureRoleWritable(ctx, req.Id); err != nil {
		return nil, err
	}

	role := &do.SysRole{
		ID:        req.Id,
		Status:    req.Status,
		Sort:      req.Sort,
		Name:      req.Name,
		DataScope: req.DataScope,
		Remark:    req.Remark,
		MenuIds:   req.MenuIds,
		UpdatedBy: &user.ID,
	}

	err := s.roleUc.Update(ctx, role)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateRoleReply{}, nil
}

func (s *RoleService) DeleteRole(ctx context.Context, req *pb.DeleteRoleRequest) (*pb.DeleteRoleReply, error) {
	id := req.Id
	if err := s.ensureRoleWritable(ctx, id); err != nil {
		return nil, err
	}
	err := s.roleUc.Delete(ctx, id)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteRoleReply{}, nil
}

func (s *RoleService) GetRole(ctx context.Context, req *pb.GetRoleRequest) (*pb.GetRoleReply, error) {
	role, err := s.roleUc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	pbRole := &pb.RoleInfo{}
	utils.ObjConvert(role, pbRole)
	return &pb.GetRoleReply{Role: pbRole}, nil
}

func (s *RoleService) ListRole(ctx context.Context, req *pb.ListRoleRequest) (*pb.ListRoleReply, error) {
	roles, total, err := s.roleUc.List(ctx, req.PageSize, req.Current, req.Name, req.Status)
	if err != nil {
		return nil, err
	}
	pbRoles := make([]*pb.RoleInfo, 0)
	for _, role := range roles {
		pbRole := &pb.RoleInfo{}
		utils.ObjConvert(role, pbRole)
		pbRoles = append(pbRoles, pbRole)
	}
	return &pb.ListRoleReply{Items: pbRoles, Total: int32(total)}, nil
}

func (s *RoleService) ListRolePermission(ctx context.Context, req *pb.ListRolePermissionRequest) (*pb.ListRolePermissionReply, error) {
	menus, err := s.roleUc.ListRolePermissionMenus(ctx, req.RoleId)
	if err != nil {
		return nil, err
	}

	// 将菜单列表转换为 proto 格式
	pbMenus := make([]*pb.RoleMenuInfo, 0, len(menus))
	for _, menu := range menus {
		pbMenu := &pb.RoleMenuInfo{}
		utils.ObjConvert(menu, pbMenu)
		pbMenus = append(pbMenus, pbMenu)
	}

	return &pb.ListRolePermissionReply{
		Items: pbMenus,
	}, nil
}

func (s *RoleService) UpdateRoleMenus(ctx context.Context, req *pb.UpdateRoleMenusRequest) (*pb.UpdateRoleMenusReply, error) {
	if err := s.ensureRoleWritable(ctx, req.RoleId); err != nil {
		return nil, err
	}
	// 业务逻辑（包括存在性验证）委托给 usecase 层
	if err := s.roleUc.UpdateRoleMenus(ctx, req.RoleId, req.MenuIds); err != nil {
		return nil, err
	}

	return &pb.UpdateRoleMenusReply{}, nil
}

func (s *RoleService) UpdateRoleDataScope(ctx context.Context, req *pb.UpdateRoleDataScopeRequest) (*pb.UpdateRoleDataScopeReply, error) {
	if err := s.ensureRoleWritable(ctx, req.RoleId); err != nil {
		return nil, err
	}

	if err := s.roleUc.Update(ctx, &do.SysRole{
		ID:               req.RoleId,
		DataScope:        &req.DataScope,
		DataScopeDeptIDs: req.DataScopeDeptIds,
	}); err != nil {
		return nil, err
	}

	return &pb.UpdateRoleDataScopeReply{}, nil
}

func (s *RoleService) UpdateUserRole(ctx context.Context, req *pb.UpdateUserRoleRequest) (*pb.UpdateUserRoleReply, error) {
	// 验证基本参数
	if req.UserId <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", req.UserId)
	}

	// 使用roleUc更新用户角色
	if err := s.roleUc.UpdateUserRoles(ctx, req.UserId, req.RoleIds); err != nil {
		return nil, err
	}

	return &pb.UpdateUserRoleReply{}, nil
}

func (s *RoleService) ListRoleMenus(ctx context.Context, req *pb.ListRoleMenusRequest) (*pb.ListRoleMenusReply, error) {
	// 获取角色的菜单ID列表
	menuIds, err := s.roleUc.GetMenuIdsByRoleId(ctx, req.RoleId)
	if err != nil {
		return nil, err
	}

	return &pb.ListRoleMenusReply{
		MenuIds: menuIds,
	}, nil
}

func (s *RoleService) ListUserRoles(ctx context.Context, req *pb.ListUserRolesRequest) (*pb.ListUserRolesReply, error) {
	// 获取用户的角色ID列表
	roleIds, err := s.roleUc.ListUserRoles(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &pb.ListUserRolesReply{
		RoleIds: roleIds,
	}, nil
}

func (s *RoleService) CheckAPIPermission(ctx context.Context, req *pb.CheckAPIPermissionRequest) (*pb.CheckAPIPermissionReply, error) {
	// 参数校验在 service 层进行
	if req.RoleId <= 0 || req.Path == "" {
		return &pb.CheckAPIPermissionReply{
			HasPermission: false,
		}, nil
	}

	// 业务逻辑委托给 usecase 层
	hasPermission, err := s.roleUc.CheckAPIPermission(ctx, req.RoleId, req.Path)
	if err != nil {
		return nil, err
	}

	return &pb.CheckAPIPermissionReply{
		HasPermission: hasPermission,
	}, nil
}

func (s *RoleService) GetUserPermission(ctx context.Context, req *pb.GetUserPermissionRequest) (*pb.GetUserPermissionReply, error) {
	// 检查请求参数
	if req.UserId <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", req.UserId)
	}

	// 获取用户拥有的所有菜单ID
	menuIds, err := s.roleUc.ListUserPermission(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	// 获取用户拥有的完整菜单对象
	menuObjs, err := s.roleUc.ListUserPermissionMenus(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	// 转换完整菜单对象为proto格式
	pbMenus := make([]*pb.RoleMenuInfo, 0, len(menuObjs))
	for _, menu := range menuObjs {
		pbMenu := &pb.RoleMenuInfo{}
		utils.ObjConvert(menu, pbMenu)
		pbMenus = append(pbMenus, pbMenu)
	}

	return &pb.GetUserPermissionReply{
		MenuIds: menuIds,
	}, nil
}
