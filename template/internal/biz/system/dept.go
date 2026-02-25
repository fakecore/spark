package system

import (
	"context"

	"spark/internal/data/dal/model"
	"spark/internal/data/do"
	clog "spark/pkg/clog"
)

type DeptRepo interface {
	Create(ctx context.Context, dept *model.SysDept) error
	Update(ctx context.Context, dept *do.SysDept) error
	Delete(ctx context.Context, id int64) error
	Get(ctx context.Context, id int64) (*model.SysDept, error)
	List(ctx context.Context, current int32, pageSize int32, deptName *string, status *int32) ([]*model.SysDept, int64, error)
}

type SystemDeptUsecase struct {
	repo   DeptRepo
	logger clog.Logger
}

func NewSystemDeptUsecase(repo DeptRepo, logger clog.Logger) *SystemDeptUsecase {
	return &SystemDeptUsecase{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new department
func (uc *SystemDeptUsecase) Create(ctx context.Context, dept *model.SysDept) error {
	return uc.repo.Create(ctx, dept)
}

// Update updates an existing department
func (uc *SystemDeptUsecase) Update(ctx context.Context, dept *do.SysDept) error {
	return uc.repo.Update(ctx, dept)
}

// Delete deletes a department
func (uc *SystemDeptUsecase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

// Get retrieves a department by ID
func (uc *SystemDeptUsecase) Get(ctx context.Context, id int64) (*model.SysDept, error) {
	return uc.repo.Get(ctx, id)
}

// List retrieves a list of departments
func (uc *SystemDeptUsecase) List(ctx context.Context, current int32, pageSize int32, deptName *string, status *int32) ([]*model.SysDept, int64, error) {
	return uc.repo.List(ctx, current, pageSize, deptName, status)
}
