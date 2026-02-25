package kvstore

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	c *redis.Client
}

func NewRedisStore(c *redis.Client) *RedisStore { return &RedisStore{c: c} }

func (s *RedisStore) Backend() string { return "redis" }

func (s *RedisStore) Health(ctx context.Context) (bool, string) {
	if s.c == nil {
		return false, "nil client"
	}
	if err := s.c.Ping(ctx).Err(); err != nil {
		return false, fmt.Sprintf("ping: %v", err)
	}
	return true, "ok"
}

func (s *RedisStore) Get(ctx context.Context, key string) (string, error) {
	if s.c == nil {
		return "", fmt.Errorf("redis client is nil")
	}
	v, err := s.c.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return v, nil
}

func (s *RedisStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if s.c == nil {
		return fmt.Errorf("redis client is nil")
	}
	return s.c.Set(ctx, key, value, ttl).Err()
}

func (s *RedisStore) Del(ctx context.Context, keys ...string) error {
	if s.c == nil {
		return fmt.Errorf("redis client is nil")
	}
	return s.c.Del(ctx, keys...).Err()
}

func (s *RedisStore) Keys(ctx context.Context, pattern string) ([]string, error) {
	if s.c == nil {
		return nil, fmt.Errorf("redis client is nil")
	}

	// Use SCAN to avoid blocking Redis with KEYS.
	var (
		cursor uint64
		out    []string
	)
	for {
		keys, next, err := s.c.Scan(ctx, cursor, pattern, 500).Result()
		if err != nil {
			return nil, err
		}
		out = append(out, keys...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return out, nil
}

func (s *RedisStore) AsRedis() (*redis.Client, bool) { return s.c, s.c != nil }
