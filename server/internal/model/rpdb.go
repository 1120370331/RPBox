package model

import "time"

const (
	RPDBWorkTypeItemShowcase = "item_showcase"
	RPDBWorkTypeTransmog     = "transmog"
	RPDBWorkTypeHomeShowcase = "home_showcase"

	RPDBStatusDraft     = "draft"
	RPDBStatusPending   = "pending"
	RPDBStatusPublished = "published"
	RPDBStatusArchived  = "archived"
	RPDBStatusRemoved   = "removed"

	RPDBReviewNone     = "none"
	RPDBReviewPending  = "pending"
	RPDBReviewApproved = "approved"
	RPDBReviewRejected = "rejected"

	RPDBVerificationUnverified = "unverified"
	RPDBVerificationVerified   = "verified"
	RPDBVerificationStale      = "stale"
	RPDBVerificationDisputed   = "disputed"

	RPDBListStatusWanted  = "wanted"
	RPDBListStatusFarming = "farming"
	RPDBListStatusOwned   = "owned"
	RPDBListStatusPaused  = "paused"

	RPDBVisibilityPublic  = "public"
	RPDBVisibilityGuild   = "guild"
	RPDBVisibilityPrivate = "private"
)

// RPDBWork is a player-authored RP showcase, transmog, or guide.
type RPDBWork struct {
	ID       uint `gorm:"primarykey" json:"id"`
	AuthorID uint `gorm:"index;not null" json:"author_id"`

	Type        string `gorm:"size:24;index;not null" json:"type"`
	Title       string `gorm:"size:256;index;not null" json:"title"`
	Slug        string `gorm:"size:256;index" json:"slug"`
	Summary     string `gorm:"size:512" json:"summary"`
	Content     string `gorm:"type:text" json:"content"`
	ContentType string `gorm:"size:20;default:html" json:"content_type"`

	CoverImage          string     `gorm:"type:text" json:"cover_image"`
	CoverImageUpdatedAt *time.Time `json:"cover_image_updated_at,omitempty"`

	RPUseCases        string `gorm:"type:text" json:"rp_use_cases"`
	EffectDescription string `gorm:"type:text" json:"effect_description"`
	Restrictions      string `gorm:"type:json" json:"restrictions"`
	Extra             string `gorm:"type:json" json:"extra"`

	GameVersion        string     `gorm:"size:32;index" json:"game_version"`
	Expansion          string     `gorm:"size:64;index" json:"expansion"`
	AvailabilityStatus string     `gorm:"size:24;index" json:"availability_status"`
	BindType           string     `gorm:"size:24;index" json:"bind_type"`
	Faction            string     `gorm:"size:24;index" json:"faction"`
	ArmorType          string     `gorm:"size:24;index" json:"armor_type"`
	VerificationStatus string     `gorm:"size:24;default:unverified;index" json:"verification_status"`
	LastVerifiedAt     *time.Time `gorm:"index" json:"last_verified_at"`
	VerifiedCount      int        `gorm:"default:0" json:"verified_count"`
	OutdatedCount      int        `gorm:"default:0" json:"outdated_count"`

	Status        string     `gorm:"size:20;default:draft;index" json:"status"`
	IsPublic      bool       `gorm:"default:false;index" json:"is_public"`
	Visibility    string     `gorm:"size:16;index" json:"visibility"`
	GuildID       *uint      `gorm:"index" json:"guild_id,omitempty"`
	GuildIDs      []uint     `gorm:"serializer:json;type:json" json:"guild_ids,omitempty"`
	ReviewStatus  string     `gorm:"size:20;default:none;index" json:"review_status"`
	ReviewerID    *uint      `gorm:"index" json:"reviewer_id"`
	ReviewComment string     `gorm:"size:512" json:"review_comment"`
	ReviewedAt    *time.Time `json:"reviewed_at"`

	Version       int `gorm:"default:1" json:"version"`
	ViewCount     int `gorm:"default:0" json:"view_count"`
	LikeCount     int `gorm:"default:0" json:"like_count"`
	FavoriteCount int `gorm:"default:0" json:"favorite_count"`
	CommentCount  int `gorm:"default:0" json:"comment_count"`
	ListCount     int `gorm:"default:0" json:"list_count"`
	MediaCount    int `gorm:"default:0" json:"media_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (RPDBWork) TableName() string { return "rpdb_works" }

// RPDBReference links a work to a real game object or external database.
type RPDBReference struct {
	ID     uint `gorm:"primarykey" json:"id"`
	WorkID uint `gorm:"uniqueIndex:idx_rpdb_reference_unique;index;not null" json:"work_id"`

	ExternalType      string `gorm:"size:32;uniqueIndex:idx_rpdb_reference_unique;index;not null" json:"external_type"`
	ExternalID        string `gorm:"size:64;uniqueIndex:idx_rpdb_reference_unique;index;not null" json:"external_id"`
	Name              string `gorm:"size:256;not null" json:"name"`
	Icon              string `gorm:"size:128" json:"icon"`
	Quality           string `gorm:"size:32;index" json:"quality"`
	Description       string `gorm:"type:text" json:"description"`
	AcquisitionMethod string `gorm:"type:text" json:"acquisition_method"`
	Source            string `gorm:"size:32" json:"source"`
	URL               string `gorm:"type:text" json:"url"`
	Locale            string `gorm:"size:16" json:"locale"`
	IsPrimary         bool   `gorm:"default:false;index" json:"is_primary"`
	SortOrder         int    `gorm:"default:0" json:"sort_order"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (RPDBReference) TableName() string { return "rpdb_references" }

// RPDBMedia stores player-provided visual evidence for a work.
type RPDBMedia struct {
	ID       uint  `gorm:"primarykey" json:"id"`
	WorkID   uint  `gorm:"index;not null" json:"work_id"`
	AuthorID *uint `gorm:"index" json:"author_id"`

	Type         string `gorm:"size:20;index;not null" json:"type"`
	URL          string `gorm:"type:text;not null" json:"url"`
	ThumbnailURL string `gorm:"type:text" json:"thumbnail_url"`
	Caption      string `gorm:"size:256" json:"caption"`
	SortOrder    int    `gorm:"default:0" json:"sort_order"`
	Width        int    `gorm:"default:0" json:"width"`
	Height       int    `gorm:"default:0" json:"height"`
	Duration     int    `gorm:"default:0" json:"duration"`
	Meta         string `gorm:"type:json" json:"meta"`

	ReviewStatus  string     `gorm:"size:20;default:pending;index" json:"review_status"`
	ReviewerID    *uint      `gorm:"index" json:"reviewer_id"`
	ReviewComment string     `gorm:"size:512" json:"review_comment"`
	ReviewedAt    *time.Time `json:"reviewed_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (RPDBMedia) TableName() string { return "rpdb_media" }

// RPDBTransmogSlot stores one equipment slot in a transmog work.
type RPDBTransmogSlot struct {
	ID          uint   `gorm:"primarykey" json:"id"`
	WorkID      uint   `gorm:"index;not null" json:"work_id"`
	ReferenceID *uint  `gorm:"index" json:"reference_id"`
	Slot        string `gorm:"size:32;index;not null" json:"slot"`
	Role        string `gorm:"size:20;default:required;index" json:"role"`
	Name        string `gorm:"size:256" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Source      string `gorm:"size:256" json:"source"`
	WowheadURL  string `gorm:"type:text" json:"wowhead_url"`
	Variant     string `gorm:"size:256" json:"variant"`
	Note        string `gorm:"size:512" json:"note"`
	SortOrder   int    `gorm:"default:0" json:"sort_order"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (RPDBTransmogSlot) TableName() string { return "rpdb_transmog_slots" }

// RPDBGuideStep stores a structured acquisition step and optional coordinates.
type RPDBGuideStep struct {
	ID           uint    `gorm:"primarykey" json:"id"`
	WorkID       uint    `gorm:"index;not null" json:"work_id"`
	SortOrder    int     `gorm:"default:0" json:"sort_order"`
	Title        string  `gorm:"size:128" json:"title"`
	Body         string  `gorm:"type:text" json:"body"`
	Zone         string  `gorm:"size:128;index" json:"zone"`
	MapID        string  `gorm:"size:64" json:"map_id"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Label        string  `gorm:"size:128" json:"label"`
	Prerequisite string  `gorm:"type:text" json:"prerequisite"`
	Meta         string  `gorm:"type:json" json:"meta"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (RPDBGuideStep) TableName() string { return "rpdb_guide_steps" }

// RPDBTag associates an existing shared tag with a work.
type RPDBTag struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	WorkID    uint      `gorm:"uniqueIndex:idx_rpdb_work_tag;not null" json:"work_id"`
	TagID     uint      `gorm:"uniqueIndex:idx_rpdb_work_tag;index;not null" json:"tag_id"`
	AddedBy   uint      `gorm:"index" json:"added_by"`
	CreatedAt time.Time `json:"created_at"`
}

func (RPDBTag) TableName() string { return "rpdb_tags" }

type RPDBLike struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	WorkID    uint      `gorm:"uniqueIndex:idx_rpdb_like_user;not null" json:"work_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_rpdb_like_user;index;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (RPDBLike) TableName() string { return "rpdb_likes" }

type RPDBFavorite struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	WorkID    uint      `gorm:"uniqueIndex:idx_rpdb_favorite_user;not null" json:"work_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_rpdb_favorite_user;index;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (RPDBFavorite) TableName() string { return "rpdb_favorites" }

type RPDBView struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	WorkID    uint      `gorm:"uniqueIndex:idx_rpdb_view_user;not null" json:"work_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_rpdb_view_user;index;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (RPDBView) TableName() string { return "rpdb_views" }

// RPDBViewEvent records daily unique detail-page views for heat ranking.
// Each logged-in user contributes at most once per work per calendar day.
type RPDBViewEvent struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	WorkID    uint      `gorm:"uniqueIndex:idx_rpdb_view_event_daily,priority:1;index:idx_rpdb_view_event_work_time,priority:1;not null" json:"work_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_rpdb_view_event_daily,priority:2;index;not null" json:"user_id"`
	ViewDate  string    `gorm:"size:10;uniqueIndex:idx_rpdb_view_event_daily,priority:3;not null" json:"view_date"` // YYYY-MM-DD local date
	CreatedAt time.Time `gorm:"index:idx_rpdb_view_event_work_time,priority:2" json:"created_at"`
}

func (RPDBViewEvent) TableName() string { return "rpdb_view_events" }

// RPDBComment supports threaded discussion beneath a work.
type RPDBComment struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	WorkID    uint      `gorm:"index;not null" json:"work_id"`
	AuthorID  uint      `gorm:"index;not null" json:"author_id"`
	ParentID  *uint     `gorm:"index" json:"parent_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	LikeCount int       `gorm:"default:0" json:"like_count"`
	Status    string    `gorm:"size:20;default:published;index" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (RPDBComment) TableName() string { return "rpdb_comments" }

type RPDBCommentLike struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CommentID uint      `gorm:"uniqueIndex:idx_rpdb_comment_like_user;not null" json:"comment_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_rpdb_comment_like_user;index;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (RPDBCommentLike) TableName() string { return "rpdb_comment_likes" }

// RPDBList is a user-owned RPDB collection checklist.
type RPDBList struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Description string    `gorm:"size:512" json:"description"`
	IsDefault   bool      `gorm:"default:false;index" json:"is_default"`
	IsPublic    bool      `gorm:"default:false;index" json:"is_public"`
	ItemCount   int       `gorm:"default:0" json:"item_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (RPDBList) TableName() string { return "rpdb_lists" }

type RPDBListEntry struct {
	ID          uint       `gorm:"primarykey" json:"id"`
	ListID      uint       `gorm:"uniqueIndex:idx_rpdb_list_work;not null" json:"list_id"`
	WorkID      uint       `gorm:"uniqueIndex:idx_rpdb_list_work;index;not null" json:"work_id"`
	CharacterID *uint      `gorm:"index" json:"character_id"`
	Status      string     `gorm:"size:20;default:wanted;index" json:"status"`
	Priority    int        `gorm:"default:0" json:"priority"`
	Quantity    int        `gorm:"default:1" json:"quantity"`
	Note        string     `gorm:"size:512" json:"note"`
	SortOrder   int        `gorm:"default:0" json:"sort_order"`
	AcquiredAt  *time.Time `json:"acquired_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (RPDBListEntry) TableName() string { return "rpdb_list_entries" }

// RPDBRevision stores a reviewable change to an existing published work.
type RPDBRevision struct {
	ID            uint       `gorm:"primarykey" json:"id"`
	WorkID        uint       `gorm:"index;not null" json:"work_id"`
	ProposerID    uint       `gorm:"index;not null" json:"proposer_id"`
	BaseVersion   int        `gorm:"not null" json:"base_version"`
	Payload       string     `gorm:"type:json;not null" json:"payload"`
	ChangeSummary string     `gorm:"size:512" json:"change_summary"`
	Status        string     `gorm:"size:20;default:pending;index" json:"status"`
	ReviewerID    *uint      `gorm:"index" json:"reviewer_id"`
	ReviewComment string     `gorm:"size:512" json:"review_comment"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	AppliedAt     *time.Time `json:"applied_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (RPDBRevision) TableName() string { return "rpdb_revisions" }

// RPDBVerification records whether a user confirmed or disputed a work version.
type RPDBVerification struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	WorkID      uint      `gorm:"uniqueIndex:idx_rpdb_verification_user;not null" json:"work_id"`
	UserID      uint      `gorm:"uniqueIndex:idx_rpdb_verification_user;index;not null" json:"user_id"`
	WorkVersion int       `gorm:"default:1" json:"work_version"`
	Result      string    `gorm:"size:20;not null" json:"result"`
	Comment     string    `gorm:"size:512" json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (RPDBVerification) TableName() string { return "rpdb_verifications" }

// RPDBSet is a public, player-authored themed collection.
type RPDBSet struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	AuthorID     uint      `gorm:"index;not null" json:"author_id"`
	Name         string    `gorm:"size:128;not null" json:"name"`
	Description  string    `gorm:"type:text" json:"description"`
	CoverImage   string    `gorm:"type:text" json:"cover_image"`
	Type         string    `gorm:"size:32;index" json:"type"`
	IsPublic     bool      `gorm:"default:true;index" json:"is_public"`
	Status       string    `gorm:"size:20;default:pending;index" json:"status"`
	ReviewStatus string    `gorm:"size:20;default:pending;index" json:"review_status"`
	ItemCount    int       `gorm:"default:0" json:"item_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (RPDBSet) TableName() string { return "rpdb_sets" }

type RPDBSetWork struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	SetID     uint      `gorm:"uniqueIndex:idx_rpdb_set_work;not null" json:"set_id"`
	WorkID    uint      `gorm:"uniqueIndex:idx_rpdb_set_work;index;not null" json:"work_id"`
	Role      string    `gorm:"size:20;default:required" json:"role"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	Note      string    `gorm:"size:256" json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

func (RPDBSetWork) TableName() string { return "rpdb_set_works" }
