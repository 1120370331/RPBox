package api

import (
	"context"

	"github.com/rpbox/server/internal/database"
	"gorm.io/gorm"
)

func (s *Server) mutatePostListsGlobal(
	ctx context.Context,
	mutate func(tx *gorm.DB) error,
) error {
	if s != nil && s.postMutations != nil {
		return s.postMutations.Global(ctx, mutate)
	}
	return database.DB.WithContext(ctx).Transaction(mutate)
}

func (s *Server) mutatePostListsViewer(
	ctx context.Context,
	viewerID uint,
	mutate func(tx *gorm.DB) error,
) error {
	if s != nil && s.postMutations != nil {
		return s.postMutations.Viewer(ctx, viewerID, mutate)
	}
	return database.DB.WithContext(ctx).Transaction(mutate)
}
