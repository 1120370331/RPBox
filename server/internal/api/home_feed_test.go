package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func newHomeFeedDB(t *testing.T) (*gorm.DB, model.User, model.User, model.User) {
	t.Helper()
	db := testutil.NewTestDB(t, &model.User{}, &model.Post{}, &model.Item{}, &model.RPDBWork{},
		&model.PostLike{}, &model.PostFavorite{}, &model.PostView{}, &model.PostTag{},
		&model.ItemLike{}, &model.ItemFavorite{}, &model.ItemView{}, &model.ItemTag{},
		&model.RPDBLike{}, &model.RPDBFavorite{}, &model.RPDBViewEvent{}, &model.RPDBTag{},
		&model.UserBlock{}, &model.UserHiddenContent{})
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })
	users := []model.User{
		{Username: "reader", Email: "reader@home.test", PassHash: "hash"},
		{Username: "writer", Email: "writer@home.test", PassHash: "hash"},
		{Username: "creator", Email: "creator@home.test", PassHash: "hash"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	return db, users[0], users[1], users[2]
}

func homeCreate(t *testing.T, db *gorm.DB, value interface{}) {
	t.Helper()
	if err := db.Create(value).Error; err != nil {
		t.Fatal(err)
	}
}

func TestHomeFeaturedUsesSevenDayInteractionsAndEnforcesVisibility(t *testing.T) {
	db, viewer, author, other := newHomeFeedDB(t)
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	old := model.Post{AuthorID: author.ID, Title: "Lifetime popular", Content: "secret body", Status: "published", ReviewStatus: "approved", IsPublic: true, LikeCount: 99999, CreatedAt: now.Add(-365 * 24 * time.Hour)}
	post := model.Post{AuthorID: author.ID, Title: "Recent post", Content: "body", Status: "published", ReviewStatus: "approved", IsPublic: true, CoverImage: "data:image/png;base64,SECRET", CreatedAt: now.Add(-30 * 24 * time.Hour)}
	item := model.Item{AuthorID: other.ID, Name: "Recent favorite", Status: "published", ReviewStatus: "approved", IsPublic: true}
	work := model.RPDBWork{AuthorID: other.ID, Title: "Recent work", Status: "published", ReviewStatus: "approved", Visibility: "public", IsPublic: true}
	for _, value := range []interface{}{&old, &post, &item, &work} {
		homeCreate(t, db, value)
	}
	homeCreate(t, db, &model.PostLike{PostID: old.ID, UserID: viewer.ID, CreatedAt: now.Add(-8 * 24 * time.Hour)})
	homeCreate(t, db, &model.PostLike{PostID: old.ID, UserID: other.ID, CreatedAt: now.Add(time.Hour)})
	homeCreate(t, db, &model.PostLike{PostID: post.ID, UserID: viewer.ID, CreatedAt: now.Add(-time.Hour)})
	homeCreate(t, db, &model.ItemFavorite{ItemID: item.ID, UserID: viewer.ID, CreatedAt: now.Add(-time.Hour)})
	homeCreate(t, db, &model.RPDBLike{WorkID: work.ID, UserID: viewer.ID, CreatedAt: now.Add(-time.Hour)})
	homeCreate(t, db, &model.RPDBViewEvent{WorkID: work.ID, UserID: viewer.ID, ViewDate: "2026-10-22", CreatedAt: now.Add(-time.Hour)})
	feed, err := buildHomeFeed(db, viewer.ID, false, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Entries) != 3 || feed.Entries[0].Kind != "item" || feed.Entries[0].WeeklyScore != 5 || feed.Entries[1].Kind != "rpdb" || feed.Entries[1].WeeklyScore != 4 || feed.Entries[2].ID != post.ID || feed.Entries[2].WeeklyScore != 3 {
		t.Fatalf("incorrect seven-day ranking: %+v", feed.Entries)
	}
	payload, _ := json.Marshal(feed)
	if strings.Contains(string(payload), "base64") || strings.Contains(string(payload), "secret body") {
		t.Fatalf("feed leaked image data or body: %s", payload)
	}
	// Public feeds cannot expose these states even to their author.
	for _, mutation := range []map[string]interface{}{
		{"is_public": false}, {"status": "draft"}, {"review_status": "pending"},
	} {
		hidden := model.Post{AuthorID: author.ID, Title: "Hidden", Content: "body", Status: "published", ReviewStatus: "approved", IsPublic: true}
		homeCreate(t, db, &hidden)
		if err := db.Model(&hidden).Updates(mutation).Error; err != nil {
			t.Fatal(err)
		}
		homeCreate(t, db, &model.PostFavorite{PostID: hidden.ID, UserID: viewer.ID, CreatedAt: now})
	}
	if err := db.Model(&work).Update("visibility", "guild").Error; err != nil {
		t.Fatal(err)
	}
	homeCreate(t, db, &model.UserHiddenContent{UserID: viewer.ID, TargetType: reportTargetItem, TargetID: item.ID})
	homeCreate(t, db, &model.UserBlock{BlockerID: viewer.ID, BlockedUserID: author.ID})
	for _, personalized := range []bool{false, true} {
		filtered, err := buildHomeFeed(db, viewer.ID, personalized, nil, now)
		if err != nil || len(filtered.Entries) != 0 {
			t.Fatalf("hidden/private/blocked content appeared: %+v err=%v", filtered, err)
		}
	}
	if err := db.Model(&other).Update("account_deleted_at", now).Error; err != nil {
		t.Fatal(err)
	}
	deleted, err := buildHomeFeed(db, author.ID, false, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range deleted.Entries {
		if entry.AuthorID == other.ID {
			t.Fatal("deleted author's content appeared")
		}
	}
}

func TestHomeRecommendationsUsePersonalInterestsAndRotateTypedIDs(t *testing.T) {
	db, viewer, author, other := newHomeFeedDB(t)
	now := time.Now()
	seed := model.Post{AuthorID: author.ID, Title: "Read before", Content: "body", Category: "novel", Status: "published", ReviewStatus: "approved", IsPublic: true}
	match := model.Post{AuthorID: other.ID, Title: "Another novel", Content: "body", Category: "novel", Status: "published", ReviewStatus: "approved", IsPublic: true}
	unrelated := model.Post{AuthorID: other.ID, Title: "Other category", Content: "body", Category: "other", Status: "published", ReviewStatus: "approved", IsPublic: true}
	item := model.Item{AuthorID: other.ID, Name: "Shared tag", Status: "published", ReviewStatus: "approved", IsPublic: true}
	for _, value := range []interface{}{&seed, &match, &unrelated, &item} {
		homeCreate(t, db, value)
	}
	homeCreate(t, db, &model.PostFavorite{PostID: seed.ID, UserID: viewer.ID, CreatedAt: now.Add(-time.Hour)})
	homeCreate(t, db, &model.PostTag{PostID: seed.ID, TagID: 42})
	homeCreate(t, db, &model.ItemTag{ItemID: item.ID, TagID: 42})
	feed, err := buildHomeFeed(db, viewer.ID, true, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if !feed.Personalized || !feed.HasMore || len(feed.Entries) != 3 || feed.Entries[0].Kind != "item" || feed.Entries[1].ID != match.ID || feed.Entries[0].Reason != "interest" {
		t.Fatalf("interests not reflected: %+v", feed)
	}
	excluded := map[string]bool{}
	for _, entry := range feed.Entries {
		excluded[fmt.Sprintf("%s:%d", entry.Kind, entry.ID)] = true
	}
	next, err := buildHomeFeed(db, viewer.ID, true, excluded, now)
	if err != nil || len(next.Entries) != 1 {
		t.Fatalf("rotation failed: %+v err=%v", next, err)
	}
	for _, entry := range next.Entries {
		if excluded[fmt.Sprintf("%s:%d", entry.Kind, entry.ID)] {
			t.Fatal("rotation repeated an excluded content key")
		}
	}
	cold, err := buildHomeFeed(db, other.ID, true, nil, now)
	if err != nil || cold.Personalized || len(cold.Entries) == 0 {
		t.Fatalf("new user fallback failed: %+v err=%v", cold, err)
	}
}

func TestHomeFeedHTTPRequiresAuthAndValidatesExclusions(t *testing.T) {
	db, viewer, _, _ := newHomeFeedDB(t)
	s := newTestServer(t, db)
	for _, path := range []string{"/api/v1/home/featured", "/api/v1/home/recommendations"} {
		if response := performRequest(s.router, http.MethodGet, path, nil, ""); response.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated request status=%d", response.Code)
		}
		response := performRequest(s.router, http.MethodGet, path, nil, newTestToken(t, viewer))
		if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(response.Body.String(), `"entries":[]`) {
			t.Fatalf("empty feed contract: %d %s", response.Code, response.Body.String())
		}
	}
	for _, exclude := range []string{"post:abc", "invalid:1", "post:0", strings.Repeat("post:1,", 61)} {
		response := performRequest(s.router, http.MethodGet, "/api/v1/home/recommendations?exclude="+exclude, nil, newTestToken(t, viewer))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid exclusion accepted: %s -> %d", exclude, response.Code)
		}
	}
}
