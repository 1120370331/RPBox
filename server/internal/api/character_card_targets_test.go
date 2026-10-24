package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/model"
)

func TestCharacterCardCloudTargetsAtomicAndEditionIsolated(t *testing.T) {
	db, server, owner, other := newCharacterCardTestServer(t)
	token := newTestToken(t, owner)
	backups := []model.AccountBackup{
		{UserID: owner.ID, AccountID: "WOW1", GameFlavor: "_retail_", ProfilesData: `{"same":{"player":{"characteristics":{"FN":"old"}},"custom":{"keep":1}},"unrelated":{"profileName":"keep"}}`, Version: 1},
		{UserID: owner.ID, AccountID: "WOW1", GameFlavor: "_classic_era_", ProfilesData: `{"same":{"player":{"characteristics":{"FN":"classic"}},"custom":{"keep":2}}}`, Version: 1},
		{UserID: other.ID, AccountID: "WOW1", GameFlavor: "_retail_", ProfilesData: `{}`, Version: 1},
	}
	if err := db.Create(&backups).Error; err != nil {
		t.Fatal(err)
	}
	created := performRequest(server.router, http.MethodPost, "/api/v1/character-cards", map[string]interface{}{"source_type": "blank"}, token)
	if created.Code != 201 {
		t.Fatal(created.Body.String())
	}
	var body struct {
		Card model.CharacterCard `json:"character_card"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	endpoint := fmt.Sprintf("/api/v1/character-cards/%d/save-cloud", body.Card.ID)
	payload := map[string]interface{}{"first_name": "new", "cloud_targets": []characterCardSaveTarget{
		{BackupID: backups[0].ID, ProfileID: "same", ExpectedVersion: 1}, {BackupID: backups[1].ID, ProfileID: "same", ExpectedVersion: 1},
	}}
	response := performRequest(server.router, http.MethodPut, endpoint, payload, token)
	if response.Code != 200 {
		t.Fatalf("save: %d %s", response.Code, response.Body.String())
	}
	for i := 0; i < 2; i++ {
		var backup model.AccountBackup
		if err := db.First(&backup, backups[i].ID).Error; err != nil {
			t.Fatal(err)
		}
		var profiles map[string]map[string]interface{}
		if err := json.Unmarshal([]byte(backup.ProfilesData), &profiles); err != nil {
			t.Fatal(err)
		}
		if backup.Version != 2 || profiles["same"]["custom"].(map[string]interface{})["keep"] != float64(i+1) {
			t.Fatalf("lost target-specific data: %+v", backup)
		}
		if profiles["same"]["player"].(map[string]interface{})["characteristics"].(map[string]interface{})["FN"] != "new" {
			t.Fatal("profile not updated")
		}
		var count int64
		db.Model(&model.AccountBackupVersion{}).Where("backup_id = ?", backup.ID).Count(&count)
		if count != 1 {
			t.Fatalf("history %d", count)
		}
	}
	var card model.CharacterCard
	db.First(&card, body.Card.ID)
	if card.SourceBackupID != nil || card.SourceAccountID != "" {
		t.Fatal("save rewrote provenance")
	}
	var bindingCount int64
	db.Model(&model.CharacterCardTarget{}).Where("character_card_id = ?", card.ID).Count(&bindingCount)
	if bindingCount != 2 {
		t.Fatalf("bindings %d", bindingCount)
	}
	// A late unauthorized target must roll back even the earlier valid target and card edit.
	payload["first_name"] = "must-roll-back"
	payload["cloud_targets"] = []characterCardSaveTarget{{BackupID: backups[0].ID, ProfileID: "same", ExpectedVersion: 2}, {BackupID: backups[2].ID, ProfileID: "same", ExpectedVersion: 1}}
	response = performRequest(server.router, http.MethodPut, endpoint, payload, token)
	if response.Code != 404 {
		t.Fatalf("ownership: %d %s", response.Code, response.Body.String())
	}
	db.First(&card, card.ID)
	if card.FirstName != "new" {
		t.Fatal("card partially saved")
	}
	var first model.AccountBackup
	db.First(&first, backups[0].ID)
	if first.Version != 2 {
		t.Fatal("first target partially saved")
	}
	payload["cloud_targets"] = []characterCardSaveTarget{{BackupID: backups[0].ID, ProfileID: "same", ExpectedVersion: 1}}
	if response = performRequest(server.router, http.MethodPut, endpoint, payload, token); response.Code != 409 {
		t.Fatalf("stale version: %d", response.Code)
	}
	payload["cloud_targets"] = []characterCardSaveTarget{}
	if response = performRequest(server.router, http.MethodPut, endpoint, payload, token); response.Code != 400 {
		t.Fatalf("empty targets: %d", response.Code)
	}
	// Registering an existing target must never reset its data.
	response = performRequest(server.router, http.MethodPost, "/api/v1/account-backup-targets", map[string]string{"account_id": "WOW1", "game_flavor": "_retail_"}, token)
	if response.Code != 200 {
		t.Fatalf("ensure: %d %s", response.Code, response.Body.String())
	}
	db.First(&first, backups[0].ID)
	if first.Version != 2 {
		t.Fatal("registration overwrote archive")
	}
	for _, flavor := range []string{"_retail_", "_classic_era_"} {
		response = performRequest(server.router, http.MethodGet, "/api/v1/account-backups/WOW1?game_flavor="+flavor, nil, token)
		if response.Code != 200 {
			t.Fatalf("edition lookup: %d %s", response.Code, response.Body.String())
		}
	}
}
