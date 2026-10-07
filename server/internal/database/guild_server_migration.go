package database

import (
	"fmt"

	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
)

// migrateGuildServer normalizes legacy NULL values after AutoMigrate adds the
// optional server column. Existing names are preserved; no name is inferred.
func migrateGuildServer(db *gorm.DB) error {
	if err := db.Model(&model.Guild{}).Where("server IS NULL").UpdateColumn("server", "").Error; err != nil {
		return fmt.Errorf("normalize legacy guild server values: %w", err)
	}
	return nil
}
