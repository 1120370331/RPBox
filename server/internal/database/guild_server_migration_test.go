package database

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newGuildServerMigrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("RPBOX_GUILD_TEST_POSTGRES_DSN")
	if dsn == "" {
		return testutil.NewTestDB(t)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	schema := fmt.Sprintf("guild_server_migration_%d", time.Now().UnixNano())
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Errorf("cleanup test schema: %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	if err := db.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestGuildServerLegacyMigration(t *testing.T) {
	db := newGuildServerMigrationDB(t)
	// Start with the complete historical schema except the new server column.
	if err := db.AutoMigrate(&model.Guild{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropColumn(&model.Guild{}, "Server"); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasColumn(&model.Guild{}, "Server") {
		t.Fatal("legacy fixture must not have server column")
	}
	if err := db.Exec("INSERT INTO guilds (id, name, owner_id, description, invite_code) VALUES (1, ?, 10, ?, ?)", "Legacy Guild", "Preserved description", "legacy-invite").Error; err != nil {
		t.Fatal(err)
	}
	// Same order as Init: GORM schema migration followed by checked normalization.
	if err := db.AutoMigrate(&model.Guild{}); err != nil {
		t.Fatal(err)
	}
	if err := migrateGuildServer(db); err != nil {
		t.Fatal(err)
	}
	var guild model.Guild
	if err := db.First(&guild, 1).Error; err != nil {
		t.Fatal(err)
	}
	if guild.Server != "" || guild.Name != "Legacy Guild" || guild.Description != "Preserved description" || guild.OwnerID != 10 || guild.InviteCode != "legacy-invite" {
		t.Fatalf("legacy record changed unexpectedly: %#v", guild)
	}
	if err := db.Model(&guild).UpdateColumn("server", "已有服").Error; err != nil {
		t.Fatal(err)
	}
	// Preserve real names and normalize explicit NULL without touching timestamps.
	if err := db.Exec("INSERT INTO guilds (id, name, owner_id, server, invite_code) VALUES (2, ?, 20, NULL, ?)", "Null Guild", "null-invite").Error; err != nil {
		t.Fatal(err)
	}
	before := guild.UpdatedAt
	for i := 0; i < 2; i++ {
		if err := db.AutoMigrate(&model.Guild{}); err != nil {
			t.Fatal(err)
		}
		if err := migrateGuildServer(db); err != nil {
			t.Fatal(err)
		}
	}
	var rows []model.Guild
	if err := db.Order("id").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Server != "已有服" || rows[1].Server != "" || !rows[0].UpdatedAt.Equal(before) {
		t.Fatalf("repeat migration lost or changed records: %#v", rows)
	}
	var nullCount int64
	if err := db.Model(&model.Guild{}).Where("server IS NULL").Count(&nullCount).Error; err != nil || nullCount != 0 {
		t.Fatalf("NULL normalization failed: count=%d err=%v", nullCount, err)
	}
	if db.Dialector.Name() == "postgres" {
		var maxLength int
		if err := db.Raw("SELECT character_maximum_length FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'guilds' AND column_name = 'server'").Scan(&maxLength).Error; err != nil || maxLength != 128 {
			t.Fatalf("postgres server column must be varchar(128): length=%d err=%v", maxLength, err)
		}
	}
}

func TestGuildServerMigrationErrors(t *testing.T) {
	db := newGuildServerMigrationDB(t)
	if err := db.AutoMigrate(&model.Guild{}); err != nil {
		t.Fatal(err)
	}
	want := errors.New("injected migration write failure")
	if err := db.Callback().Update().Before("gorm:update").Register("reject_guild_server_migration", func(tx *gorm.DB) {
		tx.AddError(want)
	}); err != nil {
		t.Fatal(err)
	}
	if err := migrateGuildServer(db); !errors.Is(err, want) {
		t.Fatalf("migration failure must propagate: %v", err)
	}
}

func TestGuildServerMigrationMissingColumnErrors(t *testing.T) {
	db := newGuildServerMigrationDB(t)
	if err := db.Exec("CREATE TABLE guilds (id integer PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateGuildServer(db); err == nil {
		t.Fatal("normalization must fail when schema migration has not succeeded")
	}
}
