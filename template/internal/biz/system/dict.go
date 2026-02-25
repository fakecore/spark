package system

import (
	"context"

	"spark/internal/data/dal/model"
	"spark/internal/data/do"
	clog "spark/pkg/clog"
	"spark/pkg/page"
)

// DictRepo defines the repository interface for dictionary data access.
// It combines operations for both dictionary types and dictionary values.
type DictRepo interface {
	// Type Operations
	CreateType(ctx context.Context, dictType *model.SysDictType) error
	UpdateType(ctx context.Context, dictType *do.SysDictType) error
	DeleteType(ctx context.Context, id int64) error
	GetType(ctx context.Context, id int64) (*model.SysDictType, error)
	GetTypeByCode(ctx context.Context, typeCode string) (*model.SysDictType, error)
	ListPageType(ctx context.Context, pageParam *page.Param, condition map[string]interface{}) ([]*model.SysDictType, int64, error)
	GetAllTypes(ctx context.Context) ([]*model.SysDictType, error)

	// Value Operations
	CreateValue(ctx context.Context, dictData *model.SysDictValue) error
	UpdateValue(ctx context.Context, dictData *do.SysDictValue) error
	DeleteValue(ctx context.Context, id int64) error
	GetValue(ctx context.Context, id int64) (*model.SysDictValue, error)
	ListPageValue(ctx context.Context, pageParam *page.Param, condition map[string]interface{}) ([]*model.SysDictValue, int64, error)
	GetValuesByType(ctx context.Context, dictTypeCode string) ([]*model.SysDictValue, error)
}

// DictUsecase manages the business logic for dictionaries.
type DictUsecase struct {
	repo   DictRepo
	logger clog.Logger
}

// NewSystemDictUsecase creates a new SystemDictUsecase.
func NewSystemDictUsecase(repo DictRepo, logger clog.Logger) *DictUsecase {
	return &DictUsecase{
		repo:   repo,
		logger: logger,
	}
}

// CreateDictType creates a new dictionary type.
func (uc *DictUsecase) CreateDictType(ctx context.Context, dictType *model.SysDictType) error {
	// TODO: Add business logic validation if needed (e.g., check if TypeCode already exists)
	return uc.repo.CreateType(ctx, dictType)
}

// UpdateDictType updates an existing dictionary type.
func (uc *DictUsecase) UpdateDictType(ctx context.Context, dictType *do.SysDictType) error {
	// TODO: Add business logic validation if needed
	return uc.repo.UpdateType(ctx, dictType)
}

// DeleteDictType deletes a dictionary type.
func (uc *DictUsecase) DeleteDictType(ctx context.Context, id int64) error {
	// TODO: Add business logic if needed (e.g., check if values exist for this type before deletion)
	return uc.repo.DeleteType(ctx, id)
}

// GetDictType retrieves a dictionary type by its ID.
func (uc *DictUsecase) GetDictType(ctx context.Context, id int64) (*model.SysDictType, error) {
	return uc.repo.GetType(ctx, id)
}

// GetDictTypeByCode retrieves a dictionary type by its code.
func (uc *DictUsecase) GetDictTypeByCode(ctx context.Context, typeCode string) (*model.SysDictType, error) {
	return uc.repo.GetTypeByCode(ctx, typeCode)
}

// ListDictTypes retrieves a paginated list of dictionary types.
func (uc *DictUsecase) ListDictTypes(ctx context.Context, pageParam *page.Param, condition map[string]interface{}) ([]*model.SysDictType, int64, error) {
	return uc.repo.ListPageType(ctx, pageParam, condition)
}

// GetAllDictTypes retrieves all dictionary types.
func (uc *DictUsecase) GetAllDictTypes(ctx context.Context) ([]*model.SysDictType, error) {
	return uc.repo.GetAllTypes(ctx)
}

// --- Value Operations ---

// CreateDictValue creates a new dictionary value.
func (uc *DictUsecase) CreateDictValue(ctx context.Context, dictData *model.SysDictValue) error {
	// TODO: Add business logic validation if needed (e.g., check if value exists for this type, check if type exists)
	return uc.repo.CreateValue(ctx, dictData)
}

// UpdateDictValue updates an existing dictionary value.
func (uc *DictUsecase) UpdateDictValue(ctx context.Context, dictData *do.SysDictValue) error {
	// TODO: Add business logic validation if needed
	return uc.repo.UpdateValue(ctx, dictData)
}

// DeleteDictValue deletes a dictionary value.
func (uc *DictUsecase) DeleteDictValue(ctx context.Context, id int64) error {
	return uc.repo.DeleteValue(ctx, id)
}

// GetDictValue retrieves a dictionary value by its ID.
func (uc *DictUsecase) GetDictValue(ctx context.Context, id int64) (*model.SysDictValue, error) {
	return uc.repo.GetValue(ctx, id)
}

// ListDictValues retrieves a paginated list of dictionary values.
func (uc *DictUsecase) ListDictValues(ctx context.Context, pageParam *page.Param, condition map[string]interface{}) ([]*model.SysDictValue, int64, error) {
	return uc.repo.ListPageValue(ctx, pageParam, condition)
}

// GetDictValuesByType retrieves all dictionary values for a specific type code.
func (uc *DictUsecase) GetDictValuesByType(ctx context.Context, dictTypeCode string) ([]*model.SysDictValue, error) {
	return uc.repo.GetValuesByType(ctx, dictTypeCode)
}
