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
		&model.Post{},
		&model.Item{},
		&model.Guild{},
		&model.GuildMember{},
		&model.Story{},
		&model.StoryEntry{},
		&model.Profile{},
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

	posts := []model.Post{
		{AuthorID: user.ID, Title: "approved post", Content: "body", Status: "published", ReviewStatus: "approved", ViewCount: 25, LikeCount: 40},
		{AuthorID: user.ID, Title: "draft post", Content: "draft", Status: "draft", ReviewStatus: "approved", ViewCount: 999, LikeCount: 999},
		{AuthorID: other.ID, Title: "other post", Content: "other", Status: "published", ReviewStatus: "approved", ViewCount: 300, LikeCount: 300},
	}
	if err := db.Create(&posts).Error; err != nil {
		t.Fatalf("create posts: %v", err)
	}

	items := []model.Item{
		{AuthorID: user.ID, Name: "approved item", Type: "item", Status: "published", ReviewStatus: "approved", Downloads: 10, LikeCount: 60},
		{AuthorID: user.ID, Name: "draft item", Type: "item", Status: "draft", ReviewStatus: "approved", Downloads: 999, LikeCount: 999},
		{AuthorID: other.ID, Name: "other item", Type: "item", Status: "published", ReviewStatus: "approved", Downloads: 500, LikeCount: 500},
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatalf("create items: %v", err)
	}

	approvedGuild := model.Guild{Name: "approved guild", OwnerID: other.ID, Status: "approved", InviteCode: "approved1"}
	pendingGuild := model.Guild{Name: "pending guild", OwnerID: user.ID, Status: "pending", InviteCode: "pending1"}
	if err := db.Create(&approvedGuild).Error; err != nil {
		t.Fatalf("create approved guild: %v", err)
	}
	if err := db.Create(&pendingGuild).Error; err != nil {
		t.Fatalf("create pending guild: %v", err)
	}
	memberships := []model.GuildMember{
		{GuildID: approvedGuild.ID, UserID: user.ID, Role: "member", JoinedAt: nowForTest()},
		{GuildID: pendingGuild.ID, UserID: user.ID, Role: "owner", JoinedAt: nowForTest()},
		{GuildID: approvedGuild.ID, UserID: other.ID, Role: "owner", JoinedAt: nowForTest()},
	}
	if err := db.Create(&memberships).Error; err != nil {
		t.Fatalf("create guild memberships: %v", err)
	}

	profile := model.Profile{ID: "profile-1", UserID: user.ID, ProfileName: "Profile", Checksum: "abc"}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("create profile: %v", err)
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
	token := newTestToken(t, user)

	assertStats := func(path string) {
		t.Helper()
		resp := performRequest(server.router, http.MethodGet, path, nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, resp.Code, resp.Body.String())
		}

		var payload struct {
			PostCount             int64 `json:"post_count"`
			GuildCount            int64 `json:"guild_count"`
			ItemCount             int64 `json:"item_count"`
			StoryCount            int64 `json:"story_count"`
			StoryEntryCount       int64 `json:"story_entry_count"`
			ProfileCount          int64 `json:"profile_count"`
			MaxPostViews          int64 `json:"max_post_views"`
			MaxItemDownloads      int64 `json:"max_item_downloads"`
			TotalLikes            int64 `json:"total_likes"`
			TotalItemDownloads    int64 `json:"total_item_downloads"`
			TotalSignInDays       int   `json:"total_sign_in_days"`
			ConsecutiveSignInDays int   `json:"consecutive_sign_in_days"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if payload.PostCount != 1 {
			t.Fatalf("expected post_count 1 approved post, got %d", payload.PostCount)
		}
		if payload.GuildCount != 1 {
			t.Fatalf("expected guild_count 1 approved membership, got %d", payload.GuildCount)
		}
		if payload.ItemCount != 1 {
			t.Fatalf("expected item_count 1 approved item, got %d", payload.ItemCount)
		}
		if payload.StoryCount != 1 {
			t.Fatalf("expected story_count 1 archive, got %d", payload.StoryCount)
		}
		if payload.StoryEntryCount != 3 {
			t.Fatalf("expected story_entry_count 3 lines, got %d", payload.StoryEntryCount)
		}
		if payload.ProfileCount != 1 {
			t.Fatalf("expected profile_count 1, got %d", payload.ProfileCount)
		}
		if payload.MaxPostViews != 25 {
			t.Fatalf("expected max_post_views 25, got %d", payload.MaxPostViews)
		}
		if payload.MaxItemDownloads != 10 {
			t.Fatalf("expected max_item_downloads 10, got %d", payload.MaxItemDownloads)
		}
		if payload.TotalLikes != 100 {
			t.Fatalf("expected total_likes 100 from approved post and item, got %d", payload.TotalLikes)
		}
		if payload.TotalItemDownloads != 10 {
			t.Fatalf("expected total_item_downloads 10, got %d", payload.TotalItemDownloads)
		}
		if payload.TotalSignInDays != 2 {
			t.Fatalf("expected total sign-in days 2, got %d", payload.TotalSignInDays)
		}
		if payload.ConsecutiveSignInDays != 2 {
			t.Fatalf("expected consecutive sign-in days 2, got %d", payload.ConsecutiveSignInDays)
		}
	}

	assertStats(fmt.Sprintf("/api/v1/users/%d", user.ID))
	assertStats("/api/v1/user/info")
}

func nowForTest() time.Time {
	return time.Now()
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
