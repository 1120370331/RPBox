package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func TestCharacterCardCreatesBlankAndMapsOwnedBackupProfile(t *testing.T) {
	db, server, owner, other := newCharacterCardTestServer(t)
	profilesData := characterCardProfilesData(`{
		"profileName":"银月档案",
		"player":{"characteristics":{
			"FN":"艾琳","LN":"晨星","TI":"巡林客","FT":"银月城远行巡林客",
			"RA":"高等精灵","CL":"游侠","EC":"琥珀色","EH":"A1B2C3",
			"AG":"127","HE":"一米七二","WE":"轻盈","BP":"银月城","RE":"暴风城",
			"RS":2,"IC":"ability_hunter_beastcall","CH":"DDBB88"
		}}
	}`)
	backup := model.AccountBackup{UserID: owner.ID, AccountID: "account-owner", ProfilesData: profilesData, ProfilesCount: 1, Checksum: "sum"}
	otherBackup := model.AccountBackup{UserID: other.ID, AccountID: "account-other", ProfilesData: profilesData, ProfilesCount: 1, Checksum: "sum-other"}
	if err := db.Create(&[]*model.AccountBackup{&backup, &otherBackup}).Error; err != nil {
		t.Fatalf("create backups: %v", err)
	}

	ownerToken := newTestToken(t, owner)
	sourcesResp := performRequest(server.router, http.MethodGet, "/api/v1/character-card-sources", nil, ownerToken)
	if sourcesResp.Code != http.StatusOK {
		t.Fatalf("sources expected 200, got %d body=%s", sourcesResp.Code, sourcesResp.Body.String())
	}
	var sourcesPayload struct {
		Sources []characterCardSourceDTO `json:"sources"`
	}
	if err := json.Unmarshal(sourcesResp.Body.Bytes(), &sourcesPayload); err != nil {
		t.Fatalf("decode sources: %v", err)
	}
	if len(sourcesPayload.Sources) != 1 {
		t.Fatalf("expected only the owner's one source, got %+v", sourcesPayload.Sources)
	}
	var sourceContract struct {
		Sources []map[string]interface{} `json:"sources"`
	}
	if err := json.Unmarshal(sourcesResp.Body.Bytes(), &sourceContract); err != nil || len(sourceContract.Sources) != 1 {
		t.Fatalf("decode source contract: err=%v payload=%+v", err, sourceContract)
	}
	for _, key := range []string{"backup_id", "account_id", "profile_id"} {
		if _, exists := sourceContract.Sources[0][key]; !exists {
			t.Fatalf("source response missing frontend contract key %q: %s", key, sourcesResp.Body.String())
		}
	}
	for _, legacyKey := range []string{"source_backup_id", "source_account_id", "source_profile_id"} {
		if _, exists := sourceContract.Sources[0][legacyKey]; exists {
			t.Fatalf("source response unexpectedly used card provenance key %q: %s", legacyKey, sourcesResp.Body.String())
		}
	}
	source := sourcesPayload.Sources[0]
	if source.SourceBackupID != backup.ID || source.SourceAccountID != backup.AccountID || source.SourceProfileID != "profile-exact" {
		t.Fatalf("unexpected source identity: %+v", source)
	}
	if source.DisplayName != "艾琳 晨星" || source.Title != "巡林客" || source.Icon != "ability_hunter_beastcall" {
		t.Fatalf("unexpected source summary: %+v", source)
	}
	if source.ClassColor != "DDBB88" || source.NameColor != "DDBB88" {
		t.Fatalf("source did not expose the single TRP3 CH color through both compatibility fields: %+v", source)
	}
	if strings.Contains(sourcesResp.Body.String(), "player") || strings.Contains(sourcesResp.Body.String(), "characteristics") {
		t.Fatalf("source response leaked raw profile data: %s", sourcesResp.Body.String())
	}

	blankResp := performRequest(server.router, http.MethodPost, "/api/v1/character-cards", map[string]interface{}{
		"source_type": "blank",
	}, ownerToken)
	if blankResp.Code != http.StatusCreated {
		t.Fatalf("blank create expected 201, got %d body=%s", blankResp.Code, blankResp.Body.String())
	}
	blank := decodeCharacterCardResponse(t, blankResp)
	if blank.Status != model.CharacterCardStatusDraft || blank.Visibility != model.CharacterCardVisibilityPrivate || blank.SourceBackupID != nil {
		t.Fatalf("unexpected blank card: %+v", blank)
	}

	importResp := performRequest(server.router, http.MethodPost, "/api/v1/character-cards", map[string]interface{}{
		"source_type":       "backup",
		"source_backup_id":  backup.ID,
		"source_profile_id": "profile-exact",
	}, ownerToken)
	if importResp.Code != http.StatusCreated {
		t.Fatalf("backup create expected 201, got %d body=%s", importResp.Code, importResp.Body.String())
	}
	imported := decodeCharacterCardResponse(t, importResp)
	if imported.FirstName != "艾琳" || imported.LastName != "晨星" || imported.DisplayName != "艾琳 晨星" ||
		imported.FullTitle != "银月城远行巡林客" || imported.Race != "高等精灵" || imported.Class != "游侠" ||
		imported.EyeColor != "琥珀色" || imported.EyeColorHex != "A1B2C3" || imported.Age != "127" ||
		imported.Height != "一米七二" || imported.Weight != "轻盈" || imported.Birthplace != "银月城" ||
		imported.Residence != "暴风城" || imported.RelationshipStatus != "2" || imported.ClassColor != "DDBB88" || imported.NameColor != "DDBB88" {
		t.Fatalf("TRP3 field mapping mismatch: %+v", imported)
	}
	if imported.SourceBackupID == nil || *imported.SourceBackupID != backup.ID || imported.SourceAccountID != backup.AccountID || imported.SourceProfileID != "profile-exact" {
		t.Fatalf("source tracking mismatch: %+v", imported)
	}
	if strings.Contains(importResp.Body.String(), `"portrait_image":"`) {
		t.Fatalf("response leaked portrait storage value: %s", importResp.Body.String())
	}

	var storedBackup model.AccountBackup
	if err := db.First(&storedBackup, backup.ID).Error; err != nil {
		t.Fatalf("reload backup: %v", err)
	}
	if storedBackup.ProfilesData != profilesData {
		t.Fatal("import unexpectedly modified the backup")
	}
}

func TestCharacterCardBadOrForeignSourceDoesNotCreateRecord(t *testing.T) {
	db, server, owner, other := newCharacterCardTestServer(t)
	validBackup := model.AccountBackup{
		UserID: owner.ID, AccountID: "valid", Checksum: "valid",
		ProfilesData: characterCardProfilesData(`{"profileName":"Valid","player":{"characteristics":{"FN":"有效"}}}`),
	}
	corruptBackup := model.AccountBackup{UserID: owner.ID, AccountID: "corrupt", Checksum: "corrupt", ProfilesData: `{"profile-exact":`}
	foreignBackup := model.AccountBackup{
		UserID: other.ID, AccountID: "foreign", Checksum: "foreign",
		ProfilesData: characterCardProfilesData(`{"profileName":"Foreign","player":{"characteristics":{"FN":"他人"}}}`),
	}
	unsafeAccountBackup := model.AccountBackup{
		UserID: owner.ID, AccountID: "../escape", Checksum: "unsafe-account",
		ProfilesData: characterCardProfilesData(`{"profileName":"Unsafe","player":{"characteristics":{"FN":"越界"}}}`),
	}
	if err := db.Create(&[]*model.AccountBackup{&validBackup, &corruptBackup, &foreignBackup, &unsafeAccountBackup}).Error; err != nil {
		t.Fatalf("create backups: %v", err)
	}
	token := newTestToken(t, owner)

	tests := []struct {
		name      string
		backupID  uint
		profileID string
		wantCode  int
	}{
		{name: "foreign backup", backupID: foreignBackup.ID, profileID: "profile-exact", wantCode: http.StatusNotFound},
		{name: "missing profile", backupID: validBackup.ID, profileID: "missing", wantCode: http.StatusNotFound},
		{name: "corrupt profiles json", backupID: corruptBackup.ID, profileID: "profile-exact", wantCode: http.StatusBadRequest},
		{name: "unsafe account directory", backupID: unsafeAccountBackup.ID, profileID: "profile-exact", wantCode: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp := performRequest(server.router, http.MethodPost, "/api/v1/character-cards", map[string]interface{}{
				"source_type":       "backup",
				"source_backup_id":  test.backupID,
				"source_profile_id": test.profileID,
			}, token)
			if resp.Code != test.wantCode {
				t.Fatalf("expected %d, got %d body=%s", test.wantCode, resp.Code, resp.Body.String())
			}
		})
	}

	var count int64
	if err := db.Model(&model.CharacterCard{}).Count(&count).Error; err != nil {
		t.Fatalf("count cards: %v", err)
	}
	if count != 0 {
		t.Fatalf("bad sources left %d partial records", count)
	}

	sourcesResp := performRequest(server.router, http.MethodGet, "/api/v1/character-card-sources", nil, token)
	if sourcesResp.Code != http.StatusOK || strings.Contains(sourcesResp.Body.String(), "../escape") {
		t.Fatalf("unsafe account directory appeared in sources: code=%d body=%s", sourcesResp.Code, sourcesResp.Body.String())
	}
	for _, accountID := range []string{"", ".", "..", "account/other", `account\other`, " account", "account.name"} {
		if err := validateCharacterCardSourceAccountID(accountID); err == nil {
			t.Fatalf("unsafe account directory %q was accepted", accountID)
		}
	}
	for _, accountID := range []string{"123456789#1", "Account_One", "账号-1"} {
		if err := validateCharacterCardSourceAccountID(accountID); err != nil {
			t.Fatalf("safe account directory %q was rejected: %v", accountID, err)
		}
	}
}

func TestCharacterCardOptionalCharacterRelationIsOwnedAndNotOneToOne(t *testing.T) {
	db, server, owner, other := newCharacterCardTestServer(t)
	ownedCharacter := model.Character{UserID: owner.ID, RefID: "owned-character"}
	foreignCharacter := model.Character{UserID: other.ID, RefID: "foreign-character"}
	if err := db.Create(&[]*model.Character{&ownedCharacter, &foreignCharacter}).Error; err != nil {
		t.Fatalf("create characters: %v", err)
	}
	cards := []model.CharacterCard{
		{UserID: owner.ID, Status: model.CharacterCardStatusDraft, Visibility: model.CharacterCardVisibilityPrivate},
		{UserID: owner.ID, Status: model.CharacterCardStatusDraft, Visibility: model.CharacterCardVisibilityPrivate},
	}
	if err := db.Create(&cards).Error; err != nil {
		t.Fatalf("create cards: %v", err)
	}
	token := newTestToken(t, owner)
	for _, card := range cards {
		path := "/api/v1/character-cards/" + strconv.FormatUint(uint64(card.ID), 10)
		resp := performRequest(server.router, http.MethodPut, path, map[string]interface{}{"character_id": ownedCharacter.ID}, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("associate card %d expected 200, got %d body=%s", card.ID, resp.Code, resp.Body.String())
		}
	}
	var linkedCount int64
	if err := db.Model(&model.CharacterCard{}).Where("character_id = ?", ownedCharacter.ID).Count(&linkedCount).Error; err != nil || linkedCount != 2 {
		t.Fatalf("expected non-unique optional relation for two cards: count=%d err=%v", linkedCount, err)
	}

	path := "/api/v1/character-cards/" + strconv.FormatUint(uint64(cards[0].ID), 10)
	foreignResp := performRequest(server.router, http.MethodPut, path, map[string]interface{}{"character_id": foreignCharacter.ID}, token)
	if foreignResp.Code != http.StatusBadRequest {
		t.Fatalf("foreign character relation expected 400, got %d body=%s", foreignResp.Code, foreignResp.Body.String())
	}
	clearResp := performRequest(server.router, http.MethodPut, path, map[string]interface{}{"character_id": nil}, token)
	if clearResp.Code != http.StatusOK {
		t.Fatalf("null should clear optional relation: code=%d body=%s", clearResp.Code, clearResp.Body.String())
	}
	var cleared model.CharacterCard
	if err := db.First(&cleared, cards[0].ID).Error; err != nil || cleared.CharacterID != nil {
		t.Fatalf("character relation was not cleared: err=%v card=%+v", err, cleared)
	}
}

func TestCharacterCardOwnershipVisibilityAndPublicWall(t *testing.T) {
	db, server, owner, other := newCharacterCardTestServer(t)
	backupID := uint(77)
	cards := []model.CharacterCard{
		{UserID: owner.ID, DisplayName: "公开卡", Status: model.CharacterCardStatusPublished, Visibility: model.CharacterCardVisibilityPublic, SortOrder: 2, SourceBackupID: &backupID, SourceAccountID: "private-account", SourceProfileID: "private-profile", BackgroundStory: `<p>公开卡完整背景</p>`, FirstImpression: `<p>公开卡第一印象</p>`, OtherContent: `<p>公开卡其他资料</p>`},
		{UserID: owner.ID, DisplayName: "私密卡", Status: model.CharacterCardStatusPublished, Visibility: model.CharacterCardVisibilityPrivate, SortOrder: 1, BackgroundStory: `<p>私密卡完整背景</p>`},
		{UserID: owner.ID, DisplayName: "公开草稿", Status: model.CharacterCardStatusDraft, Visibility: model.CharacterCardVisibilityPublic, SortOrder: 0, OtherContent: `<p>草稿完整资料</p>`},
	}
	if err := db.Create(&cards).Error; err != nil {
		t.Fatalf("create cards: %v", err)
	}
	ownerToken := newTestToken(t, owner)
	otherToken := newTestToken(t, other)

	publicPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(cards[0].ID), 10)
	privatePath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(cards[1].ID), 10)
	publicDraftPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(cards[2].ID), 10)
	if resp := performRequest(server.router, http.MethodGet, publicPath, nil, ""); resp.Code != http.StatusOK {
		t.Fatalf("visitor public detail expected 200, got %d body=%s", resp.Code, resp.Body.String())
	} else if strings.Contains(resp.Body.String(), "private-account") || strings.Contains(resp.Body.String(), "private-profile") || strings.Contains(resp.Body.String(), "source_backup_id") {
		t.Fatalf("public detail leaked owner-only source metadata: %s", resp.Body.String())
	} else if !strings.Contains(resp.Body.String(), `"background_story"`) || !strings.Contains(resp.Body.String(), "公开卡完整背景") || !strings.Contains(resp.Body.String(), `"first_impression"`) || !strings.Contains(resp.Body.String(), "公开卡第一印象") || !strings.Contains(resp.Body.String(), `"other_content"`) || !strings.Contains(resp.Body.String(), "公开卡其他资料") {
		t.Fatalf("public detail omitted full rich text: %s", resp.Body.String())
	}
	for _, path := range []string{privatePath, publicDraftPath} {
		if resp := performRequest(server.router, http.MethodGet, path, nil, otherToken); resp.Code != http.StatusNotFound {
			t.Fatalf("unauthorized detail %s expected 404, got %d body=%s", path, resp.Code, resp.Body.String())
		}
	}
	if resp := performRequest(server.router, http.MethodGet, privatePath, nil, ownerToken); resp.Code != http.StatusOK {
		t.Fatalf("owner private detail expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	if resp := performRequest(server.router, http.MethodGet, "/api/v1/character-cards", nil, ownerToken); resp.Code != http.StatusOK {
		t.Fatalf("owner list expected 200, got %d body=%s", resp.Code, resp.Body.String())
	} else {
		var payload struct {
			Cards []characterCardDTO `json:"character_cards"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil || len(payload.Cards) != 3 {
			t.Fatalf("owner list expected all cards: err=%v payload=%+v", err, payload)
		}
		assertCharacterCardListOmitsRichText(t, resp.Body.String())
	}
	wallPath := "/api/v1/users/" + strconv.FormatUint(uint64(owner.ID), 10) + "/character-cards"
	if resp := performRequest(server.router, http.MethodGet, wallPath, nil, ""); resp.Code != http.StatusOK {
		t.Fatalf("public wall expected 200, got %d body=%s", resp.Code, resp.Body.String())
	} else {
		var payload struct {
			Cards []characterCardDTO `json:"character_cards"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil || len(payload.Cards) != 1 || payload.Cards[0].ID != cards[0].ID {
			t.Fatalf("public wall leaked non-public cards: err=%v payload=%+v", err, payload)
		}
		assertCharacterCardListOmitsRichText(t, resp.Body.String())
	}

	updateResp := performRequest(server.router, http.MethodPut, privatePath, map[string]interface{}{"display_name": "越权修改"}, otherToken)
	if updateResp.Code != http.StatusNotFound {
		t.Fatalf("foreign update expected 404, got %d body=%s", updateResp.Code, updateResp.Body.String())
	}
	deleteResp := performRequest(server.router, http.MethodDelete, privatePath, nil, otherToken)
	if deleteResp.Code != http.StatusNotFound {
		t.Fatalf("foreign delete expected 404, got %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	syncResp := performRequest(server.router, http.MethodPost, privatePath+"/sync-from-trp3", nil, otherToken)
	if syncResp.Code != http.StatusNotFound {
		t.Fatalf("foreign sync expected 404, got %d body=%s", syncResp.Code, syncResp.Body.String())
	}
	var unchanged model.CharacterCard
	if err := db.First(&unchanged, cards[1].ID).Error; err != nil || unchanged.DisplayName != "私密卡" {
		t.Fatalf("foreign CRUD changed card: err=%v card=%+v", err, unchanged)
	}
}

func TestCharacterCardSyncOnlyOverwritesTRP3BasicFields(t *testing.T) {
	db, server, owner, _ := newCharacterCardTestServer(t)
	backup := model.AccountBackup{
		UserID: owner.ID, AccountID: "sync-account", Checksum: "before",
		ProfilesData: characterCardProfilesData(`{"profileName":"Original Profile","player":{"characteristics":{"FN":"原名","LN":"原姓","RA":"人类","CL":"法师","WE":"原体重"}}}`),
	}
	if err := db.Create(&backup).Error; err != nil {
		t.Fatalf("create backup: %v", err)
	}
	token := newTestToken(t, owner)
	createResp := performRequest(server.router, http.MethodPost, "/api/v1/character-cards", map[string]interface{}{
		"source_type":       "backup",
		"source_backup_id":  backup.ID,
		"source_profile_id": "profile-exact",
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}
	created := decodeCharacterCardResponse(t, createResp)
	portraitTime := time.Now().UTC().Add(-time.Hour)
	keptPortrait := fmt.Sprintf("/uploads/character-cards/%d/portrait/kept.png", owner.ID)
	if err := db.Model(&model.CharacterCard{}).Where("id = ?", created.ID).Updates(map[string]interface{}{
		"display_name":              "RPBox 自定义名",
		"summary":                   "自定义摘要",
		"background_story":          "<p>背景故事</p>",
		"first_impression":          "<p>第一印象</p>",
		"other_content":             "<p>其他内容</p>",
		"portrait_image":            keptPortrait,
		"portrait_image_updated_at": portraitTime,
		"status":                    model.CharacterCardStatusPublished,
		"visibility":                model.CharacterCardVisibilityPublic,
		"sort_order":                9,
	}).Error; err != nil {
		t.Fatalf("customize card: %v", err)
	}

	updatedProfiles := characterCardProfilesData(`{
		"profileName":"Changed Profile",
		"player":{"characteristics":{
			"FN":"新名","LN":"新姓","TI":"新称号","FT":"新完整头衔","RA":"暗夜精灵","CL":"德鲁伊",
			"EC":"绿色","EH":"00AA11","AG":"300","HE":"很高","WE":"新体重","BP":"海加尔山",
			"RE":"月光林地","RS":"伴侣","IC":"new_icon","CH":"ABCDEF"
		}}
	}`)
	if err := db.Model(&model.AccountBackup{}).Where("id = ?", backup.ID).Update("profiles_data", updatedProfiles).Error; err != nil {
		t.Fatalf("update backup: %v", err)
	}

	syncPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(created.ID), 10) + "/sync-from-trp3"
	syncResp := performRequest(server.router, http.MethodPost, syncPath, nil, token)
	if syncResp.Code != http.StatusOK {
		t.Fatalf("sync expected 200, got %d body=%s", syncResp.Code, syncResp.Body.String())
	}
	var card model.CharacterCard
	if err := db.First(&card, created.ID).Error; err != nil {
		t.Fatalf("reload synced card: %v", err)
	}
	if card.FirstName != "新名" || card.LastName != "新姓" || card.Title != "新称号" || card.FullTitle != "新完整头衔" ||
		card.Race != "暗夜精灵" || card.Class != "德鲁伊" || card.EyeColor != "绿色" || card.EyeColorHex != "00AA11" ||
		card.Age != "300" || card.Height != "很高" || card.Weight != "新体重" || card.Birthplace != "海加尔山" ||
		card.Residence != "月光林地" || card.RelationshipStatus != "伴侣" || card.Icon != "new_icon" || card.ClassColor != "ABCDEF" || card.NameColor != "ABCDEF" {
		t.Fatalf("sync did not map all basic fields: %+v", card)
	}
	if card.DisplayName != "RPBox 自定义名" || card.Summary != "自定义摘要" || card.BackgroundStory != "<p>背景故事</p>" ||
		card.FirstImpression != "<p>第一印象</p>" || card.OtherContent != "<p>其他内容</p>" ||
		card.PortraitImage != keptPortrait || card.PortraitImageUpdatedAt == nil || !card.PortraitImageUpdatedAt.Equal(portraitTime) ||
		card.Status != model.CharacterCardStatusPublished || card.Visibility != model.CharacterCardVisibilityPublic || card.SortOrder != 9 {
		t.Fatalf("sync overwrote RPBox-only fields: %+v", card)
	}
}

func TestCharacterCardImpressionsImportVisibilityAndSyncBoundary(t *testing.T) {
	db, server, owner, moderator := newCharacterCardTestServer(t)
	if err := db.Model(&model.User{}).Where("id = ?", moderator.ID).Update("role", "moderator").Error; err != nil {
		t.Fatalf("promote moderator: %v", err)
	}
	backup := model.AccountBackup{
		UserID: owner.ID, AccountID: "impression-account", Checksum: "before",
		ProfilesData: characterCardProfilesData(`{
			"profileName":"五槽档案",
			"player":{
				"characteristics":{"FN":"初始名","LN":"星影","RA":"人类"},
				"misc":{"PE":{
					"1":{"AC":true,"IC":"ability_stealth","TI":"步伐轻盈","TX":"走路几乎没有声音。"},
					"2":{"AC":false,"IC":"inv_misc_note_01","TI":"隐藏线索","TX":"只有所有者应看到。"},
					"5":{"AC":true,"IC":"spell_fire_fire","TI":"灼热气息","TX":"靠近时能感到暖意。"}
				}}
			}
		}`),
	}
	if err := db.Create(&backup).Error; err != nil {
		t.Fatalf("create backup: %v", err)
	}
	token := newTestToken(t, owner)
	createResp := performRequest(server.router, http.MethodPost, "/api/v1/character-cards", map[string]interface{}{
		"source_type":       "backup",
		"source_backup_id":  backup.ID,
		"source_profile_id": "profile-exact",
	}, token)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create from backup expected 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}
	created := decodeCharacterCardResponse(t, createResp)
	assertFiveCharacterCardImpressionSlots(t, created.Impressions)
	if !created.Impressions[0].Active || created.Impressions[0].Title != "步伐轻盈" || created.Impressions[0].Text != "走路几乎没有声音。" || created.Impressions[0].TRP3Icon != "ability_stealth" {
		t.Fatalf("slot 1 PE mapping mismatch: %+v", created.Impressions[0])
	}
	if created.Impressions[1].Active || created.Impressions[1].Title != "隐藏线索" {
		t.Fatalf("inactive PE slot should still be returned to owner: %+v", created.Impressions[1])
	}
	if created.Impressions[2].Active || created.Impressions[2].Title != "" || !created.Impressions[4].Active {
		t.Fatalf("missing/fifth PE slot mapping mismatch: %+v", created.Impressions)
	}

	requests := impressionRequestsFromDTOs(created.Impressions)
	requests[0].Title = "RPBox 自定义印象"
	requests[0].Text = "同步基础资料时必须保留。"
	for index := 1; index < len(requests); index++ {
		requests[index].Active = false
	}
	cardPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(created.ID), 10)
	updateResp := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{
		"impressions":      requests,
		"first_impression": "<p>其他备注（兼容旧数据）</p>",
		"status":           model.CharacterCardStatusPublished,
		"visibility":       model.CharacterCardVisibilityPublic,
	}, token)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update impressions expected 200, got %d body=%s", updateResp.Code, updateResp.Body.String())
	}
	updated := decodeCharacterCardResponse(t, updateResp)
	assertFiveCharacterCardImpressionSlots(t, updated.Impressions)
	if updated.FirstImpression != "<p>其他备注（兼容旧数据）</p>" {
		t.Fatalf("legacy first_impression was not retained as other notes: %+v", updated)
	}
	if updated.ReviewStatus != model.CharacterCardReviewPending {
		t.Fatalf("public save must enter moderation: %+v", updated)
	}
	if hidden := performRequest(server.router, http.MethodGet, cardPath, nil, ""); hidden.Code != http.StatusNotFound {
		t.Fatalf("unapproved public card expected 404, got %d body=%s", hidden.Code, hidden.Body.String())
	}
	moderatorToken := newTestToken(t, moderator)
	reviewResp := performRequest(server.router, http.MethodPost, "/api/v1/moderator/review/character-cards/"+strconv.FormatUint(uint64(created.ID), 10), map[string]interface{}{
		"action": "approve",
	}, moderatorToken)
	if reviewResp.Code != http.StatusOK {
		t.Fatalf("approve character card expected 200, got %d body=%s", reviewResp.Code, reviewResp.Body.String())
	}

	publicResp := performRequest(server.router, http.MethodGet, cardPath, nil, "")
	if publicResp.Code != http.StatusOK {
		t.Fatalf("public detail expected 200, got %d body=%s", publicResp.Code, publicResp.Body.String())
	}
	publicCard := decodeCharacterCardResponse(t, publicResp)
	if len(publicCard.Impressions) != 1 || publicCard.Impressions[0].Slot != 1 || publicCard.Impressions[0].Title != "RPBox 自定义印象" {
		t.Fatalf("public detail should return active impressions only: %+v", publicCard.Impressions)
	}
	ownerResp := performRequest(server.router, http.MethodGet, cardPath, nil, token)
	if ownerResp.Code != http.StatusOK {
		t.Fatalf("owner detail expected 200, got %d body=%s", ownerResp.Code, ownerResp.Body.String())
	}
	assertFiveCharacterCardImpressionSlots(t, decodeCharacterCardResponse(t, ownerResp).Impressions)
	ownerList := performRequest(server.router, http.MethodGet, "/api/v1/character-cards", nil, token)
	var ownerListPayload struct {
		Cards []characterCardDTO `json:"character_cards"`
	}
	if ownerList.Code != http.StatusOK || json.Unmarshal(ownerList.Body.Bytes(), &ownerListPayload) != nil || len(ownerListPayload.Cards) != 1 {
		t.Fatalf("owner list failed: code=%d body=%s", ownerList.Code, ownerList.Body.String())
	}
	assertFiveCharacterCardImpressionSlots(t, ownerListPayload.Cards[0].Impressions)
	wallResp := performRequest(server.router, http.MethodGet, "/api/v1/users/"+strconv.FormatUint(uint64(owner.ID), 10)+"/character-cards", nil, "")
	var wallPayload struct {
		Cards []characterCardDTO `json:"character_cards"`
	}
	if wallResp.Code != http.StatusOK || json.Unmarshal(wallResp.Body.Bytes(), &wallPayload) != nil || len(wallPayload.Cards) != 1 || len(wallPayload.Cards[0].Impressions) != 1 {
		t.Fatalf("public wall did not filter impressions: code=%d body=%s", wallResp.Code, wallResp.Body.String())
	}

	updatedProfiles := characterCardProfilesData(`{
		"profileName":"变更后的档案",
		"player":{
			"characteristics":{"FN":"同步新名","LN":"星影","RA":"暗夜精灵"},
			"misc":{"PE":{"1":{"AC":true,"IC":"different_icon","TI":"TRP3 新印象","TX":"不得覆盖 RPBox 编辑。"}}}
		}
	}`)
	if err := db.Model(&model.AccountBackup{}).Where("id = ?", backup.ID).Update("profiles_data", updatedProfiles).Error; err != nil {
		t.Fatalf("update backup: %v", err)
	}
	syncResp := performRequest(server.router, http.MethodPost, cardPath+"/sync-from-trp3", nil, token)
	if syncResp.Code != http.StatusOK {
		t.Fatalf("sync expected 200, got %d body=%s", syncResp.Code, syncResp.Body.String())
	}
	synced := decodeCharacterCardResponse(t, syncResp)
	assertFiveCharacterCardImpressionSlots(t, synced.Impressions)
	if synced.FirstName != "同步新名" || synced.Race != "暗夜精灵" {
		t.Fatalf("sync did not update basic fields: %+v", synced)
	}
	if synced.Impressions[0].Title != "RPBox 自定义印象" || synced.Impressions[0].TRP3Icon != "ability_stealth" || synced.FirstImpression != "<p>其他备注（兼容旧数据）</p>" {
		t.Fatalf("sync crossed the RPBox impression boundary: %+v", synced)
	}
}

func TestCharacterCardImpressionValidationAndDatabaseConstraints(t *testing.T) {
	db, server, owner, _ := newCharacterCardTestServer(t)
	card := model.CharacterCard{UserID: owner.ID, Status: model.CharacterCardStatusDraft, Visibility: model.CharacterCardVisibilityPrivate}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create card: %v", err)
	}
	path := "/api/v1/character-cards/" + strconv.FormatUint(uint64(card.ID), 10)
	token := newTestToken(t, owner)
	requests := emptyCharacterCardImpressionRequests()
	requests[0].Title = strings.Repeat("题", characterCardImpressionTitleMax)
	requests[0].Text = strings.Repeat("文", characterCardImpressionTextMax)
	if resp := performRequest(server.router, http.MethodPut, path, map[string]interface{}{"impressions": requests}, token); resp.Code != http.StatusOK {
		t.Fatalf("80/500 boundary expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	tests := []struct {
		name     string
		requests []characterCardImpressionRequest
	}{
		{name: "title too long", requests: func() []characterCardImpressionRequest {
			value := emptyCharacterCardImpressionRequests()
			value[0].Title = strings.Repeat("题", characterCardImpressionTitleMax+1)
			return value
		}()},
		{name: "text too long", requests: func() []characterCardImpressionRequest {
			value := emptyCharacterCardImpressionRequests()
			value[0].Text = strings.Repeat("文", characterCardImpressionTextMax+1)
			return value
		}()},
		{name: "missing slot", requests: emptyCharacterCardImpressionRequests()[:4]},
		{name: "duplicate slot", requests: func() []characterCardImpressionRequest {
			value := emptyCharacterCardImpressionRequests()
			value[4].Slot = 1
			return value
		}()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp := performRequest(server.router, http.MethodPut, path, map[string]interface{}{"impressions": test.requests}, token)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("invalid impressions expected 400, got %d body=%s", resp.Code, resp.Body.String())
			}
		})
	}

	duplicate := model.CharacterCardImpression{CharacterCardID: card.ID, Slot: 1}
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("composite unique constraint accepted duplicate card/slot")
	}
	invalidSlot := model.CharacterCardImpression{CharacterCardID: card.ID, Slot: 6}
	if err := db.Create(&invalidSlot).Error; err == nil {
		t.Fatal("slot check constraint accepted slot 6")
	}
}

func TestCharacterCardPortraitPermissionsCachingAndRemovalVersion(t *testing.T) {
	db, server, owner, other := newCharacterCardTestServer(t)
	publicPortraitPath := writeCharacterCardTestPNG(t, server, fmt.Sprintf("character-cards/%d/portrait/public.png", owner.ID))
	privatePortraitPath := writeCharacterCardTestPNG(t, server, fmt.Sprintf("character-cards/%d/portrait/private.png", owner.ID))
	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	publicCard := model.CharacterCard{
		UserID: owner.ID, DisplayName: "公开肖像", PortraitImage: publicPortraitPath, PortraitImageUpdatedAt: &oldTime,
		Status: model.CharacterCardStatusPublished, Visibility: model.CharacterCardVisibilityPublic,
	}
	privateCard := model.CharacterCard{
		UserID: owner.ID, DisplayName: "私密肖像", PortraitImage: privatePortraitPath, PortraitImageUpdatedAt: &oldTime,
		Status: model.CharacterCardStatusPublished, Visibility: model.CharacterCardVisibilityPrivate,
	}
	if err := db.Create(&[]*model.CharacterCard{&publicCard, &privateCard}).Error; err != nil {
		t.Fatalf("create portrait cards: %v", err)
	}

	ownerToken := newTestToken(t, owner)
	publicImagePath := "/api/v1/images/character-card-portrait/" + strconv.FormatUint(uint64(publicCard.ID), 10)
	publicResp := performCharacterCardRequest(server, http.MethodGet, publicImagePath, "", "")
	if publicResp.Code != http.StatusOK || publicResp.Header().Get("ETag") == "" || publicResp.Header().Get("Cache-Control") != "private, max-age=3600" {
		t.Fatalf("unexpected public image response: code=%d cache=%q etag=%q body=%s", publicResp.Code, publicResp.Header().Get("Cache-Control"), publicResp.Header().Get("ETag"), publicResp.Body.String())
	}
	etag := publicResp.Header().Get("ETag")
	notModified := performCharacterCardRequest(server, http.MethodGet, publicImagePath, "", etag)
	if notModified.Code != http.StatusNotModified || notModified.Header().Get("ETag") != etag || notModified.Header().Get("Cache-Control") != "private, max-age=3600" {
		t.Fatalf("unexpected 304 response: code=%d headers=%v", notModified.Code, notModified.Header())
	}
	oldVersionedPath := publicImagePath + "?v=" + strconv.FormatInt(oldTime.UnixNano(), 10)
	versioned := performCharacterCardRequest(server, http.MethodGet, oldVersionedPath, "", "")
	if versioned.Code != http.StatusOK || versioned.Header().Get("Cache-Control") != "private, max-age=31536000, immutable" || strings.Contains(versioned.Header().Get("Cache-Control"), "public") {
		t.Fatalf("public portrait was eligible for shared immutable cache: code=%d cache=%q", versioned.Code, versioned.Header().Get("Cache-Control"))
	}
	publicCardPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(publicCard.ID), 10)
	privacyResp := performRequest(server.router, http.MethodPut, publicCardPath, map[string]interface{}{"visibility": model.CharacterCardVisibilityPrivate}, ownerToken)
	if privacyResp.Code != http.StatusOK {
		t.Fatalf("public to private transition expected 200, got %d body=%s", privacyResp.Code, privacyResp.Body.String())
	}
	privatePublicCard := decodeCharacterCardResponse(t, privacyResp)
	if privatePublicCard.PortraitImageUpdatedAt == nil || !privatePublicCard.PortraitImageUpdatedAt.After(oldTime) {
		t.Fatalf("public to private transition did not rotate portrait version: %+v", privatePublicCard)
	}
	if staleVisitor := performCharacterCardRequest(server, http.MethodGet, oldVersionedPath, "", ""); staleVisitor.Code != http.StatusNotFound {
		t.Fatalf("old public portrait URL remained readable after privacy transition: %d", staleVisitor.Code)
	}
	newOwnerPath := publicImagePath + "?v=" + strconv.FormatInt(privatePublicCard.PortraitImageUpdatedAt.UnixNano(), 10)
	if ownerResp := performCharacterCardRequest(server, http.MethodGet, newOwnerPath, ownerToken, ""); ownerResp.Code != http.StatusOK || ownerResp.Header().Get("Cache-Control") != "private, max-age=31536000, immutable" {
		t.Fatalf("owner private version expected private immutable cache, got code=%d cache=%q", ownerResp.Code, ownerResp.Header().Get("Cache-Control"))
	}

	privateImagePath := "/api/v1/images/character-card-portrait/" + strconv.FormatUint(uint64(privateCard.ID), 10)
	if resp := performCharacterCardRequest(server, http.MethodGet, privateImagePath, "", ""); resp.Code != http.StatusNotFound {
		t.Fatalf("visitor private portrait expected 404, got %d", resp.Code)
	}
	if resp := performCharacterCardRequest(server, http.MethodGet, privateImagePath, newTestToken(t, other), ""); resp.Code != http.StatusNotFound {
		t.Fatalf("other user private portrait expected 404, got %d", resp.Code)
	}
	if resp := performCharacterCardRequest(server, http.MethodGet, privateImagePath, ownerToken, ""); resp.Code != http.StatusOK || resp.Header().Get("Cache-Control") != "private, max-age=3600" {
		t.Fatalf("owner private portrait expected private 200, got code=%d cache=%q", resp.Code, resp.Header().Get("Cache-Control"))
	}

	cardPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(privateCard.ID), 10)
	remoteResp := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{
		"portrait_image_url": "https://evil.example/portrait.png",
	}, ownerToken)
	if remoteResp.Code != http.StatusBadRequest {
		t.Fatalf("remote portrait expected 400, got %d body=%s", remoteResp.Code, remoteResp.Body.String())
	}
	clearResp := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{
		"portrait_image_url": "",
	}, ownerToken)
	if clearResp.Code != http.StatusOK {
		t.Fatalf("clear portrait expected 200, got %d body=%s", clearResp.Code, clearResp.Body.String())
	}
	cleared := decodeCharacterCardResponse(t, clearResp)
	if cleared.PortraitImageURL != "" || cleared.PortraitImageUpdatedAt == nil || !cleared.PortraitImageUpdatedAt.After(oldTime) {
		t.Fatalf("portrait removal did not update DTO version: %+v", cleared)
	}
	var stored model.CharacterCard
	if err := db.First(&stored, privateCard.ID).Error; err != nil || stored.PortraitImage != "" || stored.PortraitImageUpdatedAt == nil || !stored.PortraitImageUpdatedAt.After(oldTime) {
		t.Fatalf("portrait removal not persisted: err=%v card=%+v", err, stored)
	}
	if resp := performCharacterCardRequest(server, http.MethodGet, privateImagePath, ownerToken, ""); resp.Code != http.StatusNotFound {
		t.Fatalf("removed portrait expected 404 despite prior cache, got %d", resp.Code)
	}
}

func TestCharacterCardPortraitUploadArchivesPendingBehindPermissionAwareProxy(t *testing.T) {
	db, server, owner, other := newCharacterCardTestServer(t)
	genericUploadPath := writeCharacterCardTestPNG(t, server, "images/new-portrait.png")
	foreignProtectedPath := writeCharacterCardTestPNG(t, server, fmt.Sprintf("character-cards/%d/portrait/private.png", other.ID))
	stalePendingPath := writeCharacterCardTestPNG(t, server, fmt.Sprintf("character-cards/%d/pending/stale.png", other.ID))
	stalePendingFile := characterCardTestUploadFile(server, stalePendingPath)
	staleTime := time.Now().Add(-characterCardPortraitPendingTTL - time.Hour)
	if err := os.Chtimes(stalePendingFile, staleTime, staleTime); err != nil {
		t.Fatalf("age stale pending portrait: %v", err)
	}
	card := model.CharacterCard{
		UserID: owner.ID, DisplayName: "受保护肖像",
		Status: model.CharacterCardStatusDraft, Visibility: model.CharacterCardVisibilityPrivate,
	}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create card: %v", err)
	}
	token := newTestToken(t, owner)
	cardPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(card.ID), 10)
	genericResp := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{
		"portrait_image_url": genericUploadPath,
	}, token)
	if genericResp.Code != http.StatusBadRequest {
		t.Fatalf("generic public upload expected 400, got %d body=%s", genericResp.Code, genericResp.Body.String())
	}
	foreignResp := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{
		"portrait_image_url": foreignProtectedPath,
	}, token)
	if foreignResp.Code != http.StatusBadRequest {
		t.Fatalf("foreign protected portrait expected 400, got %d body=%s", foreignResp.Code, foreignResp.Body.String())
	}
	dataResp := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{
		"portrait_image_url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(characterCardTestPNGBytes(t, 3, 5)),
	}, token)
	if dataResp.Code != http.StatusBadRequest {
		t.Fatalf("inline portrait expected dedicated upload error, got %d body=%s", dataResp.Code, dataResp.Body.String())
	}

	uploadResp := performCharacterCardPortraitUpload(t, server.router, token, "portrait.png", "application/octet-stream", characterCardTestPNGBytes(t, 3, 5))
	if uploadResp.Code != http.StatusCreated {
		t.Fatalf("portrait upload expected 201, got %d body=%s", uploadResp.Code, uploadResp.Body.String())
	}
	pendingRef := decodeCharacterCardPortraitUploadRef(t, uploadResp)
	wantPendingPrefix := fmt.Sprintf("/uploads/character-cards/%d/pending/", owner.ID)
	if !strings.HasPrefix(pendingRef, wantPendingPrefix) {
		t.Fatalf("portrait upload did not return owner pending reference: %q", pendingRef)
	}
	if direct := performCharacterCardRequest(server, http.MethodGet, pendingRef, token, ""); direct.Code != http.StatusNotFound {
		t.Fatalf("pending portrait backing object expected generic route 404, got %d", direct.Code)
	}
	if _, err := os.Stat(stalePendingFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expired unused pending portrait was not cleaned: %v", err)
	}
	pendingFile := characterCardTestUploadFile(server, pendingRef)
	if _, err := os.Stat(pendingFile); err != nil {
		t.Fatalf("pending portrait missing before archive: %v", err)
	}

	resp := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{
		"portrait_image_url": pendingRef,
	}, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("set portrait expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	dto := decodeCharacterCardResponse(t, resp)
	if !strings.Contains(dto.PortraitImageURL, "/api/v1/images/character-card-portrait/") || strings.Contains(dto.PortraitImageURL, "/uploads/") {
		t.Fatalf("DTO did not use protected image endpoint: %+v", dto)
	}
	var stored model.CharacterCard
	if err := db.First(&stored, card.ID).Error; err != nil {
		t.Fatalf("reload card: %v", err)
	}
	wantPrefix := "/uploads/character-cards/" + strconv.FormatUint(uint64(owner.ID), 10) + "/portrait/"
	if !strings.HasPrefix(stored.PortraitImage, wantPrefix) || stored.PortraitImage == pendingRef {
		t.Fatalf("portrait was not copied into protected storage: %q", stored.PortraitImage)
	}
	if _, err := os.Stat(pendingFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pending portrait was not cleaned after successful archive: %v", err)
	}
	storedPath := stored.PortraitImage
	storedVersion := stored.PortraitImageUpdatedAt
	resave := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{
		"portrait_image_url": dto.PortraitImageURL,
		"summary":            "只修改摘要",
	}, token)
	if resave.Code != http.StatusOK {
		t.Fatalf("resubmit current portrait proxy expected 200, got %d body=%s", resave.Code, resave.Body.String())
	}
	if err := db.First(&stored, card.ID).Error; err != nil {
		t.Fatalf("reload resaved card: %v", err)
	}
	if stored.PortraitImage != storedPath || stored.PortraitImageUpdatedAt == nil || storedVersion == nil || !stored.PortraitImageUpdatedAt.Equal(*storedVersion) {
		t.Fatalf("current proxy URL was treated as a portrait change: before=%q/%v after=%q/%v", storedPath, storedVersion, stored.PortraitImage, stored.PortraitImageUpdatedAt)
	}
	if direct := performCharacterCardRequest(server, http.MethodGet, stored.PortraitImage, token, ""); direct.Code != http.StatusNotFound {
		t.Fatalf("protected backing object expected generic route 404, got %d", direct.Code)
	}
	proxyPath := "/api/v1/images/character-card-portrait/" + strconv.FormatUint(uint64(card.ID), 10)
	if visitor := performCharacterCardRequest(server, http.MethodGet, proxyPath, "", ""); visitor.Code != http.StatusNotFound {
		t.Fatalf("visitor draft portrait expected 404, got %d", visitor.Code)
	}
	if ownerResp := performCharacterCardRequest(server, http.MethodGet, proxyPath, token, ""); ownerResp.Code != http.StatusOK {
		t.Fatalf("owner protected portrait expected 200, got %d body=%s", ownerResp.Code, ownerResp.Body.String())
	}

	retryUpload := performCharacterCardPortraitUpload(t, server.router, token, "retry.png", "image/png", characterCardTestPNGBytes(t, 3, 5))
	if retryUpload.Code != http.StatusCreated {
		t.Fatalf("retry portrait upload expected 201, got %d body=%s", retryUpload.Code, retryUpload.Body.String())
	}
	retryRef := decodeCharacterCardPortraitUploadRef(t, retryUpload)
	failedSave := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{
		"portrait_image_url": retryRef,
		"status":             "archived",
	}, token)
	if failedSave.Code != http.StatusBadRequest {
		t.Fatalf("invalid save expected 400, got %d body=%s", failedSave.Code, failedSave.Body.String())
	}
	if _, err := os.Stat(characterCardTestUploadFile(server, retryRef)); err != nil {
		t.Fatalf("failed save should retain pending portrait for retry: %v", err)
	}
}

func TestCharacterCardPortraitUploadValidatesAuthBytesMIMEAndDimensions(t *testing.T) {
	_, server, owner, _ := newCharacterCardTestServer(t)
	validPNG := characterCardTestPNGBytes(t, 3, 5)

	unauthorized := performCharacterCardPortraitUpload(t, server.router, "", "portrait.png", "image/png", validPNG)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated portrait upload expected 401, got %d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
	token := newTestToken(t, owner)
	tests := []struct {
		name        string
		contentType string
		data        []byte
		wantCode    int
	}{
		{name: "corrupt bytes", contentType: "image/png", data: []byte("not an image"), wantCode: http.StatusBadRequest},
		{name: "MIME mismatch", contentType: "image/jpeg", data: validPNG, wantCode: http.StatusBadRequest},
		{name: "oversized bytes", contentType: "application/octet-stream", data: make([]byte, characterCardPortraitMaxBytes+1), wantCode: http.StatusRequestEntityTooLarge},
		{name: "oversized dimensions", contentType: "image/png", data: characterCardTestPNGBytes(t, characterCardPortraitMaxSide+1, 1), wantCode: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp := performCharacterCardPortraitUpload(t, server.router, token, "portrait.png", test.contentType, test.data)
			if resp.Code != test.wantCode {
				t.Fatalf("invalid portrait expected %d, got %d body=%s", test.wantCode, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestCharacterCardPortraitGalleryModerationKeepsApprovedSnapshotAssets(t *testing.T) {
	db, server, owner, moderator := newCharacterCardTestServer(t)
	if err := db.Model(&model.User{}).Where("id = ?", moderator.ID).Update("role", "moderator").Error; err != nil {
		t.Fatalf("promote moderator: %v", err)
	}
	ownerToken := newTestToken(t, owner)
	moderatorToken := newTestToken(t, moderator)
	createResp := performRequest(server.router, http.MethodPost, "/api/v1/character-cards", map[string]interface{}{"source_type": "blank"}, ownerToken)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create card expected 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}
	card := decodeCharacterCardResponse(t, createResp)
	cardPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(card.ID), 10)

	addPortrait := func(name string, width int) characterCardDTO {
		t.Helper()
		upload := performCharacterCardPortraitUpload(t, server.router, ownerToken, name, "image/png", characterCardTestPNGBytes(t, width, 5))
		if upload.Code != http.StatusCreated {
			t.Fatalf("upload portrait expected 201, got %d body=%s", upload.Code, upload.Body.String())
		}
		resp := performRequest(server.router, http.MethodPost, cardPath+"/portraits", map[string]interface{}{
			"image_ref": decodeCharacterCardPortraitUploadRef(t, upload),
		}, ownerToken)
		if resp.Code != http.StatusCreated {
			t.Fatalf("add gallery portrait expected 201, got %d body=%s", resp.Code, resp.Body.String())
		}
		return decodeCharacterCardResponse(t, resp)
	}

	card = addPortrait("approved.png", 4)
	if len(card.Portraits) != 1 || !card.Portraits[0].IsCover || card.Portraits[0].ImageURL == "" || card.PortraitImageURL == "" {
		t.Fatalf("first gallery image did not become compatible cover: %+v", card)
	}
	firstPortrait := card.Portraits[0]
	var firstStored model.CharacterCardPortrait
	if err := db.First(&firstStored, firstPortrait.ID).Error; err != nil {
		t.Fatalf("load first portrait: %v", err)
	}
	firstFile := characterCardTestUploadFile(server, firstStored.Image)

	publish := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{
		"display_name": "已审版本", "status": model.CharacterCardStatusPublished, "visibility": model.CharacterCardVisibilityPublic,
	}, ownerToken)
	if publish.Code != http.StatusOK || decodeCharacterCardResponse(t, publish).ReviewStatus != model.CharacterCardReviewPending {
		t.Fatalf("public save expected pending: code=%d body=%s", publish.Code, publish.Body.String())
	}
	if visitor := performCharacterCardRequest(server, http.MethodGet, firstPortrait.ImageURL, "", ""); visitor.Code != http.StatusNotFound {
		t.Fatalf("pending gallery image leaked to visitor: %d", visitor.Code)
	}
	if moderatorView := performCharacterCardRequest(server, http.MethodGet, firstPortrait.ImageURL, moderatorToken, ""); moderatorView.Code != http.StatusOK {
		t.Fatalf("moderator could not inspect pending gallery image: %d body=%s", moderatorView.Code, moderatorView.Body.String())
	}
	queue := performRequest(server.router, http.MethodGet, "/api/v1/moderator/review/character-cards", nil, moderatorToken)
	if queue.Code != http.StatusOK || !strings.Contains(queue.Body.String(), `"display_name":"已审版本"`) {
		t.Fatalf("moderator queue missing pending card: code=%d body=%s", queue.Code, queue.Body.String())
	}
	approve := performRequest(server.router, http.MethodPost, "/api/v1/moderator/review/character-cards/"+strconv.FormatUint(uint64(card.ID), 10), map[string]interface{}{"action": "approve"}, moderatorToken)
	if approve.Code != http.StatusOK {
		t.Fatalf("first approval expected 200, got %d body=%s", approve.Code, approve.Body.String())
	}
	approvedImage := performCharacterCardRequest(server, http.MethodGet, firstPortrait.ImageURL, "", "")
	if approvedImage.Code != http.StatusOK || approvedImage.Header().Get("ETag") == "" || approvedImage.Header().Get("Cache-Control") != "private, max-age=31536000, immutable" {
		t.Fatalf("approved gallery image cache contract mismatch: code=%d headers=%v", approvedImage.Code, approvedImage.Header())
	}
	if cached := performCharacterCardRequest(server, http.MethodGet, firstPortrait.ImageURL, "", approvedImage.Header().Get("ETag")); cached.Code != http.StatusNotModified {
		t.Fatalf("gallery image ETag expected 304, got %d headers=%v", cached.Code, cached.Header())
	}

	card = addPortrait("pending.png", 7)
	if len(card.Portraits) != 2 || card.ReviewStatus != model.CharacterCardReviewPending {
		t.Fatalf("editing approved gallery should retain working copy and enter pending: %+v", card)
	}
	secondPortrait := card.Portraits[1]
	coverResp := performRequest(server.router, http.MethodPut, cardPath+"/portraits/"+strconv.FormatUint(uint64(secondPortrait.ID), 10)+"/cover", nil, ownerToken)
	if coverResp.Code != http.StatusOK {
		t.Fatalf("set gallery cover expected 200, got %d body=%s", coverResp.Code, coverResp.Body.String())
	}
	coverCard := decodeCharacterCardResponse(t, coverResp)
	if len(coverCard.Portraits) != 2 || coverCard.Portraits[0].ID != secondPortrait.ID || !coverCard.Portraits[0].IsCover {
		t.Fatalf("set cover did not reorder gallery: %+v", coverCard.Portraits)
	}
	orderResp := performRequest(server.router, http.MethodPut, cardPath+"/portraits/order", map[string]interface{}{
		"portrait_ids": []uint{firstPortrait.ID, secondPortrait.ID},
	}, ownerToken)
	if orderResp.Code != http.StatusOK || decodeCharacterCardResponse(t, orderResp).Portraits[0].ID != firstPortrait.ID {
		t.Fatalf("explicit gallery order expected first portrait restored: code=%d body=%s", orderResp.Code, orderResp.Body.String())
	}
	if update := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{"display_name": "未审版本"}, ownerToken); update.Code != http.StatusOK {
		t.Fatalf("update pending card expected 200, got %d body=%s", update.Code, update.Body.String())
	}
	deleteFirst := performRequest(server.router, http.MethodDelete, cardPath+"/portraits/"+strconv.FormatUint(uint64(firstPortrait.ID), 10), nil, ownerToken)
	if deleteFirst.Code != http.StatusOK {
		t.Fatalf("delete approved working portrait expected 200, got %d body=%s", deleteFirst.Code, deleteFirst.Body.String())
	}
	if _, err := os.Stat(firstFile); err != nil {
		t.Fatalf("working-copy deletion destroyed asset still referenced by approved snapshot: %v", err)
	}
	publicDuringReview := performRequest(server.router, http.MethodGet, cardPath, nil, "")
	if publicDuringReview.Code != http.StatusOK {
		t.Fatalf("existing approved snapshot should remain public during review: %d body=%s", publicDuringReview.Code, publicDuringReview.Body.String())
	}
	approvedView := decodeCharacterCardResponse(t, publicDuringReview)
	if approvedView.DisplayName != "已审版本" || len(approvedView.Portraits) != 1 || approvedView.Portraits[0].ID != firstPortrait.ID {
		t.Fatalf("public reader saw unreviewed working copy: %+v", approvedView)
	}
	if visitor := performCharacterCardRequest(server, http.MethodGet, firstPortrait.ImageURL, "", ""); visitor.Code != http.StatusOK {
		t.Fatalf("old approved image stopped serving during pending edit: %d", visitor.Code)
	}
	if visitor := performCharacterCardRequest(server, http.MethodGet, secondPortrait.ImageURL, "", ""); visitor.Code != http.StatusNotFound {
		t.Fatalf("new pending image leaked publicly: %d", visitor.Code)
	}
	if moderatorView := performCharacterCardRequest(server, http.MethodGet, secondPortrait.ImageURL, moderatorToken, ""); moderatorView.Code != http.StatusOK {
		t.Fatalf("moderator could not inspect new pending image: %d", moderatorView.Code)
	}

	reject := performRequest(server.router, http.MethodPost, "/api/v1/moderator/review/character-cards/"+strconv.FormatUint(uint64(card.ID), 10), map[string]interface{}{
		"action": "reject", "comment": "请调整",
	}, moderatorToken)
	if reject.Code != http.StatusOK || decodeCharacterCardResponse(t, reject).ReviewStatus != model.CharacterCardReviewRejected {
		t.Fatalf("reject expected rejected working copy: code=%d body=%s", reject.Code, reject.Body.String())
	}
	if visitor := performCharacterCardRequest(server, http.MethodGet, firstPortrait.ImageURL, "", ""); visitor.Code != http.StatusOK {
		t.Fatalf("rejection should retain old approved image: %d", visitor.Code)
	}

	resubmit := performRequest(server.router, http.MethodPut, cardPath, map[string]interface{}{"summary": "已修正"}, ownerToken)
	if resubmit.Code != http.StatusOK || decodeCharacterCardResponse(t, resubmit).ReviewStatus != model.CharacterCardReviewPending {
		t.Fatalf("edit after rejection should resubmit: code=%d body=%s", resubmit.Code, resubmit.Body.String())
	}
	approve = performRequest(server.router, http.MethodPost, "/api/v1/moderator/review/character-cards/"+strconv.FormatUint(uint64(card.ID), 10), map[string]interface{}{"action": "approve"}, moderatorToken)
	if approve.Code != http.StatusOK {
		t.Fatalf("replacement approval expected 200, got %d body=%s", approve.Code, approve.Body.String())
	}
	finalPublic := performRequest(server.router, http.MethodGet, cardPath, nil, "")
	finalCard := decodeCharacterCardResponse(t, finalPublic)
	if finalPublic.Code != http.StatusOK || finalCard.DisplayName != "未审版本" || len(finalCard.Portraits) != 1 || finalCard.Portraits[0].ID != secondPortrait.ID || !finalCard.Portraits[0].IsCover {
		t.Fatalf("approved replacement snapshot mismatch: code=%d card=%+v", finalPublic.Code, finalCard)
	}
	if oldImage := performCharacterCardRequest(server, http.MethodGet, firstPortrait.ImageURL, "", ""); oldImage.Code != http.StatusNotFound {
		t.Fatalf("old portrait proxy remained public after replacement approval: %d", oldImage.Code)
	}
	if _, err := os.Stat(firstFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old snapshot asset was not cleaned after replacement approval: %v", err)
	}
}

func TestCharacterCardImpressionImagesUsePrivateArchiveProxyAndCleanup(t *testing.T) {
	db, server, owner, other := newCharacterCardTestServer(t)
	ownerToken := newTestToken(t, owner)
	otherToken := newTestToken(t, other)
	createCard := func(token string) characterCardDTO {
		t.Helper()
		resp := performRequest(server.router, http.MethodPost, "/api/v1/character-cards", map[string]interface{}{"source_type": "blank"}, token)
		if resp.Code != http.StatusCreated {
			t.Fatalf("blank card create expected 201, got %d body=%s", resp.Code, resp.Body.String())
		}
		return decodeCharacterCardResponse(t, resp)
	}
	ownerCard := createCard(ownerToken)
	otherCard := createCard(otherToken)
	ownerPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(ownerCard.ID), 10)
	otherPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(otherCard.ID), 10)
	validImageBytes := characterCardTestPNGBytes(t, 3, 5)
	if resp := performCharacterCardImpressionUpload(t, server.router, "", characterCardImpressionKindIcon, "icon.png", "image/png", validImageBytes); resp.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated impression upload expected 401, got %d body=%s", resp.Code, resp.Body.String())
	}
	if resp := performCharacterCardImpressionUpload(t, server.router, ownerToken, "portrait", "icon.png", "image/png", validImageBytes); resp.Code != http.StatusBadRequest {
		t.Fatalf("invalid impression kind expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	inlineRequests := emptyCharacterCardImpressionRequests()
	inlineRequests[0].IconImageURL = "data:image/png;base64," + base64.StdEncoding.EncodeToString(characterCardTestPNGBytes(t, 3, 5))
	if resp := performRequest(server.router, http.MethodPut, ownerPath, map[string]interface{}{"impressions": inlineRequests}, ownerToken); resp.Code != http.StatusBadRequest {
		t.Fatalf("inline impression image expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	iconUpload := performCharacterCardImpressionUpload(t, server.router, ownerToken, characterCardImpressionKindIcon, "icon.png", "image/png", characterCardTestPNGBytes(t, 4, 4))
	imageUpload := performCharacterCardImpressionUpload(t, server.router, ownerToken, characterCardImpressionKindImage, "scene.png", "image/png", characterCardTestPNGBytes(t, 8, 5))
	if iconUpload.Code != http.StatusCreated || imageUpload.Code != http.StatusCreated {
		t.Fatalf("impression uploads failed: icon=%d/%s image=%d/%s", iconUpload.Code, iconUpload.Body.String(), imageUpload.Code, imageUpload.Body.String())
	}
	iconRef := decodeCharacterCardImpressionUploadRef(t, iconUpload)
	imageRef := decodeCharacterCardImpressionUploadRef(t, imageUpload)
	if !strings.Contains(iconRef, "/pending/icon/") || !strings.Contains(imageRef, "/pending/image/") {
		t.Fatalf("kind-specific pending references missing: icon=%q image=%q", iconRef, imageRef)
	}
	if resp := performRequest(server.router, http.MethodPut, ownerPath, map[string]interface{}{"portrait_image_url": iconRef}, ownerToken); resp.Code != http.StatusBadRequest {
		t.Fatalf("impression icon pending reference used as portrait expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
	if _, err := os.Stat(characterCardTestUploadFile(server, iconRef)); err != nil {
		t.Fatalf("rejected impression icon reference was consumed as portrait: %v", err)
	}

	portraitUpload := performCharacterCardPortraitUpload(t, server.router, ownerToken, "portrait.png", "image/png", validImageBytes)
	if portraitUpload.Code != http.StatusCreated {
		t.Fatalf("portrait upload for cross-type check expected 201, got %d body=%s", portraitUpload.Code, portraitUpload.Body.String())
	}
	portraitRef := decodeCharacterCardPortraitUploadRef(t, portraitUpload)
	portraitAsIcon := emptyCharacterCardImpressionRequests()
	portraitAsIcon[0].IconImageURL = portraitRef
	if resp := performRequest(server.router, http.MethodPut, ownerPath, map[string]interface{}{"impressions": portraitAsIcon}, ownerToken); resp.Code != http.StatusBadRequest {
		t.Fatalf("portrait pending reference used as impression icon expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
	if _, err := os.Stat(characterCardTestUploadFile(server, portraitRef)); err != nil {
		t.Fatalf("rejected portrait reference was consumed as impression icon: %v", err)
	}

	foreignRequests := emptyCharacterCardImpressionRequests()
	foreignRequests[0].IconImageURL = iconRef
	if resp := performRequest(server.router, http.MethodPut, otherPath, map[string]interface{}{"impressions": foreignRequests}, otherToken); resp.Code != http.StatusBadRequest {
		t.Fatalf("cross-user pending reference expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
	if _, err := os.Stat(characterCardTestUploadFile(server, iconRef)); err != nil {
		t.Fatalf("rejected cross-user update consumed owner's pending image: %v", err)
	}
	mismatchedKind := emptyCharacterCardImpressionRequests()
	mismatchedKind[0].ImageURL = iconRef
	if resp := performRequest(server.router, http.MethodPut, ownerPath, map[string]interface{}{"impressions": mismatchedKind}, ownerToken); resp.Code != http.StatusBadRequest {
		t.Fatalf("icon pending reference used as image expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	requests := emptyCharacterCardImpressionRequests()
	requests[0].Active = true
	requests[0].Title = "自定义图片印象"
	requests[0].IconImageURL = iconRef
	requests[0].ImageURL = imageRef
	updateResp := performRequest(server.router, http.MethodPut, ownerPath, map[string]interface{}{"impressions": requests}, ownerToken)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("archive impression images expected 200, got %d body=%s", updateResp.Code, updateResp.Body.String())
	}
	updated := decodeCharacterCardResponse(t, updateResp)
	assertFiveCharacterCardImpressionSlots(t, updated.Impressions)
	slot := updated.Impressions[0]
	if !strings.Contains(slot.IconImageURL, "/images/character-card-impression-icon/") || !strings.Contains(slot.ImageURL, "/images/character-card-impression-image/") || slot.IconImageUpdatedAt == nil || slot.ImageUpdatedAt == nil {
		t.Fatalf("impression DTO did not expose versioned proxy URLs: %+v", slot)
	}
	for _, pending := range []string{iconRef, imageRef} {
		if _, err := os.Stat(characterCardTestUploadFile(server, pending)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("successful archive did not remove pending file %q: %v", pending, err)
		}
	}
	var stored model.CharacterCardImpression
	if err := db.Where("character_card_id = ? AND slot = ?", ownerCard.ID, 1).First(&stored).Error; err != nil {
		t.Fatalf("load stored impression: %v", err)
	}
	if !strings.Contains(stored.IconImage, "/impression-icon/") || !strings.Contains(stored.Image, "/impression-image/") {
		t.Fatalf("images were not copied into kind-specific private archive: %+v", stored)
	}
	iconFile := characterCardTestUploadFile(server, stored.IconImage)
	imageFile := characterCardTestUploadFile(server, stored.Image)
	for _, backing := range []string{stored.IconImage, stored.Image} {
		if resp := performCharacterCardRequest(server, http.MethodGet, backing, ownerToken, ""); resp.Code != http.StatusNotFound {
			t.Fatalf("protected backing object expected 404, got %d for %s", resp.Code, backing)
		}
	}

	ownerImage := performCharacterCardRequest(server, http.MethodGet, slot.ImageURL, ownerToken, "")
	if ownerImage.Code != http.StatusOK || ownerImage.Header().Get("ETag") == "" || ownerImage.Header().Get("Cache-Control") != "private, max-age=31536000, immutable" {
		t.Fatalf("owner proxy response mismatch: code=%d headers=%v body=%s", ownerImage.Code, ownerImage.Header(), ownerImage.Body.String())
	}
	etag := ownerImage.Header().Get("ETag")
	if cached := performCharacterCardRequest(server, http.MethodGet, slot.ImageURL, ownerToken, etag); cached.Code != http.StatusNotModified || cached.Header().Get("ETag") != etag {
		t.Fatalf("impression proxy ETag expected 304, got code=%d headers=%v", cached.Code, cached.Header())
	}
	if visitor := performCharacterCardRequest(server, http.MethodGet, slot.ImageURL, "", ""); visitor.Code != http.StatusNotFound {
		t.Fatalf("visitor draft impression image expected 404, got %d", visitor.Code)
	}

	publishRequests := impressionRequestsFromDTOs(updated.Impressions)
	publishResp := performRequest(server.router, http.MethodPut, ownerPath, map[string]interface{}{
		"display_name": "公开图片卡", "status": model.CharacterCardStatusPublished,
		"visibility": model.CharacterCardVisibilityPublic, "impressions": publishRequests,
	}, ownerToken)
	if publishResp.Code != http.StatusOK {
		t.Fatalf("publish image card expected 200, got %d body=%s", publishResp.Code, publishResp.Body.String())
	}
	published := decodeCharacterCardResponse(t, publishResp)
	publicImageURL := published.Impressions[0].ImageURL
	if visitor := performCharacterCardRequest(server, http.MethodGet, publicImageURL, "", ""); visitor.Code != http.StatusOK || strings.Contains(visitor.Header().Get("Cache-Control"), "public") {
		t.Fatalf("active public impression image should be readable only through private cache: code=%d cache=%q", visitor.Code, visitor.Header().Get("Cache-Control"))
	}

	inactiveRequests := impressionRequestsFromDTOs(published.Impressions)
	inactiveRequests[0].Active = false
	inactiveResp := performRequest(server.router, http.MethodPut, ownerPath, map[string]interface{}{"impressions": inactiveRequests}, ownerToken)
	if inactiveResp.Code != http.StatusOK {
		t.Fatalf("disable impression expected 200, got %d body=%s", inactiveResp.Code, inactiveResp.Body.String())
	}
	inactive := decodeCharacterCardResponse(t, inactiveResp)
	if inactive.Impressions[0].ImageUpdatedAt == nil || published.Impressions[0].ImageUpdatedAt == nil || !inactive.Impressions[0].ImageUpdatedAt.After(*published.Impressions[0].ImageUpdatedAt) {
		t.Fatalf("active transition did not rotate image cache version: before=%v after=%v", published.Impressions[0].ImageUpdatedAt, inactive.Impressions[0].ImageUpdatedAt)
	}
	if visitor := performCharacterCardRequest(server, http.MethodGet, publicImageURL, "", ""); visitor.Code != http.StatusNotFound {
		t.Fatalf("disabled impression image remained publicly readable: %d", visitor.Code)
	}
	if ownerView := performCharacterCardRequest(server, http.MethodGet, inactive.Impressions[0].ImageURL, ownerToken, ""); ownerView.Code != http.StatusOK {
		t.Fatalf("owner should retain access to inactive impression image: %d", ownerView.Code)
	}

	deleteResp := performRequest(server.router, http.MethodDelete, ownerPath, nil, ownerToken)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete card expected 200, got %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	for _, file := range []string{iconFile, imageFile} {
		if _, err := os.Stat(file); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("card deletion did not remove impression image %q: %v", file, err)
		}
	}
	var impressionCount int64
	if err := db.Model(&model.CharacterCardImpression{}).Where("character_card_id = ?", ownerCard.ID).Count(&impressionCount).Error; err != nil || impressionCount != 0 {
		t.Fatalf("card deletion left impression rows: count=%d err=%v", impressionCount, err)
	}
}

func TestCharacterCardRejectsInvalidStateAndUnsafeRichText(t *testing.T) {
	db, server, owner, _ := newCharacterCardTestServer(t)
	card := model.CharacterCard{UserID: owner.ID, Status: model.CharacterCardStatusDraft, Visibility: model.CharacterCardVisibilityPrivate}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create card: %v", err)
	}
	token := newTestToken(t, owner)
	path := "/api/v1/character-cards/" + strconv.FormatUint(uint64(card.ID), 10)

	tests := []map[string]interface{}{
		{"status": "archived"},
		{"visibility": "friends"},
		{"background_story": `<p onclick="steal()">unsafe</p>`},
		{"other_content": `<script>alert(1)</script>`},
		{"first_impression": `<a href="java\nscript:alert(1)">unsafe</a>`},
		{"background_story": `<p style="position: fixed; inset: 0">unsafe</p>`},
		{"other_content": `<p style="text-align: center; background-image: url(javascript:alert(1))">unsafe</p>`},
		{"first_impression": `<p style="text-align: var(--alignment)">unsafe</p>`},
		{"class_color": "ABCDEF", "name_color": "123456"},
		{"display_name": strings.Repeat("人", 257)},
	}
	for _, body := range tests {
		resp := performRequest(server.router, http.MethodPut, path, body, token)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("invalid body %+v expected 400, got %d body=%s", body, resp.Code, resp.Body.String())
		}
	}
	var stored model.CharacterCard
	if err := db.First(&stored, card.ID).Error; err != nil {
		t.Fatalf("reload card: %v", err)
	}
	if stored.Status != model.CharacterCardStatusDraft || stored.Visibility != model.CharacterCardVisibilityPrivate || stored.BackgroundStory != "" {
		t.Fatalf("invalid update was persisted: %+v", stored)
	}

	safeRichText := `<h2 style="text-align: center">标题</h2><p style="text-align: justify;">正文</p>`
	resp := performRequest(server.router, http.MethodPut, path, map[string]interface{}{"background_story": safeRichText}, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("Tiptap text alignment expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if err := db.First(&stored, card.ID).Error; err != nil {
		t.Fatalf("reload safely updated card: %v", err)
	}
	if stored.BackgroundStory != safeRichText {
		t.Fatalf("safe rich text was not persisted: %q", stored.BackgroundStory)
	}
	colorResp := performRequest(server.router, http.MethodPut, path, map[string]interface{}{"class_color": "#aabbcc"}, token)
	if colorResp.Code != http.StatusOK {
		t.Fatalf("class_color compatibility update expected 200, got %d body=%s", colorResp.Code, colorResp.Body.String())
	}
	colored := decodeCharacterCardResponse(t, colorResp)
	if colored.ClassColor != "AABBCC" || colored.NameColor != "AABBCC" {
		t.Fatalf("single TRP3 CH color was not mirrored to both fields: %+v", colored)
	}
}

func TestCharacterCardShareRequiresApprovedPublicVersion(t *testing.T) {
	db, server, owner, _ := newCharacterCardTestServer(t)
	cards := []model.CharacterCard{
		{
			UserID: owner.ID, DisplayName: "Approved", Status: model.CharacterCardStatusPublished,
			Visibility: model.CharacterCardVisibilityPublic, ReviewStatus: model.CharacterCardReviewApproved,
		},
		{
			UserID: owner.ID, DisplayName: "Pending", Status: model.CharacterCardStatusPublished,
			Visibility: model.CharacterCardVisibilityPublic, ReviewStatus: model.CharacterCardReviewPending,
		},
		{
			UserID: owner.ID, DisplayName: "Private", Status: model.CharacterCardStatusPublished,
			Visibility: model.CharacterCardVisibilityPrivate, ReviewStatus: model.CharacterCardReviewApproved,
		},
	}
	if err := db.Create(&cards).Error; err != nil {
		t.Fatalf("create share cards: %v", err)
	}

	approvedPath := "/api/v1/character-cards/" + strconv.FormatUint(uint64(cards[0].ID), 10) + "/share"
	approved := performRequest(server.router, http.MethodGet, approvedPath, nil, "")
	if approved.Code != http.StatusOK {
		t.Fatalf("approved public share expected 200, got %d body=%s", approved.Code, approved.Body.String())
	}
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(approved.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode share response: %v", err)
	}
	wantPath := "/character-cards/" + strconv.FormatUint(uint64(cards[0].ID), 10)
	if payload.Path != wantPath {
		t.Fatalf("expected share path %q, got %q", wantPath, payload.Path)
	}

	for _, card := range cards[1:] {
		path := "/api/v1/character-cards/" + strconv.FormatUint(uint64(card.ID), 10) + "/share"
		resp := performRequest(server.router, http.MethodGet, path, nil, "")
		if resp.Code != http.StatusConflict {
			t.Fatalf("ineligible card %d expected 409, got %d body=%s", card.ID, resp.Code, resp.Body.String())
		}
	}
}

func newCharacterCardTestServer(t *testing.T) (*gorm.DB, *Server, model.User, model.User) {
	t.Helper()
	db := testutil.NewTestDB(t,
		&model.User{}, &model.AccountBackup{}, &model.AccountBackupVersion{}, &model.Character{},
		&model.CharacterCard{}, &model.CharacterCardPortrait{}, &model.CharacterCardImpression{}, &model.CharacterCardPublication{},
	)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get test database handle: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	database.DB = db
	owner := model.User{Username: "card-owner", Email: "card-owner@example.com", PassHash: "hash"}
	other := model.User{Username: "card-other", Email: "card-other@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&owner, &other}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	return db, newTestServer(t, db), owner, other
}

func characterCardProfilesData(profile string) string {
	return `{"profile-exact":` + profile + `}`
}

func decodeCharacterCardResponse(t *testing.T, resp *httptest.ResponseRecorder) characterCardDTO {
	t.Helper()
	var payload struct {
		Card characterCardDTO `json:"character_card"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode character card response: %v body=%s", err, resp.Body.String())
	}
	return payload.Card
}

func assertFiveCharacterCardImpressionSlots(t *testing.T, impressions []characterCardImpressionDTO) {
	t.Helper()
	if len(impressions) != characterCardImpressionSlotCount {
		t.Fatalf("expected five impression slots, got %+v", impressions)
	}
	for index, impression := range impressions {
		if impression.Slot != uint8(index+1) {
			t.Fatalf("impression slots are not fixed in 1..5 order: %+v", impressions)
		}
	}
}

func impressionRequestsFromDTOs(impressions []characterCardImpressionDTO) []characterCardImpressionRequest {
	requests := make([]characterCardImpressionRequest, 0, len(impressions))
	for _, impression := range impressions {
		requests = append(requests, characterCardImpressionRequest{
			Slot:         impression.Slot,
			Active:       impression.Active,
			Title:        impression.Title,
			Text:         impression.Text,
			TRP3Icon:     impression.TRP3Icon,
			IconImageURL: impression.IconImageURL,
			ImageURL:     impression.ImageURL,
		})
	}
	return requests
}

func emptyCharacterCardImpressionRequests() []characterCardImpressionRequest {
	requests := make([]characterCardImpressionRequest, 0, characterCardImpressionSlotCount)
	for slot := 1; slot <= characterCardImpressionSlotCount; slot++ {
		requests = append(requests, characterCardImpressionRequest{Slot: uint8(slot)})
	}
	return requests
}

func writeCharacterCardTestPNG(t *testing.T, server *Server, relative string) string {
	t.Helper()
	target := filepath.Join(server.cfg.Storage.Path, "uploads", filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatalf("create image dir: %v", err)
	}
	if err := os.WriteFile(target, characterCardTestPNGBytes(t, 3, 5), 0644); err != nil {
		t.Fatalf("write png: %v", err)
	}
	return "/uploads/" + strings.TrimPrefix(filepath.ToSlash(relative), "/")
}

func characterCardTestPNGBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	if width > 1 && height > 1 {
		img.Set(1, 1, color.RGBA{R: 180, G: 115, B: 51, A: 255})
	}
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return data.Bytes()
}

func performCharacterCardPortraitUpload(t *testing.T, router http.Handler, token, filename, contentType string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename="%s"`, filepath.Base(filename)))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create portrait upload part: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write portrait upload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close portrait upload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/character-card-portrait", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func performCharacterCardImpressionUpload(t *testing.T, router http.Handler, token, kind, filename, contentType string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("kind", kind); err != nil {
		t.Fatalf("write impression kind: %v", err)
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename="%s"`, filepath.Base(filename)))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create impression image upload part: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write impression image: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close impression image upload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/character-card-impression-image", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func decodeCharacterCardPortraitUploadRef(t *testing.T, resp *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Reference string `json:"portrait_image_ref"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil || payload.Reference == "" {
		t.Fatalf("decode portrait upload response: err=%v body=%s", err, resp.Body.String())
	}
	return payload.Reference
}

func decodeCharacterCardImpressionUploadRef(t *testing.T, resp *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Reference string `json:"image_ref"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil || payload.Reference == "" {
		t.Fatalf("decode impression image upload response: err=%v body=%s", err, resp.Body.String())
	}
	return payload.Reference
}

func characterCardTestUploadFile(server *Server, reference string) string {
	key := uploadsKeyFromPath(reference)
	return filepath.Join(server.cfg.Storage.Path, "uploads", filepath.FromSlash(key))
}

func assertCharacterCardListOmitsRichText(t *testing.T, body string) {
	t.Helper()
	for _, field := range []string{"background_story", "first_impression", "other_content"} {
		if strings.Contains(body, `"`+field+`"`) {
			t.Fatalf("character card list leaked full rich-text field %q: %s", field, body)
		}
	}
}

func performCharacterCardRequest(server *Server, method, requestPath, token, etag string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, requestPath, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp := httptest.NewRecorder()
	server.router.ServeHTTP(resp, req)
	return resp
}
