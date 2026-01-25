package cache

import (
	"context"
	"errors"
	"time"
)

// ErrCacheMiss indicates the key is not present in cache.
var ErrCacheMiss = errors.New("cache miss")

// Fetcher loads data when cache is missing.
type Fetcher func(ctx context.Context) (interface{}, error)

// Cache defines a minimal cache interface for the service.
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	MGet(ctx context.Context, keys []string, dest []interface{}) error
	IncrBy(ctx context.Context, key string, n int64, ttl time.Duration) (int64, error)
	Fetch(ctx context.Context, key string, ttl time.Duration, dest interface{}, loader Fetcher) error
	Version(ctx context.Context, name string) (int64, error)
	BumpVersion(ctx context.Context, name string) (int64, error)
	Close() error
}
