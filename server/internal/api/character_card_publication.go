package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type characterCardSnapshot struct {
	Card        characterCardSnapshotCard         `json:"card"`
	Impressions []characterCardSnapshotImpression `json:"impressions"`
	Portraits   []characterCardSnapshotPortrait   `json:"portraits"`
}

var errCharacterCardPublishRequiresPublic = errors.New("character card publish requires public status")

type characterCardSnapshotCard struct {
	ID     uint `json:"id"`
	UserID uint `json:"user_id"`

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
	ClassColor         string `json:"class_color"`
	NameColor          string `json:"name_color"`

	Summary         string `json:"summary"`
	BackgroundStory string `json:"background_story"`
	FirstImpression string `json:"first_impression"`
	OtherContent    string `json:"other_content"`

	PortraitImage          string     `json:"portrait_image"`
	PortraitImageUpdatedAt *time.Time `json:"portrait_image_updated_at"`
	SortOrder              int        `json:"sort_order"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type characterCardSnapshotImpression struct {
	ID     uint  `json:"id"`
	Slot   uint8 `json:"slot"`
	Active bool  `json:"active"`

	Title    string `json:"title"`
	Text     string `json:"text"`
	TRP3Icon string `json:"trp3_icon"`

	IconImage          string     `json:"icon_image"`
	IconImageUpdatedAt *time.Time `json:"icon_image_updated_at"`
	Image              string     `json:"image"`
	ImageUpdatedAt     *time.Time `json:"image_updated_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type characterCardSnapshotPortrait struct {
	ID             uint       `json:"id"`
	SortOrder      int        `json:"sort_order"`
	Image          string     `json:"image"`
	ImageUpdatedAt *time.Time `json:"image_updated_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func normalizedCharacterCardReviewStatus(raw string) string {
	switch strings.TrimSpace(raw) {
	case model.CharacterCardReviewPending, model.CharacterCardReviewApproved, model.CharacterCardReviewRejected:
		return strings.TrimSpace(raw)
	default:
		return model.CharacterCardReviewNone
	}
}

func characterCardSnapshotFromModels(card model.CharacterCard, impressions []model.CharacterCardImpression, portraits []model.CharacterCardPortrait) characterCardSnapshot {
	color := canonicalCharacterCardTRP3Color(card.ClassColor, card.NameColor)
	snapshot := characterCardSnapshot{
		Card: characterCardSnapshotCard{
			ID: card.ID, UserID: card.UserID,
			FirstName: card.FirstName, LastName: card.LastName, DisplayName: card.DisplayName,
			Title: card.Title, FullTitle: card.FullTitle, Race: card.Race, Class: card.Class,
			EyeColor: card.EyeColor, EyeColorHex: card.EyeColorHex, Age: card.Age,
			Height: card.Height, Weight: card.Weight, Birthplace: card.Birthplace,
			Residence: card.Residence, RelationshipStatus: card.RelationshipStatus,
			Icon: card.Icon, ClassColor: color, NameColor: color,
			Summary: card.Summary, BackgroundStory: card.BackgroundStory,
			FirstImpression: card.FirstImpression, OtherContent: card.OtherContent,
			PortraitImage: card.PortraitImage, PortraitImageUpdatedAt: card.PortraitImageUpdatedAt,
			SortOrder: card.SortOrder, CreatedAt: card.CreatedAt, UpdatedAt: card.UpdatedAt,
		},
		Impressions: make([]characterCardSnapshotImpression, 0, len(impressions)),
		Portraits:   make([]characterCardSnapshotPortrait, 0, len(portraits)),
	}
	for _, row := range fixedCharacterCardImpressions(card.ID, impressions) {
		snapshot.Impressions = append(snapshot.Impressions, characterCardSnapshotImpression{
			ID: row.ID, Slot: row.Slot, Active: row.Active, Title: row.Title, Text: row.Text,
			TRP3Icon: row.TRP3Icon, IconImage: row.IconImage, IconImageUpdatedAt: row.IconImageUpdatedAt,
			Image: row.Image, ImageUpdatedAt: row.ImageUpdatedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	sort.SliceStable(portraits, func(i, j int) bool {
		if portraits[i].SortOrder == portraits[j].SortOrder {
			return portraits[i].ID < portraits[j].ID
		}
		return portraits[i].SortOrder < portraits[j].SortOrder
	})
	for _, row := range portraits {
		snapshot.Portraits = append(snapshot.Portraits, characterCardSnapshotPortrait{
			ID: row.ID, SortOrder: row.SortOrder, Image: row.Image,
			ImageUpdatedAt: row.ImageUpdatedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	return snapshot
}

func (snapshot characterCardSnapshot) models() (model.CharacterCard, []model.CharacterCardImpression, []model.CharacterCardPortrait) {
	data := snapshot.Card
	card := model.CharacterCard{
		ID: data.ID, UserID: data.UserID,
		FirstName: data.FirstName, LastName: data.LastName, DisplayName: data.DisplayName,
		Title: data.Title, FullTitle: data.FullTitle, Race: data.Race, Class: data.Class,
		EyeColor: data.EyeColor, EyeColorHex: data.EyeColorHex, Age: data.Age,
		Height: data.Height, Weight: data.Weight, Birthplace: data.Birthplace,
		Residence: data.Residence, RelationshipStatus: data.RelationshipStatus,
		Icon: data.Icon, ClassColor: data.ClassColor, NameColor: data.NameColor,
		Summary: data.Summary, BackgroundStory: data.BackgroundStory,
		FirstImpression: data.FirstImpression, OtherContent: data.OtherContent,
		PortraitImage: data.PortraitImage, PortraitImageUpdatedAt: data.PortraitImageUpdatedAt,
		Status: model.CharacterCardStatusPublished, Visibility: model.CharacterCardVisibilityPublic,
		SortOrder: data.SortOrder, ReviewStatus: model.CharacterCardReviewApproved,
		CreatedAt: data.CreatedAt, UpdatedAt: data.UpdatedAt,
	}
	impressions := make([]model.CharacterCardImpression, 0, len(snapshot.Impressions))
	for _, row := range snapshot.Impressions {
		impressions = append(impressions, model.CharacterCardImpression{
			ID: row.ID, CharacterCardID: card.ID, Slot: row.Slot, Active: row.Active,
			Title: row.Title, Text: row.Text, TRP3Icon: row.TRP3Icon,
			IconImage: row.IconImage, IconImageUpdatedAt: row.IconImageUpdatedAt,
			Image: row.Image, ImageUpdatedAt: row.ImageUpdatedAt,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	portraits := make([]model.CharacterCardPortrait, 0, len(snapshot.Portraits))
	for _, row := range snapshot.Portraits {
		portraits = append(portraits, model.CharacterCardPortrait{
			ID: row.ID, CharacterCardID: card.ID, SortOrder: row.SortOrder,
			Image: row.Image, ImageUpdatedAt: row.ImageUpdatedAt,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	return card, impressions, portraits
}

func captureCharacterCardSnapshot(tx *gorm.DB, cardID uint) (characterCardSnapshot, error) {
	var card model.CharacterCard
	if err := tx.First(&card, cardID).Error; err != nil {
		return characterCardSnapshot{}, err
	}
	impressionsByCard, err := loadCharacterCardImpressions(tx, []uint{card.ID})
	if err != nil {
		return characterCardSnapshot{}, err
	}
	portraits, err := ensureCharacterCardPortraitRows(tx, card)
	if err != nil {
		return characterCardSnapshot{}, err
	}
	return characterCardSnapshotFromModels(card, impressionsByCard[card.ID], portraits), nil
}

func saveCharacterCardPublication(tx *gorm.DB, snapshot characterCardSnapshot, reviewerID uint, approvedAt time.Time) error {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	publication := model.CharacterCardPublication{
		CharacterCardID: snapshot.Card.ID,
		UserID:          snapshot.Card.UserID,
		Payload:         string(payload),
		ApprovedBy:      &reviewerID,
		ApprovedAt:      approvedAt.UTC(),
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "character_card_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"user_id", "payload", "approved_by", "approved_at", "updated_at",
		}),
	}).Create(&publication).Error
}

func loadCharacterCardPublication(tx *gorm.DB, cardID uint) (characterCardSnapshot, model.CharacterCardPublication, error) {
	var publication model.CharacterCardPublication
	if err := tx.Where("character_card_id = ?", cardID).First(&publication).Error; err != nil {
		return characterCardSnapshot{}, publication, err
	}
	var snapshot characterCardSnapshot
	if err := json.Unmarshal([]byte(publication.Payload), &snapshot); err != nil || snapshot.Card.ID != cardID || snapshot.Card.UserID != publication.UserID {
		if err == nil {
			err = errors.New("publication identity mismatch")
		}
		return characterCardSnapshot{}, publication, err
	}
	return snapshot, publication, nil
}

func saveCharacterCardSubmission(tx *gorm.DB, snapshot characterCardSnapshot, submittedAt time.Time) error {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	submission := model.CharacterCardSubmission{
		CharacterCardID: snapshot.Card.ID,
		UserID:          snapshot.Card.UserID,
		Payload:         string(payload),
		SubmittedAt:     submittedAt.UTC(),
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "character_card_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"user_id", "payload", "submitted_at", "updated_at",
		}),
	}).Create(&submission).Error
}

func loadCharacterCardSubmission(tx *gorm.DB, cardID uint) (characterCardSnapshot, model.CharacterCardSubmission, error) {
	var submission model.CharacterCardSubmission
	if err := tx.Where("character_card_id = ?", cardID).First(&submission).Error; err != nil {
		return characterCardSnapshot{}, submission, err
	}
	var snapshot characterCardSnapshot
	if err := json.Unmarshal([]byte(submission.Payload), &snapshot); err != nil || snapshot.Card.ID != cardID || snapshot.Card.UserID != submission.UserID {
		if err == nil {
			err = errors.New("submission identity mismatch")
		}
		return characterCardSnapshot{}, submission, err
	}
	return snapshot, submission, nil
}

// ensureCharacterCardApprovedSnapshotBeforeMutation upgrades a legacy public
// card to the approved-snapshot model before its live working copy changes.
func ensureCharacterCardApprovedSnapshotBeforeMutation(tx *gorm.DB, card model.CharacterCard) error {
	if card.Status != model.CharacterCardStatusPublished || card.Visibility != model.CharacterCardVisibilityPublic {
		return nil
	}
	status := normalizedCharacterCardReviewStatus(card.ReviewStatus)
	if status != model.CharacterCardReviewApproved {
		return nil
	}
	var count int64
	if err := tx.Model(&model.CharacterCardPublication{}).Where("character_card_id = ?", card.ID).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	snapshot, err := captureCharacterCardSnapshot(tx, card.ID)
	if err != nil {
		return err
	}
	return saveCharacterCardPublication(tx, snapshot, 0, card.UpdatedAt)
}

func resetCharacterCardReviewWhenWithdrawn(card *model.CharacterCard, updates map[string]interface{}) {
	if card.Status == model.CharacterCardStatusPublished && card.Visibility == model.CharacterCardVisibilityPublic {
		return
	}
	card.ReviewStatus = model.CharacterCardReviewNone
	card.ReviewerID = nil
	card.ReviewComment = ""
	card.ReviewedAt = nil
	updates["review_status"] = model.CharacterCardReviewNone
	updates["reviewer_id"] = nil
	updates["review_comment"] = ""
	updates["reviewed_at"] = nil
}

func isCharacterCardModerator(viewerID uint) bool {
	if viewerID == 0 {
		return false
	}
	var user model.User
	if err := database.DB.Select("id", "role", "account_deleted_at").First(&user, viewerID).Error; err != nil || user.AccountDeletedAt != nil {
		return false
	}
	return user.Role == "moderator" || user.Role == "admin"
}

func (s *Server) loadOwnerCharacterCardDTO(cardID, userID uint, includeRichText bool) (characterCardDTO, error) {
	var card model.CharacterCard
	if err := database.DB.Where("id = ? AND user_id = ?", cardID, userID).First(&card).Error; err != nil {
		return characterCardDTO{}, err
	}
	impressionsByCard, err := loadCharacterCardImpressions(database.DB, []uint{card.ID})
	if err != nil {
		return characterCardDTO{}, err
	}
	portraitsByCard, err := loadCharacterCardPortraits(database.DB, []uint{card.ID})
	if err != nil {
		return characterCardDTO{}, err
	}
	return s.buildCharacterCardDTO(card, impressionsByCard[card.ID], portraitsByCard[card.ID], true, includeRichText), nil
}

func (s *Server) loadPublicCharacterCardDTO(card model.CharacterCard, includeRichText bool) (characterCardDTO, bool, error) {
	snapshot, _, err := loadCharacterCardPublication(database.DB, card.ID)
	if err == nil {
		publishedCard, impressions, portraits := snapshot.models()
		return s.buildCharacterCardDTO(publishedCard, impressions, portraits, false, includeRichText), true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return characterCardDTO{}, false, err
	}
	// Approved legacy cards created before immutable publications remain visible
	// until their first edit freezes the working aggregate into a publication.
	status := normalizedCharacterCardReviewStatus(card.ReviewStatus)
	if card.Status != model.CharacterCardStatusPublished || card.Visibility != model.CharacterCardVisibilityPublic ||
		status != model.CharacterCardReviewApproved {
		return characterCardDTO{}, false, nil
	}
	impressionsByCard, loadErr := loadCharacterCardImpressions(database.DB, []uint{card.ID})
	if loadErr != nil {
		return characterCardDTO{}, false, loadErr
	}
	portraitsByCard, loadErr := loadCharacterCardPortraits(database.DB, []uint{card.ID})
	if loadErr != nil {
		return characterCardDTO{}, false, loadErr
	}
	return s.buildCharacterCardDTO(card, impressionsByCard[card.ID], portraitsByCard[card.ID], false, includeRichText), true, nil
}

func (s *Server) loadProtectedCharacterCardPortrait(cardID, viewerID uint) (string, time.Time, error) {
	var card model.CharacterCard
	if err := database.DB.First(&card, cardID).Error; err != nil {
		return "", time.Time{}, err
	}
	if viewerID != 0 && viewerID == card.UserID {
		if strings.TrimSpace(card.PortraitImage) == "" {
			return "", time.Time{}, gorm.ErrRecordNotFound
		}
		version := card.UpdatedAt
		if card.PortraitImageUpdatedAt != nil {
			version = *card.PortraitImageUpdatedAt
		}
		return card.PortraitImage, version, nil
	}
	if isCharacterCardModerator(viewerID) {
		if normalizedCharacterCardReviewStatus(card.ReviewStatus) == model.CharacterCardReviewPending {
			if snapshot, _, submissionErr := loadCharacterCardSubmission(database.DB, card.ID); submissionErr == nil {
				if strings.TrimSpace(snapshot.Card.PortraitImage) == "" {
					return "", time.Time{}, gorm.ErrRecordNotFound
				}
				version := snapshot.Card.UpdatedAt
				if snapshot.Card.PortraitImageUpdatedAt != nil {
					version = *snapshot.Card.PortraitImageUpdatedAt
				}
				return snapshot.Card.PortraitImage, version, nil
			} else if !errors.Is(submissionErr, gorm.ErrRecordNotFound) {
				return "", time.Time{}, submissionErr
			}
		}
		if strings.TrimSpace(card.PortraitImage) == "" {
			return "", time.Time{}, gorm.ErrRecordNotFound
		}
		version := card.UpdatedAt
		if card.PortraitImageUpdatedAt != nil {
			version = *card.PortraitImageUpdatedAt
		}
		return card.PortraitImage, version, nil
	}
	snapshot, _, err := loadCharacterCardPublication(database.DB, card.ID)
	if err == nil {
		if strings.TrimSpace(snapshot.Card.PortraitImage) == "" {
			return "", time.Time{}, gorm.ErrRecordNotFound
		}
		version := snapshot.Card.UpdatedAt
		if snapshot.Card.PortraitImageUpdatedAt != nil {
			version = *snapshot.Card.PortraitImageUpdatedAt
		}
		return snapshot.Card.PortraitImage, version, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", time.Time{}, err
	}
	if !canViewCharacterCard(card, viewerID) || strings.TrimSpace(card.PortraitImage) == "" {
		return "", time.Time{}, gorm.ErrRecordNotFound
	}
	version := card.UpdatedAt
	if card.PortraitImageUpdatedAt != nil {
		version = *card.PortraitImageUpdatedAt
	}
	return card.PortraitImage, version, nil
}

func (s *Server) loadProtectedCharacterCardGalleryPortrait(cardID, portraitID, viewerID uint) (string, time.Time, error) {
	var card model.CharacterCard
	if err := database.DB.First(&card, cardID).Error; err != nil {
		return "", time.Time{}, err
	}
	if viewerID != 0 && viewerID == card.UserID {
		var portrait model.CharacterCardPortrait
		if err := database.DB.Where("id = ? AND character_card_id = ?", portraitID, card.ID).First(&portrait).Error; err != nil {
			return "", time.Time{}, err
		}
		version := portrait.UpdatedAt
		if portrait.ImageUpdatedAt != nil {
			version = *portrait.ImageUpdatedAt
		}
		return portrait.Image, version, nil
	}
	if isCharacterCardModerator(viewerID) {
		if normalizedCharacterCardReviewStatus(card.ReviewStatus) == model.CharacterCardReviewPending {
			if snapshot, _, submissionErr := loadCharacterCardSubmission(database.DB, card.ID); submissionErr == nil {
				for _, portrait := range snapshot.Portraits {
					if portrait.ID != portraitID || strings.TrimSpace(portrait.Image) == "" {
						continue
					}
					version := portrait.UpdatedAt
					if portrait.ImageUpdatedAt != nil {
						version = *portrait.ImageUpdatedAt
					}
					return portrait.Image, version, nil
				}
				return "", time.Time{}, gorm.ErrRecordNotFound
			} else if !errors.Is(submissionErr, gorm.ErrRecordNotFound) {
				return "", time.Time{}, submissionErr
			}
		}
		var portrait model.CharacterCardPortrait
		if err := database.DB.Where("id = ? AND character_card_id = ?", portraitID, card.ID).First(&portrait).Error; err != nil {
			return "", time.Time{}, err
		}
		version := portrait.UpdatedAt
		if portrait.ImageUpdatedAt != nil {
			version = *portrait.ImageUpdatedAt
		}
		return portrait.Image, version, nil
	}
	snapshot, _, err := loadCharacterCardPublication(database.DB, card.ID)
	if err == nil {
		for _, portrait := range snapshot.Portraits {
			if portrait.ID != portraitID || strings.TrimSpace(portrait.Image) == "" {
				continue
			}
			version := portrait.UpdatedAt
			if portrait.ImageUpdatedAt != nil {
				version = *portrait.ImageUpdatedAt
			}
			return portrait.Image, version, nil
		}
		return "", time.Time{}, gorm.ErrRecordNotFound
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) || !canViewCharacterCard(card, viewerID) {
		if err == nil {
			err = gorm.ErrRecordNotFound
		}
		return "", time.Time{}, err
	}
	var portrait model.CharacterCardPortrait
	if err := database.DB.Where("id = ? AND character_card_id = ?", portraitID, card.ID).First(&portrait).Error; err != nil {
		return "", time.Time{}, err
	}
	version := portrait.UpdatedAt
	if portrait.ImageUpdatedAt != nil {
		version = *portrait.ImageUpdatedAt
	}
	return portrait.Image, version, nil
}

func characterCardSnapshotAssetPaths(snapshot characterCardSnapshot) map[string]struct{} {
	paths := make(map[string]struct{})
	add := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			paths[value] = struct{}{}
		}
	}
	add(snapshot.Card.PortraitImage)
	for _, row := range snapshot.Portraits {
		add(row.Image)
	}
	for _, row := range snapshot.Impressions {
		add(row.IconImage)
		add(row.Image)
	}
	return paths
}

// publishCharacterCard freezes the owner's current cloud working copy as the
// one review candidate. Repeated calls replace that candidate, while ordinary
// auto-saves continue changing only the editable aggregate.
func (s *Server) publishCharacterCard(c *gin.Context) {
	cardID, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	userID := c.GetUint("userID")
	now := time.Now().UTC()
	var card model.CharacterCard
	var replacedSubmission *characterCardSnapshot

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", cardID, userID).
			First(&card).Error; err != nil {
			return err
		}
		if card.Status != model.CharacterCardStatusPublished || card.Visibility != model.CharacterCardVisibilityPublic {
			return errCharacterCardPublishRequiresPublic
		}
		if err := ensureCharacterCardApprovedSnapshotBeforeMutation(tx, card); err != nil {
			return err
		}
		if existing, _, err := loadCharacterCardSubmission(tx, card.ID); err == nil {
			copy := existing
			replacedSubmission = &copy
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		snapshot, err := captureCharacterCardSnapshot(tx, card.ID)
		if err != nil {
			return err
		}
		if err := saveCharacterCardSubmission(tx, snapshot, now); err != nil {
			return err
		}
		updates := map[string]interface{}{
			"review_status":  model.CharacterCardReviewPending,
			"reviewer_id":    nil,
			"review_comment": "",
			"reviewed_at":    nil,
		}
		return tx.Model(&model.CharacterCard{}).
			Where("id = ? AND user_id = ?", card.ID, userID).
			Updates(updates).Error
	})
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		case errors.Is(err, errCharacterCardPublishRequiresPublic):
			c.JSON(http.StatusConflict, gin.H{"error": "请先将人物卡设置为已发布且公开可见"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "提交人物卡审核失败"})
		}
		return
	}

	if replacedSubmission != nil {
		for asset := range characterCardSnapshotAssetPaths(*replacedSubmission) {
			s.cleanupCharacterCardAssetIfUnreferenced(c, userID, card.ID, asset)
		}
	}
	dto, err := s.loadOwnerCharacterCardDTO(card.ID, userID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "已提交最新人物卡版本审核", "character_card": dto})
}

func (s *Server) cleanupCharacterCardAssetIfUnreferenced(c *gin.Context, userID, cardID uint, raw string) {
	canonical, ok := s.characterCardInternalUploadPath(c, raw)
	if !ok {
		return
	}
	expectedPrefix := fmt.Sprintf("/uploads/character-cards/%d/", userID)
	policyPath := pathPolicy(canonical)
	if !strings.HasPrefix(policyPath, expectedPrefix) {
		return
	}
	var card model.CharacterCard
	if err := database.DB.First(&card, cardID).Error; err == nil {
		if card.PortraitImage == canonical {
			return
		}
		var count int64
		if err := database.DB.Model(&model.CharacterCardPortrait{}).Where("character_card_id = ? AND image = ?", cardID, canonical).Count(&count).Error; err == nil && count > 0 {
			return
		}
		if err := database.DB.Model(&model.CharacterCardImpression{}).
			Where("character_card_id = ? AND (icon_image = ? OR image = ?)", cardID, canonical, canonical).Count(&count).Error; err == nil && count > 0 {
			return
		}
	}
	if snapshot, _, err := loadCharacterCardPublication(database.DB, cardID); err == nil {
		if _, exists := characterCardSnapshotAssetPaths(snapshot)[canonical]; exists {
			return
		}
	}
	if snapshot, _, err := loadCharacterCardSubmission(database.DB, cardID); err == nil {
		if _, exists := characterCardSnapshotAssetPaths(snapshot)[canonical]; exists {
			return
		}
	}
	if key := uploadsKeyFromPath(canonical); key != "" {
		s.deleteUploadKey(key)
	}
}

func pathPolicy(raw string) string {
	return strings.ToLower(strings.ReplaceAll(raw, `\`, "/"))
}

func (s *Server) listPendingCharacterCards(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	status := strings.TrimSpace(c.DefaultQuery("status", model.CharacterCardReviewPending))
	if status != model.CharacterCardReviewPending && status != model.CharacterCardReviewApproved && status != model.CharacterCardReviewRejected {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status 无效"})
		return
	}
	query := database.DB.Model(&model.CharacterCard{}).Where("review_status = ?", status)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询人物卡审核队列失败"})
		return
	}
	var cards []model.CharacterCard
	if err := query.Order("updated_at ASC, id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&cards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询人物卡审核队列失败"})
		return
	}
	impressionsByCard, err := loadCharacterCardImpressions(database.DB, characterCardIDs(cards))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡第一印象失败"})
		return
	}
	portraitsByCard, err := loadCharacterCardPortraits(database.DB, characterCardIDs(cards))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡角色大图失败"})
		return
	}
	userIDs := make([]uint, 0, len(cards))
	for _, card := range cards {
		userIDs = append(userIDs, card.UserID)
	}
	var users []model.User
	if len(userIDs) > 0 {
		_ = database.DB.Where("id IN ?", userIDs).Find(&users).Error
	}
	userByID := make(map[uint]model.User, len(users))
	for _, user := range users {
		userByID[user.ID] = user
	}
	type reviewDTO struct {
		characterCardDTO
		OwnerName      string `json:"owner_name"`
		OwnerNameColor string `json:"owner_name_color"`
		OwnerNameBold  bool   `json:"owner_name_bold"`
	}
	result := make([]reviewDTO, 0, len(cards))
	for _, card := range cards {
		owner := userByID[card.UserID]
		color, bold := userDisplayStyle(owner)
		dto := s.buildCharacterCardDTO(card, impressionsByCard[card.ID], portraitsByCard[card.ID], true, true)
		if snapshot, _, submissionErr := loadCharacterCardSubmission(database.DB, card.ID); submissionErr == nil {
			submittedCard, submittedImpressions, submittedPortraits := snapshot.models()
			submittedCard.ReviewStatus = card.ReviewStatus
			submittedCard.ReviewComment = card.ReviewComment
			submittedCard.ReviewedAt = card.ReviewedAt
			dto = s.buildCharacterCardDTO(submittedCard, submittedImpressions, submittedPortraits, true, true)
		} else if !errors.Is(submissionErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡审核版本失败"})
			return
		}
		result = append(result, reviewDTO{
			characterCardDTO: dto,
			OwnerName:        owner.Username, OwnerNameColor: color, OwnerNameBold: bold,
		})
	}
	c.JSON(http.StatusOK, gin.H{"character_cards": result, "total": total})
}

func (s *Server) reviewCharacterCard(c *gin.Context) {
	cardID, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil || (req.Action != "approve" && req.Action != "reject") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的审核操作"})
		return
	}
	comment := strings.TrimSpace(req.Comment)
	if len([]rune(comment)) > 512 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "审核意见不能超过 512 个字符"})
		return
	}
	reviewerID := c.GetUint("userID")
	now := time.Now().UTC()
	var card model.CharacterCard
	var oldSnapshot *characterCardSnapshot
	var reviewedSnapshot *characterCardSnapshot
	reviewedName := ""
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&card, cardID).Error; err != nil {
			return err
		}
		if normalizedCharacterCardReviewStatus(card.ReviewStatus) != model.CharacterCardReviewPending {
			return errors.New("character card review is no longer pending")
		}
		if existing, _, err := loadCharacterCardPublication(tx, card.ID); err == nil {
			copy := existing
			oldSnapshot = &copy
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		snapshot, _, submissionErr := loadCharacterCardSubmission(tx, card.ID)
		if errors.Is(submissionErr, gorm.ErrRecordNotFound) {
			// Compatibility for review requests created before frozen submission
			// snapshots were introduced.
			var err error
			snapshot, err = captureCharacterCardSnapshot(tx, card.ID)
			if err != nil {
				return err
			}
		} else if submissionErr != nil {
			return submissionErr
		}
		copy := snapshot
		reviewedSnapshot = &copy
		reviewedName = snapshot.Card.DisplayName
		updates := map[string]interface{}{
			"reviewer_id": reviewerID, "review_comment": comment, "reviewed_at": now,
		}
		if req.Action == "approve" {
			if card.Status != model.CharacterCardStatusPublished || card.Visibility != model.CharacterCardVisibilityPublic {
				return errors.New("character card is no longer public")
			}
			if err := saveCharacterCardPublication(tx, snapshot, reviewerID, now); err != nil {
				return err
			}
			updates["review_status"] = model.CharacterCardReviewApproved
		} else {
			updates["review_status"] = model.CharacterCardReviewRejected
		}
		if err := tx.Where("character_card_id = ?", card.ID).Delete(&model.CharacterCardSubmission{}).Error; err != nil {
			return err
		}
		return tx.Model(&model.CharacterCard{}).Where("id = ? AND review_status = ?", card.ID, model.CharacterCardReviewPending).Updates(updates).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		} else if strings.Contains(err.Error(), "no longer") {
			c.JSON(http.StatusConflict, gin.H{"error": "该人物卡审核状态已变化，请刷新后重试"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "审核人物卡失败"})
		}
		return
	}
	if req.Action == "approve" && oldSnapshot != nil {
		for asset := range characterCardSnapshotAssetPaths(*oldSnapshot) {
			s.cleanupCharacterCardAssetIfUnreferenced(c, card.UserID, card.ID, asset)
		}
	}
	if req.Action == "reject" && reviewedSnapshot != nil {
		for asset := range characterCardSnapshotAssetPaths(*reviewedSnapshot) {
			s.cleanupCharacterCardAssetIfUnreferenced(c, card.UserID, card.ID, asset)
		}
	}
	if reviewedName == "" {
		reviewedName = card.DisplayName
	}
	logAdminAction(c, "review_character_card", "character_card", card.ID, reviewedName, map[string]interface{}{
		"action": req.Action, "comment": comment,
	})
	notifyModerationResult(card.UserID, "character_card", card.ID, "人物卡《"+reviewedName+"》", req.Action, comment)
	dto, loadErr := s.loadOwnerCharacterCardDTO(card.ID, card.UserID, true)
	if loadErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "审核完成", "character_card": dto})
}
