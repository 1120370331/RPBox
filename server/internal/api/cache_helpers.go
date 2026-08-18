package api

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rpbox/server/internal/cache"
)

const (
	// v2 uses canonical filter identities and non-reusable Redis generations.
	postListCacheName = "post:list:v2"
	rpdbListCacheName = "rpdb:list"
)

func (s *Server) userProfileCacheKey(userID string) string {
	return cache.Key("user", "public", "v2", userID)
}

func (s *Server) invalidateUserProfileCache(ctx context.Context, userID uint) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, s.userProfileCacheKey(fmt.Sprint(userID)))
}

func (s *Server) bumpPostListCache(ctx context.Context) {
	if s.cache == nil {
		return
	}

	// A database commit must not leave list caches stale merely because the
	// client disconnected before invalidation ran. Keep the fallback bounded so
	// Redis problems never turn a successful content mutation into a failure.
	if ctx == nil || ctx.Err() != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
		defer cancel()
	}
	if _, err := s.cache.BumpVersion(ctx, postListCacheName); err != nil {
		log.Printf("[Cache] failed to invalidate %s: %v", postListCacheName, err)
	}
}

func (s *Server) bumpRPDBListCache(ctx context.Context) {
	if s.cache == nil {
		return
	}
	_, _ = s.cache.BumpVersion(ctx, rpdbListCacheName)
}
