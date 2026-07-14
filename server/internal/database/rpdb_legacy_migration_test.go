package database

import (
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func TestMigrateLegacyRPDBLikesPreservesRowsAndClearsCanonicalName(t *testing.T) {
	db := testutil.NewTestDB(t)
	createLegacyRPDBLikesTable(t, db)

	createdAt := time.Date(2026, time.July, 10, 12, 30, 0, 0, time.UTC)
	if err := db.Exec(
		"INSERT INTO rpdb_likes (id, entry_id, user_id, created_at) VALUES (?, ?, ?, ?)",
		7, 42, 9, createdAt,
	).Error; err != nil {
		t.Fatalf("insert legacy RPDB like: %v", err)
	}

	if err := migrateLegacyRPDBLikes(db); err != nil {
		t.Fatalf("migrate legacy RPDB likes: %v", err)
	}

	if db.Migrator().HasTable("rpdb_likes") {
		t.Fatal("canonical rpdb_likes name still exists before AutoMigrate")
	}
	if !db.Migrator().HasTable("rpdb_entry_likes_legacy") {
		t.Fatal("legacy RPDB likes table was not renamed")
	}

	var preserved struct {
		ID        uint
		EntryID   uint
		UserID    uint
		CreatedAt time.Time
	}
	if err := db.Table("rpdb_entry_likes_legacy").First(&preserved).Error; err != nil {
		t.Fatalf("load preserved legacy RPDB like: %v", err)
	}
	if preserved.ID != 7 || preserved.EntryID != 42 || preserved.UserID != 9 || !preserved.CreatedAt.Equal(createdAt) {
		t.Fatalf("legacy RPDB like changed during rename: %+v", preserved)
	}

	if err := db.AutoMigrate(&model.RPDBLike{}); err != nil {
		t.Fatalf("auto migrate canonical RPDB likes: %v", err)
	}

	columns := tableColumnNames(t, db, "rpdb_likes")
	if !columns["work_id"] || !columns["user_id"] {
		t.Fatalf("canonical rpdb_likes missing new columns: %v", columns)
	}
	if columns["entry_id"] {
		t.Fatalf("canonical rpdb_likes retained legacy entry_id column: %v", columns)
	}

	if err := migrateLegacyRPDBLikes(db); err != nil {
		t.Fatalf("repeat legacy RPDB likes migration: %v", err)
	}
	if db.Migrator().HasTable("rpdb_entry_likes_legacy_2") {
		t.Fatal("repeat migration renamed the clean canonical table")
	}
	if !db.Migrator().HasTable("rpdb_likes") {
		t.Fatal("repeat migration removed the clean canonical table")
	}
}

func TestMigrateLegacyRPDBLikesUsesIncrementingSuffixOnConflict(t *testing.T) {
	db := testutil.NewTestDB(t)
	createLegacyRPDBLikesTable(t, db)
	if err := db.Exec("CREATE TABLE rpdb_entry_likes_legacy (marker TEXT NOT NULL)").Error; err != nil {
		t.Fatalf("create conflicting legacy table: %v", err)
	}
	if err := db.Exec("INSERT INTO rpdb_likes (entry_id, user_id, created_at) VALUES (?, ?, ?)", 11, 12, time.Now().UTC()).Error; err != nil {
		t.Fatalf("insert legacy RPDB like: %v", err)
	}

	if err := migrateLegacyRPDBLikes(db); err != nil {
		t.Fatalf("migrate legacy RPDB likes with conflict: %v", err)
	}

	if !db.Migrator().HasTable("rpdb_entry_likes_legacy_2") {
		t.Fatal("legacy RPDB likes table did not use an incremented suffix")
	}
	var count int64
	if err := db.Table("rpdb_entry_likes_legacy_2").Count(&count).Error; err != nil {
		t.Fatalf("count suffixed legacy RPDB likes: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one preserved legacy row, got %d", count)
	}
}

func createLegacyRPDBLikesTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`CREATE TABLE rpdb_likes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		entry_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		created_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create legacy rpdb_likes: %v", err)
	}
}

func tableColumnNames(t *testing.T, db *gorm.DB, table string) map[string]bool {
	t.Helper()
	columnTypes, err := db.Migrator().ColumnTypes(table)
	if err != nil {
		t.Fatalf("list columns for %s: %v", table, err)
	}
	columns := make(map[string]bool, len(columnTypes))
	for _, columnType := range columnTypes {
		columns[columnType.Name()] = true
	}
	return columns
}
