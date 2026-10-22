package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
)

func TestCharacterCardNativeAboutRoundTripAndPublication(t *testing.T) {
	db, s, owner, other := newCharacterCardTestServer(t)
	card := model.CharacterCard{UserID: owner.ID, DisplayName: "原档案", Status: "published", Visibility: "public", ReviewStatus: "approved", BackgroundStory: "旧版正文"}
	if err := db.Create(&card).Error; err != nil {
		t.Fatal(err)
	}
	if err := saveCharacterCardPublication(db, characterCardSnapshotFromModels(card, nil, nil), owner.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	about := json.RawMessage(`{"TE":2,"BK":7,"MU":123,"T1":{"TX":"{h1:c}保留模板一{/h1}"},"T2":[{"TX":"{col:11aa33}原生格式{/col}{icon:inv_misc_book_09:25}","IC":"inv_misc_book_09","BK":19}],"T3":{"PH":{"TX":"外貌"},"PS":{"TX":"性格"},"HI":{"TX":"历史"}},"custom":{"preserved":true}}`)
	path := fmt.Sprintf("/api/v1/character-cards/%d", card.ID)
	token := newTestToken(t, owner)
	unauthorized := performRequest(s.router, http.MethodPut, path, map[string]interface{}{"trp3_about": about}, newTestToken(t, other))
	if unauthorized.Code == http.StatusOK {
		t.Fatal("another user updated native template")
	}
	resp := performRequest(s.router, http.MethodPut, path, map[string]interface{}{"trp3_about": about, "icon": "inv_misc_book_09"}, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("save=%d %s", resp.Code, resp.Body.String())
	}
	var saved model.CharacterCard
	if err := db.First(&saved, card.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.TRP3AboutJSON != string(about) || saved.BackgroundStory != card.BackgroundStory {
		t.Fatalf("source changed: %+v", saved)
	}
	ownerRead := performRequest(s.router, http.MethodGet, path, nil, token)
	if !strings.Contains(ownerRead.Body.String(), "trp3_about") {
		t.Fatal("owner cannot reload native template")
	}
	publicRead := performRequest(s.router, http.MethodGet, path, nil, "")
	if strings.Contains(publicRead.Body.String(), "原生格式") {
		t.Fatal("unreviewed native text leaked through published snapshot")
	}
	profile, err := buildTRP3ProfileFromCharacterCard(saved, nil, map[string]interface{}{"player": map[string]interface{}{"about": map[string]interface{}{"v": float64(3)}, "unknown": "keep"}})
	if err != nil {
		t.Fatal(err)
	}
	player := profile["player"].(map[string]interface{})
	encoded, err := json.Marshal(player["about"])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "原生格式") || !strings.Contains(string(encoded), `"MU":123`) || player["unknown"] != "keep" {
		t.Fatalf("lossy export: %s", encoded)
	}
	snapshot := characterCardSnapshotFromModels(saved, nil, nil)
	restored, _, _ := snapshot.models()
	if restored.TRP3AboutJSON != saved.TRP3AboutJSON {
		t.Fatal("publication snapshot lost native template")
	}
	if err := saveCharacterCardPublication(db, snapshot, owner.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	approvedRead := performRequest(s.router, http.MethodGet, path, nil, "")
	if !strings.Contains(approvedRead.Body.String(), "原生格式") {
		t.Fatal("approved native template missing from archive")
	}
	// Existing rich-text clients preserve native metadata on unrelated saves.
	resp = performRequest(s.router, http.MethodPut, path, map[string]interface{}{"background_story": "旧版正文", "title": "新头衔"}, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("legacy save: %s", resp.Body.String())
	}
	if err := db.First(&saved, card.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.TRP3AboutJSON == "" {
		t.Fatal("unrelated legacy edit erased native template")
	}
	// Explicitly replacing the old biography makes the ordinary editor authoritative.
	resp = performRequest(s.router, http.MethodPut, path, map[string]interface{}{"background_story": "新版正文"}, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("native replacement: %s", resp.Body.String())
	}
	if err := db.First(&saved, card.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.TRP3AboutJSON != "" {
		t.Fatal("stale template survived a biography replacement")
	}
}

func TestCharacterCardNativeAboutRejectsInvalidShapes(t *testing.T) {
	for _, raw := range []string{`[]`, `{"TE":4}`, `{"TE":1,"T1":{"TX":123}}`, `{"TE":2,"T2":{}}`, strings.Repeat("x", characterCardRichTextMaxBytes+1)} {
		if _, err := normalizeCharacterCardTRP3About(json.RawMessage(raw)); err == nil {
			t.Fatalf("accepted invalid template: %.80s", raw)
		}
	}
}
