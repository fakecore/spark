package kvstore

import (
	"context"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
)

type MemoryStore struct {
	c *gocache.Cache
}

func NewMemoryStore(defaultTTL time.Duration) *MemoryStore {
	if defaultTTL <= 0 {
		defaultTTL = 24 * time.Hour
	}
	// Cleanup interval doesn't need to be aggressive.
	return &MemoryStore{c: gocache.New(defaultTTL, 10*time.Minute)}
}

func (s *MemoryStore) Backend() string { return "memory" }

func (s *MemoryStore) Health(ctx context.Context) (bool, string) { return true, "ok" }

func (s *MemoryStore) Get(ctx context.Context, key string) (string, error) {
	v, ok := s.c.Get(key)
	if !ok {
		return "", ErrNotFound
	}
	str, ok := v.(string)
	if !ok {
		// We only store strings; treat other types as missing.
		return "", ErrNotFound
	}
	return str, nil
}

func (s *MemoryStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = gocache.DefaultExpiration
	}
	s.c.Set(key, value, ttl)
	return nil
}

func (s *MemoryStore) Del(ctx context.Context, keys ...string) error {
	for _, k := range keys {
		s.c.Delete(k)
	}
	return nil
}

func (s *MemoryStore) Keys(ctx context.Context, pattern string) ([]string, error) {
	// DoorX uses pattern like "token:<uid>:*". We support "prefix*" and exact match.
	prefix := pattern
	isPrefix := false
	if strings.HasSuffix(pattern, "*") {
		isPrefix = true
		prefix = strings.TrimSuffix(pattern, "*")
	}

	items := s.c.Items()
	out := make([]string, 0, len(items))
	for k := range items {
		if isPrefix {
			if strings.HasPrefix(k, prefix) {
				out = append(out, k)
			}
		} else if k == prefix {
			out = append(out, k)
		}
	}
	return out, nil
}

func (s *MemoryStore) AsRedis() (*redis.Client, bool) { return nil, false }
