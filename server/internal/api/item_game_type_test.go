package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestCreateTRP3GameMarketItem(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Item{})
	user := model.User{Username: "game-maker", Email: "game-maker@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	server := newTestServer(t, db)
	token := newTestToken(t, user)

	resp := performRequest(server.router, http.MethodPost, "/api/v1/items", map[string]interface{}{
		"name":        "艾泽拉斯桌游",
		"type":        "game",
		"import_code": "TRP3_GAME_DATA",
		"status":      "draft",
	}, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected game creation to return 201, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Data model.Item `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Type != "game" {
		t.Fatalf("expected game type, got %q", payload.Data.Type)
	}

	invalid := performRequest(server.router, http.MethodPost, "/api/v1/items", map[string]interface{}{
		"name":        "Unknown type",
		"type":        "minigame-unknown",
		"import_code": "DATA",
		"status":      "draft",
	}, token)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid type to return 400, got %d body=%s", invalid.Code, invalid.Body.String())
	}
}
