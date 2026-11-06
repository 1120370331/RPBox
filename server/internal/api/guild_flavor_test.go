package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/model"
)

func TestPublicGuildGameFlavorFilter(t *testing.T) {
	db, server, owner := newGuildServerFixture(t)
	token := newTestToken(t, owner)

	create := func(name string, flavors []string) model.Guild {
		t.Helper()
		body := map[string]interface{}{"name": name}
		if flavors != nil {
			body["game_flavors"] = flavors
		}
		response := performRequest(server.router, http.MethodPost, "/api/v1/guilds", body, token)
		if response.Code != http.StatusCreated {
			t.Fatalf("create %s: status=%d body=%s", name, response.Code, response.Body.String())
		}
		var guild model.Guild
		if err := json.Unmarshal(response.Body.Bytes(), &guild); err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&guild).Updates(map[string]interface{}{"status": "approved", "is_public": true}).Error; err != nil {
			t.Fatal(err)
		}
		return guild
	}

	retail := create("Retail guild", nil)
	both := create("Cross-server guild", []string{"_forever_", "_retail_"})
	forever := model.Guild{Name: "Infinite guild", GameFlavor: "_forever_", GameFlavors: []string{"_forever_"}, OwnerID: owner.ID, InviteCode: "infinite-test", Status: "approved", IsPublic: true}
	if err := db.Create(&forever).Error; err != nil {
		t.Fatal(err)
	}
	if retail.GameFlavor != "_retail_" || forever.GameFlavor != "_forever_" || both.GameFlavor != "_retail_" || len(both.GameFlavors) != 2 {
		t.Fatalf("created flavors: retail=%v infinite=%v both=%v", retail.GameFlavors, forever.GameFlavors, both.GameFlavors)
	}
	// Existing rows may have only the legacy single-flavor column.
	if err := db.Model(&model.Guild{}).Where("id = ?", retail.ID).Update("game_flavors", nil).Error; err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		query string
		want  []uint
	}{
		{"", []uint{retail.ID, forever.ID, both.ID}},
		{"?game_flavor=_retail_", []uint{retail.ID, both.ID}},
		{"?game_flavor=_forever_", []uint{forever.ID, both.ID}},
		{"?game_flavor=_retail_&game_flavor=_forever_", []uint{retail.ID, forever.ID, both.ID}},
	} {
		response := performRequest(server.router, http.MethodGet, "/api/v1/public/guilds"+tc.query, nil, "")
		if response.Code != http.StatusOK {
			t.Fatalf("list %q: status=%d body=%s", tc.query, response.Code, response.Body.String())
		}
		var payload struct {
			Guilds []model.Guild `json:"guilds"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		found := make(map[uint]bool)
		for _, guild := range payload.Guilds {
			found[guild.ID] = true
		}
		if len(found) != len(tc.want) {
			t.Fatalf("list %q: found=%v want=%v", tc.query, found, tc.want)
		}
		for _, id := range tc.want {
			if !found[id] {
				t.Fatalf("list %q missing guild %d", tc.query, id)
			}
		}
	}

	invalid := performRequest(server.router, http.MethodGet, "/api/v1/public/guilds?game_flavor=_classic_", nil, "")
	if invalid.Code != http.StatusBadRequest {
		t.Fatal(fmt.Sprintf("invalid flavor status=%d body=%s", invalid.Code, invalid.Body.String()))
	}

	changed := performRequest(server.router, http.MethodPut, fmt.Sprintf("/api/v1/guilds/%d", retail.ID), map[string]interface{}{"game_flavors": []string{"_forever_", "_retail_"}}, token)
	if changed.Code != http.StatusOK {
		t.Fatalf("update flavor: status=%d body=%s", changed.Code, changed.Body.String())
	}
	var stored model.Guild
	if err := db.First(&stored, retail.ID).Error; err != nil || stored.GameFlavor != "_retail_" || len(stored.GameFlavors) != 2 {
		t.Fatalf("stored flavors=%v primary=%q err=%v", stored.GameFlavors, stored.GameFlavor, err)
	}
	invalidUpdate := performRequest(server.router, http.MethodPut, fmt.Sprintf("/api/v1/guilds/%d", retail.ID), map[string]interface{}{"game_flavors": []string{}}, token)
	if invalidUpdate.Code != http.StatusBadRequest {
		t.Fatalf("empty flavors status=%d body=%s", invalidUpdate.Code, invalidUpdate.Body.String())
	}
}
