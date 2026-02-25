package kvstore

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type FallbackHook func(op string, err error)

// AutoStore tries Redis first and falls back to memory on Redis errors.
// It is meant for To-C installs where availability is more important than strict consistency.
type AutoStore struct {
	redis *RedisStore
	mem   *MemoryStore
	hook  FallbackHook
}

func NewAutoStore(redisClient *redis.Client, mem *MemoryStore, hook FallbackHook) *AutoStore {
	var rs *RedisStore
	if redisClient != nil {
		rs = NewRedisStore(redisClient)
	}
	if mem == nil {
		mem = NewMemoryStore(24 * time.Hour)
	}
	return &AutoStore{redis: rs, mem: mem, hook: hook}
}

func (s *AutoStore) Backend() string {
	if s.redis != nil {
		return "auto(redis+memory)"
	}
	return "auto(memory)"
}

func (s *AutoStore) Health(ctx context.Context) (bool, string) {
	if s.redis == nil {
		return true, "memory (no redis configured)"
	}
	if ok, detail := s.redis.Health(ctx); ok {
		return true, "redis ok"
	} else {
		return true, fmt.Sprintf("memory fallback (%s)", detail)
	}
}

func (s *AutoStore) Get(ctx context.Context, key string) (string, error) {
	if s.redis != nil {
		v, err := s.redis.Get(ctx, key)
		if err == nil {
			return v, nil
		}
		// For auto mode we prefer availability: fall back to memory on both
		// Redis errors and Redis misses. This keeps sessions usable during Redis
		// restarts/outages for single-process To-C installs.
		if s.hook != nil && err != ErrNotFound {
			s.hook("get", err)
		}
	}
	return s.mem.Get(ctx, key)
}

func (s *AutoStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if s.redis != nil {
		err := s.redis.Set(ctx, key, value, ttl)
		if err == nil {
			// Mirror into memory so reads can fall back during Redis downtime.
			_ = s.mem.Set(ctx, key, value, ttl)
			return nil
		}
		if s.hook != nil {
			s.hook("set", err)
		}
	}
	return s.mem.Set(ctx, key, value, ttl)
}

func (s *AutoStore) Del(ctx context.Context, keys ...string) error {
	var lastErr error
	if s.redis != nil {
		if err := s.redis.Del(ctx, keys...); err != nil {
			lastErr = err
			if s.hook != nil {
				s.hook("del", err)
			}
		}
	}
	if err := s.mem.Del(ctx, keys...); err != nil && lastErr == nil {
		lastErr = err
	}
	return lastErr
}

func (s *AutoStore) Keys(ctx context.Context, pattern string) ([]string, error) {
	if s.redis != nil {
		keys, err := s.redis.Keys(ctx, pattern)
		if err == nil {
			return keys, nil
		}
		if s.hook != nil {
			s.hook("keys", err)
		}
	}
	return s.mem.Keys(ctx, pattern)
}

func (s *AutoStore) AsRedis() (*redis.Client, bool) {
	if s.redis == nil {
		return nil, false
	}
	return s.redis.AsRedis()
}
