package kvstore

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestMemoryStore_BasicsAndKeys(t *testing.T) {
	s := NewMemoryStore(50 * time.Millisecond)
	ctx := context.Background()

	if err := s.Set(ctx, "token:1:aaa", "v1", 50*time.Millisecond); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := s.Set(ctx, "token:1:bbb", "v2", 0); err != nil {
		t.Fatalf("set: %v", err)
	}

	v, err := s.Get(ctx, "token:1:aaa")
	if err != nil || v != "v1" {
		t.Fatalf("get: v=%q err=%v", v, err)
	}

	keys, err := s.Keys(ctx, "token:1:*")
	if err != nil {
		t.Fatalf("keys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d (%v)", len(keys), keys)
	}

	if err := s.Del(ctx, "token:1:aaa"); err != nil {
		t.Fatalf("del: %v", err)
	}
	if _, err := s.Get(ctx, "token:1:aaa"); err == nil {
		t.Fatalf("expected not found after del")
	}
}

func TestRedisStore_BasicsAndKeys(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
	})

	s := NewRedisStore(rdb)
	ctx := context.Background()

	if err := s.Set(ctx, "token:2:aaa", "v1", time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := s.Set(ctx, "token:2:bbb", "v2", time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, err := s.Get(ctx, "token:2:aaa")
	if err != nil || v != "v1" {
		t.Fatalf("get: v=%q err=%v", v, err)
	}
	keys, err := s.Keys(ctx, "token:2:*")
	if err != nil {
		t.Fatalf("keys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d (%v)", len(keys), keys)
	}
	if err := s.Del(ctx, "token:2:aaa"); err != nil {
		t.Fatalf("del: %v", err)
	}
	if _, err := s.Get(ctx, "token:2:aaa"); err == nil {
		t.Fatalf("expected not found after del")
	}
}

func TestAutoStore_FallbackToMemoryWhenRedisDown(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		DialTimeout:  150 * time.Millisecond,
		ReadTimeout:  150 * time.Millisecond,
		WriteTimeout: 150 * time.Millisecond,
	})
	mem := NewMemoryStore(time.Hour)

	fallbacks := 0
	s := NewAutoStore(rdb, mem, func(op string, err error) { fallbacks++ })
	ctx := context.Background()

	if err := s.Set(ctx, "token:3:aaa", "v1", time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}

	// Simulate Redis outage.
	mr.Close()

	// This should fall back to memory.
	if err := s.Set(ctx, "token:3:bbb", "v2", time.Minute); err != nil {
		t.Fatalf("set fallback: %v", err)
	}
	v, err := s.Get(ctx, "token:3:bbb")
	if err != nil || v != "v2" {
		t.Fatalf("get fallback: v=%q err=%v", v, err)
	}
	if fallbacks == 0 {
		t.Fatalf("expected fallback hook to be called")
	}
}

func TestAutoStore_FallbackToMemoryWhenRedisMiss(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
	})
	mem := NewMemoryStore(time.Hour)
	s := NewAutoStore(rdb, mem, nil)
	ctx := context.Background()

	// Simulate data loss on Redis side (e.g. restart) while memory still has the key.
	if err := mem.Set(ctx, "token:4:aaa", "v1", time.Minute); err != nil {
		t.Fatalf("mem set: %v", err)
	}

	v, err := s.Get(ctx, "token:4:aaa")
	if err != nil || v != "v1" {
		t.Fatalf("get fallback on miss: v=%q err=%v", v, err)
	}
}
