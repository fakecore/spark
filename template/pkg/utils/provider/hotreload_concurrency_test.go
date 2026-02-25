package provider

import (
	"context"
	"testing"
	"time"

	"spark/pkg/clog"
)

func TestDatabaseProvider_ReadPathNotBlockedByUpdateLock(t *testing.T) {
	p := NewDatabaseProvider(&clog.NoOpLogger{})
	m := p.(*databaseManager)

	m.mu.Lock()
	defer m.mu.Unlock()

	done := make(chan struct{})
	go func() {
		_, _ = p.DB()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DB read path blocked by update lock")
	}
}

func TestKVStoreProvider_ReadPathNotBlockedByUpdateLock(t *testing.T) {
	p := NewKVStoreProvider(&clog.NoOpLogger{})
	m := p.(*kvstoreManager)

	m.mu.Lock()
	defer m.mu.Unlock()

	done := make(chan struct{})
	go func() {
		_, _ = p.Get(context.Background(), "k")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("KV read path blocked by update lock")
	}
}
