package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

func TestModeratorCanArchiveReportWithReporterInfo(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Story{},
		&model.ContentReport{},
		&model.AdminActionLog{},
	)

	author := model.User{Username: "archive-author", Email: "archive-author@example.com", PassHash: "hash", Role: "user"}
	reporter := model.User{
		Username:           "archive-reporter",
		Email:              "archive-reporter@example.com",
		PassHash:           "hash",
		Role:               "user",
		Avatar:             "/uploads/users/reporter.png",
		AvatarReviewStatus: "approved",
	}
	moderator := model.User{Username: "archive-moderator", Email: "archive-mod@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &reporter, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	now := time.Now()
	story := model.Story{
		UserID:      author.ID,
		Title:       "可归档剧情",
		Description: "低质量举报不应删除的剧情",
		StartTime:   now,
		EndTime:     now,
		IsPublic:    true,
		ShareCode:   "archiveabc",
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	server := newTestServer(t, db)
	reportResp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/reports",
		map[string]interface{}{
			"target_type": "story",
			"target_id":   story.ID,
			"reason":      "story_content",
			"detail":      "证据不足的剧情举报",
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
			ID      uint   `json:"id"`
			Status  string `json:"status"`
			Reasons []struct {
				ReporterID     uint   `json:"reporter_id"`
				ReporterName   string `json:"reporter_name"`
				ReporterAvatar string `json:"reporter_avatar"`
				Reason         string `json:"reason"`
			} `json:"reasons"`
			ReportDetails []struct {
				ReporterID     uint   `json:"reporter_id"`
				ReporterName   string `json:"reporter_name"`
				ReporterAvatar string `json:"reporter_avatar"`
			} `json:"reports"`
		} `json:"reports"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listPayload.Total != 1 || len(listPayload.Reports) != 1 {
		t.Fatalf("expected one report, got total=%d len=%d", listPayload.Total, len(listPayload.Reports))
	}
	report := listPayload.Reports[0]
	if len(report.Reasons) != 1 {
		t.Fatalf("expected one reason, got %#v", report.Reasons)
	}
	reason := report.Reasons[0]
	if reason.ReporterID != reporter.ID || reason.ReporterName != reporter.Username {
		t.Fatalf("unexpected reporter fields: %#v", reason)
	}
	if !strings.Contains(reason.ReporterAvatar, fmt.Sprintf("/api/v1/images/user-avatar/%d", reporter.ID)) {
		t.Fatalf("expected safe reporter avatar URL, got %q", reason.ReporterAvatar)
	}
	if len(report.ReportDetails) != 1 || report.ReportDetails[0].ReporterID != reporter.ID {
		t.Fatalf("expected reports alias to include reporter info, got %#v", report.ReportDetails)
	}

	reviewResp := performRequest(
		server.router,
		http.MethodPost,
		fmt.Sprintf("/api/v1/moderator/reports/%d/review", report.ID),
		map[string]interface{}{"action": "archive", "comment": "证据不足，忽略归档"},
		newTestToken(t, moderator),
	)
	if reviewResp.Code != http.StatusOK {
		t.Fatalf("expected archive 200, got %d body=%s", reviewResp.Code, reviewResp.Body.String())
	}

	if err := db.First(&model.Story{}, story.ID).Error; err != nil {
		t.Fatalf("expected story to remain after archive: %v", err)
	}

	var authorAfter model.User
	if err := db.First(&authorAfter, author.ID).Error; err != nil {
		t.Fatalf("load author: %v", err)
	}
	if authorAfter.IsMuted || authorAfter.IsBanned {
		t.Fatalf("expected archive not to punish author, muted=%v banned=%v", authorAfter.IsMuted, authorAfter.IsBanned)
	}

	var storedReport model.ContentReport
	if err := db.First(&storedReport, report.ID).Error; err != nil {
		t.Fatalf("load stored report: %v", err)
	}
	if storedReport.Status != "archived" {
		t.Fatalf("expected report archived, got %q", storedReport.Status)
	}
	if !strings.Contains(storedReport.ReviewComment, "忽略归档举报") {
		t.Fatalf("expected archive review comment, got %q", storedReport.ReviewComment)
	}

	archivedListResp := performRequest(
		server.router,
		http.MethodGet,
		"/api/v1/moderator/reports?target_scope=story&status=archived",
		nil,
		newTestToken(t, moderator),
	)
	if archivedListResp.Code != http.StatusOK {
		t.Fatalf("expected archived list 200, got %d body=%s", archivedListResp.Code, archivedListResp.Body.String())
	}

	var archivedListPayload struct {
		Reports []struct {
			ID     uint   `json:"id"`
			Status string `json:"status"`
		} `json:"reports"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(archivedListResp.Body.Bytes(), &archivedListPayload); err != nil {
		t.Fatalf("decode archived list response: %v", err)
	}
	if archivedListPayload.Total != 1 || len(archivedListPayload.Reports) != 1 {
		t.Fatalf("expected one archived report, got total=%d len=%d", archivedListPayload.Total, len(archivedListPayload.Reports))
	}
	if archivedListPayload.Reports[0].ID != report.ID || archivedListPayload.Reports[0].Status != "archived" {
		t.Fatalf("unexpected archived report: %#v", archivedListPayload.Reports[0])
	}
}
