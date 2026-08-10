package model

import "time"

const (
	CharacterCardStatusDraft     = "draft"
	CharacterCardStatusPublished = "published"

	CharacterCardVisibilityPrivate = "private"
	CharacterCardVisibilityPublic  = "public"
)

// CharacterCard is an RPBox-native character profile used for authoring and
// public presentation. It deliberately has no one-to-one constraint with the
// story archive Character model.
type CharacterCard struct {
	ID     uint `gorm:"primarykey" json:"id"`
	UserID uint `gorm:"index;not null" json:"user_id"`

	CharacterID     *uint  `gorm:"index" json:"character_id"`
	SourceBackupID  *uint  `gorm:"index" json:"source_backup_id"`
	SourceAccountID string `gorm:"size:32" json:"source_account_id"`
	SourceProfileID string `gorm:"size:128" json:"source_profile_id"`

	FirstName          string `gorm:"size:128" json:"first_name"`
	LastName           string `gorm:"size:128" json:"last_name"`
	DisplayName        string `gorm:"size:256" json:"display_name"`
	Title              string `gorm:"size:128" json:"title"`
	FullTitle          string `gorm:"size:256" json:"full_title"`
	Race               string `gorm:"size:64" json:"race"`
	Class              string `gorm:"size:64" json:"class"`
	EyeColor           string `gorm:"size:64" json:"eye_color"`
	EyeColorHex        string `gorm:"size:16" json:"eye_color_hex"`
	Age                string `gorm:"size:64" json:"age"`
	Height             string `gorm:"size:64" json:"height"`
	Weight             string `gorm:"size:64" json:"weight"`
	Birthplace         string `gorm:"size:256" json:"birthplace"`
	Residence          string `gorm:"size:256" json:"residence"`
	RelationshipStatus string `gorm:"size:64" json:"relationship_status"`
	Icon               string `gorm:"size:128" json:"icon"`
	NameColor          string `gorm:"size:16" json:"name_color"`

	Summary         string `gorm:"size:1000" json:"summary"`
	BackgroundStory string `gorm:"type:text" json:"background_story"`
	FirstImpression string `gorm:"type:text" json:"first_impression"`
	OtherContent    string `gorm:"type:text" json:"other_content"`

	// PortraitImage is an internal storage reference. API responses expose only
	// the permission-checked image endpoint built by the CharacterCard DTO.
	PortraitImage          string     `gorm:"size:512" json:"-"`
	PortraitImageUpdatedAt *time.Time `json:"portrait_image_updated_at"`

	Status     string `gorm:"size:20;not null;default:draft;index" json:"status"`
	Visibility string `gorm:"size:20;not null;default:private;index" json:"visibility"`
	SortOrder  int    `gorm:"default:0;index" json:"sort_order"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
