package kvstore

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("kvstore: not found")

// Store is a minimal KV interface used by auth/session/token logic.
// It is intentionally tiny so we can support Redis and in-memory backends.
type Store interface {
	Backend() string
	Health(ctx context.Context) (ok bool, detail string)

	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error

	// Keys returns keys matching a pattern.
	// Implementations may support only the common "prefix*" pattern used by DoorX.
	Keys(ctx context.Context, pattern string) ([]string, error)

	// AsRedis returns the underlying redis client if backed by Redis.
	AsRedis() (*redis.Client, bool)
}
