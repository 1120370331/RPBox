package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestExistingStoryEntrySourceIDsIsLightweightAndOwnerScoped(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Story{}, &model.StoryEntry{})

	owner := model.User{Username: "source-owner", Email: "source-owner@example.com", PassHash: "hash"}
	other := model.User{Username: "source-other", Email: "source-other@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&owner, &other}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	story := model.Story{UserID: owner.ID, Title: "source lookup"}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}
	entries := []model.StoryEntry{
		{StoryID: story.ID, SourceID: "alpha", Content: "secret alpha content", SortOrder: 1},
		{StoryID: story.ID, SourceID: "beta", Content: "secret beta content", SortOrder: 2},
		{StoryID: story.ID, SourceID: "", Content: "source-less content", SortOrder: 3},
	}
	if err := db.Create(&entries).Error; err != nil {
		t.Fatalf("create entries: %v", err)
	}

	server := newTestServer(t, db)
	path := fmt.Sprintf("/api/v1/stories/%d/entries/existing-source-ids", story.ID)
	resp := performRequest(server.router, http.MethodPost, path, map[string]interface{}{
		"source_ids": []string{" alpha ", "", "missing", "beta", "alpha", "   "},
	}, newTestToken(t, owner))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var rawPayload map[string]json.RawMessage
	if err := json.Unmarshal(resp.Body.Bytes(), &rawPayload); err != nil {
		t.Fatalf("decode response object: %v", err)
	}
	if len(rawPayload) != 1 {
		t.Fatalf("expected lightweight response with one field, got keys=%v", reflect.ValueOf(rawPayload).MapKeys())
	}
	if _, exists := rawPayload["entries"]; exists {
		t.Fatal("lightweight response must not include entries")
	}
	if _, exists := rawPayload["content"]; exists {
		t.Fatal("lightweight response must not include content")
	}
	if strings.Contains(resp.Body.String(), "secret alpha content") || strings.Contains(resp.Body.String(), "secret beta content") {
		t.Fatal("lightweight response leaked story entry content")
	}

	var sourceIDs []string
	if err := json.Unmarshal(rawPayload["source_ids"], &sourceIDs); err != nil {
		t.Fatalf("decode source_ids: %v", err)
	}
	if expected := []string{"alpha", "beta"}; !reflect.DeepEqual(sourceIDs, expected) {
		t.Fatalf("expected source_ids %v, got %v", expected, sourceIDs)
	}

	notOwnerResp := performRequest(server.router, http.MethodPost, path, map[string]interface{}{
		"source_ids": []string{"alpha"},
	}, newTestToken(t, other))
	if notOwnerResp.Code != http.StatusNotFound {
		t.Fatalf("expected non-owner lookup 404, got %d body=%s", notOwnerResp.Code, notOwnerResp.Body.String())
	}

	tooMany := make([]string, existingSourceIDLookupLimit+1)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("source-%d", i)
	}
	limitResp := performRequest(server.router, http.MethodPost, path, map[string]interface{}{
		"source_ids": tooMany,
	}, newTestToken(t, owner))
	if limitResp.Code != http.StatusBadRequest {
		t.Fatalf("expected over-limit lookup 400, got %d body=%s", limitResp.Code, limitResp.Body.String())
	}
}

func TestAddStoryEntriesSkipsDuplicateSourceIDsAndCountsActualCreates(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Story{},
		&model.StoryEntry{},
		&model.UserDailyActivity{},
		&model.UserActivityLog{},
	)

	owner := model.User{Username: "entry-owner", Email: "entry-owner@example.com", PassHash: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	baseTime := time.Date(2026, time.July, 22, 10, 0, 0, 0, time.UTC)
	story := model.Story{
		UserID:    owner.ID,
		Title:     "idempotent archive",
		StartTime: baseTime,
		EndTime:   baseTime,
	}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, owner)
	path := fmt.Sprintf("/api/v1/stories/%d/entries", story.ID)

	firstRequest := []map[string]interface{}{
		{
			"source_id": " source-1 ",
			"type":      "dialogue",
			"speaker":   "Alice",
			"content":   "first archive line",
			"channel":   "SAY",
			"timestamp": baseTime.Format(time.RFC3339),
		},
	}
	firstResp := performRequest(server.router, http.MethodPost, path, firstRequest, token)
	assertStoryEntryWriteCounts(t, firstResp, http.StatusCreated, 1, 1, 0)

	var storyAfterFirst model.Story
	if err := db.First(&storyAfterFirst, story.ID).Error; err != nil {
		t.Fatalf("reload story after first add: %v", err)
	}
	updatedAfterFirst := storyAfterFirst.UpdatedAt

	retryResp := performRequest(server.router, http.MethodPost, path, firstRequest, token)
	assertStoryEntryWriteCounts(t, retryResp, http.StatusCreated, 1, 0, 1)

	var storyAfterRetry model.Story
	if err := db.First(&storyAfterRetry, story.ID).Error; err != nil {
		t.Fatalf("reload story after retry: %v", err)
	}
	if !storyAfterRetry.UpdatedAt.Equal(updatedAfterFirst) {
		t.Fatalf("duplicate-only retry changed story updated_at: before=%s after=%s", updatedAfterFirst, storyAfterRetry.UpdatedAt)
	}

	batchRequest := []map[string]interface{}{
		{
			"source_id": "source-2",
			"content":   "second source",
			"timestamp": baseTime.Add(time.Minute).Format(time.RFC3339),
		},
		{
			"source_id": " source-2 ",
			"content":   "same-request duplicate with a later timestamp",
			"timestamp": baseTime.Add(24 * time.Hour).Format(time.RFC3339),
		},
		{
			"source_id": "",
			"content":   "source-less one",
			"timestamp": baseTime.Add(2 * time.Minute).Format(time.RFC3339),
		},
		{
			"source_id": "   ",
			"content":   "source-less two",
			"timestamp": baseTime.Add(3 * time.Minute).Format(time.RFC3339),
		},
	}
	batchResp := performRequest(server.router, http.MethodPost, path, batchRequest, token)
	assertStoryEntryWriteCounts(t, batchResp, http.StatusCreated, 4, 3, 1)

	var storedEntries []model.StoryEntry
	if err := db.Where("story_id = ?", story.ID).Order("sort_order ASC").Find(&storedEntries).Error; err != nil {
		t.Fatalf("list stored entries: %v", err)
	}
	if len(storedEntries) != 4 {
		t.Fatalf("expected four actual entries, got %d", len(storedEntries))
	}
	expectedSourceIDs := []string{"source-1", "source-2", "", ""}
	for i, entry := range storedEntries {
		if entry.SourceID != expectedSourceIDs[i] {
			t.Fatalf("entry %d expected source_id %q, got %q", i, expectedSourceIDs[i], entry.SourceID)
		}
		if entry.SortOrder != i+1 {
			t.Fatalf("entry %d expected continuous sort_order %d, got %d", i, i+1, entry.SortOrder)
		}
	}

	var sourceOneCount int64
	if err := db.Model(&model.StoryEntry{}).
		Where("story_id = ? AND source_id = ?", story.ID, "source-1").
		Count(&sourceOneCount).Error; err != nil {
		t.Fatalf("count source-1: %v", err)
	}
	if sourceOneCount != 1 {
		t.Fatalf("expected retried source-1 once, got %d", sourceOneCount)
	}

	var sourceTwoCount int64
	if err := db.Model(&model.StoryEntry{}).
		Where("story_id = ? AND source_id = ?", story.ID, "source-2").
		Count(&sourceTwoCount).Error; err != nil {
		t.Fatalf("count source-2: %v", err)
	}
	if sourceTwoCount != 1 {
		t.Fatalf("expected same-request source-2 once, got %d", sourceTwoCount)
	}

	var sourceLessCount int64
	if err := db.Model(&model.StoryEntry{}).
		Where("story_id = ? AND source_id = ''", story.ID).
		Count(&sourceLessCount).Error; err != nil {
		t.Fatalf("count source-less entries: %v", err)
	}
	if sourceLessCount != 2 {
		t.Fatalf("expected empty source_id entries to remain repeatable, got %d", sourceLessCount)
	}

	var storedStory model.Story
	if err := db.First(&storedStory, story.ID).Error; err != nil {
		t.Fatalf("reload final story: %v", err)
	}
	if !storedStory.StartTime.Equal(baseTime) {
		t.Fatalf("expected start_time %s, got %s", baseTime, storedStory.StartTime)
	}
	expectedEndTime := baseTime.Add(3 * time.Minute)
	if !storedStory.EndTime.Equal(expectedEndTime) {
		t.Fatalf("expected end_time %s, got %s", expectedEndTime, storedStory.EndTime)
	}

	var activity model.UserDailyActivity
	if err := db.Where("user_id = ?", owner.ID).First(&activity).Error; err != nil {
		t.Fatalf("load archive activity: %v", err)
	}
	if activity.StoryArchiveEntries != 4 {
		t.Fatalf("expected activity to count four actual creates, got %d", activity.StoryArchiveEntries)
	}
}

func assertStoryEntryWriteCounts(
	t *testing.T,
	resp *httptest.ResponseRecorder,
	wantStatus int,
	wantCount int,
	wantCreated int,
	wantSkipped int,
) {
	t.Helper()

	if resp.Code != wantStatus {
		t.Fatalf("expected status %d, got %d body=%s", wantStatus, resp.Code, resp.Body.String())
	}

	var payload struct {
		Count        int `json:"count"`
		CreatedCount int `json:"created_count"`
		SkippedCount int `json:"skipped_count"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode add entries response: %v", err)
	}
	if payload.Count != wantCount || payload.CreatedCount != wantCreated || payload.SkippedCount != wantSkipped {
		t.Fatalf(
			"expected counts count=%d created=%d skipped=%d, got count=%d created=%d skipped=%d",
			wantCount,
			wantCreated,
			wantSkipped,
			payload.Count,
			payload.CreatedCount,
			payload.SkippedCount,
		)
	}
}
