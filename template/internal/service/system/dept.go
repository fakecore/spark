package system

import (
	"context"

	"spark/internal/biz/system"
	"spark/internal/common/constants"
	"spark/internal/data/dal/model"
	"spark/internal/data/do"
	"spark/pkg/utils"

	pb "spark/api/system/v1"
)

type DeptService struct {
	pb.UnimplementedDeptServer

	uc *system.SystemDeptUsecase
}

func NewDeptService(uc *system.SystemDeptUsecase) *DeptService {
	return &DeptService{uc: uc}
}

func (s *DeptService) CreateDept(ctx context.Context, req *pb.CreateDeptRequest) (*pb.CreateDeptReply, error) {
	dept := &model.SysDept{
		ParentID: req.ParentId,
		DeptName: req.DeptName,
		Sort:     req.Sort,
		Leader:   req.Leader,
		Phone:    req.Phone,
		Email:    req.Email,
		Status:   utils.GetDataOrDefault(req.Status, constants.CommonStatus_ACTIVE),
	}

	err := s.uc.Create(ctx, dept)
	if err != nil {
		return nil, err
	}

	return &pb.CreateDeptReply{}, nil
}

func (s *DeptService) UpdateDept(ctx context.Context, req *pb.UpdateDeptRequest) (*pb.UpdateDeptReply, error) {
	dept := &do.SysDept{
		ID:       req.Id,
		DeptName: req.DeptName,
		OrderNum: req.Sort,
		Leader:   req.Leader,
		Phone:    req.Phone,
		Email:    req.Email,
		Status:   req.Status,
	}

	err := s.uc.Update(ctx, dept)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateDeptReply{}, nil
}

func (s *DeptService) DeleteDept(ctx context.Context, req *pb.DeleteDeptRequest) (*pb.DeleteDeptReply, error) {
	err := s.uc.Delete(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteDeptReply{}, nil
}

func (s *DeptService) GetDept(ctx context.Context, req *pb.GetDeptRequest) (*pb.GetDeptReply, error) {
	dept, err := s.uc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.GetDeptReply{
		Dept: convertDeptModelToInfo(dept),
	}, nil
}

func (s *DeptService) ListDept(ctx context.Context, req *pb.ListDeptRequest) (*pb.ListDeptReply, error) {
	depts, total, err := s.uc.List(ctx, req.Current, req.PageSize, req.DeptName, req.Status)
	if err != nil {
		return nil, err
	}

	// 构建部门树
	deptTree := buildDeptTree(depts, utils.GetDataOrDefault(req.ParentId, 0))

	return &pb.ListDeptReply{
		Items: deptTree,
		Total: int32(total),
	}, nil
}

// convertDeptModelToInfo 将数据模型转换为 proto 消息
func convertDeptModelToInfo(dept *model.SysDept) *pb.DeptInfo {
	if dept == nil {
		return nil
	}

	info := &pb.DeptInfo{}
	utils.ObjConvert(dept, info)

	return info
}

// buildDeptTree 构建部门树形结构
func buildDeptTree(depts []*model.SysDept, parentId int64) []*pb.DeptInfo {
	var tree []*pb.DeptInfo

	for _, dept := range depts {
		if dept.ParentID == parentId {
			deptInfo := convertDeptModelToInfo(dept)
			// 递归构建子部门
			deptInfo.Children = buildDeptTree(depts, dept.ID)
			tree = append(tree, deptInfo)
		}
	}

	return tree
}
