package system

import (
	"context"

	v1 "spark/api/system/v1"
	"spark/internal/biz/system"
	"spark/internal/data/dal/model"
	"spark/internal/data/do"
	clog "spark/pkg/clog"
	"spark/pkg/page"
)

type DictService struct {
	v1.UnimplementedDictServer

	dictUsecase *system.DictUsecase
	logger      clog.Logger
}

func NewDictService(
	dictUsecase *system.DictUsecase,
	logger clog.Logger,
) *DictService {
	return &DictService{
		dictUsecase: dictUsecase,
		logger:      logger,
	}
}

// CreateDictType 创建字典类型
func (s *DictService) CreateDictType(ctx context.Context, req *v1.CreateDictTypeRequest) (*v1.CreateDictTypeReply, error) {
	dictType := &model.SysDictType{
		Name:     req.DictName,
		TypeCode: req.DictCode,
		Status:   req.Status,
		Remark:   req.Remark,
	}
	err := s.dictUsecase.CreateDictType(ctx, dictType)
	if err != nil {
		return nil, err
	}
	return &v1.CreateDictTypeReply{}, nil
}

// UpdateDictType 更新字典类型
func (s *DictService) UpdateDictType(ctx context.Context, req *v1.UpdateDictTypeRequest) (*v1.UpdateDictTypeReply, error) {
	dictType := &do.SysDictType{
		ID:       req.GetId(),
		Name:     &req.DictName,
		TypeCode: &req.DictCode,
		Status:   &req.Status,
		Remark:   req.Remark,
	}
	err := s.dictUsecase.UpdateDictType(ctx, dictType)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateDictTypeReply{}, nil
}

// DeleteDictType 删除字典类型
func (s *DictService) DeleteDictType(ctx context.Context, req *v1.DeleteDictTypeRequest) (*v1.DeleteDictTypeReply, error) {
	err := s.dictUsecase.DeleteDictType(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.DeleteDictTypeReply{}, nil
}

// GetDictType 获取字典类型
func (s *DictService) GetDictType(ctx context.Context, req *v1.GetDictTypeRequest) (*v1.GetDictTypeReply, error) {
	info, err := s.dictUsecase.GetDictType(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.GetDictTypeReply{
		DictType: &v1.DictTypeInfo{
			Id:        info.ID,
			DictName:  info.Name,
			DictCode:  info.TypeCode,
			Status:    info.Status,
			Remark:    info.Remark,
			CreatedAt: info.CreatedAt,
			UpdatedAt: info.UpdatedAt,
		},
	}, nil
}

// ListDictType 获取字典类型列表
func (s *DictService) ListDictType(ctx context.Context, req *v1.ListDictTypeRequest) (*v1.ListDictTypeReply, error) {
	pageParam := &page.Param{
		PageSize: req.GetPageSize(),
		Current:  req.GetCurrent(),
	}
	condition := make(map[string]interface{})
	if req.GetDictName() != "" {
		condition["name"] = req.GetDictName()
	}
	if req.GetDictType() != "" {
		condition["type_code"] = req.GetDictType()
	}
	if req.GetStatus() != 0 {
		condition["status"] = req.GetStatus()
	}

	list, total, err := s.dictUsecase.ListDictTypes(ctx, pageParam, condition)
	if err != nil {
		return nil, err
	}

	items := make([]*v1.DictTypeInfo, 0, len(list))
	for _, info := range list {
		items = append(items, &v1.DictTypeInfo{
			Id:        info.ID,
			DictName:  info.Name,
			DictCode:  info.TypeCode,
			Status:    info.Status,
			Remark:    info.Remark,
			CreatedAt: info.CreatedAt,
			UpdatedAt: info.UpdatedAt,
		})
	}
	return &v1.ListDictTypeReply{
		Items: items,
		Total: int32(total),
	}, nil
}

// CreateDictData 创建字典数据
func (s *DictService) CreateDictData(ctx context.Context, req *v1.CreateDictDataRequest) (*v1.CreateDictDataReply, error) {
	dictData := &model.SysDictValue{
		Label:     req.DictLabel,
		Value:     req.DictValue,
		DictCode:  req.DictType,
		Sort:      req.Sort,
		CSSClass:  &req.CssClass,
		ListClass: &req.ListClass,
		Status:    req.Status,
		Remark:    req.Remark,
	}
	if req.IsDefault {
		dictData.IsDefault = 1
	} else {
		dictData.IsDefault = 0
	}
	err := s.dictUsecase.CreateDictValue(ctx, dictData)
	if err != nil {
		return nil, err
	}
	return &v1.CreateDictDataReply{}, nil
}

// UpdateDictData 更新字典数据
func (s *DictService) UpdateDictData(ctx context.Context, req *v1.UpdateDictDataRequest) (*v1.UpdateDictDataReply, error) {
	dictData := &do.SysDictValue{
		ID: req.Id,
	}
	if req.DictLabel != "" {
		dictData.DictLabel = &req.DictLabel
	}
	if req.DictValue != "" {
		dictData.DictValue = &req.DictValue
	}
	if req.DictType != "" {
		dictData.DictType = &req.DictType
	}
	sort := req.Sort
	if sort != 0 {
		dictData.Sort = &sort
	}
	if req.CssClass != "" {
		cssClass := req.CssClass
		dictData.CSSClass = &cssClass
	}
	if req.ListClass != "" {
		listClass := req.ListClass
		dictData.ListClass = &listClass
	}
	dictData.IsDefault = &req.IsDefault
	if req.Status != 0 {
		dictData.Status = &req.Status
	}
	if req.Remark != nil {
		dictData.Remark = req.Remark
	}
	err := s.dictUsecase.UpdateDictValue(ctx, dictData)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateDictDataReply{}, nil
}

// DeleteDictData 删除字典数据
func (s *DictService) DeleteDictData(ctx context.Context, req *v1.DeleteDictDataRequest) (*v1.DeleteDictDataReply, error) {
	err := s.dictUsecase.DeleteDictValue(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.DeleteDictDataReply{}, nil
}

// GetDictData 获取字典数据
func (s *DictService) GetDictData(ctx context.Context, req *v1.GetDictDataRequest) (*v1.GetDictDataReply, error) {
	info, err := s.dictUsecase.GetDictValue(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	var isDefault bool
	if info.IsDefault == 1 {
		isDefault = true
	}
	var sort int32
	if info.Sort != 0 {
		sort = info.Sort
	}
	var cssClass, listClass string
	if info.CSSClass != nil {
		cssClass = *info.CSSClass
	}
	if info.ListClass != nil {
		listClass = *info.ListClass
	}
	return &v1.GetDictDataReply{
		DictData: &v1.DictDataInfo{
			Id:        info.ID,
			DictLabel: info.Label,
			DictValue: info.Value,
			DictCode:  info.DictCode,
			Sort:      sort,
			CssClass:  cssClass,
			ListClass: listClass,
			IsDefault: isDefault,
			Status:    info.Status,
			Remark:    info.Remark,
			CreatedAt: info.CreatedAt,
			UpdatedAt: info.UpdatedAt,
		},
	}, nil
}

// ListDictData 获取字典数据列表
func (s *DictService) ListDictData(ctx context.Context, req *v1.ListDictDataRequest) (*v1.ListDictDataReply, error) {
	pageParam := &page.Param{
		PageSize: req.GetPageSize(),
		Current:  req.GetCurrent(),
	}
	condition := make(map[string]interface{})
	if req.GetDictType() != "" {
		condition["dict_type"] = req.GetDictType()
	}
	if req.GetDictLabel() != "" {
		condition["dict_label"] = req.GetDictLabel()
	}
	if req.GetStatus() != 0 {
		condition["status"] = req.GetStatus()
	}
	list, total, err := s.dictUsecase.ListDictValues(ctx, pageParam, condition)
	if err != nil {
		return nil, err
	}
	items := make([]*v1.DictDataInfo, 0, len(list))
	for _, info := range list {
		var isDefault bool
		if info.IsDefault == 1 {
			isDefault = true
		}
		var sort int32
		if info.Sort != 0 {
			sort = info.Sort
		}
		var cssClass, listClass string
		if info.CSSClass != nil {
			cssClass = *info.CSSClass
		}
		if info.ListClass != nil {
			listClass = *info.ListClass
		}
		items = append(items, &v1.DictDataInfo{
			Id:        info.ID,
			DictLabel: info.Label,
			DictValue: info.Value,
			DictCode:  info.DictCode,
			Sort:      sort,
			CssClass:  cssClass,
			ListClass: listClass,
			IsDefault: isDefault,
			Status:    info.Status,
			Remark:    info.Remark,
			CreatedAt: info.CreatedAt,
			UpdatedAt: info.UpdatedAt,
		})
	}
	return &v1.ListDictDataReply{
		Total: int32(total),
		Items: items,
	}, nil
}
