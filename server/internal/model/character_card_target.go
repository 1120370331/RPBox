package model

import "time"

// CharacterCardTarget maps an independent RPBox card to a versioned account profile.
// Source* fields on CharacterCard remain import provenance, not save destinations.
type CharacterCardTarget struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CharacterCardID uint      `gorm:"index;uniqueIndex:idx_card_backup" json:"character_card_id"`
	BackupID        uint      `gorm:"uniqueIndex:idx_card_backup;uniqueIndex:idx_backup_profile" json:"backup_id"`
	ProfileID       string    `gorm:"size:128;uniqueIndex:idx_backup_profile" json:"profile_id"`
	UpdatedAt       time.Time `json:"updated_at"`
}
