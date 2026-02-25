package system

import (
	"context"
	"encoding/json"
	"fmt"

	"spark/internal/data/dal/model"
	clog "spark/pkg/clog"
)

type ConfigRepo interface {
	GetByKey(ctx context.Context, key string) (*model.SysConfig, error)
	GetByKeys(ctx context.Context, keys []string) ([]*model.SysConfig, error)
	List(ctx context.Context, pageSize, current int32, name *string, key *string, kind *int32, status *int32) ([]*model.SysConfig, int32, error)
	Create(ctx context.Context, config *model.SysConfig) error
	Update(ctx context.Context, config *model.SysConfig) error
	Delete(ctx context.Context, id int64) error
	Get(ctx context.Context, id int64) (*model.SysConfig, error)
}

type SystemConfigUsecase struct {
	repo   ConfigRepo
	logger clog.Logger
}

func NewSystemConfigUsecase(repo ConfigRepo, logger clog.Logger) *SystemConfigUsecase {
	return &SystemConfigUsecase{repo: repo, logger: logger}
}

// GetConfigValue 获取配置值
func (uc *SystemConfigUsecase) GetConfigValue(ctx context.Context, key string) (string, error) {
	config, err := uc.repo.GetByKey(ctx, key)
	if err != nil {
		return "", err
	}

	if config == nil || config.Value == nil {
		return "", fmt.Errorf("config not found for key: %s", key)
	}

	return *config.Value, nil
}

// GetConfigAsJSON 获取配置值并解析为JSON
func (uc *SystemConfigUsecase) GetConfigAsJSON(ctx context.Context, key string, result interface{}) error {
	valueStr, err := uc.GetConfigValue(ctx, key)
	if err != nil {
		return err
	}

	if valueStr == "" {
		return fmt.Errorf("empty config value for key: %s", key)
	}

	return json.Unmarshal([]byte(valueStr), result)
}

func (uc *SystemConfigUsecase) Create(ctx context.Context, config *model.SysConfig) error {
	return uc.repo.Create(ctx, config)
}

func (uc *SystemConfigUsecase) Update(ctx context.Context, config *model.SysConfig) error {
	return uc.repo.Update(ctx, config)
}

func (uc *SystemConfigUsecase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *SystemConfigUsecase) Get(ctx context.Context, id int64) (*model.SysConfig, error) {
	return uc.repo.Get(ctx, id)
}

func (uc *SystemConfigUsecase) List(ctx context.Context, pageSize, current int32, name *string, key *string, kind *int32, status *int32) ([]*model.SysConfig, int32, error) {
	return uc.repo.List(ctx, pageSize, current, name, key, kind, status)
}

// UpdateConfigValueByKey 根据配置键更新配置值
func (uc *SystemConfigUsecase) UpdateConfigValueByKey(ctx context.Context, key string, value string) error {
	// 1. 先获取现有配置
	config, err := uc.repo.GetByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("获取配置失败: %w", err)
	}

	if config == nil {
		return fmt.Errorf("配置不存在: %s", key)
	}

	// 2. 更新值
	config.Value = &value

	// 3. 保存
	return uc.repo.Update(ctx, config)
}
