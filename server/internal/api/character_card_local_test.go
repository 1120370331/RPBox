package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/rpbox/server/internal/model"
)

func TestLocalCharacterCardImportIsPrivateAndDoesNotReplaceBackups(t *testing.T) {
	db, s, owner, other := newCharacterCardTestServer(t)
	backup := model.AccountBackup{UserID: owner.ID, AccountID: "local-account", ProfilesData: `{"preserved":true}`, Checksum: "keep"}
	if err := db.Create(&backup).Error; err != nil {
		t.Fatal(err)
	}
	profile := json.RawMessage(`{"user_id":999,"status":"published","profileName":"本地原卡","player":{"characteristics":{"FN":"米拉","LN":"铜枝","IC":"inv_misc_book_09","CL":"猎人"},"about":{"TE":1,"T1":{"TX":"{h1:c}原始格式{/h1}"}},"misc":{"PE":{"1":{"AC":true,"TI":"第一印象","TX":"原卡说明","IC":"inv_misc_book_09"}}}}}`)
	body := map[string]interface{}{"source_type": "local", "source_account_id": "local-account", "source_profile_id": "local-id", "local_profile": profile, "user_id": other.ID}
	response := performRequest(s.router, http.MethodPost, "/api/v1/character-cards", body, newTestToken(t, owner))
	if response.Code != http.StatusCreated {
		t.Fatalf("import=%d %s", response.Code, response.Body.String())
	}
	card := decodeCharacterCardResponse(t, response)
	if card.UserID != owner.ID || card.Status != "draft" || card.Visibility != "private" || card.SourceBackupID != nil {
		t.Fatalf("unsafe ownership/provenance: %+v", card)
	}
	if card.SourceAccountID != "local-account" || card.SourceProfileID != "local-id" || card.Icon != "inv_misc_book_09" {
		t.Fatalf("lost local source: %+v", card)
	}
	if !strings.Contains(string(card.TRP3About), "{h1:c}原始格式{/h1}") || len(card.Impressions) != 5 || card.Impressions[0].Title != "第一印象" {
		t.Fatal("native content was not imported")
	}
	var stored model.AccountBackup
	if err := db.First(&stored, backup.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ProfilesData != backup.ProfilesData || stored.Checksum != "keep" {
		t.Fatal("local import replaced a cloud backup")
	}
	for _, changes := range []map[string]interface{}{
		{"source_account_id": "../unsafe"}, {"source_profile_id": "../unsafe"}, {"source_backup_id": backup.ID}, {"local_profile": map[string]interface{}{"player": map[string]interface{}{}}},
	} {
		invalid := map[string]interface{}{}
		for key, value := range body {
			invalid[key] = value
		}
		for key, value := range changes {
			invalid[key] = value
		}
		result := performRequest(s.router, http.MethodPost, "/api/v1/character-cards", invalid, newTestToken(t, owner))
		if result.Code != http.StatusBadRequest {
			t.Fatalf("invalid import accepted: %d %s", result.Code, result.Body.String())
		}
	}
	var count int64
	if err := db.Model(&model.CharacterCard{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("invalid requests left partial cards: %d", count)
	}
}
