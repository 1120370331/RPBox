package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
	"github.com/rpbox/server/internal/testutil"
)

func TestUserProfileAchievementStatsUseSignInDaysAndStoryEntries(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Story{},
		&model.StoryEntry{},
		&model.UserDailyActivity{},
		&model.UserActivityLog{},
	)
	user := model.User{Username: "achievement-user", Email: "achievement@example.com", PassHash: "hash"}
	other := model.User{Username: "other-user", Email: "other@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}

	story := model.Story{UserID: user.ID, Title: "archived"}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}
	otherStory := model.Story{UserID: other.ID, Title: "other"}
	if err := db.Create(&otherStory).Error; err != nil {
		t.Fatalf("create other story: %v", err)
	}
	entries := []model.StoryEntry{
		{StoryID: story.ID, Content: "line 1"},
		{StoryID: story.ID, Content: "line 2"},
		{StoryID: story.ID, Content: "line 3"},
		{StoryID: otherStory.ID, Content: "other line"},
	}
	if err := db.Create(&entries).Error; err != nil {
		t.Fatalf("create story entries: %v", err)
	}

	now := time.Now()
	signedAt := now
	for _, day := range []time.Time{service.DayStart(now), service.DayStart(now).AddDate(0, 0, -1)} {
		if err := db.Create(&model.UserDailyActivity{
			UserID:       user.ID,
			ActivityDate: day,
			SignedInAt:   &signedAt,
		}).Error; err != nil {
			t.Fatalf("create daily activity: %v", err)
		}
	}

	server := newTestServer(t, db)
	resp := performRequest(server.router, http.MethodGet, fmt.Sprintf("/api/v1/users/%d", user.ID), nil, newTestToken(t, user))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		StoryCount            int64 `json:"story_count"`
		StoryEntryCount       int64 `json:"story_entry_count"`
		TotalSignInDays       int   `json:"total_sign_in_days"`
		ConsecutiveSignInDays int   `json:"consecutive_sign_in_days"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.StoryCount != 1 {
		t.Fatalf("expected story_count 1 archive, got %d", payload.StoryCount)
	}
	if payload.StoryEntryCount != 3 {
		t.Fatalf("expected story_entry_count 3 lines, got %d", payload.StoryEntryCount)
	}
	if payload.TotalSignInDays != 2 {
		t.Fatalf("expected total sign-in days 2, got %d", payload.TotalSignInDays)
	}
	if payload.ConsecutiveSignInDays != 2 {
		t.Fatalf("expected consecutive sign-in days 2, got %d", payload.ConsecutiveSignInDays)
	}
}

func TestSignInDailyReturnsAchievementSignInStats(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.UserDailyActivity{}, &model.UserActivityLog{})
	user := model.User{Username: "sign-in-user", Email: "signin@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	server := newTestServer(t, db)
	resp := performRequest(server.router, http.MethodPost, "/api/v1/user/sign-in", nil, newTestToken(t, user))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Granted               bool `json:"granted"`
		SignedInToday         bool `json:"signed_in_today"`
		TotalSignInDays       int  `json:"total_sign_in_days"`
		ConsecutiveSignInDays int  `json:"consecutive_sign_in_days"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Granted || !payload.SignedInToday {
		t.Fatalf("expected granted signed-in response, got %+v", payload)
	}
	if payload.TotalSignInDays != 1 {
		t.Fatalf("expected total sign-in days 1, got %d", payload.TotalSignInDays)
	}
	if payload.ConsecutiveSignInDays != 1 {
		t.Fatalf("expected consecutive sign-in days 1, got %d", payload.ConsecutiveSignInDays)
	}
}
