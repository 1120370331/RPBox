package api

import (
	"context"
	"fmt"

	"github.com/rpbox/server/internal/cache"
)

const (
	postListCacheName = "post:list"
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
	_, _ = s.cache.BumpVersion(ctx, postListCacheName)
}

func (s *Server) bumpRPDBListCache(ctx context.Context) {
	if s.cache == nil {
		return
	}
	_, _ = s.cache.BumpVersion(ctx, rpdbListCacheName)
}
