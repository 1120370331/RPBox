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

	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	DisplayName        string `json:"display_name"`
	Title              string `json:"title"`
	FullTitle          string `json:"full_title"`
	Race               string `json:"race"`
	Class              string `json:"class"`
	EyeColor           string `json:"eye_color"`
	EyeColorHex        string `json:"eye_color_hex"`
	Age                string `json:"age"`
	Height             string `json:"height"`
	Weight             string `json:"weight"`
	Birthplace         string `json:"birthplace"`
	Residence          string `json:"residence"`
	RelationshipStatus string `json:"relationship_status"`
	Icon               string `json:"icon"`
	NameColor          string `json:"name_color"`

	Summary         string `json:"summary"`
	BackgroundStory string `json:"background_story,omitempty"`
	FirstImpression string `json:"first_impression,omitempty"`
	OtherContent    string `json:"other_content,omitempty"`

	PortraitImageURL       string     `json:"portrait_image_url"`
	PortraitImageUpdatedAt *time.Time `json:"portrait_image_updated_at"`
	Status                 string     `json:"status"`
	Visibility             string     `json:"visibility"`
	SortOrder              int        `json:"sort_order"`
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
	BackupUpdatedAt time.Time `json:"backup_updated_at"`
}

type createCharacterCardRequest struct {
	SourceType      string `json:"source_type"`
	SourceBackupID  *uint  `json:"source_backup_id"`
	SourceProfileID string `json:"source_profile_id"`
}

type updateCharacterCardRequest struct {
	CharacterID optionalCharacterCardUint `json:"character_id"`

	FirstName          *string `json:"first_name"`
	LastName           *string `json:"last_name"`
	DisplayName        *string `json:"display_name"`
	Title              *string `json:"title"`
	FullTitle          *string `json:"full_title"`
	Race               *string `json:"race"`
	Class              *string `json:"class"`
	EyeColor           *string `json:"eye_color"`
	EyeColorHex        *string `json:"eye_color_hex"`
	Age                *string `json:"age"`
	Height             *string `json:"height"`
	Weight             *string `json:"weight"`
	Birthplace         *string `json:"birthplace"`
	Residence          *string `json:"residence"`
	RelationshipStatus *string `json:"relationship_status"`
	Icon               *string `json:"icon"`
	NameColor          *string `json:"name_color"`

	Summary         *string `json:"summary"`
	BackgroundStory *string `json:"background_story"`
	FirstImpression *string `json:"first_impression"`
	OtherContent    *string `json:"other_content"`

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
	NameColor          string
}

type parsedTRP3CharacterCardProfile struct {
	ProfileName string
	Fields      trp3CharacterCardFields
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

	result := make([]characterCardDTO, 0, len(cards))
	for _, card := range cards {
		result = append(result, s.buildCharacterCardDTO(card, true, false))
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
		if err := database.DB.Create(&card).Error; err != nil {
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
			card.DisplayName = importedCharacterCardDisplayName(profile)
			if err := validateCharacterCard(card); err != nil {
				return fmt.Errorf("%w: %v", errCharacterCardSourceCorrupt, err)
			}
			return tx.Create(&card).Error
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

	c.JSON(http.StatusCreated, gin.H{"character_card": s.buildCharacterCardDTO(card, true, true)})
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
	if !canViewCharacterCard(card, viewerID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"character_card": s.buildCharacterCardDTO(card, viewerID == card.UserID, true)})
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
	applyCharacterCardStringUpdate(req.NameColor, &candidate.NameColor, "name_color", updates, true)
	applyCharacterCardStringUpdate(req.Summary, &candidate.Summary, "summary", updates, true)
	applyCharacterCardStringUpdate(req.BackgroundStory, &candidate.BackgroundStory, "background_story", updates, false)
	applyCharacterCardStringUpdate(req.FirstImpression, &candidate.FirstImpression, "first_impression", updates, false)
	applyCharacterCardStringUpdate(req.OtherContent, &candidate.OtherContent, "other_content", updates, false)
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

	newPortraitGenerated := false
	pendingPortraitToCleanup := ""
	if portraitSet {
		normalized, err := s.normalizeCharacterCardPortrait(c, userID, card, portraitValue)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		normalizedPortrait := normalized.Path
		newPortraitGenerated = normalized.Generated
		pendingPortraitToCleanup = normalized.PendingSource
		if normalizedPortrait != card.PortraitImage {
			now := time.Now().UTC()
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

	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"character_card": s.buildCharacterCardDTO(card, true, true)})
		return
	}

	result := database.DB.Model(&model.CharacterCard{}).
		Where("id = ? AND user_id = ?", card.ID, userID).
		Updates(updates)
	if result.Error != nil || result.RowsAffected == 0 {
		if newPortraitGenerated {
			s.cleanupOwnedCharacterCardPortrait(c, userID, candidate.PortraitImage)
		}
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新人物卡失败"})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		}
		return
	}

	if portraitSet && candidate.PortraitImage != card.PortraitImage {
		s.cleanupOwnedCharacterCardPortrait(c, userID, card.PortraitImage)
	}
	if pendingPortraitToCleanup != "" {
		s.cleanupOwnedCharacterCardPendingPortrait(c, userID, pendingPortraitToCleanup)
	}
	if err := database.DB.First(&card, card.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"character_card": s.buildCharacterCardDTO(card, true, true)})
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
	if err := database.DB.Where("id = ? AND user_id = ?", card.ID, userID).Delete(&model.CharacterCard{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除人物卡失败"})
		return
	}

	// Posts contain stable IDs in their rich text and are intentionally not
	// rewritten or deleted when a card disappears.
	s.cleanupOwnedCharacterCardPortrait(c, userID, card.PortraitImage)
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
		if err := validateCharacterCard(candidate); err != nil {
			return fmt.Errorf("%w: %v", errCharacterCardSourceCorrupt, err)
		}

		updates := trp3CharacterCardUpdateMap(profile.Fields)
		if err := tx.Model(&model.CharacterCard{}).
			Where("id = ? AND user_id = ?", card.ID, userID).
			Updates(updates).Error; err != nil {
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

	c.JSON(http.StatusOK, gin.H{"character_card": s.buildCharacterCardDTO(card, true, true)})
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
	if err := database.DB.Where("user_id = ? AND status = ? AND visibility = ?", userID, model.CharacterCardStatusPublished, model.CharacterCardVisibilityPublic).
		Omit("background_story", "first_impression", "other_content").
		Order("sort_order ASC, updated_at DESC, id DESC").
		Find(&cards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询人物卡失败"})
		return
	}

	result := make([]characterCardDTO, 0, len(cards))
	for _, card := range cards {
		result = append(result, s.buildCharacterCardDTO(card, false, false))
	}
	c.JSON(http.StatusOK, gin.H{"character_cards": result})
}

func (s *Server) buildCharacterCardDTO(card model.CharacterCard, ownerView, includeRichText bool) characterCardDTO {
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
		NameColor:              card.NameColor,
		Summary:                card.Summary,
		PortraitImageURL:       s.characterCardPortraitURL(card),
		PortraitImageUpdatedAt: card.PortraitImageUpdatedAt,
		Status:                 card.Status,
		Visibility:             card.Visibility,
		SortOrder:              card.SortOrder,
		CreatedAt:              card.CreatedAt,
		UpdatedAt:              card.UpdatedAt,
	}
	if ownerView {
		dto.CharacterID = card.CharacterID
		dto.SourceBackupID = card.SourceBackupID
		dto.SourceAccountID = card.SourceAccountID
		dto.SourceProfileID = card.SourceProfileID
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
	return card.Status == model.CharacterCardStatusPublished && card.Visibility == model.CharacterCardVisibilityPublic
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
			NameColor:          values["CH"],
		},
	}, nil
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
	card.NameColor = fields.NameColor
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
	if len(data) == 0 {
		return "", errors.New("角色大图不能为空")
	}
	if len(data) > characterCardPortraitMaxBytes {
		return "", errors.New("角色大图不能超过 20MB")
	}
	detectedType := strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0])
	if detectedType == "image/jpg" {
		detectedType = "image/jpeg"
	}
	allowed := map[string]struct{}{
		"image/jpeg": {}, "image/png": {}, "image/gif": {}, "image/webp": {},
	}
	if _, ok := allowed[detectedType]; !ok {
		return "", errors.New("角色大图仅支持 JPEG、PNG、GIF 或 WebP")
	}
	declaredType = strings.TrimSpace(strings.Split(declaredType, ";")[0])
	if declaredType == "image/jpg" {
		declaredType = "image/jpeg"
	}
	if declaredType != "" && declaredType != "application/octet-stream" && declaredType != detectedType {
		return "", errors.New("角色大图 MIME 与文件内容不一致")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return "", errors.New("角色大图文件已损坏")
	}
	if config.Width > characterCardPortraitMaxSide || config.Height > characterCardPortraitMaxSide || int64(config.Width)*int64(config.Height) > characterCardPortraitMaxPixels {
		return "", errors.New("角色大图尺寸过大")
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
