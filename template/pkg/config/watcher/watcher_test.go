package watcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"spark/internal/conf"
	"spark/pkg/clog"
	"spark/pkg/utils/oss"
	"spark/pkg/utils/provider"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func TestConfigWatcherReloadConfig_UpdatesProvidersOnChange(t *testing.T) {
	configPath := writeWatcherConfig(t, "dsn_v1")

	bc := &conf.Bootstrap{
		Data: &conf.Data{
			Database: &conf.Data_Database{Driver: "postgres", Source: "dsn_v1"},
			Redis:    &conf.Data_Redis{Addr: "127.0.0.1:6379", Mode: "memory"},
		},
		Oss: &conf.OSS{Service: "memory"},
	}

	ossP := &ossProviderStub{}
	dbP := &dbProviderStub{}
	kvP := &kvProviderStub{}

	w := NewConfigWatcher(&clog.NoOpLogger{}, configPath, bc, ossP, dbP, kvP)
	ctx := context.Background()

	w.reloadConfig(ctx)
	ossBefore := ossP.calls()
	dbBefore := dbP.calls()
	kvBefore := kvP.calls()

	writeWatcherConfigToPath(t, configPath, "dsn_v2")
	w.reloadConfig(ctx)

	if got := ossP.calls(); got != ossBefore+1 {
		t.Fatalf("expected oss provider updates +1 after config change, before=%d got=%d", ossBefore, got)
	}
	if got := dbP.calls(); got != dbBefore+1 {
		t.Fatalf("expected db provider updates +1 after config change, before=%d got=%d", dbBefore, got)
	}
	if got := kvP.calls(); got != kvBefore+1 {
		t.Fatalf("expected kv provider updates +1 after config change, before=%d got=%d", kvBefore, got)
	}

	if got := w.bc.GetData().GetDatabase().GetSource(); got != "dsn_v2" {
		t.Fatalf("expected watcher bootstrap updated to dsn_v2, got %q", got)
	}
}

func TestConfigWatcherReloadConfig_KeepOldConfigWhenAnyProviderFails(t *testing.T) {
	configPath := writeWatcherConfig(t, "dsn_v1")

	bc := &conf.Bootstrap{
		Data: &conf.Data{
			Database: &conf.Data_Database{Driver: "postgres", Source: "dsn_v1"},
			Redis:    &conf.Data_Redis{Addr: "127.0.0.1:6379", Mode: "memory"},
		},
		Oss: &conf.OSS{Service: "memory"},
	}

	ossP := &ossProviderStub{}
	dbP := &dbProviderStub{updateErr: errors.New("db update failed")}
	kvP := &kvProviderStub{}

	w := NewConfigWatcher(&clog.NoOpLogger{}, configPath, bc, ossP, dbP, kvP)
	ctx := context.Background()

	writeWatcherConfigToPath(t, configPath, "dsn_v2")
	w.reloadConfig(ctx)

	if got := w.bc.GetData().GetDatabase().GetSource(); got != "dsn_v1" {
		t.Fatalf("expected old bootstrap kept on failed update, got %q", got)
	}

	if got := dbP.calls(); got != 1 {
		t.Fatalf("expected db provider called once, got %d", got)
	}
	if got := ossP.calls(); got != 1 {
		t.Fatalf("expected oss provider still called once, got %d", got)
	}
	if got := kvP.calls(); got != 1 {
		t.Fatalf("expected kv provider still called once, got %d", got)
	}
}

func writeWatcherConfig(t *testing.T, dsn string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	writeWatcherConfigToPath(t, path, dsn)
	return path
}

func writeWatcherConfigToPath(t *testing.T, path string, dsn string) {
	t.Helper()
	content := "data:\n" +
		"  database:\n" +
		"    driver: \"postgres\"\n" +
		"    source: \"" + dsn + "\"\n" +
		"  redis:\n" +
		"    mode: \"memory\"\n" +
		"    addr: \"127.0.0.1:6379\"\n" +
		"oss:\n" +
		"  service: \"memory\"\n"

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
}

type ossProviderStub struct {
	mu        sync.Mutex
	updates   int
	updateErr error
}

func (s *ossProviderStub) GenUploadUrl(ctx context.Context, fileName string, fileSize int64, mimeType string, bizType int32, forceSts bool) (*oss.GenUploadUrlReply, error) {
	return nil, nil
}

func (s *ossProviderStub) DeleteObject(ctx context.Context, bucket string, objectPath string) error {
	return nil
}

func (s *ossProviderStub) GetStatus() oss.OssStatus {
	return oss.OssStatusReady
}

func (s *ossProviderStub) Update(ctx context.Context, cfg *conf.OSS) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates++
	return s.updateErr
}

func (s *ossProviderStub) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updates
}

type dbProviderStub struct {
	mu        sync.Mutex
	updates   int
	updateErr error
}

func (s *dbProviderStub) DB() (*gorm.DB, error) {
	return nil, nil
}

func (s *dbProviderStub) GetStatus() provider.DatabaseStatus {
	return provider.DatabaseStatusReady
}

func (s *dbProviderStub) Update(ctx context.Context, cfg *conf.Data_Database) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates++
	return s.updateErr
}

func (s *dbProviderStub) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updates
}

type kvProviderStub struct {
	mu      sync.Mutex
	updates int
}

func (s *kvProviderStub) Backend() string { return "memory" }

func (s *kvProviderStub) Health(ctx context.Context) (bool, string) { return true, "ok" }

func (s *kvProviderStub) Get(ctx context.Context, key string) (string, error) { return "", nil }

func (s *kvProviderStub) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return nil
}

func (s *kvProviderStub) Del(ctx context.Context, keys ...string) error { return nil }

func (s *kvProviderStub) Keys(ctx context.Context, pattern string) ([]string, error) { return nil, nil }

func (s *kvProviderStub) AsRedis() (*redis.Client, bool) { return nil, false }

func (s *kvProviderStub) GetStatus() provider.KVStoreStatus { return provider.KVStoreStatusReady }

func (s *kvProviderStub) Update(ctx context.Context, cfg *conf.Data_Redis) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates++
	return nil
}

func (s *kvProviderStub) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updates
}
