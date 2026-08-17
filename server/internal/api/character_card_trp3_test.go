package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/rpbox/server/internal/model"
)

func TestValidateTRP3ProfileIDAcceptsDefaultProfileMarker(t *testing.T) {
	valid := []string{
		"0111210250fhJ7*",
		"profile-exact",
		"RPBOX_12_ABC123",
	}
	for _, profileID := range valid {
		if err := validateTRP3ProfileID(profileID); err != nil {
			t.Errorf("expected profile ID %q to be valid: %v", profileID, err)
		}
	}

	invalid := []string{
		"*",
		"profile*internal",
		"profile**",
		"../escape",
	}
	for _, profileID := range invalid {
		if err := validateTRP3ProfileID(profileID); err == nil {
			t.Errorf("expected profile ID %q to be rejected", profileID)
		}
	}
}

func TestCharacterCardTRP3LuaWriteBackAlwaysSnapshotsAndPreservesRawFile(t *testing.T) {
	db, server, owner, other := newCharacterCardTestServer(t)
	originalProfiles := characterCardProfilesData(`{
		"profileName":"原档案",
		"player":{
			"characteristics":{"FN":"原名","CL":"法师","CH":"112233","CUSTOM":{"nested":7}},
			"misc":{"PE":{"1":{"AC":true,"TI":"来源印象"}},"ST":{"style":"保留"}},
			"unknownPlayer":{"enabled":true}
		},
		"unknown":{"text":"保留 { 花括号 }"}
	}`)
	originalRaw := `-- braces in a comment { must not affect scanning }
TRP3_Profiles = {
  ["profile-exact"] = {
    profileName = "原档案",
    player = { characteristics = { FN = "原名", CH = "112233" } },
    unknown = "quoted } brace"
  }
}
TRP3_Characters = { ["角色-服务器"] = { profileID = "profile-exact" } }
`
	backup := model.AccountBackup{
		UserID: owner.ID, AccountID: "writeback-account", ProfilesData: originalProfiles, ProfilesCount: 1,
		RawTrp3Lua: originalRaw, Checksum: "original-checksum", Version: 1,
	}
	if err := db.Create(&backup).Error; err != nil {
		t.Fatalf("create backup: %v", err)
	}
	ownerToken := newTestToken(t, owner)
	otherToken := newTestToken(t, other)
	createResp := performRequest(server.router, http.MethodPost, "/api/v1/character-cards", map[string]interface{}{
		"source_type": "backup", "source_backup_id": backup.ID, "source_profile_id": "profile-exact",
	}, ownerToken)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create card expected 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}
	card := decodeCharacterCardResponse(t, createResp)
	cardPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(card.ID), 10)
	impressions := emptyCharacterCardImpressionRequests()
	impressions[0] = characterCardImpressionRequest{Slot: 1, Active: true, Title: "第一眼", Text: "来自 RPBox", TRP3Icon: "spell_arcane"}
	update := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{
		"first_name": "写回名", "class": "德鲁伊", "class_color": "#aabbcc", "impressions": impressions,
		"summary": "RPBox 摘要", "background_story": "RPBox 富文本", "other_content": "RPBox 扩展内容",
	}, ownerToken)
	if update.Code != http.StatusOK {
		t.Fatalf("update card expected 200, got %d body=%s", update.Code, update.Body.String())
	}
	updated := decodeCharacterCardResponse(t, update)
	if updated.ClassColor != "AABBCC" || updated.NameColor != "AABBCC" {
		t.Fatalf("CH compatibility fields diverged before export: %+v", updated)
	}

	exportResp := performRequest(server.router, http.MethodGet, cardPath+"/trp3-lua", nil, ownerToken)
	if exportResp.Code != http.StatusOK || !strings.Contains(exportResp.Body.String(), `CH = \"AABBCC\"`) || strings.Count(exportResp.Body.String(), `CH =`) != 1 {
		t.Fatalf("TRP3 Lua export did not contain exactly one canonical CH: code=%d body=%s", exportResp.Code, exportResp.Body.String())
	}
	var exportPayload struct {
		Profile map[string]interface{} `json:"profile"`
	}
	if err := json.Unmarshal(exportResp.Body.Bytes(), &exportPayload); err != nil {
		t.Fatalf("decode pure export response: %v body=%s", err, exportResp.Body.String())
	}
	exportedPlayer := exportPayload.Profile["player"].(map[string]interface{})
	exportedCharacteristics := exportedPlayer["characteristics"].(map[string]interface{})
	exportedMisc := exportedPlayer["misc"].(map[string]interface{})
	if _, exists := exportPayload.Profile["unknown"]; !exists || exportedPlayer["unknownPlayer"] == nil || exportedCharacteristics["CUSTOM"] == nil {
		t.Fatalf("pure export discarded imported unknown sections: %+v", exportPayload.Profile)
	}
	if exportedMisc["ST"] == nil || exportedMisc["PE"].(map[string]interface{})["1"].(map[string]interface{})["TI"] != "来源印象" {
		t.Fatalf("pure export changed imported player.misc: %+v", exportedMisc)
	}
	if strings.Contains(exportResp.Body.String(), "来自 RPBox") || strings.Contains(exportResp.Body.String(), "RPBox 富文本") {
		t.Fatalf("pure export leaked RPBox-only rich content or structured impressions: %s", exportResp.Body.String())
	}
	if foreign := performRequest(server.router, http.MethodGet, cardPath+"/trp3-lua", nil, otherToken); foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign export expected 404, got %d body=%s", foreign.Code, foreign.Body.String())
	}

	writeResp := performRequest(server.router, http.MethodPost, cardPath+"/write-back-trp3", map[string]interface{}{}, ownerToken)
	if writeResp.Code != http.StatusOK {
		t.Fatalf("write-back expected 200, got %d body=%s", writeResp.Code, writeResp.Body.String())
	}
	var writePayload struct {
		Backup   model.AccountBackup        `json:"backup"`
		Snapshot model.AccountBackupVersion `json:"snapshot"`
		Profile  map[string]interface{}     `json:"profile"`
		Lua      string                     `json:"lua"`
	}
	if err := json.Unmarshal(writeResp.Body.Bytes(), &writePayload); err != nil {
		t.Fatalf("decode write-back response: %v body=%s", err, writeResp.Body.String())
	}
	if writePayload.Snapshot.ID == 0 || writePayload.Snapshot.Name == "" || writePayload.Snapshot.ContentHash == "" || writePayload.Snapshot.RawTrp3Lua != originalRaw {
		t.Fatalf("mandatory pre-write snapshot incomplete: %+v", writePayload.Snapshot)
	}
	if writePayload.Backup.Version != 2 || !strings.Contains(writePayload.Backup.RawTrp3Lua, "TRP3_Characters") || !strings.Contains(writePayload.Backup.RawTrp3Lua, "braces in a comment") {
		t.Fatalf("write-back did not preserve non-profile raw Lua content: %+v", writePayload.Backup)
	}
	profiles, err := decodeTRP3ProfilesObject(writePayload.Backup.ProfilesData)
	if err != nil {
		t.Fatalf("decode written profiles: %v", err)
	}
	profile := profiles["profile-exact"].(map[string]interface{})
	player := profile["player"].(map[string]interface{})
	characteristics := player["characteristics"].(map[string]interface{})
	if characteristics["FN"] != "写回名" || characteristics["CL"] != "德鲁伊" || characteristics["CH"] != "AABBCC" {
		t.Fatalf("written TRP3 characteristics mismatch: %+v", characteristics)
	}
	misc := player["misc"].(map[string]interface{})
	pe := misc["PE"].(map[string]interface{})
	if pe["1"].(map[string]interface{})["TI"] != "来源印象" || misc["ST"] == nil {
		t.Fatalf("write-back overwrote imported misc with RPBox impressions: %+v", misc)
	}
	if _, exists := profile["unknown"]; !exists || player["unknownPlayer"] == nil || characteristics["CUSTOM"] == nil {
		t.Fatalf("write-back discarded unknown profile sections: %+v", profile)
	}

	// A second write-back must snapshot even when the generated content is
	// unchanged; clients cannot opt out of this protection.
	secondWrite := performRequest(server.router, http.MethodPost, cardPath+"/write-back-trp3", map[string]interface{}{"snapshot_name": "写回前保护"}, ownerToken)
	if secondWrite.Code != http.StatusOK {
		t.Fatalf("second write-back expected 200, got %d body=%s", secondWrite.Code, secondWrite.Body.String())
	}
	var count int64
	if err := db.Model(&model.AccountBackupVersion{}).Where("backup_id = ?", backup.ID).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("each write-back must create one snapshot: count=%d err=%v", count, err)
	}
	unsafe := performRequest(server.router, http.MethodPost, cardPath+"/write-back-trp3", map[string]interface{}{"profile_id": "../escape"}, ownerToken)
	if unsafe.Code != http.StatusBadRequest {
		t.Fatalf("unsafe profile id expected 400, got %d body=%s", unsafe.Code, unsafe.Body.String())
	}
	if err := db.Model(&model.AccountBackupVersion{}).Where("backup_id = ?", backup.ID).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("rejected path traversal created a snapshot: count=%d err=%v", count, err)
	}
}

func TestCharacterCardTRP3WriteBackNormalizesLegacyProfilesWrapper(t *testing.T) {
	db, server, owner, _ := newCharacterCardTestServer(t)
	backup := model.AccountBackup{
		UserID: owner.ID, AccountID: "legacy-wrapper", ProfilesCount: 1, Checksum: "legacy", Version: 1,
		ProfilesData: `{"profiles":{"legacy-profile":{"profileName":"Legacy","player":{"characteristics":{"FN":"旧名","CUSTOM":"保留"},"misc":{"PE":{"1":{"TI":"旧印象"}},"OTHER":{"value":9}}},"unknown":{"enabled":true}}}}`,
	}
	if err := db.Create(&backup).Error; err != nil {
		t.Fatalf("create legacy backup: %v", err)
	}
	token := newTestToken(t, owner)
	createResp := performRequest(server.router, http.MethodPost, "/api/v1/character-cards", map[string]interface{}{
		"source_type": "backup", "source_backup_id": backup.ID, "source_profile_id": "legacy-profile",
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create card from legacy wrapper expected 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}
	card := decodeCharacterCardResponse(t, createResp)
	cardPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(card.ID), 10)
	if update := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{"first_name": "新名"}, token); update.Code != http.StatusOK {
		t.Fatalf("update legacy-backed card expected 200, got %d body=%s", update.Code, update.Body.String())
	}
	writeResp := performRequest(server.router, http.MethodPost, cardPath+"/write-back-trp3", map[string]interface{}{}, token)
	if writeResp.Code != http.StatusOK {
		t.Fatalf("legacy wrapper write-back expected 200, got %d body=%s", writeResp.Code, writeResp.Body.String())
	}
	var payload struct {
		Backup   model.AccountBackup        `json:"backup"`
		Snapshot model.AccountBackupVersion `json:"snapshot"`
	}
	if err := json.Unmarshal(writeResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode legacy write-back: %v body=%s", err, writeResp.Body.String())
	}
	if payload.Snapshot.ID == 0 {
		t.Fatal("legacy write-back did not create mandatory snapshot")
	}
	var canonical map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payload.Backup.ProfilesData), &canonical); err != nil {
		t.Fatalf("written profiles are invalid JSON: %v data=%s", err, payload.Backup.ProfilesData)
	}
	if _, mixedWrapper := canonical["profiles"]; mixedWrapper || len(canonical) != 1 || canonical["legacy-profile"] == nil {
		t.Fatalf("legacy wrapper was not normalized to canonical profile map: %s", payload.Backup.ProfilesData)
	}
	profiles, err := decodeTRP3ProfilesObject(payload.Backup.ProfilesData)
	if err != nil {
		t.Fatalf("decode normalized profiles: %v", err)
	}
	profile := profiles["legacy-profile"].(map[string]interface{})
	player := profile["player"].(map[string]interface{})
	characteristics := player["characteristics"].(map[string]interface{})
	if characteristics["FN"] != "新名" || characteristics["CUSTOM"] != "保留" || player["misc"] == nil || profile["unknown"] == nil {
		t.Fatalf("legacy write-back discarded target fields: %+v", profile)
	}
}

func TestCharacterCardTRP3RelationshipStatusIsNumericOrRejected(t *testing.T) {
	db, server, owner, _ := newCharacterCardTestServer(t)
	backup := model.AccountBackup{
		UserID: owner.ID, AccountID: "rs-account", ProfilesCount: 1, Checksum: "rs", Version: 1,
		ProfilesData: characterCardProfilesData(`{"profileName":"RS","player":{"characteristics":{"FN":"关系测试","RS":2}}}`),
	}
	if err := db.Create(&backup).Error; err != nil {
		t.Fatalf("create RS backup: %v", err)
	}
	token := newTestToken(t, owner)
	createResp := performRequest(server.router, http.MethodPost, "/api/v1/character-cards", map[string]interface{}{
		"source_type": "backup", "source_backup_id": backup.ID, "source_profile_id": "profile-exact",
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create RS card expected 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}
	card := decodeCharacterCardResponse(t, createResp)
	cardPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(card.ID), 10)
	if update := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{"relationship_status": "3"}, token); update.Code != http.StatusOK {
		t.Fatalf("set numeric RS expected 200, got %d body=%s", update.Code, update.Body.String())
	}
	exportResp := performRequest(server.router, http.MethodGet, cardPath+"/trp3-lua", nil, token)
	if exportResp.Code != http.StatusOK || !strings.Contains(exportResp.Body.String(), "RS = 3") || strings.Contains(exportResp.Body.String(), `RS = \"3\"`) {
		t.Fatalf("RS was not emitted as a Lua number: code=%d body=%s", exportResp.Code, exportResp.Body.String())
	}
	writeResp := performRequest(server.router, http.MethodPost, cardPath+"/write-back-trp3", map[string]interface{}{}, token)
	if writeResp.Code != http.StatusOK {
		t.Fatalf("numeric RS write-back expected 200, got %d body=%s", writeResp.Code, writeResp.Body.String())
	}
	var writePayload struct {
		Backup model.AccountBackup `json:"backup"`
	}
	if err := json.Unmarshal(writeResp.Body.Bytes(), &writePayload); err != nil {
		t.Fatalf("decode numeric RS write-back: %v", err)
	}
	profiles, err := decodeTRP3ProfilesObject(writePayload.Backup.ProfilesData)
	if err != nil {
		t.Fatalf("decode numeric RS profiles: %v", err)
	}
	characteristics := profiles["profile-exact"].(map[string]interface{})["player"].(map[string]interface{})["characteristics"].(map[string]interface{})
	if rs, ok := characteristics["RS"].(json.Number); !ok || rs.String() != "3" {
		t.Fatalf("written RS is not a JSON/TRP3 number: %#v", characteristics["RS"])
	}

	if update := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{"relationship_status": ""}, token); update.Code != http.StatusOK {
		t.Fatalf("clear RS expected 200, got %d body=%s", update.Code, update.Body.String())
	}
	clearResp := performRequest(server.router, http.MethodPost, cardPath+"/write-back-trp3", map[string]interface{}{}, token)
	if clearResp.Code != http.StatusOK {
		t.Fatalf("empty RS write-back expected 200, got %d body=%s", clearResp.Code, clearResp.Body.String())
	}
	if err := json.Unmarshal(clearResp.Body.Bytes(), &writePayload); err != nil {
		t.Fatalf("decode empty RS write-back: %v", err)
	}
	profiles, err = decodeTRP3ProfilesObject(writePayload.Backup.ProfilesData)
	if err != nil {
		t.Fatalf("decode cleared RS profiles: %v", err)
	}
	characteristics = profiles["profile-exact"].(map[string]interface{})["player"].(map[string]interface{})["characteristics"].(map[string]interface{})
	if _, exists := characteristics["RS"]; exists {
		t.Fatalf("empty RS should delete the target field: %+v", characteristics)
	}

	if update := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{"relationship_status": "伴侣"}, token); update.Code != http.StatusOK {
		t.Fatalf("store invalid RS fixture expected 200, got %d body=%s", update.Code, update.Body.String())
	}
	for _, request := range []struct {
		method string
		path   string
		body   interface{}
	}{
		{method: http.MethodGet, path: cardPath + "/trp3-lua"},
		{method: http.MethodPost, path: cardPath + "/write-back-trp3", body: map[string]interface{}{}},
	} {
		resp := performRequest(server.router, request.method, request.path, request.body, token)
		if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Body.String(), "relationship_status") || !strings.Contains(resp.Body.String(), "0 到 5") {
			t.Fatalf("invalid RS expected explicit 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	}
	var snapshots int64
	if err := db.Model(&model.AccountBackupVersion{}).Where("backup_id = ?", backup.ID).Count(&snapshots).Error; err != nil || snapshots != 2 {
		t.Fatalf("invalid RS should not create a write-back snapshot: count=%d err=%v", snapshots, err)
	}
}

func TestCharacterCardTRP3PureExportSupportsBlankCardWithoutSource(t *testing.T) {
	db, server, owner, _ := newCharacterCardTestServer(t)
	token := newTestToken(t, owner)
	createResp := performRequest(server.router, http.MethodPost, "/api/v1/character-cards", map[string]interface{}{"source_type": "blank"}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create blank card expected 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}
	card := decodeCharacterCardResponse(t, createResp)
	if err := db.Model(&model.CharacterCard{}).Where("id = ?", card.ID).Updates(map[string]interface{}{
		"summary": "RPBox 摘要", "background_story": "RPBox 富文本", "first_impression": "旧自由文本",
		"other_content": "RPBox 扩展内容", "portrait_image": "character-cards/private/custom.webp",
	}).Error; err != nil {
		t.Fatalf("seed RPBox-only blank-card fields: %v", err)
	}
	if err := db.Model(&model.CharacterCardImpression{}).Where("character_card_id = ? AND slot = ?", card.ID, 1).Updates(map[string]interface{}{
		"active": true, "title": "结构化印象", "text": "不应写入 TRP3", "image": "character-card-impressions/private/custom.webp",
	}).Error; err != nil {
		t.Fatalf("seed RPBox-only impression: %v", err)
	}
	cardPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(card.ID), 10)
	exportResp := performRequest(server.router, http.MethodGet, cardPath+"/trp3-lua", nil, token)
	if exportResp.Code != http.StatusOK {
		t.Fatalf("blank-card pure export expected 200, got %d body=%s", exportResp.Code, exportResp.Body.String())
	}
	var payload struct {
		ProfileID string                 `json:"profile_id"`
		Profile   map[string]interface{} `json:"profile"`
		Lua       string                 `json:"lua"`
	}
	if err := json.Unmarshal(exportResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode blank-card export: %v body=%s", err, exportResp.Body.String())
	}
	if payload.ProfileID != fmt.Sprintf("rpbox-%d", card.ID) || !strings.HasPrefix(payload.Lua, "TRP3_Profiles = {") {
		t.Fatalf("blank-card export did not synthesize a safe standalone profile: %+v", payload)
	}
	player := payload.Profile["player"].(map[string]interface{})
	if _, hasMisc := player["misc"]; hasMisc {
		t.Fatalf("blank export synthesized RPBox impressions into player.misc: %+v", player)
	}
	for _, rpboxOnly := range []string{"RPBox 摘要", "RPBox 富文本", "旧自由文本", "RPBox 扩展内容", "结构化印象", "custom.webp"} {
		if strings.Contains(exportResp.Body.String(), rpboxOnly) {
			t.Fatalf("blank export leaked RPBox-only content %q: %s", rpboxOnly, exportResp.Body.String())
		}
	}
}

func TestAccountBackupVersionOwnerLifecycleHasNoSilentTenVersionLimit(t *testing.T) {
	db, server, owner, other := newCharacterCardTestServer(t)
	backup := model.AccountBackup{
		UserID: owner.ID, AccountID: "version-account", ProfilesData: characterCardProfilesData(`{"profileName":"v0","player":{"characteristics":{"FN":"v0"}}}`),
		ProfilesCount: 1, Checksum: "v0", Version: 1,
	}
	if err := db.Create(&backup).Error; err != nil {
		t.Fatalf("create backup: %v", err)
	}
	ownerToken := newTestToken(t, owner)
	otherToken := newTestToken(t, other)
	for index := 1; index <= 12; index++ {
		profiles := characterCardProfilesData(fmt.Sprintf(`{"profileName":"v%d","player":{"characteristics":{"FN":"v%d"}}}`, index, index))
		resp := performRequest(server.router, http.MethodPost, "/api/v1/account-backups", map[string]interface{}{
			"account_id": backup.AccountID, "profiles_data": profiles, "profiles_count": 1, "checksum": fmt.Sprintf("v%d", index),
		}, ownerToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("backup update %d expected 200, got %d body=%s", index, resp.Code, resp.Body.String())
		}
	}
	versionsPath := "/api/v1/account-backups/" + backup.AccountID + "/versions"
	listResp := performRequest(server.router, http.MethodGet, versionsPath, nil, ownerToken)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list versions expected 200, got %d body=%s", listResp.Code, listResp.Body.String())
	}
	var list struct {
		Versions []model.AccountBackupVersion `json:"versions"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &list); err != nil || len(list.Versions) != 12 {
		t.Fatalf("version history was silently truncated: err=%v count=%d body=%s", err, len(list.Versions), listResp.Body.String())
	}
	if strings.Contains(listResp.Body.String(), "profiles_data") || strings.Contains(listResp.Body.String(), "raw_trp3_lua") {
		t.Fatalf("metadata list leaked heavy version content: %s", listResp.Body.String())
	}
	target := list.Versions[len(list.Versions)-1]
	detailPath := versionsPath + "/" + strconv.FormatUint(uint64(target.ID), 10)
	if foreign := performRequest(server.router, http.MethodGet, detailPath, nil, otherToken); foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign version detail expected 404, got %d body=%s", foreign.Code, foreign.Body.String())
	}
	detail := performRequest(server.router, http.MethodGet, detailPath, nil, ownerToken)
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), `"profiles_data"`) {
		t.Fatalf("owner version detail missing content: code=%d body=%s", detail.Code, detail.Body.String())
	}
	var detailPayload struct {
		Version model.AccountBackupVersion `json:"version"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &detailPayload); err != nil {
		t.Fatalf("decode version detail: %v", err)
	}
	rename := performRequest(server.router, http.MethodPut, detailPath, map[string]interface{}{"name": "基线版本"}, ownerToken)
	if rename.Code != http.StatusOK || !strings.Contains(rename.Body.String(), "基线版本") {
		t.Fatalf("rename version failed: code=%d body=%s", rename.Code, rename.Body.String())
	}

	var beforeRestore model.AccountBackup
	if err := db.First(&beforeRestore, backup.ID).Error; err != nil {
		t.Fatalf("load current backup: %v", err)
	}
	restore := performRequest(server.router, http.MethodPost, detailPath+"/restore", map[string]interface{}{"snapshot_name": "回退前保护"}, ownerToken)
	if restore.Code != http.StatusOK {
		t.Fatalf("restore version expected 200, got %d body=%s", restore.Code, restore.Body.String())
	}
	var restored model.AccountBackup
	if err := db.First(&restored, backup.ID).Error; err != nil {
		t.Fatalf("reload restored backup: %v", err)
	}
	if restored.ProfilesData == beforeRestore.ProfilesData || restored.ProfilesData != detailPayload.Version.ProfilesData {
		t.Fatalf("restore did not apply target content: target=%q restored=%q", detailPayload.Version.ProfilesData, restored.ProfilesData)
	}
	var protection model.AccountBackupVersion
	if err := db.Where("backup_id = ? AND name = ?", backup.ID, "回退前保护").First(&protection).Error; err != nil || protection.ProfilesData != beforeRestore.ProfilesData {
		t.Fatalf("restore did not protect current version first: err=%v protection=%+v", err, protection)
	}
	deleteResp := performRequest(server.router, http.MethodDelete, detailPath, nil, ownerToken)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete version expected 200, got %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	if missing := performRequest(server.router, http.MethodGet, detailPath, nil, ownerToken); missing.Code != http.StatusNotFound {
		t.Fatalf("deleted version expected 404, got %d", missing.Code)
	}
}
