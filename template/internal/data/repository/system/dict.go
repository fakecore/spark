package system

import (
	"context"
	"errors"
	"fmt"

	"spark/internal/biz/system"
	"spark/internal/data/dal/model"
	"spark/internal/data/dal/query"

	"spark/internal/data/do"
	clog "spark/pkg/clog"
	"spark/pkg/page"

	"gorm.io/gorm"
)

// Ensure dictRepo implements the biz.DictRepo interface.
var _ system.DictRepo = (*dictRepo)(nil)

type dictRepo struct {
	logger clog.Logger
	query  *query.Query
}

// NewSystemDictRepo creates a new dictionary repository instance.
func NewSystemDictRepo(q *query.Query, logger clog.Logger) system.DictRepo {
	return &dictRepo{
		logger: logger,
		query:  q,
	}
}

// --- Type Operations ---

// CreateType creates a dictionary type.
func (r *dictRepo) CreateType(ctx context.Context, dictType *model.SysDictType) error {
	q := r.query.SysDictType
	err := q.WithContext(ctx).Create(dictType)
	if err != nil {
		r.logger.Error(ctx, "[dict] 创建字典类型失败", clog.Err(err))
		return fmt.Errorf("创建字典类型失败: %w", err)
	}
	return nil
}

// UpdateType updates a dictionary type.
func (r *dictRepo) UpdateType(ctx context.Context, dictType *do.SysDictType) error {
	q := r.query.SysDictType
	values := make(map[string]interface{})
	if dictType.Name != nil {
		values[q.Name.ColumnName().String()] = *dictType.Name
	}
	if dictType.TypeCode != nil {
		values[q.TypeCode.ColumnName().String()] = *dictType.TypeCode
	}
	if dictType.Status != nil {
		values[q.Status.ColumnName().String()] = *dictType.Status
	}
	if dictType.Remark != nil {
		values[q.Remark.ColumnName().String()] = *dictType.Remark
	}
	if len(values) == 0 {
		return nil
	}
	_, err := q.WithContext(ctx).
		Where(q.ID.Eq(dictType.ID)).
		Updates(values)
	if err != nil {
		r.logger.Error(ctx, "[dict] 更新字典类型失败", clog.Int64("id", dictType.ID), clog.Err(err))
		return fmt.Errorf("更新字典类型失败: %w", err)
	}
	return nil
}

// DeleteType deletes a dictionary type.
func (r *dictRepo) DeleteType(ctx context.Context, id int64) error {
	q := r.query.SysDictType
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Delete()
	if err != nil {
		r.logger.Error(ctx, "[dict] 删除字典类型失败", clog.Int64("id", id), clog.Err(err))
		return fmt.Errorf("删除字典类型失败: %w", err)
	}
	return nil
}

// GetType gets a dictionary type by ID.
func (r *dictRepo) GetType(ctx context.Context, id int64) (*model.SysDictType, error) {
	q := r.query.SysDictType
	res, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("字典类型不存在, id=%d", id) // Consider using a biz layer error type
		}
		r.logger.Error(ctx, "[dict] 查询字典类型失败", clog.Int64("id", id), clog.Err(err))
		return nil, fmt.Errorf("查询字典类型失败: %w", err)
	}
	return res, nil
}

// GetTypeByCode gets a dictionary type by TypeCode.
func (r *dictRepo) GetTypeByCode(ctx context.Context, typeCode string) (*model.SysDictType, error) {
	q := r.query.SysDictType
	res, err := q.WithContext(ctx).Where(q.TypeCode.Eq(typeCode)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("字典类型不存在, code=%s", typeCode) // Consider using a biz layer error type
		}
		r.logger.Error(ctx, "[dict] 查询字典类型失败", clog.String("code", typeCode), clog.Err(err))
		return nil, fmt.Errorf("查询字典类型失败: %w", err)
	}
	return res, nil
}

// ListPageType gets a paginated list of dictionary types.
func (r *dictRepo) ListPageType(ctx context.Context, pageParam *page.Param, condition map[string]interface{}) ([]*model.SysDictType, int64, error) {
	t := r.query.SysDictType
	q := t.WithContext(ctx)
	if v, ok := condition["dict_name"]; ok {
		if name, ok2 := v.(string); ok2 && name != "" {
			q = q.Where(t.Name.Like("%" + name + "%"))
		}
	}
	if v, ok := condition["dict_type"]; ok {
		if typeCode, ok2 := v.(string); ok2 && typeCode != "" {
			q = q.Where(t.TypeCode.Like("%" + typeCode + "%"))
		}
	}
	if v, ok := condition["status"]; ok {
		if status, ok2 := v.(int32); ok2 && status != 0 { // Assuming 0 means 'all statuses'
			q = q.Where(t.Status.Eq(status))
		}
	}

	countQuery := q
	total, err := countQuery.Count()
	if err != nil {
		r.logger.Error(ctx, "[dict] 查询字典类型总数失败", clog.Err(err))
		return nil, 0, fmt.Errorf("查询字典类型总数失败: %w", err)
	}

	if total == 0 {
		return []*model.SysDictType{}, 0, nil
	}

	res, err := q.Order(t.ID.Desc()).
		Offset(int((pageParam.GetCurrent() - 1) * pageParam.GetPageSize())).
		Limit(int(pageParam.GetPageSize())).
		Find()
	if err != nil {
		r.logger.Error(ctx, "[dict] 分页查询字典类型列表失败", clog.Err(err))
		return nil, 0, fmt.Errorf("分页查询字典类型列表失败: %w", err)
	}

	return res, total, nil
}

// GetAllTypes gets all dictionary types.
func (r *dictRepo) GetAllTypes(ctx context.Context) ([]*model.SysDictType, error) {
	q := r.query.SysDictType
	res, err := q.WithContext(ctx).Find()
	if err != nil {
		r.logger.Error(ctx, "[dict] 查询所有字典类型失败", clog.Err(err))
		return nil, fmt.Errorf("查询所有字典类型失败: %w", err)
	}
	return res, nil
}

// --- Value Operations ---

// CreateValue creates a dictionary value.
func (r *dictRepo) CreateValue(ctx context.Context, dictData *model.SysDictValue) error {
	q := r.query.SysDictValue
	err := q.WithContext(ctx).Create(dictData)
	if err != nil {
		r.logger.Error(ctx, "[dict] 创建字典数据失败", clog.Err(err))
		return fmt.Errorf("创建字典数据失败: %w", err)
	}
	return nil
}

// UpdateValue updates a dictionary value.
func (r *dictRepo) UpdateValue(ctx context.Context, dictData *do.SysDictValue) error {
	q := r.query.SysDictValue
	// TODO: Consider using Updates(dictData) directly if fields align and zero values are handled correctly.
	// Building a map avoids accidentally zeroing fields not present in the DO.
	values := make(map[string]interface{})
	if dictData.DictTypeID != nil {
		values[q.DictCode.ColumnName().String()] = *dictData.DictTypeID // Assuming DictTypeID maps to DictCode
	}
	if dictData.DictLabel != nil {
		values[q.Label.ColumnName().String()] = *dictData.DictLabel
	}
	if dictData.DictValue != nil {
		values[q.Value.ColumnName().String()] = *dictData.DictValue
	}
	if dictData.DictType != nil {
		values["kind"] = *dictData.DictType
	}
	if dictData.Sort != nil {
		values[q.Sort.ColumnName().String()] = *dictData.Sort
	}
	if dictData.CSSClass != nil {
		values[q.CSSClass.ColumnName().String()] = *dictData.CSSClass
	}
	if dictData.ListClass != nil {
		values[q.ListClass.ColumnName().String()] = *dictData.ListClass
	}
	if dictData.IsDefault != nil {
		if *dictData.IsDefault {
			values[q.IsDefault.ColumnName().String()] = 1
		} else {
			values[q.IsDefault.ColumnName().String()] = 0
		}
	}
	if dictData.Status != nil {
		values[q.Status.ColumnName().String()] = *dictData.Status
	}
	if dictData.Remark != nil {
		values[q.Remark.ColumnName().String()] = *dictData.Remark
	}
	if len(values) == 0 {
		return nil
	}

	_, err := q.WithContext(ctx).
		Where(q.ID.Eq(dictData.ID)).
		Updates(values)
	if err != nil {
		r.logger.Error(ctx, "[dict] 更新字典数据失败", clog.Int64("id", dictData.ID), clog.Err(err))
		return fmt.Errorf("更新字典数据失败: %w", err)
	}
	return nil
}

// DeleteValue deletes a dictionary value.
func (r *dictRepo) DeleteValue(ctx context.Context, id int64) error {
	q := r.query.SysDictValue
	_, err := q.WithContext(ctx).
		Where(q.ID.Eq(id)).
		Delete()
	if err != nil {
		r.logger.Error(ctx, "[dict] 删除字典数据失败", clog.Int64("id", id), clog.Err(err))
		return fmt.Errorf("删除字典数据失败: %w", err)
	}
	return nil
}

// GetValue gets a dictionary value by ID.
func (r *dictRepo) GetValue(ctx context.Context, id int64) (*model.SysDictValue, error) {
	q := r.query.SysDictValue
	res, err := q.WithContext(ctx).
		Where(q.ID.Eq(id)).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("字典数据不存在, id=%d", id) // Consider using a biz layer error type
		}
		r.logger.Error(ctx, "[dict] 查询字典数据失败", clog.Int64("id", id), clog.Err(err))
		return nil, fmt.Errorf("查询字典数据失败: %w", err)
	}
	return res, nil
}

// ListPageValue gets a paginated list of dictionary values.
func (r *dictRepo) ListPageValue(ctx context.Context, pageParam *page.Param, condition map[string]interface{}) ([]*model.SysDictValue, int64, error) {
	vq := r.query.SysDictValue
	q := vq.WithContext(ctx)

	if v, ok := condition["dict_type"]; ok {
		if dictType, ok2 := v.(string); ok2 && dictType != "" {
			q = q.Where(vq.DictCode.Eq(dictType)) // Assuming condition["dict_type"] refers to the TypeCode
		}
	}
	if v, ok := condition["dict_label"]; ok {
		if label, ok2 := v.(string); ok2 && label != "" {
			q = q.Where(vq.Label.Like("%" + label + "%"))
		}
	}
	if v, ok := condition["status"]; ok {
		if status, ok2 := v.(int32); ok2 && status != 0 { // Assuming 0 means 'all statuses'
			q = q.Where(vq.Status.Eq(status))
		}
	}

	countQuery := q
	total, err := countQuery.Count()
	if err != nil {
		r.logger.Error(ctx, "[dict] 查询字典数据总数失败", clog.Err(err))
		return nil, 0, fmt.Errorf("查询字典数据总数失败: %w", err)
	}

	if total == 0 {
		return []*model.SysDictValue{}, 0, nil
	}

	res, err := q.Order(vq.Sort).
		Offset(int((pageParam.GetCurrent() - 1) * pageParam.GetPageSize())).
		Limit(int(pageParam.GetPageSize())).
		Find()
	if err != nil {
		r.logger.Error(ctx, "[dict] 分页查询字典数据列表失败", clog.Err(err))
		return nil, 0, fmt.Errorf("分页查询字典数据列表失败: %w", err)
	}

	return res, total, nil
}

// GetValuesByType gets all dictionary values for a specific type code.
func (r *dictRepo) GetValuesByType(ctx context.Context, dictTypeCode string) ([]*model.SysDictValue, error) {
	q := r.query.SysDictValue
	res, err := q.WithContext(ctx).Where(q.DictCode.Eq(dictTypeCode)).Order(q.Sort).Find()
	if err != nil {
		r.logger.Error(ctx, "[dict] 查询字典数据失败", clog.String("typeCode", dictTypeCode), clog.Err(err))
		return nil, fmt.Errorf("查询字典数据失败: %w", err)
	}
	return res, nil
}
