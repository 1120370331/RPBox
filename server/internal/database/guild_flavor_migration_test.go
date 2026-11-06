package database

import (
	"testing"

	"github.com/rpbox/server/internal/model"
)

func TestGuildFlavorMigrationPreservesExistingGuild(t *testing.T) {
	db := newGuildServerMigrationDB(t)
	if err := db.AutoMigrate(&model.Guild{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropColumn(&model.Guild{}, "GameFlavor"); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO guilds (id, name, owner_id, server, invite_code) VALUES (1, ?, 10, ?, ?)", "Existing Guild", "Moon Guard", "existing-invite").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Guild{}); err != nil {
		t.Fatal(err)
	}
	var guild model.Guild
	if err := db.First(&guild, 1).Error; err != nil {
		t.Fatal(err)
	}
	if guild.Name != "Existing Guild" || guild.Server != "Moon Guard" || guild.GameFlavor != "_retail_" {
		t.Fatalf("migrated guild=%+v", guild)
	}
}
