package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func TestStoryReportCanBeListedAndDeletedByModerator(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Story{},
		&model.StoryEntry{},
		&model.StoryMusicTrack{},
		&model.StoryMusicTrackStory{},
		&model.StoryMusicSegment{},
		&model.StoryTag{},
		&model.ContentReport{},
		&model.AdminActionLog{},
	)

	author := model.User{Username: "story-author", Email: "author@example.com", PassHash: "hash", Role: "user"}
	reporter := model.User{Username: "reporter", Email: "reporter@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "moderator", Email: "moderator@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &reporter, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	now := time.Now()
	story := model.Story{
		UserID:      author.ID,
		Title:       "违规剧情",
		Description: "需要版主审核的剧情描述",
		StartTime:   now,
		EndTime:     now,
		IsPublic:    true,
		ShareCode:   "storyabc",
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	entries := []model.StoryEntry{
		{StoryID: story.ID, SourceID: "line-1", Type: "dialogue", Speaker: "Alice", Content: "第一句", Channel: "SAY", Timestamp: now, SortOrder: 1},
		{StoryID: story.ID, SourceID: "line-2", Type: "dialogue", Speaker: "Bob", Content: "被举报的台词", Channel: "YELL", Timestamp: now.Add(time.Second), SortOrder: 2},
	}
	if err := db.Create(&entries).Error; err != nil {
		t.Fatalf("create entries: %v", err)
	}

	track := model.StoryMusicTrack{
		UserID:   author.ID,
		Name:     "problem-track",
		FileName: "track.mp3",
		MimeType: "audio/mpeg",
		URL:      "/music/track.mp3",
		Color:    "#B87333",
		Volume:   0.75,
	}
	if err := db.Create(&track).Error; err != nil {
		t.Fatalf("create track: %v", err)
	}
	if err := db.Create(&model.StoryMusicTrackStory{StoryID: story.ID, TrackID: track.ID}).Error; err != nil {
		t.Fatalf("attach track: %v", err)
	}
	if err := db.Create(&model.StoryMusicSegment{
		StoryID:      story.ID,
		TrackID:      track.ID,
		StartEntryID: entries[1].ID,
		Loop:         true,
		AutoPlay:     true,
		Volume:       0.75,
	}).Error; err != nil {
		t.Fatalf("create segment: %v", err)
	}

	server := newTestServer(t, db)
	reportResp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/reports",
		map[string]interface{}{
			"target_type": "story",
			"target_id":   story.ID,
			"reason":      "story_audio",
			"detail":      fmt.Sprintf("辅助条目：第 2 条 #%d", entries[1].ID),
		},
		newTestToken(t, reporter),
	)
	if reportResp.Code != http.StatusOK {
		t.Fatalf("expected create report 200, got %d body=%s", reportResp.Code, reportResp.Body.String())
	}

	listResp := performRequest(
		server.router,
		http.MethodGet,
		"/api/v1/moderator/reports?target_scope=story",
		nil,
		newTestToken(t, moderator),
	)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected list reports 200, got %d body=%s", listResp.Code, listResp.Body.String())
	}

	var listPayload struct {
		Reports []struct {
			ID                uint   `json:"id"`
			TargetType        string `json:"target_type"`
			TargetTitle       string `json:"target_title"`
			TargetAuthorName  string `json:"target_author_name"`
			TargetPreviewText string `json:"target_preview_text"`
			TargetURL         string `json:"target_url"`
			Reasons           []struct {
				Reason string `json:"reason"`
				Detail string `json:"detail"`
			} `json:"reasons"`
		} `json:"reports"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listPayload.Total != 1 || len(listPayload.Reports) != 1 {
		t.Fatalf("expected one story report, got total=%d len=%d", listPayload.Total, len(listPayload.Reports))
	}
	report := listPayload.Reports[0]
	if report.TargetType != "story" || report.TargetTitle != story.Title {
		t.Fatalf("unexpected report target: type=%q title=%q", report.TargetType, report.TargetTitle)
	}
	if report.TargetAuthorName != author.Username {
		t.Fatalf("expected author name %q, got %q", author.Username, report.TargetAuthorName)
	}
	if report.TargetPreviewText != story.Description {
		t.Fatalf("expected preview %q, got %q", story.Description, report.TargetPreviewText)
	}
	if report.TargetURL != "/story/"+story.ShareCode {
		t.Fatalf("expected story target url, got %q", report.TargetURL)
	}
	if len(report.Reasons) != 1 || report.Reasons[0].Reason != "story_audio" {
		t.Fatalf("expected story_audio reason, got %#v", report.Reasons)
	}

	reviewResp := performRequest(
		server.router,
		http.MethodPost,
		fmt.Sprintf("/api/v1/moderator/reports/%d/review", report.ID),
		map[string]interface{}{"action": "delete_content", "comment": "confirmed"},
		newTestToken(t, moderator),
	)
	if reviewResp.Code != http.StatusOK {
		t.Fatalf("expected review 200, got %d body=%s", reviewResp.Code, reviewResp.Body.String())
	}

	if err := db.First(&model.Story{}, story.ID).Error; err == nil {
		t.Fatalf("expected story to be deleted")
	} else if err != gorm.ErrRecordNotFound {
		t.Fatalf("load deleted story: %v", err)
	}

	var remainingEntries int64
	db.Model(&model.StoryEntry{}).Where("story_id = ?", story.ID).Count(&remainingEntries)
	if remainingEntries != 0 {
		t.Fatalf("expected story entries deleted, got %d", remainingEntries)
	}

	var remainingSegments int64
	db.Model(&model.StoryMusicSegment{}).Where("story_id = ?", story.ID).Count(&remainingSegments)
	if remainingSegments != 0 {
		t.Fatalf("expected story music segments deleted, got %d", remainingSegments)
	}

	var storedReport model.ContentReport
	if err := db.First(&storedReport, report.ID).Error; err != nil {
		t.Fatalf("load stored report: %v", err)
	}
	if storedReport.Status != "resolved" {
		t.Fatalf("expected report resolved, got %q", storedReport.Status)
	}
}
