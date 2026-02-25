package system

import (
	"context"
	"errors"
	"fmt"

	"spark/internal/data/dal/model"
	"spark/internal/data/do"
	clog "spark/pkg/clog"

	"gorm.io/gorm"
)

type SystemPostRepo interface {
	Get(ctx context.Context, id int64) (*model.SysPost, error)
	List(ctx context.Context, pageSize, current int32, postCode, postName *string, status *int32) ([]*model.SysPost, int64, error)
	Create(ctx context.Context, post *model.SysPost) error
	Update(ctx context.Context, post *do.SysPost) error
	Delete(ctx context.Context, id int64) error
}

type SystemPostUsecase struct {
	repo   SystemPostRepo
	logger clog.Logger
}

func NewSystemPostUsecase(repo SystemPostRepo, logger clog.Logger) *SystemPostUsecase {
	return &SystemPostUsecase{repo: repo, logger: logger}
}

func (uc *SystemPostUsecase) Get(ctx context.Context, id int64) (*model.SysPost, error) {
	return uc.repo.Get(ctx, id)
}

func (uc *SystemPostUsecase) List(ctx context.Context, pageSize, current int32, postCode, postName *string, status *int32) ([]*model.SysPost, int64, error) {
	return uc.repo.List(ctx, pageSize, current, postCode, postName, status)
}

func (uc *SystemPostUsecase) Create(ctx context.Context, post *model.SysPost) error {
	return uc.repo.Create(ctx, post)
}

func (uc *SystemPostUsecase) Update(ctx context.Context, post *do.SysPost) error {
	err := uc.repo.Update(ctx, post)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Error(ctx, "update failed: post not found", clog.Err(err))
			return fmt.Errorf("post not found: %w", err)
		}
		return fmt.Errorf("update failed: %w", err)
	}
	return nil
}

func (uc *SystemPostUsecase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}
