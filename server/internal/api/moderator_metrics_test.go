package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestBasicMetricsIncludesCustomCharacterCardCount(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.CharacterCard{})
	admin := model.User{Username: "metrics-admin", Email: "metrics-admin@example.com", PassHash: "hash", Role: "admin"}
	owner := model.User{Username: "card-owner", Email: "card-owner@example.com", PassHash: "hash", Role: "user"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := db.Create(&[]model.CharacterCard{
		{UserID: owner.ID, Status: model.CharacterCardStatusDraft, Visibility: model.CharacterCardVisibilityPrivate},
		{UserID: owner.ID, Status: model.CharacterCardStatusPublished, Visibility: model.CharacterCardVisibilityPublic},
	}).Error; err != nil {
		t.Fatalf("create character cards: %v", err)
	}

	server := newTestServer(t, db)
	resp := performRequest(server.router, http.MethodGet, "/api/v1/moderator/metrics/basic", nil, newTestToken(t, admin))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		CustomCharacterCards int64 `json:"custom_character_cards"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.CustomCharacterCards != 2 {
		t.Fatalf("expected 2 custom character cards, got %d", payload.CustomCharacterCards)
	}
}
