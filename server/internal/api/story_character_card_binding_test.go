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

func TestStoryEntryCharacterCardBindingIsOwnerScopedAndExclusive(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Story{},
		&model.StoryEntry{},
		&model.Character{},
		&model.CharacterCard{},
		&model.UserDailyActivity{},
		&model.UserActivityLog{},
	)
	owner := model.User{Username: "binding-owner", Email: "binding-owner@example.com", PassHash: "hash"}
	other := model.User{Username: "binding-other", Email: "binding-other@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&owner, &other}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	storyTime := time.Date(2026, time.August, 11, 10, 0, 0, 0, time.UTC)
	story := model.Story{UserID: owner.ID, Title: "binding story", StartTime: storyTime, EndTime: storyTime}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}
	ownerCard := model.CharacterCard{UserID: owner.ID, DisplayName: "Owner RPBox", Status: model.CharacterCardStatusDraft, Visibility: model.CharacterCardVisibilityPrivate}
	foreignCard := model.CharacterCard{UserID: other.ID, DisplayName: "Foreign RPBox", Status: model.CharacterCardStatusDraft, Visibility: model.CharacterCardVisibilityPrivate}
	if err := db.Create(&[]*model.CharacterCard{&ownerCard, &foreignCard}).Error; err != nil {
		t.Fatalf("create character cards: %v", err)
	}
	ownerCharacter := model.Character{UserID: owner.ID, RefID: "owner-trp3", FirstName: "Owner TRP3"}
	foreignCharacter := model.Character{UserID: other.ID, RefID: "foreign-trp3", FirstName: "Foreign TRP3"}
	if err := db.Create(&[]*model.Character{&ownerCharacter, &foreignCharacter}).Error; err != nil {
		t.Fatalf("create characters: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, owner)
	entriesPath := fmt.Sprintf("/api/v1/stories/%d/entries", story.ID)
	create := performRequest(server.router, http.MethodPost, entriesPath, []map[string]interface{}{{
		"content":           "Bound to RPBox",
		"speaker":           "Owner RPBox",
		"type":              "dialogue",
		"character_card_id": ownerCard.ID,
	}}, token)
	if create.Code != http.StatusCreated {
		t.Fatalf("expected owned RPBox binding 201, got %d body=%s", create.Code, create.Body.String())
	}

	var entry model.StoryEntry
	if err := db.Where("story_id = ?", story.ID).First(&entry).Error; err != nil {
		t.Fatalf("load created entry: %v", err)
	}
	if entry.CharacterCardID == nil || *entry.CharacterCardID != ownerCard.ID || entry.CharacterID != nil {
		t.Fatalf("expected exclusive RPBox binding %d, got character=%v character_card=%v", ownerCard.ID, entry.CharacterID, entry.CharacterCardID)
	}

	foreignCreate := performRequest(server.router, http.MethodPost, entriesPath, []map[string]interface{}{{
		"content":           "Must not bind foreign card",
		"character_card_id": foreignCard.ID,
	}}, token)
	if foreignCreate.Code != http.StatusNotFound {
		t.Fatalf("expected foreign RPBox binding 404, got %d body=%s", foreignCreate.Code, foreignCreate.Body.String())
	}

	conflictCreate := performRequest(server.router, http.MethodPost, entriesPath, []map[string]interface{}{{
		"content":           "Ambiguous binding",
		"character_id":      ownerCharacter.ID,
		"character_card_id": ownerCard.ID,
	}}, token)
	if conflictCreate.Code != http.StatusBadRequest {
		t.Fatalf("expected dual binding 400, got %d body=%s", conflictCreate.Code, conflictCreate.Body.String())
	}

	narrationCreate := performRequest(server.router, http.MethodPost, entriesPath, []map[string]interface{}{{
		"content":           "Narration cannot carry a character binding",
		"type":              "narration",
		"character_card_id": ownerCard.ID,
	}}, token)
	if narrationCreate.Code != http.StatusBadRequest {
		t.Fatalf("expected narration binding 400, got %d body=%s", narrationCreate.Code, narrationCreate.Body.String())
	}

	entryPath := fmt.Sprintf("%s/%d", entriesPath, entry.ID)
	switchToTRP3 := performRequest(server.router, http.MethodPut, entryPath, map[string]interface{}{
		"character_id":      ownerCharacter.ID,
		"character_card_id": nil,
	}, token)
	if switchToTRP3.Code != http.StatusOK {
		t.Fatalf("expected TRP3 switch 200, got %d body=%s", switchToTRP3.Code, switchToTRP3.Body.String())
	}
	if err := db.First(&entry, entry.ID).Error; err != nil {
		t.Fatalf("reload switched entry: %v", err)
	}
	if entry.CharacterID == nil || *entry.CharacterID != ownerCharacter.ID || entry.CharacterCardID != nil {
		t.Fatalf("expected exclusive TRP3 binding %d, got character=%v character_card=%v", ownerCharacter.ID, entry.CharacterID, entry.CharacterCardID)
	}

	foreignUpdate := performRequest(server.router, http.MethodPut, entryPath, map[string]interface{}{
		"character_id": foreignCharacter.ID,
	}, token)
	if foreignUpdate.Code != http.StatusNotFound {
		t.Fatalf("expected foreign TRP3 binding 404, got %d body=%s", foreignUpdate.Code, foreignUpdate.Body.String())
	}
	if err := db.First(&entry, entry.ID).Error; err != nil {
		t.Fatalf("reload after rejected update: %v", err)
	}
	if entry.CharacterID == nil || *entry.CharacterID != ownerCharacter.ID {
		t.Fatal("rejected foreign update changed the existing binding")
	}

	switchToNarration := performRequest(server.router, http.MethodPut, entryPath, map[string]interface{}{
		"type": "narration",
	}, token)
	if switchToNarration.Code != http.StatusOK {
		t.Fatalf("expected narration switch 200, got %d body=%s", switchToNarration.Code, switchToNarration.Body.String())
	}
	if err := db.First(&entry, entry.ID).Error; err != nil {
		t.Fatalf("reload narration entry: %v", err)
	}
	if entry.CharacterID != nil || entry.CharacterCardID != nil {
		t.Fatalf("narration switch must clear both bindings, got character=%v character_card=%v", entry.CharacterID, entry.CharacterCardID)
	}

	rebindRPBox := performRequest(server.router, http.MethodPut, entryPath, map[string]interface{}{
		"type":              "dialogue",
		"character_card_id": ownerCard.ID,
	}, token)
	if rebindRPBox.Code != http.StatusOK {
		t.Fatalf("expected RPBox rebind 200, got %d body=%s", rebindRPBox.Code, rebindRPBox.Body.String())
	}

	clear := performRequest(server.router, http.MethodPut, entryPath, map[string]interface{}{
		"character_id":      nil,
		"character_card_id": nil,
	}, token)
	if clear.Code != http.StatusOK {
		t.Fatalf("expected binding clear 200, got %d body=%s", clear.Code, clear.Body.String())
	}
	if err := db.First(&entry, entry.ID).Error; err != nil {
		t.Fatalf("reload cleared entry: %v", err)
	}
	if entry.CharacterID != nil || entry.CharacterCardID != nil {
		t.Fatalf("expected both bindings cleared, got character=%v character_card=%v", entry.CharacterID, entry.CharacterCardID)
	}

	var count int64
	if err := db.Model(&model.StoryEntry{}).Where("story_id = ?", story.ID).Count(&count).Error; err != nil {
		t.Fatalf("count entries: %v", err)
	}
	if count != 1 {
		t.Fatalf("rejected binding requests must not create entries, got %d", count)
	}
}

func TestPublicStoryOnlyExposesApprovedRPBoxCharacterCards(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Story{},
		&model.StoryEntry{},
		&model.CharacterCard{},
		&model.CharacterCardImpression{},
		&model.CharacterCardPortrait{},
		&model.CharacterCardPublication{},
		&model.StoryMusicTrack{},
		&model.StoryMusicTrackStory{},
		&model.StoryMusicSegment{},
	)
	owner := model.User{Username: "public-story-owner", Email: "public-story-owner@example.com", PassHash: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	story := model.Story{UserID: owner.ID, Title: "public story", IsPublic: true, ShareCode: "cardtest"}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}
	approved := model.CharacterCard{
		UserID: owner.ID, DisplayName: "Approved RPBox", Summary: "Visible summary",
		Status: model.CharacterCardStatusPublished, Visibility: model.CharacterCardVisibilityPublic,
		ReviewStatus: model.CharacterCardReviewApproved,
	}
	private := model.CharacterCard{
		UserID: owner.ID, DisplayName: "Private RPBox", Summary: "Must stay private",
		Status: model.CharacterCardStatusDraft, Visibility: model.CharacterCardVisibilityPrivate,
	}
	if err := db.Create(&[]*model.CharacterCard{&approved, &private}).Error; err != nil {
		t.Fatalf("create cards: %v", err)
	}
	entries := []model.StoryEntry{
		{StoryID: story.ID, CharacterCardID: &approved.ID, Speaker: "Approved snapshot name", Content: "public line", SortOrder: 1},
		{StoryID: story.ID, CharacterCardID: &private.ID, Speaker: "Private snapshot name", Content: "private-card line", SortOrder: 2},
	}
	if err := db.Create(&entries).Error; err != nil {
		t.Fatalf("create entries: %v", err)
	}

	server := newTestServer(t, db)
	resp := performRequest(server.router, http.MethodGet, "/api/v1/public/stories/cardtest", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected public story 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	var payload struct {
		Entries        []model.StoryEntry          `json:"entries"`
		CharacterCards map[string]characterCardDTO `json:"character_cards"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode public story: %v", err)
	}
	approvedKey := fmt.Sprintf("%d", approved.ID)
	privateKey := fmt.Sprintf("%d", private.ID)
	if card, ok := payload.CharacterCards[approvedKey]; !ok || card.DisplayName != "Approved RPBox" {
		t.Fatalf("approved card missing from public story: %#v", payload.CharacterCards)
	}
	if _, ok := payload.CharacterCards[privateKey]; ok {
		t.Fatal("private card leaked through public story payload")
	}
	if len(payload.Entries) != 2 {
		t.Fatalf("expected two story entries, got %d", len(payload.Entries))
	}
	if payload.Entries[0].CharacterCardID == nil || *payload.Entries[0].CharacterCardID != approved.ID {
		t.Fatal("approved card binding was unexpectedly removed")
	}
	if payload.Entries[1].CharacterCardID != nil {
		t.Fatal("private card identifier leaked through its public story entry")
	}
	if payload.Entries[1].Speaker != "Private snapshot name" {
		t.Fatal("privacy redaction must retain the historical speaker snapshot")
	}
}
