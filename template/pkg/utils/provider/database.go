package provider

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"spark/internal/conf"
	"spark/pkg/clog"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DatabaseProvider interface {
	DatabaseService
	GetStatus() DatabaseStatus
	Update(ctx context.Context, cfg *conf.Data_Database) error
}

type DatabaseStatus int

const (
	DatabaseStatusUninitialized DatabaseStatus = iota
	DatabaseStatusReady
	DatabaseStatusFailed
)

type DatabaseService interface {
	DB() (*gorm.DB, error)
}

type databaseAdapter struct {
	db *gorm.DB
}

func (a *databaseAdapter) DB() (*gorm.DB, error) {
	return a.db, nil
}

func (a *databaseAdapter) Close() error {
	if a == nil || a.db == nil {
		return nil
	}
	sqlDB, err := a.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

type databaseManager struct {
	logger clog.Logger
	mu     sync.Mutex
	db     atomic.Pointer[databaseServiceRef]
	status atomic.Int32
}

type databaseServiceRef struct {
	service DatabaseService
}

func NewDatabaseProvider(logger clog.Logger) DatabaseProvider {
	m := &databaseManager{logger: logger}
	m.db.Store(&databaseServiceRef{service: &noOpDatabaseService{logger: logger, reason: "not configured"}})
	m.status.Store(int32(DatabaseStatusUninitialized))
	return m
}

func (m *databaseManager) DB() (*gorm.DB, error) {
	return m.currentService().DB()
}

func (m *databaseManager) GetStatus() DatabaseStatus {
	return DatabaseStatus(m.status.Load())
}

func (m *databaseManager) Update(ctx context.Context, cfg *conf.Data_Database) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldService := m.currentService()

	if cfg == nil || cfg.Driver == "" || cfg.Source == "" {
		m.db.Store(&databaseServiceRef{service: &noOpDatabaseService{logger: m.logger, reason: "not configured"}})
		m.status.Store(int32(DatabaseStatusReady))
		m.logger.Info(ctx, "database: not configured")
		m.closeServiceLater(oldService)
		return nil
	}

	if err := m.validateConfig(cfg); err != nil {
		m.status.Store(int32(DatabaseStatusFailed))
		m.logger.Error(ctx, "database: config invalid", clog.Err(err))
		return err
	}

	adapter, err := m.createService(ctx, cfg)
	if err != nil {
		m.status.Store(int32(DatabaseStatusFailed))
		m.logger.Error(ctx, "database: create failed", clog.Err(err))
		return err
	}

	if adapter == nil {
		m.status.Store(int32(DatabaseStatusFailed))
		m.logger.Error(ctx, "database: create returned nil")
		return fmt.Errorf("create returned nil")
	}

	gormDB, err := adapter.DB()
	if err != nil {
		m.status.Store(int32(DatabaseStatusFailed))
		m.logger.Error(ctx, "database: get gorm.DB failed", clog.Err(err))
		return err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		m.status.Store(int32(DatabaseStatusFailed))
		m.logger.Error(ctx, "database: get sql.DB failed", clog.Err(err))
		return err
	}

	if err := sqlDB.Ping(); err != nil {
		m.status.Store(int32(DatabaseStatusFailed))
		m.logger.Error(ctx, "database: ping failed", clog.Err(err))
		return err
	}

	m.db.Store(&databaseServiceRef{service: adapter})
	m.status.Store(int32(DatabaseStatusReady))
	m.logger.Info(ctx, "database: updated", clog.String("driver", cfg.Driver))
	m.closeServiceLater(oldService)
	return nil
}

func (m *databaseManager) currentService() DatabaseService {
	ref := m.db.Load()
	if ref != nil && ref.service != nil {
		return ref.service
	}
	return &noOpDatabaseService{logger: m.logger, reason: "not configured"}
}

func (m *databaseManager) closeServiceLater(oldService DatabaseService) {
	closer, ok := oldService.(interface{ Close() error })
	if !ok {
		return
	}

	go func() {
		time.Sleep(30 * time.Second)
		if err := closer.Close(); err != nil {
			m.logger.Warn(context.Background(), "database: close previous connection failed", clog.Err(err))
		}
	}()
}

func (m *databaseManager) validateConfig(cfg *conf.Data_Database) error {
	switch cfg.Driver {
	case "postgres", "postgresql", "mysql":
		if cfg.Source == "" {
			return fmt.Errorf("database source is required")
		}
	default:
		return fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
	return nil
}

func (m *databaseManager) createService(ctx context.Context, cfg *conf.Data_Database) (DatabaseService, error) {
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "postgres", "postgresql":
		dialector = postgres.Open(cfg.Source)
	case "mysql":
		dialector = mysql.Open(cfg.Source)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(int(cfg.MaxIdleConns))
	sqlDB.SetMaxOpenConns(int(cfg.MaxOpenConns))

	return &databaseAdapter{db: db}, nil
}

type noOpDatabaseService struct {
	logger clog.Logger
	reason string
}

func (n *noOpDatabaseService) DB() (*gorm.DB, error) {
	return nil, fmt.Errorf("database service unavailable: %s", n.reason)
}
