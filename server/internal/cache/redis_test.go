package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCache(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	t.Helper()
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{
		Addr: server.Addr(),
	})
	return NewRedisCache(client, Options{}), server
}

func TestRedisCacheGetSetDel(t *testing.T) {
	cache, server := newTestCache(t)
	defer server.Close()
	defer cache.Close()

	ctx := context.Background()
	key := Key("test", "getset")
	value := map[string]string{"hello": "world"}

	if err := cache.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}

	var got map[string]string
	if err := cache.Get(ctx, key, &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got["hello"] != "world" {
		t.Fatalf("unexpected value: %v", got)
	}

	if err := cache.Del(ctx, key); err != nil {
		t.Fatalf("del: %v", err)
	}
	if err := cache.Get(ctx, key, &got); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expected cache miss, got %v", err)
	}
}

func TestRedisCacheFetch(t *testing.T) {
	cache, server := newTestCache(t)
	defer server.Close()
	defer cache.Close()

	ctx := context.Background()
	key := Key("test", "fetch")
	calls := 0
	loader := func(ctx context.Context) (interface{}, error) {
		calls++
		return map[string]string{"value": "ok"}, nil
	}

	var first map[string]string
	if err := cache.Fetch(ctx, key, time.Minute, &first, loader); err != nil {
		t.Fatalf("fetch first: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}

	var second map[string]string
	if err := cache.Fetch(ctx, key, time.Minute, &second, loader); err != nil {
		t.Fatalf("fetch second: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected cached result, got %d calls", calls)
	}
}

func TestRedisCacheVersioning(t *testing.T) {
	cache, server := newTestCache(t)
	defer server.Close()
	defer cache.Close()

	ctx := context.Background()
	version, err := cache.Version(ctx, "post:list")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if version != 1 {
		t.Fatalf("expected version 1, got %d", version)
	}

	version, err = cache.BumpVersion(ctx, "post:list")
	if err != nil {
		t.Fatalf("bump version: %v", err)
	}
	if version != 2 {
		t.Fatalf("expected version 2, got %d", version)
	}

	version, err = cache.Version(ctx, "post:list")
	if err != nil {
		t.Fatalf("version after bump: %v", err)
	}
	if version != 2 {
		t.Fatalf("expected version 2, got %d", version)
	}
}

func TestRedisCacheMGet(t *testing.T) {
	cache, server := newTestCache(t)
	defer server.Close()
	defer cache.Close()

	ctx := context.Background()
	keyA := Key("test", "mget", "a")
	keyB := Key("test", "mget", "b")

	if err := cache.Set(ctx, keyA, map[string]int{"a": 1}, time.Minute); err != nil {
		t.Fatalf("set a: %v", err)
	}
	if err := cache.Set(ctx, keyB, map[string]int{"b": 2}, time.Minute); err != nil {
		t.Fatalf("set b: %v", err)
	}

	var outA map[string]int
	var outB map[string]int
	if err := cache.MGet(ctx, []string{keyA, keyB}, []interface{}{&outA, &outB}); err != nil {
		t.Fatalf("mget: %v", err)
	}
	if outA["a"] != 1 || outB["b"] != 2 {
		t.Fatalf("unexpected values: %v %v", outA, outB)
	}
}

func TestRedisCacheIncrBy(t *testing.T) {
	cache, server := newTestCache(t)
	defer server.Close()
	defer cache.Close()

	ctx := context.Background()
	key := Key("test", "incr")

	value, err := cache.IncrBy(ctx, key, 2, time.Minute)
	if err != nil {
		t.Fatalf("incr: %v", err)
	}
	if value != 2 {
		t.Fatalf("expected 2, got %d", value)
	}

	ttl := server.TTL(key)
	if ttl <= 0 {
		t.Fatalf("expected ttl to be set, got %v", ttl)
	}
}
