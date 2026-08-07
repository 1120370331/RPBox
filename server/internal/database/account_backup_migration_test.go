package database

import (
	"testing"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

type legacyAccountBackup struct {
	ID        uint   `gorm:"primarykey"`
	UserID    uint   `gorm:"index;not null"`
	AccountID string `gorm:"size:32;uniqueIndex:idx_user_account"`
}

func (legacyAccountBackup) TableName() string {
	return "account_backups"
}

func TestMigrateAccountBackupUniqueIndex(t *testing.T) {
	db := testutil.NewTestDB(t, &model.AccountBackup{}, &model.AccountBackupVersion{})

	if err := db.Exec("DROP INDEX IF EXISTS idx_user_account").Error; err != nil {
		t.Fatalf("drop current index: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX idx_user_account ON account_backups(account_id)").Error; err != nil {
		t.Fatalf("create legacy index: %v", err)
	}

	type indexRow struct {
		Seq     int
		Name    string
		Unique  int
		Origin  string
		Partial int
	}
	var before []indexRow
	if err := db.Raw("PRAGMA index_list(account_backups)").Scan(&before).Error; err != nil {
		t.Fatalf("list indexes before migration: %v", err)
	}
	for _, idx := range before {
		if idx.Name == "idx_user_account" {
			if idx.Unique != 1 {
				t.Fatalf("expected legacy index to be unique, got %+v", idx)
			}
		}
	}

	if err := migrateAccountBackupUniqueIndex(db); err != nil {
		t.Fatalf("migrate account backup unique index: %v", err)
	}

	var after []indexRow
	if err := db.Raw("PRAGMA index_list(account_backups)").Scan(&after).Error; err != nil {
		t.Fatalf("list indexes after migration: %v", err)
	}

	var target indexRow
	found := false
	for _, idx := range after {
		if idx.Name == "idx_user_account" {
			target = idx
			found = true
			break
		}
	}
	if !found {
		t.Fatal("idx_user_account not found after migration")
	}
	if target.Unique != 1 {
		t.Fatalf("expected migrated index to remain unique, got %+v", target)
	}

	type columnRow struct {
		Seqno int
		Cid   int
		Name  string
	}
	var columns []columnRow
	if err := db.Raw("PRAGMA index_info(idx_user_account)").Scan(&columns).Error; err != nil {
		t.Fatalf("index info after migration: %v", err)
	}
	if len(columns) != 2 || columns[0].Name != "user_id" || columns[1].Name != "account_id" {
		t.Fatalf("unexpected migrated index columns: %+v", columns)
	}

	if err := db.Create(&model.AccountBackup{UserID: 1, AccountID: "acc-1", Checksum: "a"}).Error; err != nil {
		t.Fatalf("create first backup: %v", err)
	}
	if err := db.Create(&model.AccountBackup{UserID: 2, AccountID: "acc-1", Checksum: "b"}).Error; err != nil {
		t.Fatalf("create second backup with same account id: %v", err)
	}
}
