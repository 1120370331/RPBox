package api

import (
	"context"
	"fmt"

	"github.com/rpbox/server/internal/cache"
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
