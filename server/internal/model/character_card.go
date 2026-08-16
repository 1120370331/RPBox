package model

import "time"

const (
	CharacterCardStatusDraft     = "draft"
	CharacterCardStatusPublished = "published"

	CharacterCardVisibilityPrivate = "private"
	CharacterCardVisibilityPublic  = "public"

	CharacterCardReviewNone     = "none"
	CharacterCardReviewPending  = "pending"
	CharacterCardReviewApproved = "approved"
	CharacterCardReviewRejected = "rejected"
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
	ClassColor         string `gorm:"size:16" json:"class_color"`
	NameColor          string `gorm:"size:16" json:"name_color"`

	Summary         string `gorm:"size:1000" json:"summary"`
	BackgroundStory string `gorm:"type:text" json:"background_story"`
	// FirstImpression is the legacy rich-text field retained as free-form
	// "other notes" beside the structured five-slot impressions.
	FirstImpression string `gorm:"type:text" json:"first_impression"`
	OtherContent    string `gorm:"type:text" json:"other_content"`

	// PortraitImage is an internal storage reference. API responses expose only
	// the permission-checked image endpoint built by the CharacterCard DTO.
	PortraitImage          string     `gorm:"size:512" json:"-"`
	PortraitImageUpdatedAt *time.Time `json:"portrait_image_updated_at"`

	Status     string `gorm:"size:20;not null;default:draft;index" json:"status"`
	Visibility string `gorm:"size:20;not null;default:private;index" json:"visibility"`
	SortOrder  int    `gorm:"default:0;index" json:"sort_order"`

	ReviewStatus  string     `gorm:"size:20;not null;default:none;index" json:"review_status"`
	ReviewerID    *uint      `gorm:"index" json:"reviewer_id,omitempty"`
	ReviewComment string     `gorm:"size:512" json:"review_comment"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CharacterCardPortrait stores the ordered protected portrait gallery. The
// first row by SortOrder is the cover and is mirrored to CharacterCard's
// legacy PortraitImage fields for older clients and stable cover URLs.
type CharacterCardPortrait struct {
	ID              uint `gorm:"primarykey" json:"id"`
	CharacterCardID uint `gorm:"not null;index;uniqueIndex:idx_character_card_portraits_card_sort" json:"character_card_id"`
	SortOrder       int  `gorm:"not null;uniqueIndex:idx_character_card_portraits_card_sort" json:"sort_order"`

	Image          string     `gorm:"size:512;not null" json:"-"`
	ImageUpdatedAt *time.Time `json:"image_updated_at"`

	CharacterCard CharacterCard `gorm:"foreignKey:CharacterCardID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// CharacterCardPublication is the last moderator-approved immutable view of a
// card. Owners edit the live aggregate while public readers stay on this
// snapshot until the next approval, preventing unreviewed text or images from
// leaking through an already-public card.
type CharacterCardPublication struct {
	CharacterCardID uint   `gorm:"primarykey" json:"character_card_id"`
	UserID          uint   `gorm:"not null;index" json:"user_id"`
	Payload         string `gorm:"type:text;not null" json:"-"`

	ApprovedBy *uint     `gorm:"index" json:"approved_by,omitempty"`
	ApprovedAt time.Time `gorm:"not null" json:"approved_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CharacterCardSubmission is the single latest owner-submitted snapshot that
// moderators review. The editable CharacterCard aggregate may continue to
// auto-save after submission without mutating this frozen candidate. A later
// publish action replaces this row instead of creating another queue item.
type CharacterCardSubmission struct {
	CharacterCardID uint   `gorm:"primarykey" json:"character_card_id"`
	UserID          uint   `gorm:"not null;index" json:"user_id"`
	Payload         string `gorm:"type:text;not null" json:"-"`

	SubmittedAt time.Time `gorm:"not null;index" json:"submitted_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CharacterCardImpression is one of the five fixed "at first glance" slots
// attached to a character card. Image fields are private storage references;
// API responses expose them only through permission-aware image endpoints.
type CharacterCardImpression struct {
	ID              uint  `gorm:"primarykey" json:"id"`
	CharacterCardID uint  `gorm:"not null;uniqueIndex:idx_character_card_impressions_card_slot" json:"character_card_id"`
	Slot            uint8 `gorm:"not null;uniqueIndex:idx_character_card_impressions_card_slot;check:slot >= 1 AND slot <= 5" json:"slot"`

	Active   bool   `gorm:"not null;default:false" json:"active"`
	Title    string `gorm:"size:80" json:"title"`
	Text     string `gorm:"size:500" json:"text"`
	TRP3Icon string `gorm:"size:128" json:"trp3_icon"`

	IconImage          string     `gorm:"size:512" json:"-"`
	IconImageUpdatedAt *time.Time `json:"icon_image_updated_at"`
	Image              string     `gorm:"size:512" json:"-"`
	ImageUpdatedAt     *time.Time `json:"image_updated_at"`

	CharacterCard CharacterCard `gorm:"foreignKey:CharacterCardID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}
