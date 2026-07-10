package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rpbox/server/internal/cache"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func TestPostListQueryCacheKeySeparatesFilters(t *testing.T) {
	base := PostListQuery{
		ViewerID: 7,
		Page:     1,
		PageSize: 12,
		SortBy:   "created_at",
		Order:    "desc",
		Status:   "published",
	}

	pageTwo := base
	pageTwo.Page = 2
	category := base
	category.Category = "event"
	pinned := true
	pinnedQuery := base
	pinnedQuery.IsPinned = &pinned

	keys := map[string]struct{}{
		base.cacheSuffix():        {},
		pageTwo.cacheSuffix():     {},
		category.cacheSuffix():    {},
		pinnedQuery.cacheSuffix(): {},
	}
	if len(keys) != 4 {
		t.Fatalf("expected isolated keys, got %d", len(keys))
	}
}

func TestPostListQueryBypassesVolatileSorts(t *testing.T) {
	for _, sortBy := range []string{"like_count", "view_count"} {
		query := PostListQuery{SortBy: sortBy}
		if query.Cacheable() {
			t.Fatalf("%s must bypass candidate cache", sortBy)
		}
	}
}

func TestPostListCandidatesCachesIdenticalQueries(t *testing.T) {
	db, lists, redisServer := newPostListTestService(t)
	defer redisServer.Close()

	first := createPublicPost(t, db, 10, "first", time.Now().Add(-time.Minute))
	query := PostListQuery{ViewerID: 7, Page: 1, PageSize: 20}

	initial, err := lists.Candidates(context.Background(), query)
	if err != nil {
		t.Fatalf("first candidates: %v", err)
	}
	if len(initial.IDs) != 1 || initial.IDs[0] != first.ID {
		t.Fatalf("unexpected first candidates: %+v", initial)
	}

	createPublicPost(t, db, 11, "second", time.Now())

	cached, err := lists.Candidates(context.Background(), query)
	if err != nil {
		t.Fatalf("cached candidates: %v", err)
	}
	if len(cached.IDs) != 1 || cached.IDs[0] != first.ID {
		t.Fatalf("expected cached candidate page, got %+v", cached)
	}
}

func TestPostListGlobalInvalidationRefreshesCandidates(t *testing.T) {
	db, lists, redisServer := newPostListTestService(t)
	defer redisServer.Close()

	first := createPublicPost(t, db, 10, "first", time.Now().Add(-time.Minute))
	query := PostListQuery{ViewerID: 7, Page: 1, PageSize: 20}

	if _, err := lists.Candidates(context.Background(), query); err != nil {
		t.Fatalf("populate candidates: %v", err)
	}
	second := createPublicPost(t, db, 11, "second", time.Now())

	if err := lists.InvalidateGlobal(context.Background()); err != nil {
		t.Fatalf("invalidate global: %v", err)
	}

	refreshed, err := lists.Candidates(context.Background(), query)
	if err != nil {
		t.Fatalf("refreshed candidates: %v", err)
	}
	if len(refreshed.IDs) != 2 || refreshed.IDs[0] != second.ID || refreshed.IDs[1] != first.ID {
		t.Fatalf("unexpected refreshed candidates: %+v", refreshed)
	}
}

func TestPostListViewerInvalidationIsIsolated(t *testing.T) {
	db, lists, redisServer := newPostListTestService(t)
	defer redisServer.Close()

	post := createPublicPost(t, db, 20, "blocked author", time.Now())
	viewerSeven := PostListQuery{ViewerID: 7, Page: 1, PageSize: 20}
	viewerEight := PostListQuery{ViewerID: 8, Page: 1, PageSize: 20}

	for _, query := range []PostListQuery{viewerSeven, viewerEight} {
		page, err := lists.Candidates(context.Background(), query)
		if err != nil {
			t.Fatalf("populate viewer %d: %v", query.ViewerID, err)
		}
		if len(page.IDs) != 1 || page.IDs[0] != post.ID {
			t.Fatalf("unexpected initial viewer %d page: %+v", query.ViewerID, page)
		}
	}

	if err := db.Create(&model.UserBlock{BlockerID: 7, BlockedUserID: 20}).Error; err != nil {
		t.Fatalf("create block: %v", err)
	}
	if err := lists.InvalidateViewer(context.Background(), 7); err != nil {
		t.Fatalf("invalidate viewer 7: %v", err)
	}

	sevenPage, err := lists.Candidates(context.Background(), viewerSeven)
	if err != nil {
		t.Fatalf("viewer 7 candidates: %v", err)
	}
	if len(sevenPage.IDs) != 0 {
		t.Fatalf("viewer 7 should exclude blocked author: %+v", sevenPage)
	}

	eightPage, err := lists.Candidates(context.Background(), viewerEight)
	if err != nil {
		t.Fatalf("viewer 8 candidates: %v", err)
	}
	if len(eightPage.IDs) != 1 || eightPage.IDs[0] != post.ID {
		t.Fatalf("viewer 8 cache should remain isolated: %+v", eightPage)
	}
}

func TestPostListCandidatesFallsBackWhenRedisUnavailable(t *testing.T) {
	db, lists, redisServer := newPostListTestService(t)
	post := createPublicPost(t, db, 10, "database fallback", time.Now())
	redisServer.Close()

	page, err := lists.Candidates(context.Background(), PostListQuery{
		ViewerID: 7,
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("database fallback: %v", err)
	}
	if len(page.IDs) != 1 || page.IDs[0] != post.ID {
		t.Fatalf("unexpected fallback page: %+v", page)
	}
}

func TestPostListCandidatesAppliesVisibilityFilters(t *testing.T) {
	db, lists, redisServer := newPostListTestService(t)
	defer redisServer.Close()

	visible := createPublicPost(t, db, 10, "visible event", time.Now())
	visible.Category = "event"
	visible.Region = "Moon Guard"
	visible.Address = "Stormwind"
	visible.IsPinned = true
	if err := db.Save(&visible).Error; err != nil {
		t.Fatalf("update visible post: %v", err)
	}

	hidden := createPublicPost(t, db, 11, "hidden event", time.Now().Add(-time.Minute))
	hidden.Category = "event"
	if err := db.Save(&hidden).Error; err != nil {
		t.Fatalf("update hidden post: %v", err)
	}
	if err := db.Create(&model.UserHiddenContent{
		UserID:     7,
		TargetType: "post",
		TargetID:   hidden.ID,
	}).Error; err != nil {
		t.Fatalf("hide post: %v", err)
	}
	if err := db.Create(&model.PostTag{PostID: visible.ID, TagID: 42}).Error; err != nil {
		t.Fatalf("tag visible post: %v", err)
	}

	pinned := true
	page, err := lists.Candidates(context.Background(), PostListQuery{
		ViewerID: 7,
		Page:     1,
		PageSize: 20,
		Search:   "visible",
		Region:   "Moon",
		Address:  "Storm",
		TagID:    "42",
		AuthorID: "10",
		Category: "event",
		IsPinned: &pinned,
	})
	if err != nil {
		t.Fatalf("filtered candidates: %v", err)
	}
	if len(page.IDs) != 1 || page.IDs[0] != visible.ID {
		t.Fatalf("unexpected filtered page: %+v", page)
	}
}

func newPostListTestService(t *testing.T) (*gorm.DB, *PostListService, *miniredis.Miniredis) {
	t.Helper()

	db := testutil.NewTestDB(
		t,
		&model.Post{},
		&model.User{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
		&model.PostTag{},
	)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{
		Addr:         redisServer.Addr(),
		MaxRetries:   0,
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
	})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})

	return db, NewPostListService(db, cache.NewRedisCache(redisClient, cache.Options{})), redisServer
}

func createPublicPost(t *testing.T, db *gorm.DB, authorID uint, title string, createdAt time.Time) model.Post {
	t.Helper()

	var user model.User
	if err := db.FirstOrCreate(&user, model.User{
		ID:       authorID,
		Username: fmt.Sprintf("user-%d", authorID),
		Email:    fmt.Sprintf("user-%d@example.com", authorID),
	}).Error; err != nil {
		t.Fatalf("create author %d: %v", authorID, err)
	}

	post := model.Post{
		AuthorID:     authorID,
		Title:        title,
		Content:      title,
		Status:       "published",
		ReviewStatus: "approved",
		IsPublic:     true,
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post %q: %v", title, err)
	}
	return post
}
