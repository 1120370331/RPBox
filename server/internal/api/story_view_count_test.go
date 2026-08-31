package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func TestStoryViewCountTracksSuccessfulReaders(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Story{},
		&model.StoryEntry{},
		&model.StoryGuild{},
		&model.StoryMusicTrack{},
		&model.StoryMusicTrackStory{},
		&model.StoryMusicSegment{},
	)
	owner := model.User{Username: "story-owner", Email: "story-owner@example.com", PassHash: "hash"}
	viewer := model.User{Username: "story-viewer", Email: "story-viewer@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&owner, &viewer}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	story := model.Story{
		UserID:    owner.ID,
		Title:     "view-count story",
		IsPublic:  true,
		ShareCode: "viewtest",
		ViewCount: 5,
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}
	if err := db.Create(&model.StoryGuild{StoryID: story.ID, GuildID: 1, AddedBy: owner.ID}).Error; err != nil {
		t.Fatalf("archive story to guild: %v", err)
	}

	server := newTestServer(t, db)
	storyPath := fmt.Sprintf("/api/v1/stories/%d", story.ID)

	ownerResponse := performRequest(server.router, http.MethodGet, storyPath, nil, newTestToken(t, owner))
	assertStoryViewCountResponse(t, ownerResponse, http.StatusOK, 5)
	assertStoredStoryViewCount(t, db, story.ID, 5)

	viewerResponse := performRequest(server.router, http.MethodGet, storyPath, nil, newTestToken(t, viewer))
	assertStoryViewCountResponse(t, viewerResponse, http.StatusOK, 6)
	assertStoredStoryViewCount(t, db, story.ID, 6)

	publicResponse := performRequest(server.router, http.MethodGet, "/api/v1/public/stories/viewtest", nil, "")
	assertStoryViewCountResponse(t, publicResponse, http.StatusOK, 7)
	assertStoredStoryViewCount(t, db, story.ID, 7)
}

func assertStoryViewCountResponse(t *testing.T, response *httptest.ResponseRecorder, expectedStatus, expectedCount int) {
	t.Helper()
	httpResponse := response.Result()
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, httpResponse.StatusCode)
	}
	var payload struct {
		Story model.Story `json:"story"`
	}
	if err := json.NewDecoder(httpResponse.Body).Decode(&payload); err != nil {
		t.Fatalf("decode story response: %v", err)
	}
	if payload.Story.ViewCount != expectedCount {
		t.Fatalf("response view_count = %d, want %d", payload.Story.ViewCount, expectedCount)
	}
}

func assertStoredStoryViewCount(t *testing.T, db *gorm.DB, storyID uint, expectedCount int) {
	t.Helper()
	var story model.Story
	if err := db.First(&story, storyID).Error; err != nil {
		t.Fatalf("reload story: %v", err)
	}
	if story.ViewCount != expectedCount {
		t.Fatalf("stored view_count = %d, want %d", story.ViewCount, expectedCount)
	}
}
