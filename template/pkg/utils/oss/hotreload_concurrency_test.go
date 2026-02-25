package oss

import (
	"context"
	"errors"
	"testing"
	"time"

	"spark/pkg/clog"
)

func TestOssProvider_ReadPathNotBlockedByUpdateLock(t *testing.T) {
	p := NewOssProvider(&clog.NoOpLogger{})
	m := p.(*ossManager)

	m.mu.Lock()
	defer m.mu.Unlock()

	done := make(chan error, 1)
	go func() {
		err := p.DeleteObject(context.Background(), "b", "o")
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, ErrOSSUnavailable) {
			t.Fatalf("expected ErrOSSUnavailable, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("OSS read path blocked by update lock")
	}
}
