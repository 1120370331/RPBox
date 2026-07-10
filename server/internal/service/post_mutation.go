package service

import (
	"context"
	"errors"
	"log"

	"gorm.io/gorm"
)

// PostMutationService coordinates post-related transactions and list invalidation.
type PostMutationService struct {
	db    *gorm.DB
	lists *PostListService
}

// NewPostMutationService creates a post mutation coordinator.
func NewPostMutationService(db *gorm.DB, lists *PostListService) *PostMutationService {
	return &PostMutationService{db: db, lists: lists}
}

// Global commits a mutation before invalidating all public post candidate pages.
func (s *PostMutationService) Global(
	ctx context.Context,
	mutate func(tx *gorm.DB) error,
) error {
	if err := s.transaction(ctx, mutate); err != nil {
		return err
	}
	if s.lists != nil {
		if err := s.lists.InvalidateGlobal(ctx); err != nil {
			log.Printf("[Cache] post list global invalidation failed: %v", err)
		}
	}
	return nil
}

// Viewer commits a mutation before invalidating one viewer's candidate pages.
func (s *PostMutationService) Viewer(
	ctx context.Context,
	viewerID uint,
	mutate func(tx *gorm.DB) error,
) error {
	if err := s.transaction(ctx, mutate); err != nil {
		return err
	}
	if s.lists != nil {
		if err := s.lists.InvalidateViewer(ctx, viewerID); err != nil {
			log.Printf("[Cache] post list viewer invalidation failed viewer=%d: %v", viewerID, err)
		}
	}
	return nil
}

func (s *PostMutationService) transaction(
	ctx context.Context,
	mutate func(tx *gorm.DB) error,
) error {
	if s == nil || s.db == nil {
		return errors.New("post mutation service: database is not configured")
	}
	if mutate == nil {
		return errors.New("post mutation service: mutation is not configured")
	}
	return s.db.WithContext(ctx).Transaction(mutate)
}
