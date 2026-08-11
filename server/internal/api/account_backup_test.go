package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestAccountBackupAllowsSameAccountIDAcrossUsers(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.AccountBackup{}, &model.AccountBackupVersion{})
	database.DB = db

	userOne := model.User{Username: "backup-user-1", Email: "backup-user-1@example.com", PassHash: "hash"}
	userTwo := model.User{Username: "backup-user-2", Email: "backup-user-2@example.com", PassHash: "hash"}
	if err := db.Create(&userOne).Error; err != nil {
		t.Fatalf("create user one: %v", err)
	}
	if err := db.Create(&userTwo).Error; err != nil {
		t.Fatalf("create user two: %v", err)
	}

	server := newTestServer(t, db)
	tokenOne := newTestToken(t, userOne)
	tokenTwo := newTestToken(t, userTwo)

	body := map[string]any{
		"account_id":     "563986541#1",
		"profiles_data":  `{"profiles":[]}`,
		"profiles_count": 0,
		"checksum":       "sum-1",
	}

	resp := performRequest(server.router, http.MethodPost, "/api/v1/account-backups", body, tokenOne)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected user one create 201, got %d body=%s", resp.Code, resp.Body.String())
	}

	body["checksum"] = "sum-2"
	resp = performRequest(server.router, http.MethodPost, "/api/v1/account-backups", body, tokenTwo)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected user two create 201, got %d body=%s", resp.Code, resp.Body.String())
	}

	var backups []model.AccountBackup
	if err := db.Order("user_id asc").Find(&backups).Error; err != nil {
		t.Fatalf("load backups: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(backups))
	}
	if backups[0].UserID == backups[1].UserID {
		t.Fatalf("expected backups to belong to different users: %+v", backups)
	}
}

func TestAccountBackupUniqueIndexIncludesUserID(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.AccountBackup{})

	type indexRow struct {
		Seq     int
		Name    string
		Unique  int
		Origin  string
		Partial int
	}
	var indexes []indexRow
	if err := db.Raw("PRAGMA index_list(account_backups)").Scan(&indexes).Error; err != nil {
		t.Fatalf("list indexes: %v", err)
	}

	var targetIndex string
	for _, idx := range indexes {
		if idx.Name == "idx_user_account" {
			if idx.Unique != 1 {
				t.Fatalf("expected idx_user_account to be unique, got %+v", idx)
			}
			targetIndex = idx.Name
			break
		}
	}
	if targetIndex == "" {
		t.Fatal("idx_user_account not found")
	}

	type columnRow struct {
		Seqno int
		Cid   int
		Name  string
	}
	var columns []columnRow
	if err := db.Raw("PRAGMA index_info(idx_user_account)").Scan(&columns).Error; err != nil {
		t.Fatalf("index info: %v", err)
	}
	if len(columns) != 2 || columns[0].Name != "user_id" || columns[1].Name != "account_id" {
		raw, _ := json.Marshal(columns)
		t.Fatalf("unexpected index columns: %s", string(raw))
	}
}

func TestAccountBackupSnapshotReasonIsValidatedAndPersisted(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.AccountBackup{}, &model.AccountBackupVersion{})
	database.DB = db

	user := model.User{Username: "backup-reason-user", Email: "backup-reason@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	server := newTestServer(t, db)
	token := newTestToken(t, user)
	body := map[string]any{
		"account_id":     "reason-account",
		"profiles_data":  `{"profile-one":{"profileName":"First"}}`,
		"profiles_count": 1,
		"tools_count":    2,
		"checksum":       "sum-1",
	}
	if resp := performRequest(server.router, http.MethodPost, "/api/v1/account-backups", body, token); resp.Code != http.StatusCreated {
		t.Fatalf("create backup expected 201, got %d body=%s", resp.Code, resp.Body.String())
	}

	body["checksum"] = "sum-2"
	body["snapshot_reason"] = "before_manual_backup"
	if resp := performRequest(server.router, http.MethodPost, "/api/v1/account-backups", body, token); resp.Code != http.StatusOK {
		t.Fatalf("manual update expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	var version model.AccountBackupVersion
	if err := db.First(&version).Error; err != nil {
		t.Fatalf("load version: %v", err)
	}
	if version.ChangeLog != "before_manual_backup" || version.ProfilesCount != 1 || version.ToolsCount != 2 {
		t.Fatalf("snapshot metadata mismatch: %+v", version)
	}

	body["checksum"] = "sum-3"
	body["snapshot_reason"] = "client_defined_reason"
	resp := performRequest(server.router, http.MethodPost, "/api/v1/account-backups", body, token)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("invalid reason expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
	var backup model.AccountBackup
	if err := db.Where("user_id = ? AND account_id = ?", user.ID, "reason-account").First(&backup).Error; err != nil {
		t.Fatalf("reload backup: %v", err)
	}
	if backup.Checksum != "sum-2" {
		t.Fatalf("invalid reason mutated backup checksum: %s", backup.Checksum)
	}
}
