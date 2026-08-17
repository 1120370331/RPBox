package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	authpkg "github.com/rpbox/server/pkg/auth"
	_ "golang.org/x/image/webp"
	"golang.org/x/net/html"
	"gorm.io/gorm"
)

const (
	characterCardRichTextMaxBytes  = 256 << 10
	characterCardRichTextAllBytes  = 512 << 10
	characterCardPortraitMaxBytes  = 20 << 20
	characterCardPortraitMaxSide   = 12000
	characterCardPortraitMaxPixels = 60_000_000
)

var (
	errCharacterCardSourceNotFound = errors.New("character card source not found")
	errCharacterCardSourceCorrupt  = errors.New("character card source is corrupt")
)

type characterCardDTO struct {
	ID     uint `json:"id"`
	UserID uint `json:"user_id"`

	CharacterID     *uint  `json:"character_id,omitempty"`
	SourceBackupID  *uint  `json:"source_backup_id,omitempty"`
	SourceAccountID string `json:"source_account_id,omitempty"`
	SourceProfileID string `json:"source_profile_id,omitempty"`

	FirstName          string                              `json:"first_name"`
	LastName           string                              `json:"last_name"`
	DisplayName        string                              `json:"display_name"`
	Title              string                              `json:"title"`
	FullTitle          string                              `json:"full_title"`
	Race               string                              `json:"race"`
	Class              string                              `json:"class"`
	EyeColor           string                              `json:"eye_color"`
	EyeColorHex        string                              `json:"eye_color_hex"`
	Age                string                              `json:"age"`
	Height             string                              `json:"height"`
	Weight             string                              `json:"weight"`
	Birthplace         string                              `json:"birthplace"`
	Residence          string                              `json:"residence"`
	RelationshipStatus string                              `json:"relationship_status"`
	Icon               string                              `json:"icon"`
	ClassColor         string                              `json:"class_color"`
	NameColor          string                              `json:"name_color"`
	AdditionalInfo     []characterCardTRP3AdditionalInfo   `json:"additional_info"`
	PersonalityTraits  []characterCardTRP3PersonalityTrait `json:"personality_traits"`

	Summary         string                       `json:"summary"`
	BackgroundStory string                       `json:"background_story,omitempty"`
	FirstImpression string                       `json:"first_impression,omitempty"`
	OtherContent    string                       `json:"other_content,omitempty"`
	Impressions     []characterCardImpressionDTO `json:"impressions"`
	Portraits       []characterCardPortraitDTO   `json:"portraits"`

	PortraitImageURL       string     `json:"portrait_image_url"`
	PortraitImageUpdatedAt *time.Time `json:"portrait_image_updated_at"`
	Status                 string     `json:"status"`
	Visibility             string     `json:"visibility"`
	SortOrder              int        `json:"sort_order"`
	ReviewStatus           string     `json:"review_status"`
	ReviewComment          string     `json:"review_comment,omitempty"`
	ReviewedAt             *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type characterCardSourceDTO struct {
	SourceBackupID  uint      `json:"backup_id"`
	SourceAccountID string    `json:"account_id"`
	SourceProfileID string    `json:"profile_id"`
	ProfileName     string    `json:"profile_name"`
	DisplayName     string    `json:"display_name"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Title           string    `json:"title"`
	Race            string    `json:"race"`
	Class           string    `json:"class"`
	Icon            string    `json:"icon"`
	ClassColor      string    `json:"class_color"`
	NameColor       string    `json:"name_color"`
	BackupUpdatedAt time.Time `json:"backup_updated_at"`
}

type createCharacterCardRequest struct {
	SourceType      string `json:"source_type"`
	SourceBackupID  *uint  `json:"source_backup_id"`
	SourceProfileID string `json:"source_profile_id"`
}

type updateCharacterCardRequest struct {
	CharacterID optionalCharacterCardUint `json:"character_id"`

	FirstName          *string                              `json:"first_name"`
	LastName           *string                              `json:"last_name"`
	DisplayName        *string                              `json:"display_name"`
	Title              *string                              `json:"title"`
	FullTitle          *string                              `json:"full_title"`
	Race               *string                              `json:"race"`
	Class              *string                              `json:"class"`
	EyeColor           *string                              `json:"eye_color"`
	EyeColorHex        *string                              `json:"eye_color_hex"`
	Age                *string                              `json:"age"`
	Height             *string                              `json:"height"`
	Weight             *string                              `json:"weight"`
	Birthplace         *string                              `json:"birthplace"`
	Residence          *string                              `json:"residence"`
	RelationshipStatus *string                              `json:"relationship_status"`
	Icon               *string                              `json:"icon"`
	ClassColor         *string                              `json:"class_color"`
	NameColor          *string                              `json:"name_color"`
	AdditionalInfo     *[]characterCardTRP3AdditionalInfo   `json:"additional_info"`
	PersonalityTraits  *[]characterCardTRP3PersonalityTrait `json:"personality_traits"`

	Summary         *string                           `json:"summary"`
	BackgroundStory *string                           `json:"background_story"`
	FirstImpression *string                           `json:"first_impression"`
	OtherContent    *string                           `json:"other_content"`
	Impressions     *[]characterCardImpressionRequest `json:"impressions"`

	// portrait_image_url is the primary write contract. portrait_image remains
	// accepted for compatibility with callers that distinguish storage input
	// from the read-only DTO field.
	PortraitImage    optionalCharacterCardString `json:"portrait_image"`
	PortraitImageURL optionalCharacterCardString `json:"portrait_image_url"`
	Status           *string                     `json:"status"`
	Visibility       *string                     `json:"visibility"`
	SortOrder        *int                        `json:"sort_order"`
}

type optionalCharacterCardUint struct {
	Set   bool
	Value *uint
}

func (value *optionalCharacterCardUint) UnmarshalJSON(data []byte) error {
	value.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		value.Value = nil
		return nil
	}
	var parsed uint
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	value.Value = &parsed
	return nil
}

type optionalCharacterCardString struct {
	Set   bool
	Value string
}

func (value *optionalCharacterCardString) UnmarshalJSON(data []byte) error {
	value.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		value.Value = ""
		return nil
	}
	return json.Unmarshal(data, &value.Value)
}

type trp3CharacterCardFields struct {
	FirstName          string
	LastName           string
	Title              string
	FullTitle          string
	Race               string
	Class              string
	EyeColor           string
	EyeColorHex        string
	Age                string
	Height             string
	Weight             string
	Birthplace         string
	Residence          string
	RelationshipStatus string
	Icon               string
	ClassColor         string
	NameColor          string
}

type characterCardTRP3Color struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
}

type characterCardTRP3AdditionalInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Icon  string `json:"icon"`
}

type characterCardTRP3PersonalityTrait struct {
	PresetID   *int                    `json:"preset_id"`
	LeftText   string                  `json:"left_text"`
	RightText  string                  `json:"right_text"`
	LeftIcon   string                  `json:"left_icon"`
	RightIcon  string                  `json:"right_icon"`
	LeftColor  *characterCardTRP3Color `json:"left_color"`
	RightColor *characterCardTRP3Color `json:"right_color"`
	Value      int                     `json:"value"`
}

type parsedTRP3CharacterCardProfile struct {
	ProfileName       string
	Fields            trp3CharacterCardFields
	Impressions       []characterCardImpressionRequest
	AdditionalInfo    []characterCardTRP3AdditionalInfo
	PersonalityTraits []characterCardTRP3PersonalityTrait
}

func (s *Server) listCharacterCardSources(c *gin.Context) {
	userID := c.GetUint("userID")
	var backups []model.AccountBackup
	if err := database.DB.Where("user_id = ?", userID).
		Select("id", "account_id", "profiles_data", "updated_at").
		Order("updated_at DESC, id DESC").
		Find(&backups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询人物卡来源失败"})
		return
	}

	sources := make([]characterCardSourceDTO, 0)
	for _, backup := range backups {
		if err := validateCharacterCardSourceAccountID(backup.AccountID); err != nil {
			continue
		}
		profiles, err := decodeCharacterCardProfileMap(backup.ProfilesData)
		if err != nil {
			continue
		}
		profileIDs := make([]string, 0, len(profiles))
		for profileID := range profiles {
			profileIDs = append(profileIDs, profileID)
		}
		sort.Strings(profileIDs)

		for _, profileID := range profileIDs {
			if err := validatePlainCharacterCardField("profile_id", profileID, 128); err != nil {
				continue
			}
			profile, err := parseTRP3CharacterCardProfile(profiles[profileID])
			if err != nil {
				continue
			}
			if err := validatePlainCharacterCardField("profile_name", profile.ProfileName, 256); err != nil {
				continue
			}
			displayName := importedCharacterCardDisplayName(profile)
			candidate := model.CharacterCard{
				UserID:          userID,
				SourceBackupID:  &backup.ID,
				SourceAccountID: backup.AccountID,
				SourceProfileID: profileID,
				DisplayName:     displayName,
				Status:          model.CharacterCardStatusDraft,
				Visibility:      model.CharacterCardVisibilityPrivate,
			}
			applyTRP3CharacterCardFields(&candidate, profile.Fields)
			if err := validateCharacterCard(candidate); err != nil {
				continue
			}
			sources = append(sources, characterCardSourceDTO{
				SourceBackupID:  backup.ID,
				SourceAccountID: backup.AccountID,
				SourceProfileID: profileID,
				ProfileName:     profile.ProfileName,
				DisplayName:     displayName,
				FirstName:       profile.Fields.FirstName,
				LastName:        profile.Fields.LastName,
				Title:           profile.Fields.Title,
				Race:            profile.Fields.Race,
				Class:           profile.Fields.Class,
				Icon:            profile.Fields.Icon,
				ClassColor:      profile.Fields.ClassColor,
				NameColor:       profile.Fields.NameColor,
				BackupUpdatedAt: backup.UpdatedAt,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"sources": sources})
}

func (s *Server) listMyCharacterCards(c *gin.Context) {
	userID := c.GetUint("userID")
	var cards []model.CharacterCard
	if err := database.DB.Where("user_id = ?", userID).
		Omit("background_story", "first_impression", "other_content").
		Order("sort_order ASC, updated_at DESC, id DESC").
		Find(&cards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询人物卡失败"})
		return
	}
	impressionsByCard, err := loadCharacterCardImpressions(database.DB, characterCardIDs(cards))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询人物卡第一印象失败"})
		return
	}
	portraitsByCard, err := loadCharacterCardPortraits(database.DB, characterCardIDs(cards))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询人物卡角色大图失败"})
		return
	}

	result := make([]characterCardDTO, 0, len(cards))
	for _, card := range cards {
		result = append(result, s.buildCharacterCardDTO(card, impressionsByCard[card.ID], portraitsByCard[card.ID], true, false))
	}
	c.JSON(http.StatusOK, gin.H{"character_cards": result})
}

func (s *Server) createCharacterCard(c *gin.Context) {
	userID := c.GetUint("userID")
	var req createCharacterCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
		return
	}

	sourceType := strings.TrimSpace(req.SourceType)
	card := model.CharacterCard{
		UserID:     userID,
		Status:     model.CharacterCardStatusDraft,
		Visibility: model.CharacterCardVisibilityPrivate,
	}
	var impressions []model.CharacterCardImpression

	switch sourceType {
	case "blank":
		if req.SourceBackupID != nil || req.SourceProfileID != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "空白人物卡不能指定备份来源"})
			return
		}
		if err := validateCharacterCard(card); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := database.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&card).Error; err != nil {
				return err
			}
			impressions = defaultCharacterCardImpressions(card.ID)
			return tx.Omit("CharacterCard").Create(&impressions).Error
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建人物卡失败"})
			return
		}

	case "backup":
		if req.SourceBackupID == nil || *req.SourceBackupID == 0 || strings.TrimSpace(req.SourceProfileID) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请指定有效的备份和 profile"})
			return
		}
		if err := validatePlainCharacterCardField("source_profile_id", req.SourceProfileID, 128); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := database.DB.Transaction(func(tx *gorm.DB) error {
			backup, profile, err := loadCharacterCardBackupProfile(tx, userID, *req.SourceBackupID, req.SourceProfileID, "")
			if err != nil {
				return err
			}
			card.SourceBackupID = &backup.ID
			card.SourceAccountID = backup.AccountID
			card.SourceProfileID = req.SourceProfileID
			applyTRP3CharacterCardFields(&card, profile.Fields)
			if err := setCharacterCardTRP3Details(&card, profile.AdditionalInfo, profile.PersonalityTraits); err != nil {
				return fmt.Errorf("%w: %v", errCharacterCardSourceCorrupt, err)
			}
			card.DisplayName = importedCharacterCardDisplayName(profile)
			if err := validateCharacterCard(card); err != nil {
				return fmt.Errorf("%w: %v", errCharacterCardSourceCorrupt, err)
			}
			if err := tx.Create(&card).Error; err != nil {
				return err
			}
			impressions = characterCardImpressionsFromRequests(card.ID, profile.Impressions)
			return tx.Omit("CharacterCard").Create(&impressions).Error
		})
		if err != nil {
			switch {
			case errors.Is(err, errCharacterCardSourceNotFound):
				c.JSON(http.StatusNotFound, gin.H{"error": "备份来源或 profile 不存在"})
			case errors.Is(err, errCharacterCardSourceCorrupt):
				c.JSON(http.StatusBadRequest, gin.H{"error": "备份人物卡数据损坏"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"error": "创建人物卡失败"})
			}
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_type 仅支持 blank 或 backup"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"character_card": s.buildCharacterCardDTO(card, impressions, nil, true, true)})
}

func (s *Server) getCharacterCard(c *gin.Context) {
	id, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	viewerID := optionalActiveUserID(c)

	var card model.CharacterCard
	if err := database.DB.First(&card, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
		}
		return
	}
	isModerator := isCharacterCardModerator(viewerID)
	if isModerator && c.Query("review") == "submission" && normalizedCharacterCardReviewStatus(card.ReviewStatus) == model.CharacterCardReviewPending {
		if snapshot, _, submissionErr := loadCharacterCardSubmission(database.DB, card.ID); submissionErr == nil {
			submittedCard, submittedImpressions, submittedPortraits := snapshot.models()
			submittedCard.ReviewStatus = card.ReviewStatus
			submittedCard.ReviewerID = card.ReviewerID
			submittedCard.ReviewComment = card.ReviewComment
			submittedCard.ReviewedAt = card.ReviewedAt
			c.JSON(http.StatusOK, gin.H{"character_card": s.buildCharacterCardDTO(submittedCard, submittedImpressions, submittedPortraits, true, true)})
			return
		} else if !errors.Is(submissionErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡审核版本失败"})
			return
		}
	}
	ownerOrModerator := viewerID != 0 && (viewerID == card.UserID || isModerator)
	if !ownerOrModerator {
		dto, visible, err := s.loadPublicCharacterCardDTO(card, true)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
			return
		}
		if !visible {
			c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"character_card": dto})
		return
	}
	impressionsByCard, err := loadCharacterCardImpressions(database.DB, []uint{card.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡第一印象失败"})
		return
	}
	portraitsByCard, err := loadCharacterCardPortraits(database.DB, []uint{card.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡角色大图失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"character_card": s.buildCharacterCardDTO(card, impressionsByCard[card.ID], portraitsByCard[card.ID], true, true)})
}

// getCharacterCardShare returns link metadata from the approved public version.
func (s *Server) getCharacterCardShare(c *gin.Context) {
	id, ok := parseCharacterCardID(c)
	if !ok {
		return
	}

	var card model.CharacterCard
	if err := database.DB.First(&card, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
		}
		return
	}

	publicCard, visible, err := s.loadPublicCharacterCardDTO(card, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡公开版本失败"})
		return
	}
	if !visible {
		c.JSON(http.StatusConflict, gin.H{"error": "人物卡公开版本尚未通过审核，暂不可分享"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"path":       fmt.Sprintf("/character-cards/%d", card.ID),
		"title":      publicCard.DisplayName,
		"summary":    publicCard.Summary,
		"updated_at": publicCard.UpdatedAt,
	})
}

func (s *Server) updateCharacterCard(c *gin.Context) {
	id, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	userID := c.GetUint("userID")

	var card model.CharacterCard
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&card).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
		}
		return
	}

	var req updateCharacterCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
		return
	}

	candidate := card
	updates := make(map[string]interface{})
	applyCharacterCardStringUpdate(req.FirstName, &candidate.FirstName, "first_name", updates, true)
	applyCharacterCardStringUpdate(req.LastName, &candidate.LastName, "last_name", updates, true)
	applyCharacterCardStringUpdate(req.DisplayName, &candidate.DisplayName, "display_name", updates, true)
	applyCharacterCardStringUpdate(req.Title, &candidate.Title, "title", updates, true)
	applyCharacterCardStringUpdate(req.FullTitle, &candidate.FullTitle, "full_title", updates, true)
	applyCharacterCardStringUpdate(req.Race, &candidate.Race, "race", updates, true)
	applyCharacterCardStringUpdate(req.Class, &candidate.Class, "class", updates, true)
	applyCharacterCardStringUpdate(req.EyeColor, &candidate.EyeColor, "eye_color", updates, true)
	applyCharacterCardStringUpdate(req.EyeColorHex, &candidate.EyeColorHex, "eye_color_hex", updates, true)
	applyCharacterCardStringUpdate(req.Age, &candidate.Age, "age", updates, true)
	applyCharacterCardStringUpdate(req.Height, &candidate.Height, "height", updates, true)
	applyCharacterCardStringUpdate(req.Weight, &candidate.Weight, "weight", updates, true)
	applyCharacterCardStringUpdate(req.Birthplace, &candidate.Birthplace, "birthplace", updates, true)
	applyCharacterCardStringUpdate(req.Residence, &candidate.Residence, "residence", updates, true)
	applyCharacterCardStringUpdate(req.RelationshipStatus, &candidate.RelationshipStatus, "relationship_status", updates, true)
	applyCharacterCardStringUpdate(req.Icon, &candidate.Icon, "icon", updates, true)
	if err := applyCharacterCardTRP3ColorUpdate(req.ClassColor, req.NameColor, &candidate, updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	applyCharacterCardStringUpdate(req.Summary, &candidate.Summary, "summary", updates, true)
	applyCharacterCardStringUpdate(req.BackgroundStory, &candidate.BackgroundStory, "background_story", updates, false)
	applyCharacterCardStringUpdate(req.FirstImpression, &candidate.FirstImpression, "first_impression", updates, false)
	applyCharacterCardStringUpdate(req.OtherContent, &candidate.OtherContent, "other_content", updates, false)
	if req.AdditionalInfo != nil {
		if err := validateCharacterCardTRP3AdditionalInfo(*req.AdditionalInfo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		encoded, err := json.Marshal(*req.AdditionalInfo)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "附加说明格式无效"})
			return
		}
		candidate.TRP3AdditionalInfoJSON = string(encoded)
		updates["trp3_additional_info_json"] = candidate.TRP3AdditionalInfoJSON
	}
	if req.PersonalityTraits != nil {
		if err := validateCharacterCardTRP3PersonalityTraits(*req.PersonalityTraits); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		encoded, err := json.Marshal(*req.PersonalityTraits)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "个性格式无效"})
			return
		}
		candidate.TRP3PersonalityJSON = string(encoded)
		updates["trp3_personality_json"] = candidate.TRP3PersonalityJSON
	}
	applyCharacterCardStringUpdate(req.Status, &candidate.Status, "status", updates, true)
	applyCharacterCardStringUpdate(req.Visibility, &candidate.Visibility, "visibility", updates, true)
	if req.SortOrder != nil {
		candidate.SortOrder = *req.SortOrder
		updates["sort_order"] = candidate.SortOrder
	}

	if req.CharacterID.Set {
		if req.CharacterID.Value == nil || *req.CharacterID.Value == 0 {
			candidate.CharacterID = nil
			updates["character_id"] = nil
		} else {
			var count int64
			if err := database.DB.Model(&model.Character{}).
				Where("id = ? AND user_id = ?", *req.CharacterID.Value, userID).
				Count(&count).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "校验关联角色失败"})
				return
			}
			if count == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "关联角色不存在"})
				return
			}
			characterID := *req.CharacterID.Value
			candidate.CharacterID = &characterID
			updates["character_id"] = characterID
		}
	}

	portraitValue, portraitSet, err := requestedCharacterCardPortrait(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validateCharacterCard(candidate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	impressionsByCard, err := loadCharacterCardImpressions(database.DB, []uint{card.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡第一印象失败"})
		return
	}
	impressionPlan := characterCardImpressionUpdatePlan{
		Rows: fixedCharacterCardImpressions(card.ID, impressionsByCard[card.ID]),
	}
	impressionsNeedSave := false
	if req.Impressions != nil {
		impressionPlan, err = s.prepareCharacterCardImpressionUpdate(c, userID, card, impressionsByCard[card.ID], *req.Impressions)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		impressionsNeedSave = true
		// Impression rows are part of the card aggregate, so an impression-only
		// edit must also advance the card's ordering/version timestamp.
		updates["updated_at"] = time.Now().UTC()
	}

	newPortraitGenerated := false
	pendingPortraitToCleanup := ""
	var portraitRows []model.CharacterCardPortrait
	var portraitRowToDelete *model.CharacterCardPortrait
	var portraitRowToUpdate *model.CharacterCardPortrait
	var portraitRowToCreate *model.CharacterCardPortrait
	if portraitSet {
		portraitRows, err = ensureCharacterCardPortraitRows(database.DB, card)
		if err != nil {
			s.cleanupGeneratedCharacterCardImpressionImages(c, userID, impressionPlan)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取角色大图失败"})
			return
		}
		normalized, err := s.normalizeCharacterCardPortrait(c, userID, card, portraitValue)
		if err != nil {
			s.cleanupGeneratedCharacterCardImpressionImages(c, userID, impressionPlan)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		normalizedPortrait := normalized.Path
		newPortraitGenerated = normalized.Generated
		pendingPortraitToCleanup = normalized.PendingSource
		if normalizedPortrait != card.PortraitImage {
			now := time.Now().UTC()
			if normalizedPortrait == "" && len(portraitRows) > 0 {
				removed := portraitRows[0]
				portraitRowToDelete = &removed
				portraitRows = portraitRows[1:]
				if len(portraitRows) > 0 {
					normalizedPortrait = portraitRows[0].Image
				}
			} else if normalizedPortrait != "" && len(portraitRows) > 0 {
				updated := portraitRows[0]
				updated.Image = normalizedPortrait
				updated.ImageUpdatedAt = &now
				portraitRowToUpdate = &updated
				portraitRows[0] = updated
			} else if normalizedPortrait != "" {
				created := model.CharacterCardPortrait{
					CharacterCardID: card.ID,
					SortOrder:       0,
					Image:           normalizedPortrait,
					ImageUpdatedAt:  &now,
				}
				portraitRowToCreate = &created
			}
			candidate.PortraitImage = normalizedPortrait
			candidate.PortraitImageUpdatedAt = &now
			updates["portrait_image"] = normalizedPortrait
			updates["portrait_image_updated_at"] = now
		} else if normalized.Generated {
			// A generated path is always unique, but keep this conservative in case
			// storage behavior changes later.
			newPortraitGenerated = false
		}
	}
	if candidate.PortraitImage != "" && (candidate.Status != card.Status || candidate.Visibility != card.Visibility) {
		if _, alreadyVersioned := updates["portrait_image_updated_at"]; !alreadyVersioned {
			now := time.Now().UTC()
			candidate.PortraitImageUpdatedAt = &now
			updates["portrait_image_updated_at"] = now
		}
	}
	if candidate.Status != card.Status || candidate.Visibility != card.Visibility {
		if rotateCharacterCardImpressionImageVersions(impressionPlan.Rows, time.Now().UTC()) {
			impressionsNeedSave = true
		}
	}

	if len(updates) == 0 && !impressionsNeedSave {
		portraitsByCard, _ := loadCharacterCardPortraits(database.DB, []uint{card.ID})
		c.JSON(http.StatusOK, gin.H{"character_card": s.buildCharacterCardDTO(card, impressionPlan.Rows, portraitsByCard[card.ID], true, true)})
		return
	}
	resetCharacterCardReviewWhenWithdrawn(&candidate, updates)
	publicationAssetsToCheck := map[string]struct{}{}
	if candidate.Status != model.CharacterCardStatusPublished || candidate.Visibility != model.CharacterCardVisibilityPublic {
		if snapshot, _, loadErr := loadCharacterCardPublication(database.DB, card.ID); loadErr == nil {
			publicationAssetsToCheck = characterCardSnapshotAssetPaths(snapshot)
		}
		if snapshot, _, loadErr := loadCharacterCardSubmission(database.DB, card.ID); loadErr == nil {
			for asset := range characterCardSnapshotAssetPaths(snapshot) {
				publicationAssetsToCheck[asset] = struct{}{}
			}
		}
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureCharacterCardApprovedSnapshotBeforeMutation(tx, card); err != nil {
			return err
		}
		if portraitRowToDelete != nil {
			if err := tx.Delete(&model.CharacterCardPortrait{}, portraitRowToDelete.ID).Error; err != nil {
				return err
			}
			if err := persistCharacterCardPortraitSortRows(tx, portraitRows); err != nil {
				return err
			}
		}
		if portraitRowToUpdate != nil {
			if err := tx.Model(&model.CharacterCardPortrait{}).Where("id = ? AND character_card_id = ?", portraitRowToUpdate.ID, card.ID).
				Updates(map[string]interface{}{"image": portraitRowToUpdate.Image, "image_updated_at": portraitRowToUpdate.ImageUpdatedAt}).Error; err != nil {
				return err
			}
		}
		if portraitRowToCreate != nil {
			if err := tx.Create(portraitRowToCreate).Error; err != nil {
				return err
			}
		}
		if len(updates) > 0 {
			result := tx.Model(&model.CharacterCard{}).
				Where("id = ? AND user_id = ?", card.ID, userID).
				Updates(updates)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		}
		if impressionsNeedSave {
			if err := saveCharacterCardImpressions(tx, impressionPlan.Rows); err != nil {
				return err
			}
		}
		if candidate.Status != model.CharacterCardStatusPublished || candidate.Visibility != model.CharacterCardVisibilityPublic {
			if err := tx.Where("character_card_id = ?", card.ID).Delete(&model.CharacterCardSubmission{}).Error; err != nil {
				return err
			}
			if err := tx.Where("character_card_id = ?", card.ID).Delete(&model.CharacterCardPublication{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if newPortraitGenerated {
			s.cleanupOwnedCharacterCardPortrait(c, userID, candidate.PortraitImage)
		}
		s.cleanupGeneratedCharacterCardImpressionImages(c, userID, impressionPlan)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新人物卡失败"})
		}
		return
	}

	if portraitSet && candidate.PortraitImage != card.PortraitImage {
		s.cleanupCharacterCardAssetIfUnreferenced(c, userID, card.ID, card.PortraitImage)
	}
	if portraitRowToDelete != nil {
		s.cleanupCharacterCardAssetIfUnreferenced(c, userID, card.ID, portraitRowToDelete.Image)
	}
	if pendingPortraitToCleanup != "" {
		s.cleanupOwnedCharacterCardPendingPortrait(c, userID, pendingPortraitToCleanup)
	}
	s.finishCharacterCardImpressionImageUpdate(c, userID, card.ID, impressionPlan)
	for asset := range publicationAssetsToCheck {
		s.cleanupCharacterCardAssetIfUnreferenced(c, userID, card.ID, asset)
	}
	if err := database.DB.First(&card, card.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
		return
	}
	impressionsByCard, err = loadCharacterCardImpressions(database.DB, []uint{card.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡第一印象失败"})
		return
	}
	portraitsByCard, err := loadCharacterCardPortraits(database.DB, []uint{card.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡角色大图失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"character_card": s.buildCharacterCardDTO(card, impressionsByCard[card.ID], portraitsByCard[card.ID], true, true)})
}

func (s *Server) deleteCharacterCard(c *gin.Context) {
	id, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	userID := c.GetUint("userID")

	var card model.CharacterCard
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&card).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
		}
		return
	}
	impressionsByCard, err := loadCharacterCardImpressions(database.DB, []uint{card.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡第一印象失败"})
		return
	}
	impressions := impressionsByCard[card.ID]
	portraits, err := loadCharacterCardPortraitRows(database.DB, card.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡角色大图失败"})
		return
	}
	assets := map[string]struct{}{}
	addAsset := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			assets[value] = struct{}{}
		}
	}
	addAsset(card.PortraitImage)
	for _, portrait := range portraits {
		addAsset(portrait.Image)
	}
	for _, impression := range impressions {
		addAsset(impression.IconImage)
		addAsset(impression.Image)
	}
	if snapshot, _, loadErr := loadCharacterCardPublication(database.DB, card.ID); loadErr == nil {
		for asset := range characterCardSnapshotAssetPaths(snapshot) {
			addAsset(asset)
		}
	}
	if snapshot, _, loadErr := loadCharacterCardSubmission(database.DB, card.ID); loadErr == nil {
		for asset := range characterCardSnapshotAssetPaths(snapshot) {
			addAsset(asset)
		}
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable(&model.StoryEntry{}) {
			if err := tx.Model(&model.StoryEntry{}).
				Where("character_card_id = ?", card.ID).
				Update("character_card_id", nil).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("character_card_id = ?", card.ID).Delete(&model.CharacterCardPublication{}).Error; err != nil {
			return err
		}
		if err := tx.Where("character_card_id = ?", card.ID).Delete(&model.CharacterCardSubmission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("character_card_id = ?", card.ID).Delete(&model.CharacterCardPortrait{}).Error; err != nil {
			return err
		}
		if err := tx.Where("character_card_id = ?", card.ID).Delete(&model.CharacterCardImpression{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", card.ID, userID).Delete(&model.CharacterCard{}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除人物卡失败"})
		return
	}

	// Posts contain stable IDs in their rich text and are intentionally not
	// rewritten or deleted when a card disappears.
	for asset := range assets {
		s.cleanupCharacterCardAssetIfUnreferenced(c, userID, card.ID, asset)
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (s *Server) syncCharacterCardFromTRP3(c *gin.Context) {
	id, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	userID := c.GetUint("userID")

	var card model.CharacterCard
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&card).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errCharacterCardSourceNotFound
			}
			return err
		}
		if card.SourceBackupID == nil || *card.SourceBackupID == 0 || card.SourceProfileID == "" || card.SourceAccountID == "" {
			return errCharacterCardSourceNotFound
		}

		_, profile, err := loadCharacterCardBackupProfile(tx, userID, *card.SourceBackupID, card.SourceProfileID, card.SourceAccountID)
		if err != nil {
			return err
		}
		candidate := card
		applyTRP3CharacterCardFields(&candidate, profile.Fields)
		if err := setCharacterCardTRP3Details(&candidate, profile.AdditionalInfo, profile.PersonalityTraits); err != nil {
			return fmt.Errorf("%w: %v", errCharacterCardSourceCorrupt, err)
		}
		if err := validateCharacterCard(candidate); err != nil {
			return fmt.Errorf("%w: %v", errCharacterCardSourceCorrupt, err)
		}

		updates := trp3CharacterCardUpdateMap(profile.Fields)
		updates["trp3_additional_info_json"] = candidate.TRP3AdditionalInfoJSON
		updates["trp3_personality_json"] = candidate.TRP3PersonalityJSON
		updates["updated_at"] = time.Now().UTC()
		if err := ensureCharacterCardApprovedSnapshotBeforeMutation(tx, card); err != nil {
			return err
		}
		if err := tx.Model(&model.CharacterCard{}).
			Where("id = ? AND user_id = ?", card.ID, userID).
			Updates(updates).Error; err != nil {
			return err
		}
		existingImpressions, err := loadCharacterCardImpressions(tx, []uint{card.ID})
		if err != nil {
			return err
		}
		rows := fixedCharacterCardImpressions(card.ID, existingImpressions[card.ID])
		for index, request := range profile.Impressions {
			rows[index].Active = request.Active
			rows[index].Title = request.Title
			rows[index].Text = request.Text
			rows[index].TRP3Icon = request.TRP3Icon
		}
		if err := saveCharacterCardImpressions(tx, rows); err != nil {
			return err
		}
		return tx.First(&card, card.ID).Error
	})
	if err != nil {
		switch {
		case errors.Is(err, errCharacterCardSourceNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "人物卡或备份来源不存在"})
		case errors.Is(err, errCharacterCardSourceCorrupt):
			c.JSON(http.StatusBadRequest, gin.H{"error": "备份人物卡数据损坏"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "同步人物卡失败"})
		}
		return
	}

	impressionsByCard, loadErr := loadCharacterCardImpressions(database.DB, []uint{card.ID})
	if loadErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡第一印象失败"})
		return
	}
	portraitsByCard, err := loadCharacterCardPortraits(database.DB, []uint{card.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡角色大图失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"character_card": s.buildCharacterCardDTO(card, impressionsByCard[card.ID], portraitsByCard[card.ID], true, true)})
}

func (s *Server) listPublicUserCharacterCards(c *gin.Context) {
	userID64, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || userID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户 ID 无效"})
		return
	}
	userID := uint(userID64)

	var user model.User
	if err := database.DB.Select("id", "account_deleted_at").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		}
		return
	}
	if user.AccountDeletedAt != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	var cards []model.CharacterCard
	if err := database.DB.Where("user_id = ?", userID).
		Order("sort_order ASC, updated_at DESC, id DESC").
		Find(&cards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询人物卡失败"})
		return
	}
	result := make([]characterCardDTO, 0, len(cards))
	for _, card := range cards {
		dto, visible, err := s.loadPublicCharacterCardDTO(card, false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询人物卡失败"})
			return
		}
		if visible {
			result = append(result, dto)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].SortOrder != result[j].SortOrder {
			return result[i].SortOrder < result[j].SortOrder
		}
		if !result[i].UpdatedAt.Equal(result[j].UpdatedAt) {
			return result[i].UpdatedAt.After(result[j].UpdatedAt)
		}
		return result[i].ID > result[j].ID
	})
	c.JSON(http.StatusOK, gin.H{"character_cards": result})
}

func (s *Server) buildCharacterCardDTO(card model.CharacterCard, impressions []model.CharacterCardImpression, portraits []model.CharacterCardPortrait, ownerView, includeRichText bool) characterCardDTO {
	displayName := card.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName = joinedCharacterCardName(card.FirstName, card.LastName)
	}
	dto := characterCardDTO{
		ID:                     card.ID,
		UserID:                 card.UserID,
		FirstName:              card.FirstName,
		LastName:               card.LastName,
		DisplayName:            displayName,
		Title:                  card.Title,
		FullTitle:              card.FullTitle,
		Race:                   card.Race,
		Class:                  card.Class,
		EyeColor:               card.EyeColor,
		EyeColorHex:            card.EyeColorHex,
		Age:                    card.Age,
		Height:                 card.Height,
		Weight:                 card.Weight,
		Birthplace:             card.Birthplace,
		Residence:              card.Residence,
		RelationshipStatus:     card.RelationshipStatus,
		Icon:                   card.Icon,
		ClassColor:             canonicalCharacterCardTRP3Color(card.ClassColor, card.NameColor),
		NameColor:              canonicalCharacterCardTRP3Color(card.ClassColor, card.NameColor),
		AdditionalInfo:         characterCardTRP3AdditionalInfoFromCard(card),
		PersonalityTraits:      characterCardTRP3PersonalityTraitsFromCard(card),
		Summary:                card.Summary,
		Impressions:            s.buildCharacterCardImpressionDTOs(card, impressions, ownerView),
		Portraits:              s.buildCharacterCardPortraitDTOs(card, portraits),
		PortraitImageURL:       s.characterCardPortraitURL(card),
		PortraitImageUpdatedAt: card.PortraitImageUpdatedAt,
		Status:                 card.Status,
		Visibility:             card.Visibility,
		SortOrder:              card.SortOrder,
		ReviewStatus:           normalizedCharacterCardReviewStatus(card.ReviewStatus),
		CreatedAt:              card.CreatedAt,
		UpdatedAt:              card.UpdatedAt,
	}
	if ownerView {
		dto.CharacterID = card.CharacterID
		dto.SourceBackupID = card.SourceBackupID
		dto.SourceAccountID = card.SourceAccountID
		dto.SourceProfileID = card.SourceProfileID
		dto.ReviewComment = card.ReviewComment
		dto.ReviewedAt = card.ReviewedAt
	}
	if includeRichText {
		dto.BackgroundStory = card.BackgroundStory
		dto.FirstImpression = card.FirstImpression
		dto.OtherContent = card.OtherContent
	}
	return dto
}

func (s *Server) characterCardPortraitURL(card model.CharacterCard) string {
	if strings.TrimSpace(card.PortraitImage) == "" {
		return ""
	}
	versionTime := card.UpdatedAt
	if card.PortraitImageUpdatedAt != nil {
		versionTime = *card.PortraitImageUpdatedAt
	}
	imagePath := fmt.Sprintf("/api/v1/images/character-card-portrait/%d?v=%d", card.ID, versionTime.UnixNano())
	return buildAPIURL(s.cfg.Server.ApiHost, imagePath)
}

func parseCharacterCardID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "人物卡 ID 无效"})
		return 0, false
	}
	return uint(id), true
}

func canViewCharacterCard(card model.CharacterCard, viewerID uint) bool {
	if viewerID != 0 && card.UserID == viewerID {
		return true
	}
	status := normalizedCharacterCardReviewStatus(card.ReviewStatus)
	return card.Status == model.CharacterCardStatusPublished && card.Visibility == model.CharacterCardVisibilityPublic &&
		status == model.CharacterCardReviewApproved
}

func optionalActiveUserID(c *gin.Context) uint {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" || strings.TrimSpace(parts[1]) == "" {
		return 0
	}
	claims, err := authpkg.ParseToken(strings.TrimSpace(parts[1]))
	if err != nil || claims.UserID == 0 {
		return 0
	}
	userID := claims.UserID
	var user model.User
	if err := database.DB.Select("id", "account_deleted_at").First(&user, userID).Error; err != nil || user.AccountDeletedAt != nil {
		return 0
	}
	return userID
}

func loadCharacterCardBackupProfile(tx *gorm.DB, userID, backupID uint, profileID, expectedAccountID string) (model.AccountBackup, parsedTRP3CharacterCardProfile, error) {
	var backup model.AccountBackup
	if err := tx.Where("id = ? AND user_id = ?", backupID, userID).
		Select("id", "user_id", "account_id", "profiles_data").
		First(&backup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return backup, parsedTRP3CharacterCardProfile{}, errCharacterCardSourceNotFound
		}
		return backup, parsedTRP3CharacterCardProfile{}, err
	}
	if expectedAccountID != "" && backup.AccountID != expectedAccountID {
		return backup, parsedTRP3CharacterCardProfile{}, errCharacterCardSourceNotFound
	}
	if err := validateCharacterCardSourceAccountID(backup.AccountID); err != nil {
		return backup, parsedTRP3CharacterCardProfile{}, fmt.Errorf("%w: %v", errCharacterCardSourceCorrupt, err)
	}

	profiles, err := decodeCharacterCardProfileMap(backup.ProfilesData)
	if err != nil {
		return backup, parsedTRP3CharacterCardProfile{}, fmt.Errorf("%w: %v", errCharacterCardSourceCorrupt, err)
	}
	raw, exists := profiles[profileID]
	if !exists {
		return backup, parsedTRP3CharacterCardProfile{}, errCharacterCardSourceNotFound
	}
	profile, err := parseTRP3CharacterCardProfile(raw)
	if err != nil {
		return backup, parsedTRP3CharacterCardProfile{}, fmt.Errorf("%w: %v", errCharacterCardSourceCorrupt, err)
	}
	return backup, profile, nil
}

func decodeCharacterCardProfileMap(raw string) (map[string]json.RawMessage, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("empty profiles_data")
	}
	var profiles map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &profiles); err != nil || profiles == nil {
		if err == nil {
			err = errors.New("profiles_data is not an object")
		}
		return nil, err
	}

	// Accept the short-lived legacy wrapper without weakening exact profile ID
	// lookup for the current map-shaped AccountBackup payload.
	if len(profiles) == 1 {
		if wrapped, exists := profiles["profiles"]; exists {
			var nested map[string]json.RawMessage
			if err := json.Unmarshal(wrapped, &nested); err == nil && nested != nil {
				return nested, nil
			}
		}
	}
	return profiles, nil
}

func parseTRP3CharacterCardProfile(raw json.RawMessage) (parsedTRP3CharacterCardProfile, error) {
	var profileEnvelope struct {
		ProfileName string          `json:"profileName"`
		Player      json.RawMessage `json:"player"`
	}
	if err := json.Unmarshal(raw, &profileEnvelope); err != nil {
		return parsedTRP3CharacterCardProfile{}, err
	}
	if len(profileEnvelope.Player) == 0 || string(profileEnvelope.Player) == "null" {
		return parsedTRP3CharacterCardProfile{}, errors.New("missing player data")
	}
	var player struct {
		Characteristics json.RawMessage `json:"characteristics"`
		Misc            json.RawMessage `json:"misc"`
	}
	if err := json.Unmarshal(profileEnvelope.Player, &player); err != nil {
		return parsedTRP3CharacterCardProfile{}, err
	}
	if len(player.Characteristics) == 0 || string(player.Characteristics) == "null" {
		return parsedTRP3CharacterCardProfile{}, errors.New("missing player.characteristics")
	}
	var characteristics map[string]json.RawMessage
	if err := json.Unmarshal(player.Characteristics, &characteristics); err != nil || characteristics == nil {
		if err == nil {
			err = errors.New("player.characteristics is not an object")
		}
		return parsedTRP3CharacterCardProfile{}, err
	}

	read := func(name string) (string, error) {
		value, exists := characteristics[name]
		if !exists {
			return "", nil
		}
		return characterCardTRP3Scalar(name, value)
	}
	values := make(map[string]string, 16)
	for _, name := range []string{"FN", "LN", "TI", "FT", "RA", "CL", "EC", "EH", "AG", "HE", "WE", "BP", "RE", "RS", "IC", "CH"} {
		value, err := read(name)
		if err != nil {
			return parsedTRP3CharacterCardProfile{}, err
		}
		values[name] = value
	}
	// PE is an optional, independently authored TRP3 section. Ignore malformed
	// glance data instead of blocking basic-field import or sync.
	impressions, err := parseTRP3CharacterCardImpressions(player.Misc)
	if err != nil {
		impressions, _ = parseTRP3CharacterCardImpressions(nil)
	}
	additionalInfo, err := parseTRP3CharacterCardAdditionalInfo(characteristics["MI"])
	if err != nil {
		return parsedTRP3CharacterCardProfile{}, err
	}
	personalityTraits, err := parseTRP3CharacterCardPersonalityTraits(characteristics["PS"])
	if err != nil {
		return parsedTRP3CharacterCardProfile{}, err
	}

	return parsedTRP3CharacterCardProfile{
		ProfileName: strings.TrimSpace(profileEnvelope.ProfileName),
		Fields: trp3CharacterCardFields{
			FirstName:          values["FN"],
			LastName:           values["LN"],
			Title:              values["TI"],
			FullTitle:          values["FT"],
			Race:               values["RA"],
			Class:              values["CL"],
			EyeColor:           values["EC"],
			EyeColorHex:        values["EH"],
			Age:                values["AG"],
			Height:             values["HE"],
			Weight:             values["WE"],
			Birthplace:         values["BP"],
			Residence:          values["RE"],
			RelationshipStatus: values["RS"],
			Icon:               values["IC"],
			ClassColor:         values["CH"],
			NameColor:          values["CH"],
		},
		Impressions:       impressions,
		AdditionalInfo:    additionalInfo,
		PersonalityTraits: personalityTraits,
	}, nil
}

func parseTRP3CharacterCardAdditionalInfo(raw json.RawMessage) ([]characterCardTRP3AdditionalInfo, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return []characterCardTRP3AdditionalInfo{}, nil
	}
	var source []struct {
		ID int    `json:"ID"`
		NA string `json:"NA"`
		VA string `json:"VA"`
		IC string `json:"IC"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, errors.New("player.characteristics.MI is not an array")
	}
	result := make([]characterCardTRP3AdditionalInfo, 0, len(source))
	for _, item := range source {
		id := item.ID
		if id < 1 || id > 11 {
			id = 1
		}
		result = append(result, characterCardTRP3AdditionalInfo{ID: id, Name: item.NA, Value: item.VA, Icon: item.IC})
	}
	if err := validateCharacterCardTRP3AdditionalInfo(result); err != nil {
		return nil, err
	}
	return result, nil
}

func parseTRP3CharacterCardPersonalityTraits(raw json.RawMessage) ([]characterCardTRP3PersonalityTrait, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return []characterCardTRP3PersonalityTrait{}, nil
	}
	var source []struct {
		ID *int                    `json:"ID"`
		LT string                  `json:"LT"`
		RT string                  `json:"RT"`
		LI string                  `json:"LI"`
		RI string                  `json:"RI"`
		LC *characterCardTRP3Color `json:"LC"`
		RC *characterCardTRP3Color `json:"RC"`
		V2 *int                    `json:"V2"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, errors.New("player.characteristics.PS is not an array")
	}
	result := make([]characterCardTRP3PersonalityTrait, 0, len(source))
	for _, item := range source {
		value := 10
		if item.V2 != nil {
			value = *item.V2
		}
		result = append(result, characterCardTRP3PersonalityTrait{
			PresetID: item.ID, LeftText: item.LT, RightText: item.RT,
			LeftIcon: item.LI, RightIcon: item.RI, LeftColor: item.LC, RightColor: item.RC, Value: value,
		})
	}
	if err := validateCharacterCardTRP3PersonalityTraits(result); err != nil {
		return nil, err
	}
	return result, nil
}

func parseTRP3CharacterCardImpressions(rawMisc json.RawMessage) ([]characterCardImpressionRequest, error) {
	requests := make([]characterCardImpressionRequest, 0, characterCardImpressionSlotCount)
	for slot := 1; slot <= characterCardImpressionSlotCount; slot++ {
		requests = append(requests, characterCardImpressionRequest{Slot: uint8(slot)})
	}
	if len(rawMisc) == 0 || string(bytes.TrimSpace(rawMisc)) == "null" {
		return requests, nil
	}
	var misc map[string]json.RawMessage
	if err := json.Unmarshal(rawMisc, &misc); err != nil || misc == nil {
		if err == nil {
			err = errors.New("player.misc is not an object")
		}
		return nil, err
	}
	peRaw, exists := misc["PE"]
	if !exists || len(peRaw) == 0 {
		return requests, nil
	}
	trimmedPE := bytes.TrimSpace(peRaw)
	if bytes.Equal(trimmedPE, []byte("null")) || bytes.Equal(trimmedPE, []byte("[]")) {
		return requests, nil
	}
	var slots map[string]json.RawMessage
	if err := json.Unmarshal(peRaw, &slots); err != nil || slots == nil {
		if err == nil {
			err = errors.New("player.misc.PE is not an object")
		}
		return nil, err
	}
	for index := range requests {
		slotKey := strconv.Itoa(index + 1)
		slotRaw, exists := slots[slotKey]
		if !exists || len(slotRaw) == 0 || string(bytes.TrimSpace(slotRaw)) == "null" {
			continue
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(slotRaw, &fields); err != nil || fields == nil {
			continue
		}
		candidate := characterCardImpressionRequest{Slot: uint8(index + 1)}
		if activeRaw, exists := fields["AC"]; exists && !bytes.Equal(bytes.TrimSpace(activeRaw), []byte("null")) {
			if err := json.Unmarshal(activeRaw, &candidate.Active); err != nil {
				continue
			}
		}
		read := func(name string) (string, error) {
			value, exists := fields[name]
			if !exists {
				return "", nil
			}
			return characterCardTRP3Scalar("PE."+slotKey+"."+name, value)
		}
		var err error
		if candidate.TRP3Icon, err = read("IC"); err != nil {
			continue
		}
		if candidate.Title, err = read("TI"); err != nil {
			continue
		}
		if candidate.Text, err = read("TX"); err != nil {
			continue
		}
		candidate.TRP3Icon = truncateCharacterCardRunes(candidate.TRP3Icon, 128)
		candidate.Title = truncateCharacterCardRunes(candidate.Title, characterCardImpressionTitleMax)
		candidate.Text = truncateCharacterCardRunes(candidate.Text, characterCardImpressionTextMax)
		requests[index] = candidate
	}
	return validateCharacterCardImpressionRequests(requests)
}

func truncateCharacterCardRunes(value string, maxRunes int) string {
	if maxRunes < 0 || utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxRunes])
}

func characterCardTRP3Scalar(field string, raw json.RawMessage) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	switch typed := value.(type) {
	case nil:
		return "", nil
	case string:
		return strings.TrimSpace(typed), nil
	case json.Number:
		return typed.String(), nil
	default:
		return "", fmt.Errorf("TRP3 field %s is not a scalar", field)
	}
}

func applyTRP3CharacterCardFields(card *model.CharacterCard, fields trp3CharacterCardFields) {
	card.FirstName = fields.FirstName
	card.LastName = fields.LastName
	card.Title = fields.Title
	card.FullTitle = fields.FullTitle
	card.Race = fields.Race
	card.Class = fields.Class
	card.EyeColor = fields.EyeColor
	card.EyeColorHex = fields.EyeColorHex
	card.Age = fields.Age
	card.Height = fields.Height
	card.Weight = fields.Weight
	card.Birthplace = fields.Birthplace
	card.Residence = fields.Residence
	card.RelationshipStatus = fields.RelationshipStatus
	card.Icon = fields.Icon
	card.ClassColor = fields.ClassColor
	card.NameColor = fields.NameColor
}

func setCharacterCardTRP3Details(card *model.CharacterCard, additionalInfo []characterCardTRP3AdditionalInfo, personality []characterCardTRP3PersonalityTrait) error {
	if err := validateCharacterCardTRP3AdditionalInfo(additionalInfo); err != nil {
		return err
	}
	if err := validateCharacterCardTRP3PersonalityTraits(personality); err != nil {
		return err
	}
	additionalJSON, err := json.Marshal(additionalInfo)
	if err != nil {
		return err
	}
	personalityJSON, err := json.Marshal(personality)
	if err != nil {
		return err
	}
	card.TRP3AdditionalInfoJSON = string(additionalJSON)
	card.TRP3PersonalityJSON = string(personalityJSON)
	return nil
}

func characterCardTRP3AdditionalInfoFromCard(card model.CharacterCard) []characterCardTRP3AdditionalInfo {
	result := []characterCardTRP3AdditionalInfo{}
	if strings.TrimSpace(card.TRP3AdditionalInfoJSON) == "" {
		return result
	}
	if err := json.Unmarshal([]byte(card.TRP3AdditionalInfoJSON), &result); err != nil || result == nil {
		return []characterCardTRP3AdditionalInfo{}
	}
	return result
}

func characterCardTRP3PersonalityTraitsFromCard(card model.CharacterCard) []characterCardTRP3PersonalityTrait {
	result := []characterCardTRP3PersonalityTrait{}
	if strings.TrimSpace(card.TRP3PersonalityJSON) == "" {
		return result
	}
	if err := json.Unmarshal([]byte(card.TRP3PersonalityJSON), &result); err != nil || result == nil {
		return []characterCardTRP3PersonalityTrait{}
	}
	return result
}

func validateCharacterCardTRP3AdditionalInfo(items []characterCardTRP3AdditionalInfo) error {
	if len(items) > 50 {
		return errors.New("附加说明不能超过 50 项")
	}
	for _, item := range items {
		if item.ID < 1 || item.ID > 11 {
			return errors.New("附加说明类型必须是 1 到 11")
		}
		for _, field := range []struct {
			name  string
			value string
			max   int
		}{{"附加说明名称", item.Name, 80}, {"附加说明内容", item.Value, 500}, {"附加说明图标", item.Icon, 128}} {
			if err := validatePlainCharacterCardField(field.name, field.value, field.max); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateCharacterCardTRP3PersonalityTraits(items []characterCardTRP3PersonalityTrait) error {
	if len(items) > 50 {
		return errors.New("个性不能超过 50 项")
	}
	for _, item := range items {
		if item.Value < 0 || item.Value > 20 {
			return errors.New("个性倾向值必须是 0 到 20")
		}
		if item.PresetID != nil && (*item.PresetID < 1 || *item.PresetID > 11) {
			return errors.New("个性预设必须是 1 到 11")
		}
		for _, field := range []struct {
			name  string
			value string
			max   int
		}{{"个性左侧名称", item.LeftText, 80}, {"个性右侧名称", item.RightText, 80}, {"个性左侧图标", item.LeftIcon, 128}, {"个性右侧图标", item.RightIcon, 128}} {
			if err := validatePlainCharacterCardField(field.name, field.value, field.max); err != nil {
				return err
			}
		}
		for _, color := range []*characterCardTRP3Color{item.LeftColor, item.RightColor} {
			if color == nil {
				continue
			}
			if math.IsNaN(color.R) || math.IsNaN(color.G) || math.IsNaN(color.B) ||
				math.IsInf(color.R, 0) || math.IsInf(color.G, 0) || math.IsInf(color.B, 0) ||
				color.R < 0 || color.R > 1 || color.G < 0 || color.G > 1 || color.B < 0 || color.B > 1 {
				return errors.New("个性颜色分量必须在 0 到 1 之间")
			}
		}
		if item.PresetID == nil && strings.TrimSpace(item.LeftText) == "" && strings.TrimSpace(item.RightText) == "" {
			return errors.New("自定义个性必须填写至少一个倾向名称")
		}
	}
	return nil
}

func trp3CharacterCardUpdateMap(fields trp3CharacterCardFields) map[string]interface{} {
	return map[string]interface{}{
		"first_name":          fields.FirstName,
		"last_name":           fields.LastName,
		"title":               fields.Title,
		"full_title":          fields.FullTitle,
		"race":                fields.Race,
		"class":               fields.Class,
		"eye_color":           fields.EyeColor,
		"eye_color_hex":       fields.EyeColorHex,
		"age":                 fields.Age,
		"height":              fields.Height,
		"weight":              fields.Weight,
		"birthplace":          fields.Birthplace,
		"residence":           fields.Residence,
		"relationship_status": fields.RelationshipStatus,
		"icon":                fields.Icon,
		"class_color":         fields.ClassColor,
		"name_color":          fields.NameColor,
	}
}

func importedCharacterCardDisplayName(profile parsedTRP3CharacterCardProfile) string {
	if name := joinedCharacterCardName(profile.Fields.FirstName, profile.Fields.LastName); name != "" {
		return name
	}
	return strings.TrimSpace(profile.ProfileName)
}

func joinedCharacterCardName(firstName, lastName string) string {
	parts := make([]string, 0, 2)
	if firstName = strings.TrimSpace(firstName); firstName != "" {
		parts = append(parts, firstName)
	}
	if lastName = strings.TrimSpace(lastName); lastName != "" {
		parts = append(parts, lastName)
	}
	return strings.Join(parts, " ")
}

func applyCharacterCardStringUpdate(input *string, target *string, column string, updates map[string]interface{}, trim bool) {
	if input == nil {
		return
	}
	value := *input
	if trim {
		value = strings.TrimSpace(value)
	}
	*target = value
	updates[column] = value
}

func canonicalCharacterCardTRP3Color(classColor, nameColor string) string {
	value := strings.TrimSpace(classColor)
	if value == "" {
		value = strings.TrimSpace(nameColor)
	}
	value = strings.TrimPrefix(value, "#")
	return strings.ToUpper(value)
}

func applyCharacterCardTRP3ColorUpdate(classColor, nameColor *string, card *model.CharacterCard, updates map[string]interface{}) error {
	if classColor == nil && nameColor == nil {
		canonical := canonicalCharacterCardTRP3Color(card.ClassColor, card.NameColor)
		card.ClassColor = canonical
		card.NameColor = canonical
		return nil
	}
	classValue := ""
	nameValue := ""
	if classColor != nil {
		classValue = canonicalCharacterCardTRP3Color(*classColor, "")
	}
	if nameColor != nil {
		nameValue = canonicalCharacterCardTRP3Color(*nameColor, "")
	}
	if classColor != nil && nameColor != nil && classValue != nameValue {
		return errors.New("class_color 与 name_color 必须一致（TRP3 仅有一个 CH 颜色）")
	}
	canonical := classValue
	if classColor == nil {
		canonical = nameValue
	}
	card.ClassColor = canonical
	card.NameColor = canonical
	updates["class_color"] = canonical
	updates["name_color"] = canonical
	return nil
}

func validateCharacterCard(card model.CharacterCard) error {
	fields := []struct {
		name  string
		value string
		max   int
	}{
		{"first_name", card.FirstName, 128},
		{"last_name", card.LastName, 128},
		{"display_name", card.DisplayName, 256},
		{"title", card.Title, 128},
		{"full_title", card.FullTitle, 256},
		{"race", card.Race, 64},
		{"class", card.Class, 64},
		{"eye_color", card.EyeColor, 64},
		{"eye_color_hex", card.EyeColorHex, 16},
		{"age", card.Age, 64},
		{"height", card.Height, 64},
		{"weight", card.Weight, 64},
		{"birthplace", card.Birthplace, 256},
		{"residence", card.Residence, 256},
		{"relationship_status", card.RelationshipStatus, 64},
		{"icon", card.Icon, 128},
		{"class_color", card.ClassColor, 16},
		{"name_color", card.NameColor, 16},
		{"summary", card.Summary, 1000},
		{"source_account_id", card.SourceAccountID, 32},
		{"source_profile_id", card.SourceProfileID, 128},
	}
	for _, field := range fields {
		if err := validatePlainCharacterCardField(field.name, field.value, field.max); err != nil {
			return err
		}
	}
	if card.EyeColorHex != "" && !isCharacterCardHexColor(card.EyeColorHex) {
		return errors.New("eye_color_hex 必须是 6 或 8 位十六进制颜色")
	}
	if card.NameColor != "" && !isCharacterCardHexColor(card.NameColor) {
		return errors.New("name_color 必须是 6 或 8 位十六进制颜色")
	}
	if card.ClassColor != "" && !isCharacterCardHexColor(card.ClassColor) {
		return errors.New("class_color 必须是 6 或 8 位十六进制颜色")
	}
	if canonicalCharacterCardTRP3Color(card.ClassColor, "") != canonicalCharacterCardTRP3Color("", card.NameColor) {
		return errors.New("class_color 与 name_color 必须一致（TRP3 仅有一个 CH 颜色）")
	}
	if card.SourceAccountID != "" {
		if err := validateCharacterCardSourceAccountID(card.SourceAccountID); err != nil {
			return err
		}
	}
	if card.Status != model.CharacterCardStatusDraft && card.Status != model.CharacterCardStatusPublished {
		return errors.New("status 仅支持 draft 或 published")
	}
	if card.Visibility != model.CharacterCardVisibilityPrivate && card.Visibility != model.CharacterCardVisibilityPublic {
		return errors.New("visibility 仅支持 private 或 public")
	}
	if card.Status == model.CharacterCardStatusPublished && strings.TrimSpace(card.DisplayName) == "" && joinedCharacterCardName(card.FirstName, card.LastName) == "" {
		return errors.New("发布人物卡前请填写名称")
	}
	if card.SortOrder < -100000 || card.SortOrder > 100000 {
		return errors.New("sort_order 超出允许范围")
	}
	if err := validateCharacterCardTRP3AdditionalInfo(characterCardTRP3AdditionalInfoFromCard(card)); err != nil {
		return err
	}
	if err := validateCharacterCardTRP3PersonalityTraits(characterCardTRP3PersonalityTraitsFromCard(card)); err != nil {
		return err
	}

	richContents := []struct {
		name  string
		value string
	}{
		{"background_story", card.BackgroundStory},
		{"first_impression", card.FirstImpression},
		{"other_content", card.OtherContent},
	}
	totalBytes := 0
	for _, content := range richContents {
		if len(content.value) > characterCardRichTextMaxBytes {
			return fmt.Errorf("%s 内容过大", content.name)
		}
		totalBytes += len(content.value)
		if err := validateCharacterCardRichText(content.value); err != nil {
			return fmt.Errorf("%s 包含不安全内容: %w", content.name, err)
		}
	}
	if totalBytes > characterCardRichTextAllBytes {
		return errors.New("人物卡富文本总大小超出限制")
	}
	return nil
}

func validatePlainCharacterCardField(name, value string, maxRunes int) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s 不是有效文本", name)
	}
	if strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("%s 包含无效字符", name)
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return fmt.Errorf("%s 不能超过 %d 个字符", name, maxRunes)
	}
	return nil
}

func validateCharacterCardSourceAccountID(raw string) error {
	if raw == "" || strings.TrimSpace(raw) != raw || raw == "." || raw == ".." || strings.ContainsAny(raw, `/\`) {
		return errors.New("source_account_id 不是安全的账号目录名")
	}
	for _, char := range raw {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '#' || char == '_' || char == '-' {
			continue
		}
		return errors.New("source_account_id 不是安全的账号目录名")
	}
	return nil
}

func isCharacterCardHexColor(raw string) bool {
	value := strings.TrimPrefix(strings.TrimSpace(raw), "#")
	if len(value) != 6 && len(value) != 8 {
		return false
	}
	for _, char := range value {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

func validateCharacterCardRichText(content string) error {
	if content == "" {
		return nil
	}
	forbiddenTags := map[string]struct{}{
		"script": {}, "style": {}, "iframe": {}, "object": {}, "embed": {},
		"svg": {}, "math": {}, "form": {}, "input": {}, "button": {},
		"textarea": {}, "select": {}, "option": {}, "meta": {}, "link": {}, "base": {},
	}
	tokenizer := html.NewTokenizer(strings.NewReader(content))
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			if errors.Is(tokenizer.Err(), io.EOF) {
				return nil
			}
			return tokenizer.Err()
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			tagName := strings.ToLower(token.Data)
			if _, forbidden := forbiddenTags[tagName]; forbidden {
				return fmt.Errorf("不允许 <%s> 标签", tagName)
			}
			for _, attr := range token.Attr {
				name := strings.ToLower(strings.TrimSpace(attr.Key))
				if strings.HasPrefix(name, "on") || name == "srcdoc" || name == "formaction" {
					return fmt.Errorf("不允许 %s 属性", name)
				}
				if name == "href" || name == "src" || name == "xlink:href" {
					if err := validateCharacterCardRichTextURL(name, attr.Val); err != nil {
						return err
					}
				}
				if name == "style" {
					if err := validateCharacterCardRichTextStyle(attr.Val); err != nil {
						return err
					}
				}
			}
		}
	}
}

func validateCharacterCardRichTextStyle(raw string) error {
	for _, declaration := range strings.Split(raw, ";") {
		declaration = strings.TrimSpace(declaration)
		if declaration == "" {
			continue
		}
		property, value, found := strings.Cut(declaration, ":")
		if !found {
			return errors.New("style 属性格式无效")
		}
		property = strings.ToLower(strings.TrimSpace(property))
		value = strings.ToLower(strings.TrimSpace(value))
		if property != "text-align" {
			return fmt.Errorf("style 不允许 %s 属性", property)
		}
		switch value {
		case "left", "center", "right", "justify":
		default:
			return errors.New("text-align 使用了不允许的值")
		}
	}
	return nil
}

func validateCharacterCardRichTextURL(attribute, raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" || strings.HasPrefix(value, "#") {
		return nil
	}
	compact := strings.Map(func(r rune) rune {
		if r <= ' ' || r == '\u007f' {
			return -1
		}
		return r
	}, value)
	lower := strings.ToLower(compact)
	for _, dangerous := range []string{"javascript:", "vbscript:", "file:"} {
		if strings.HasPrefix(lower, dangerous) {
			return fmt.Errorf("%s 使用了不安全 URL", attribute)
		}
	}
	if strings.HasPrefix(lower, "data:") {
		if attribute != "src" || !(strings.HasPrefix(lower, "data:image/png;base64,") || strings.HasPrefix(lower, "data:image/jpeg;base64,") || strings.HasPrefix(lower, "data:image/gif;base64,") || strings.HasPrefix(lower, "data:image/webp;base64,")) {
			return fmt.Errorf("%s 使用了不允许的 data URL", attribute)
		}
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("%s URL 无效", attribute)
	}
	if parsed.Scheme == "" {
		return nil
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme == "http" || scheme == "https" {
		return nil
	}
	if attribute == "href" && (scheme == "mailto" || scheme == "tel") {
		return nil
	}
	return fmt.Errorf("%s URL 协议不受支持", attribute)
}

func requestedCharacterCardPortrait(req updateCharacterCardRequest) (string, bool, error) {
	if !req.PortraitImage.Set && !req.PortraitImageURL.Set {
		return "", false, nil
	}
	if req.PortraitImage.Set && req.PortraitImageURL.Set && strings.TrimSpace(req.PortraitImage.Value) != strings.TrimSpace(req.PortraitImageURL.Value) {
		return "", false, errors.New("portrait_image 与 portrait_image_url 不能冲突")
	}
	if req.PortraitImageURL.Set {
		return strings.TrimSpace(req.PortraitImageURL.Value), true, nil
	}
	return strings.TrimSpace(req.PortraitImage.Value), true, nil
}

type normalizedCharacterCardPortrait struct {
	Path          string
	Generated     bool
	PendingSource string
}

func (s *Server) normalizeCharacterCardPortrait(c *gin.Context, userID uint, card model.CharacterCard, raw string) (normalizedCharacterCardPortrait, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return normalizedCharacterCardPortrait{}, nil
	}
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		return normalizedCharacterCardPortrait{}, errors.New("请先使用人物卡肖像上传接口")
	}

	sourceValue := value
	currentProxy := s.isCurrentCharacterCardPortraitURL(c, card.ID, value)
	if currentProxy {
		sourceValue = strings.TrimSpace(card.PortraitImage)
		if sourceValue == "" {
			return normalizedCharacterCardPortrait{}, errors.New("角色大图不存在")
		}
	}

	canonical, kind, owned := s.characterCardOwnedPortraitStoragePath(c, userID, sourceValue)
	if owned && kind == characterCardPortraitStorageCurrent && canonical == card.PortraitImage {
		return normalizedCharacterCardPortrait{Path: canonical}, nil
	}
	if !owned {
		if !currentProxy {
			return normalizedCharacterCardPortrait{}, errors.New("角色大图只接受本人的 pending 或已归档引用")
		}
		// A proxy URL can refer to a short-lived legacy generic upload already
		// stored on this card. Migrate that trusted database value once, but never
		// accept a generic upload path directly from the request.
		var ok bool
		canonical, ok = s.characterCardInternalUploadPath(c, sourceValue)
		if !ok || isCharacterCardProtectedUploadPath(canonical) {
			return normalizedCharacterCardPortrait{}, errors.New("角色大图存储引用无效")
		}
	}
	data, contentType, err := s.loadImageBytes(c, canonical)
	if err != nil {
		return normalizedCharacterCardPortrait{}, errors.New("角色大图文件不存在")
	}
	detectedType, err := validateCharacterCardPortraitBytes(data, contentType)
	if err != nil {
		return normalizedCharacterCardPortrait{}, err
	}
	saved, err := s.saveImageBytes(c, data, detectedType, characterCardPortraitCurrentSubdir(userID))
	if err != nil {
		return normalizedCharacterCardPortrait{}, errors.New("保存角色大图失败")
	}
	protectedPath, protectedKind, ok := s.characterCardOwnedPortraitStoragePath(c, userID, saved)
	if !ok || protectedKind != characterCardPortraitStorageCurrent {
		if key := extractUploadKey(c, saved); key != "" {
			s.deleteUploadKey(key)
		}
		return normalizedCharacterCardPortrait{}, errors.New("保存角色大图失败")
	}
	result := normalizedCharacterCardPortrait{Path: protectedPath, Generated: true}
	if kind == characterCardPortraitStoragePending {
		result.PendingSource = canonical
	}
	return result, nil
}

func isCharacterCardProtectedUploadPath(raw string) bool {
	policyPath := path.Clean(strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), `\`, "/")))
	return strings.HasPrefix(policyPath, "/uploads/character-cards/")
}

func (s *Server) characterCardInternalUploadPath(c *gin.Context, raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	if strings.HasPrefix(value, "/uploads/") || strings.HasPrefix(value, "uploads/") {
		key := uploadsKeyFromPath(stripURLParams(value))
		if key == "" {
			return "", false
		}
		return path.Join("/uploads", key), true
	}

	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", false
	}
	allowedHost := isSameHost(c, parsed.Host)
	if !allowedHost && s != nil && s.cfg != nil && strings.TrimSpace(s.cfg.Server.ApiHost) != "" {
		if configured, parseErr := url.Parse(strings.TrimSpace(s.cfg.Server.ApiHost)); parseErr == nil {
			allowedHost = normalizedHost(configured.Host) == normalizedHost(parsed.Host)
		}
	}
	if !allowedHost {
		return "", false
	}
	key := uploadsKeyFromPath(parsed.Path)
	if key == "" {
		return "", false
	}
	return path.Join("/uploads", key), true
}

func (s *Server) isCurrentCharacterCardPortraitURL(c *gin.Context, cardID uint, raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	if parsed.IsAbs() {
		allowedHost := isSameHost(c, parsed.Host)
		if !allowedHost && s != nil && s.cfg != nil && strings.TrimSpace(s.cfg.Server.ApiHost) != "" {
			if configured, parseErr := url.Parse(strings.TrimSpace(s.cfg.Server.ApiHost)); parseErr == nil {
				allowedHost = normalizedHost(configured.Host) == normalizedHost(parsed.Host)
			}
		}
		if !allowedHost {
			return false
		}
	}
	expectedPath := fmt.Sprintf("/api/v1/images/character-card-portrait/%d", cardID)
	return path.Clean(parsed.Path) == expectedPath
}

func validateCharacterCardPortraitBytes(data []byte, declaredType string) (string, error) {
	return validateCharacterCardImageBytes(data, declaredType, "角色大图")
}

func validateCharacterCardImageBytes(data []byte, declaredType, label string) (string, error) {
	if label == "" {
		label = "图片"
	}
	if len(data) == 0 {
		return "", fmt.Errorf("%s不能为空", label)
	}
	if len(data) > characterCardPortraitMaxBytes {
		return "", fmt.Errorf("%s不能超过 20MB", label)
	}
	detectedType := strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0])
	if detectedType == "image/jpg" {
		detectedType = "image/jpeg"
	}
	allowed := map[string]struct{}{
		"image/jpeg": {}, "image/png": {}, "image/gif": {}, "image/webp": {},
	}
	if _, ok := allowed[detectedType]; !ok {
		return "", fmt.Errorf("%s仅支持 JPEG、PNG、GIF 或 WebP", label)
	}
	declaredType = strings.TrimSpace(strings.Split(declaredType, ";")[0])
	if declaredType == "image/jpg" {
		declaredType = "image/jpeg"
	}
	if declaredType != "" && declaredType != "application/octet-stream" && declaredType != detectedType {
		return "", fmt.Errorf("%s MIME 与文件内容不一致", label)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return "", fmt.Errorf("%s文件已损坏", label)
	}
	if config.Width > characterCardPortraitMaxSide || config.Height > characterCardPortraitMaxSide || int64(config.Width)*int64(config.Height) > characterCardPortraitMaxPixels {
		return "", fmt.Errorf("%s尺寸过大", label)
	}
	return detectedType, nil
}

func (s *Server) cleanupOwnedCharacterCardPortrait(c *gin.Context, userID uint, raw string) {
	canonical, kind, ok := s.characterCardOwnedPortraitStoragePath(c, userID, raw)
	if !ok || kind != characterCardPortraitStorageCurrent {
		return
	}
	if key := uploadsKeyFromPath(canonical); key != "" {
		s.deleteUploadKey(key)
	}
}
