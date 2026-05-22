package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestUpdateStoryMusicSegmentClearsEndEntryID(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Story{},
		&model.StoryEntry{},
		&model.StoryMusicTrack{},
		&model.StoryMusicTrackStory{},
		&model.StoryMusicSegment{},
	)

	user := model.User{Username: "owner", Email: "owner@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	now := time.Now()
	story := model.Story{
		UserID:    user.ID,
		Title:     "story",
		StartTime: now,
		EndTime:   now,
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	entries := []model.StoryEntry{
		{StoryID: story.ID, SourceID: "start", Content: "start", Timestamp: now, SortOrder: 1},
		{StoryID: story.ID, SourceID: "end", Content: "end", Timestamp: now, SortOrder: 2},
	}
	if err := db.Create(&entries).Error; err != nil {
		t.Fatalf("create entries: %v", err)
	}

	track := model.StoryMusicTrack{
		UserID:   user.ID,
		Name:     "track",
		FileName: "track.mp3",
		MimeType: "audio/mpeg",
		URL:      "/music/track.mp3",
		Color:    "#B87333",
		Volume:   0.75,
	}
	if err := db.Create(&track).Error; err != nil {
		t.Fatalf("create track: %v", err)
	}

	segment := model.StoryMusicSegment{
		StoryID:      story.ID,
		TrackID:      track.ID,
		StartEntryID: entries[0].ID,
		EndEntryID:   &entries[1].ID,
		Loop:         true,
		AutoPlay:     true,
		Volume:       0.75,
	}
	if err := db.Create(&segment).Error; err != nil {
		t.Fatalf("create segment: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, user)
	resp := performRequest(
		server.router,
		http.MethodPut,
		fmt.Sprintf("/api/v1/stories/%d/music/segments/%d", story.ID, segment.ID),
		map[string]interface{}{"endEntryId": nil},
		token,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload model.StoryMusicSegment
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.EndEntryID != nil {
		t.Fatalf("expected response endEntryId nil, got %v", *payload.EndEntryID)
	}

	var refreshed model.StoryMusicSegment
	if err := db.First(&refreshed, segment.ID).Error; err != nil {
		t.Fatalf("reload segment: %v", err)
	}
	if refreshed.EndEntryID != nil {
		t.Fatalf("expected stored endEntryId nil, got %v", *refreshed.EndEntryID)
	}
}
