package provider

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"spark/internal/conf"
	"spark/pkg/clog"
	"spark/pkg/kvstore"

	"github.com/redis/go-redis/v9"
)

type KVStoreProvider interface {
	KVStoreService
	GetStatus() KVStoreStatus
	Update(ctx context.Context, cfg *conf.Data_Redis) error
}

type KVStoreStatus int

const (
	KVStoreStatusUninitialized KVStoreStatus = iota
	KVStoreStatusReady
	KVStoreStatusFailed
)

type KVStoreService interface {
	kvstore.Store
}

type kvstoreManager struct {
	logger clog.Logger
	mu     sync.Mutex
	store  atomic.Pointer[kvStoreRef]
	status atomic.Int32
}

type kvStoreRef struct {
	store KVStoreService
}

func NewKVStoreProvider(logger clog.Logger) KVStoreProvider {
	m := &kvstoreManager{logger: logger}
	m.store.Store(&kvStoreRef{store: &noOpKVStoreService{logger: logger, reason: "not configured"}})
	m.status.Store(int32(KVStoreStatusUninitialized))
	return m
}

func (m *kvstoreManager) Backend() string {
	return m.currentStore().Backend()
}

func (m *kvstoreManager) Health(ctx context.Context) (bool, string) {
	return m.currentStore().Health(ctx)
}

func (m *kvstoreManager) Get(ctx context.Context, key string) (string, error) {
	return m.currentStore().Get(ctx, key)
}

func (m *kvstoreManager) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return m.currentStore().Set(ctx, key, value, ttl)
}

func (m *kvstoreManager) Del(ctx context.Context, keys ...string) error {
	return m.currentStore().Del(ctx, keys...)
}

func (m *kvstoreManager) Keys(ctx context.Context, pattern string) ([]string, error) {
	return m.currentStore().Keys(ctx, pattern)
}

func (m *kvstoreManager) AsRedis() (*redis.Client, bool) {
	return m.currentStore().AsRedis()
}

func (m *kvstoreManager) GetStatus() KVStoreStatus {
	return KVStoreStatus(m.status.Load())
}

func (m *kvstoreManager) Update(ctx context.Context, cfg *conf.Data_Redis) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldStore := m.currentStore()

	if cfg == nil || cfg.Addr == "" {
		m.store.Store(&kvStoreRef{store: &noOpKVStoreService{logger: m.logger, reason: "not configured"}})
		m.status.Store(int32(KVStoreStatusReady))
		m.logger.Info(ctx, "kvstore: not configured")
		m.closeRedisClientLater(oldStore)
		return nil
	}

	mode := cfg.Mode
	if mode == "" {
		mode = "auto"
	}

	if mode == "memory" {
		m.store.Store(&kvStoreRef{store: kvstore.NewMemoryStore(24 * time.Hour)})
		m.status.Store(int32(KVStoreStatusReady))
		m.logger.Info(ctx, "kvstore: using in-memory store (mode=memory)")
		m.closeRedisClientLater(oldStore)
		return nil
	}

	if err := m.validateConfig(cfg); err != nil {
		m.status.Store(int32(KVStoreStatusFailed))
		m.logger.Error(ctx, "kvstore: config invalid", clog.Err(err))
		return err
	}

	store, err := m.createService(ctx, cfg)
	if err != nil {
		if mode == "external" {
			m.status.Store(int32(KVStoreStatusFailed))
			m.logger.Error(ctx, "kvstore: create failed", clog.Err(err))
			return err
		}
		m.logger.Warn(ctx, "kvstore: unavailable, falling back to memory", clog.Err(err))
		m.store.Store(&kvStoreRef{store: kvstore.NewMemoryStore(24 * time.Hour)})
		m.status.Store(int32(KVStoreStatusReady))
		m.closeRedisClientLater(oldStore)
		return nil
	}

	m.store.Store(&kvStoreRef{store: store})
	m.status.Store(int32(KVStoreStatusReady))
	m.logger.Info(ctx, "kvstore: updated", clog.String("mode", mode))
	m.closeRedisClientLater(oldStore)
	return nil
}

func (m *kvstoreManager) currentStore() KVStoreService {
	ref := m.store.Load()
	if ref != nil && ref.store != nil {
		return ref.store
	}
	return &noOpKVStoreService{logger: m.logger, reason: "not configured"}
}

func (m *kvstoreManager) closeRedisClientLater(oldStore KVStoreService) {
	redisClient, ok := oldStore.AsRedis()
	if !ok || redisClient == nil {
		return
	}

	go func() {
		time.Sleep(30 * time.Second)
		if err := redisClient.Close(); err != nil {
			m.logger.Warn(context.Background(), "kvstore: close previous redis client failed", clog.Err(err))
		}
	}()
}

func (m *kvstoreManager) validateConfig(cfg *conf.Data_Redis) error {
	if cfg.Addr == "" {
		return fmt.Errorf("redis addr is required")
	}
	return nil
}

func (m *kvstoreManager) createService(ctx context.Context, cfg *conf.Data_Redis) (KVStoreService, error) {
	mode := cfg.Mode
	if mode == "" {
		mode = "auto"
	}

	redisdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       int(cfg.Database),
	})

	if err := redisdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	if mode == "auto" {
		mem := kvstore.NewMemoryStore(24 * time.Hour)
		return kvstore.NewAutoStore(redisdb, mem, func(op string, err error) {
			m.logger.Warn(ctx, "kvstore: fallback to memory", clog.String("op", op), clog.Err(err))
		}), nil
	}

	return kvstore.NewRedisStore(redisdb), nil
}

type noOpKVStoreService struct {
	logger clog.Logger
	reason string
}

func (n *noOpKVStoreService) Backend() string { return "noop" }
func (n *noOpKVStoreService) Health(ctx context.Context) (bool, string) {
	return false, fmt.Sprintf("unavailable: %s", n.reason)
}
func (n *noOpKVStoreService) Get(ctx context.Context, key string) (string, error) {
	return "", fmt.Errorf("kvstore unavailable: %s", n.reason)
}
func (n *noOpKVStoreService) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return fmt.Errorf("kvstore unavailable: %s", n.reason)
}
func (n *noOpKVStoreService) Del(ctx context.Context, keys ...string) error {
	return fmt.Errorf("kvstore unavailable: %s", n.reason)
}
func (n *noOpKVStoreService) Keys(ctx context.Context, pattern string) ([]string, error) {
	return nil, fmt.Errorf("kvstore unavailable: %s", n.reason)
}
func (n *noOpKVStoreService) AsRedis() (*redis.Client, bool) {
	return nil, false
}
