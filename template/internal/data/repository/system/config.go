package system

import (
	"context"
	"fmt"

	"spark/internal/biz/system"
	"spark/internal/data/dal/model"
	"spark/internal/data/dal/query"

	clog "spark/pkg/clog"
	"spark/pkg/utils"
)

type configRepo struct {
	logger clog.Logger
	query  *query.Query
}

var _ system.ConfigRepo = &configRepo{}

func NewSystemConfigRepo(q *query.Query, logger clog.Logger) system.ConfigRepo {
	return &configRepo{
		logger: logger,
		query:  q,
	}
}

// GetByKey 根据key获取配置
func (r *configRepo) GetByKey(ctx context.Context, key string) (*model.SysConfig, error) {
	config, err := r.query.SysConfig.WithContext(ctx).
		Where(r.query.SysConfig.Key.Eq(key)).
		Where(r.query.SysConfig.Status.Eq(1)).
		First()
	if err != nil {
		return nil, err
	}
	return config, nil
}

// GetByKeys 根据多个key获取配置
func (r *configRepo) GetByKeys(ctx context.Context, keys []string) ([]*model.SysConfig, error) {
	configs, err := r.query.SysConfig.WithContext(ctx).
		Where(r.query.SysConfig.Key.In(keys...)).
		Where(r.query.SysConfig.Status.Eq(1)).
		Find()
	if err != nil {
		return nil, err
	}
	return configs, nil
}

// List 获取配置列表
func (r *configRepo) List(ctx context.Context, pageSize, current int32, name *string, key *string, kind *int32, status *int32) ([]*model.SysConfig, int32, error) {
	q := r.query.SysConfig.WithContext(ctx)

	if name != nil && *name != "" {
		q = q.Where(r.query.SysConfig.Name.Like("%" + *name + "%"))
	}
	if key != nil && *key != "" {
		q = q.Where(r.query.SysConfig.Key.Like("%" + *key + "%"))
	}
	if kind != nil {
		q = q.Where(r.query.SysConfig.Kind.Eq(*kind))
	}
	if status != nil {
		q = q.Where(r.query.SysConfig.Status.Eq(*status))
	}

	// 获取总数
	total, err := q.Count()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count configs: %w", err)
	}

	// 获取分页数据
	configs, err := q.Order(r.query.SysConfig.CreatedAt.Desc()).
		Scopes(utils.Paginate(current, pageSize)).
		Find()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list configs: %w", err)
	}

	return configs, int32(total), nil
}

// Create 创建配置
func (r *configRepo) Create(ctx context.Context, config *model.SysConfig) error {
	err := r.query.SysConfig.WithContext(ctx).Create(config)
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}
	return nil
}

// Update 更新配置
func (r *configRepo) Update(ctx context.Context, config *model.SysConfig) error {
	_, err := r.query.SysConfig.WithContext(ctx).
		Where(r.query.SysConfig.ID.Eq(config.ID)).
		Updates(config)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	return nil
}

// Delete 删除配置
func (r *configRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.query.SysConfig.WithContext(ctx).
		Where(r.query.SysConfig.ID.Eq(id)).
		Delete()
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}
	return nil
}

// Get 根据ID获取配置
func (r *configRepo) Get(ctx context.Context, id int64) (*model.SysConfig, error) {
	config, err := r.query.SysConfig.WithContext(ctx).
		Where(r.query.SysConfig.ID.Eq(id)).
		First()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return config, nil
}
