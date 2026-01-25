package cache

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// Options configures RedisCache behavior.
type Options struct {
	Jitter time.Duration
}

// RedisCache implements Cache backed by Redis.
type RedisCache struct {
	client *redis.Client
	group  singleflight.Group
	jitter time.Duration
}

// NewRedisCache creates a Redis-backed cache.
func NewRedisCache(client *redis.Client, opts Options) *RedisCache {
	return &RedisCache{
		client: client,
		jitter: opts.Jitter,
	}
}

// Get reads a cached value into dest.
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	if c == nil || c.client == nil {
		return ErrCacheMiss
	}
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Set writes a value to cache with TTL.
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.setBytes(ctx, key, data, ttl)
}

// Del removes keys from cache.
func (c *RedisCache) Del(ctx context.Context, keys ...string) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

// MGet reads multiple cached values.
func (c *RedisCache) MGet(ctx context.Context, keys []string, dest []interface{}) error {
	if len(keys) != len(dest) {
		return errors.New("cache: keys and dest size mismatch")
	}
	if c == nil || c.client == nil {
		return ErrCacheMiss
	}
	values, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}
	for i, v := range values {
		if v == nil {
			return ErrCacheMiss
		}
		var data []byte
		switch typed := v.(type) {
		case string:
			data = []byte(typed)
		case []byte:
			data = typed
		default:
			return errors.New("cache: unexpected value type")
		}
		if err := json.Unmarshal(data, dest[i]); err != nil {
			return err
		}
	}
	return nil
}

// IncrBy increments a counter and sets TTL if it does not exist.
func (c *RedisCache) IncrBy(ctx context.Context, key string, n int64, ttl time.Duration) (int64, error) {
	if c == nil || c.client == nil {
		return 0, ErrCacheMiss
	}
	pipe := c.client.TxPipeline()
	incr := pipe.IncrBy(ctx, key, n)
	if ttl > 0 {
		pipe.ExpireNX(ctx, key, ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

// Fetch returns cached data or loads via loader and caches it.
func (c *RedisCache) Fetch(ctx context.Context, key string, ttl time.Duration, dest interface{}, loader Fetcher) error {
	if err := c.Get(ctx, key, dest); err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	value, err, _ := c.group.Do(key, func() (interface{}, error) {
		return loader(ctx)
	})
	if err != nil {
		return err
	}
	if value == nil {
		return errors.New("cache: loader returned nil")
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_ = c.setBytes(ctx, key, data, addJitter(ttl, c.jitter))
	return json.Unmarshal(data, dest)
}

// Version returns the current version for a cache namespace.
func (c *RedisCache) Version(ctx context.Context, name string) (int64, error) {
	if c == nil || c.client == nil {
		return 0, ErrCacheMiss
	}
	key := VersionKey(name)
	version, err := c.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		ok, setErr := c.client.SetNX(ctx, key, 1, 0).Result()
		if setErr != nil {
			return 0, setErr
		}
		if ok {
			return 1, nil
		}
		return c.client.Get(ctx, key).Int64()
	}
	if err != nil {
		return 0, err
	}
	return version, nil
}

// BumpVersion increments a cache namespace version.
func (c *RedisCache) BumpVersion(ctx context.Context, name string) (int64, error) {
	if c == nil || c.client == nil {
		return 0, ErrCacheMiss
	}
	return c.client.Incr(ctx, VersionKey(name)).Result()
}

// Close releases the underlying Redis client.
func (c *RedisCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *RedisCache) setBytes(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func addJitter(ttl, jitter time.Duration) time.Duration {
	if ttl <= 0 || jitter <= 0 {
		return ttl
	}
	delta := rand.Int63n(int64(jitter))
	return ttl + time.Duration(delta)
}
