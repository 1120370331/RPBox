package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rpbox/server/internal/cache"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func TestPostMutationGlobalInvalidatesAfterCommit(t *testing.T) {
	db := testutil.NewTestDB(t, &model.Post{})
	versions := &trackingPostListCache{}
	mutations := NewPostMutationService(db, NewPostListService(db, versions))

	err := mutations.Global(context.Background(), func(tx *gorm.DB) error {
		return tx.Create(&model.Post{
			AuthorID:     1,
			Title:        "committed",
			Content:      "committed",
			Status:       "published",
			ReviewStatus: "approved",
			IsPublic:     true,
		}).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if versions.globalBumps != 1 {
		t.Fatalf("expected one global bump, got %d", versions.globalBumps)
	}
	var count int64
	if err := db.Model(&model.Post{}).Count(&count).Error; err != nil {
		t.Fatalf("count posts: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected committed post, got %d", count)
	}
}

func TestPostMutationDoesNotInvalidateAfterRollback(t *testing.T) {
	db := testutil.NewTestDB(t, &model.Post{})
	versions := &trackingPostListCache{}
	mutations := NewPostMutationService(db, NewPostListService(db, versions))
	expected := errors.New("rollback")

	err := mutations.Global(context.Background(), func(tx *gorm.DB) error {
		if err := tx.Create(&model.Post{
			AuthorID:     1,
			Title:        "rolled back",
			Content:      "rolled back",
			Status:       "published",
			ReviewStatus: "approved",
			IsPublic:     true,
		}).Error; err != nil {
			return err
		}
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected rollback error, got %v", err)
	}
	if versions.globalBumps != 0 {
		t.Fatalf("rollback must not invalidate")
	}

	var count int64
	if err := db.Model(&model.Post{}).Count(&count).Error; err != nil {
		t.Fatalf("count posts: %v", err)
	}
	if count != 0 {
		t.Fatalf("rollback persisted %d posts", count)
	}
}

func TestPostMutationViewerInvalidatesOnlyViewerScope(t *testing.T) {
	db := testutil.NewTestDB(t, &model.UserHiddenContent{})
	versions := &trackingPostListCache{}
	mutations := NewPostMutationService(db, NewPostListService(db, versions))

	err := mutations.Viewer(context.Background(), 7, func(tx *gorm.DB) error {
		return tx.Create(&model.UserHiddenContent{
			UserID:     7,
			TargetType: "post",
			TargetID:   42,
		}).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if versions.globalBumps != 0 {
		t.Fatalf("viewer mutation must not bump global version")
	}
	if len(versions.viewerBumps) != 1 || versions.viewerBumps[0] != 7 {
		t.Fatalf("unexpected viewer bumps: %v", versions.viewerBumps)
	}
}

func TestPostMutationIgnoresInvalidationErrorAfterCommit(t *testing.T) {
	db := testutil.NewTestDB(t, &model.Post{})
	versions := &trackingPostListCache{bumpErr: errors.New("redis unavailable")}
	mutations := NewPostMutationService(db, NewPostListService(db, versions))

	err := mutations.Global(context.Background(), func(tx *gorm.DB) error {
		return tx.Create(&model.Post{
			AuthorID:     1,
			Title:        "committed",
			Content:      "committed",
			Status:       "published",
			ReviewStatus: "approved",
			IsPublic:     true,
		}).Error
	})
	if err != nil {
		t.Fatalf("invalidation error must not fail mutation: %v", err)
	}

	var count int64
	if err := db.Model(&model.Post{}).Count(&count).Error; err != nil {
		t.Fatalf("count posts: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected committed post, got %d", count)
	}
}

func TestPostMutationAllowsNilListAndCacheServices(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		lists *PostListService
	}{
		{name: "nil list service"},
		{name: "nil cache", lists: NewPostListService(nil, nil)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db := testutil.NewTestDB(t, &model.Post{})
			if testCase.lists != nil {
				testCase.lists.db = db
			}
			mutations := NewPostMutationService(db, testCase.lists)

			err := mutations.Global(context.Background(), func(tx *gorm.DB) error {
				return tx.Create(&model.Post{
					AuthorID:     1,
					Title:        testCase.name,
					Content:      testCase.name,
					Status:       "published",
					ReviewStatus: "approved",
					IsPublic:     true,
				}).Error
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

type trackingPostListCache struct {
	globalBumps int
	viewerBumps []uint
	bumpErr     error
}

func (c *trackingPostListCache) Get(context.Context, string, interface{}) error {
	return cache.ErrCacheMiss
}

func (c *trackingPostListCache) Set(context.Context, string, interface{}, time.Duration) error {
	return nil
}

func (c *trackingPostListCache) Del(context.Context, ...string) error {
	return nil
}

func (c *trackingPostListCache) MGet(context.Context, []string, []interface{}) error {
	return cache.ErrCacheMiss
}

func (c *trackingPostListCache) IncrBy(context.Context, string, int64, time.Duration) (int64, error) {
	return 0, cache.ErrCacheMiss
}

func (c *trackingPostListCache) Fetch(
	ctx context.Context,
	_ string,
	_ time.Duration,
	_ interface{},
	loader cache.Fetcher,
) error {
	_, err := loader(ctx)
	return err
}

func (c *trackingPostListCache) Version(context.Context, string) (int64, error) {
	return 1, nil
}

func (c *trackingPostListCache) BumpVersion(_ context.Context, name string) (int64, error) {
	if c.bumpErr != nil {
		return 0, c.bumpErr
	}
	if name == postListGlobalVersionName {
		c.globalBumps++
		return int64(c.globalBumps), nil
	}
	const viewerPrefix = postListViewerVersionName + ":"
	viewerID, err := strconv.ParseUint(strings.TrimPrefix(name, viewerPrefix), 10, 64)
	if err != nil || !strings.HasPrefix(name, viewerPrefix) {
		return 0, fmt.Errorf("unexpected version name %q", name)
	}
	c.viewerBumps = append(c.viewerBumps, uint(viewerID))
	return int64(len(c.viewerBumps)), nil
}

func (c *trackingPostListCache) Close() error {
	return nil
}
