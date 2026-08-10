package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
)

const (
	characterCardImpressionSlotCount = 5
	characterCardImpressionTitleMax  = 80
	characterCardImpressionTextMax   = 500

	characterCardImpressionKindIcon  = "icon"
	characterCardImpressionKindImage = "image"
)

type characterCardImpressionDTO struct {
	Slot               uint8      `json:"slot"`
	Active             bool       `json:"active"`
	Title              string     `json:"title"`
	Text               string     `json:"text"`
	TRP3Icon           string     `json:"trp3_icon"`
	IconImageURL       string     `json:"icon_image_url"`
	IconImageUpdatedAt *time.Time `json:"icon_image_updated_at"`
	ImageURL           string     `json:"image_url"`
	ImageUpdatedAt     *time.Time `json:"image_updated_at"`
}

type characterCardImpressionRequest struct {
	Slot         uint8  `json:"slot"`
	Active       bool   `json:"active"`
	Title        string `json:"title"`
	Text         string `json:"text"`
	TRP3Icon     string `json:"trp3_icon"`
	IconImageURL string `json:"icon_image_url"`
	ImageURL     string `json:"image_url"`
}

type characterCardImpressionFileRef struct {
	Kind string
	Path string
}

type characterCardImpressionUpdatePlan struct {
	Rows      []model.CharacterCardImpression
	Generated []characterCardImpressionFileRef
	Pending   []characterCardImpressionFileRef
	Replaced  []characterCardImpressionFileRef
}

func defaultCharacterCardImpressions(cardID uint) []model.CharacterCardImpression {
	rows := make([]model.CharacterCardImpression, 0, characterCardImpressionSlotCount)
	for slot := 1; slot <= characterCardImpressionSlotCount; slot++ {
		rows = append(rows, model.CharacterCardImpression{CharacterCardID: cardID, Slot: uint8(slot)})
	}
	return rows
}

func characterCardImpressionsFromRequests(cardID uint, requests []characterCardImpressionRequest) []model.CharacterCardImpression {
	rows := defaultCharacterCardImpressions(cardID)
	bySlot := make(map[uint8]characterCardImpressionRequest, len(requests))
	for _, request := range requests {
		bySlot[request.Slot] = request
	}
	for index := range rows {
		if request, exists := bySlot[rows[index].Slot]; exists {
			rows[index].Active = request.Active
			rows[index].Title = request.Title
			rows[index].Text = request.Text
			rows[index].TRP3Icon = request.TRP3Icon
		}
	}
	return rows
}

func fixedCharacterCardImpressions(cardID uint, rows []model.CharacterCardImpression) []model.CharacterCardImpression {
	bySlot := make(map[uint8]model.CharacterCardImpression, characterCardImpressionSlotCount)
	for _, row := range rows {
		if row.CharacterCardID == cardID && row.Slot >= 1 && row.Slot <= characterCardImpressionSlotCount {
			if _, exists := bySlot[row.Slot]; !exists {
				bySlot[row.Slot] = row
			}
		}
	}
	result := defaultCharacterCardImpressions(cardID)
	for index := range result {
		if row, exists := bySlot[result[index].Slot]; exists {
			result[index] = row
		}
	}
	return result
}

func loadCharacterCardImpressions(tx *gorm.DB, cardIDs []uint) (map[uint][]model.CharacterCardImpression, error) {
	result := make(map[uint][]model.CharacterCardImpression, len(cardIDs))
	if len(cardIDs) == 0 {
		return result, nil
	}
	var rows []model.CharacterCardImpression
	if err := tx.Where("character_card_id IN ?", cardIDs).
		Order("character_card_id ASC, slot ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.CharacterCardID] = append(result[row.CharacterCardID], row)
	}
	return result, nil
}

func characterCardIDs(cards []model.CharacterCard) []uint {
	ids := make([]uint, 0, len(cards))
	for _, card := range cards {
		ids = append(ids, card.ID)
	}
	return ids
}

func validateCharacterCardImpressionRequests(requests []characterCardImpressionRequest) ([]characterCardImpressionRequest, error) {
	if len(requests) != characterCardImpressionSlotCount {
		return nil, fmt.Errorf("impressions 必须完整包含 %d 个槽位", characterCardImpressionSlotCount)
	}
	bySlot := make(map[uint8]characterCardImpressionRequest, characterCardImpressionSlotCount)
	for _, request := range requests {
		if request.Slot < 1 || request.Slot > characterCardImpressionSlotCount {
			return nil, errors.New("impressions.slot 仅支持 1 到 5")
		}
		if _, exists := bySlot[request.Slot]; exists {
			return nil, errors.New("impressions.slot 不能重复")
		}
		request.Title = strings.TrimSpace(request.Title)
		request.Text = strings.TrimSpace(request.Text)
		request.TRP3Icon = strings.TrimSpace(request.TRP3Icon)
		request.IconImageURL = strings.TrimSpace(request.IconImageURL)
		request.ImageURL = strings.TrimSpace(request.ImageURL)
		if err := validatePlainCharacterCardField("impressions.title", request.Title, characterCardImpressionTitleMax); err != nil {
			return nil, err
		}
		if err := validatePlainCharacterCardField("impressions.text", request.Text, characterCardImpressionTextMax); err != nil {
			return nil, err
		}
		if err := validatePlainCharacterCardField("impressions.trp3_icon", request.TRP3Icon, 128); err != nil {
			return nil, err
		}
		bySlot[request.Slot] = request
	}
	result := make([]characterCardImpressionRequest, 0, characterCardImpressionSlotCount)
	for slot := uint8(1); slot <= characterCardImpressionSlotCount; slot++ {
		result = append(result, bySlot[slot])
	}
	return result, nil
}

func (s *Server) buildCharacterCardImpressionDTOs(card model.CharacterCard, rows []model.CharacterCardImpression, ownerView bool) []characterCardImpressionDTO {
	fixed := fixedCharacterCardImpressions(card.ID, rows)
	result := make([]characterCardImpressionDTO, 0, characterCardImpressionSlotCount)
	for _, row := range fixed {
		if !ownerView && !row.Active {
			continue
		}
		result = append(result, characterCardImpressionDTO{
			Slot:               row.Slot,
			Active:             row.Active,
			Title:              row.Title,
			Text:               row.Text,
			TRP3Icon:           row.TRP3Icon,
			IconImageURL:       s.characterCardImpressionImageURL(row, characterCardImpressionKindIcon),
			IconImageUpdatedAt: row.IconImageUpdatedAt,
			ImageURL:           s.characterCardImpressionImageURL(row, characterCardImpressionKindImage),
			ImageUpdatedAt:     row.ImageUpdatedAt,
		})
	}
	return result
}

func (s *Server) characterCardImpressionImageURL(impression model.CharacterCardImpression, kind string) string {
	var value string
	var version *time.Time
	imageType := "character-card-impression-image"
	if kind == characterCardImpressionKindIcon {
		value = impression.IconImage
		version = impression.IconImageUpdatedAt
		imageType = "character-card-impression-icon"
	} else {
		value = impression.Image
		version = impression.ImageUpdatedAt
	}
	if impression.ID == 0 || strings.TrimSpace(value) == "" {
		return ""
	}
	versionTime := impression.UpdatedAt
	if version != nil {
		versionTime = *version
	}
	imagePath := fmt.Sprintf("/api/v1/images/%s/%d?v=%d", imageType, impression.ID, versionTime.UnixNano())
	return buildAPIURL(s.cfg.Server.ApiHost, imagePath)
}

func (s *Server) uploadCharacterCardImpressionImage(c *gin.Context) {
	userID := c.GetUint("userID")
	kind := strings.ToLower(strings.TrimSpace(c.PostForm("kind")))
	if kind != characterCardImpressionKindIcon && kind != characterCardImpressionKindImage {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kind 仅支持 icon 或 image"})
		return
	}
	header, err := c.FormFile("image")
	if err != nil || header == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择印象图片"})
		return
	}
	if header.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "印象图片不能为空"})
		return
	}
	if header.Size > characterCardPortraitMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "印象图片不能超过 20MB"})
		return
	}
	file, err := header.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取印象图片失败"})
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, characterCardPortraitMaxBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取印象图片失败"})
		return
	}
	if len(data) > characterCardPortraitMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "印象图片不能超过 20MB"})
		return
	}
	contentType, err := validateCharacterCardImageBytes(data, header.Header.Get("Content-Type"), "印象图片")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cutoff := time.Now().UTC().Add(-characterCardPortraitPendingTTL)
	_, _ = s.cleanupExpiredCharacterCardPendingPortraits(cutoff)
	saved, err := s.saveImageBytes(c, data, contentType, characterCardImpressionPendingSubdir(userID, kind))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存印象图片失败"})
		return
	}
	canonical, storageKind, ok := s.characterCardOwnedImpressionStoragePath(c, userID, saved, kind)
	if !ok || storageKind != characterCardPortraitStoragePending {
		if key := extractUploadKey(c, saved); key != "" {
			s.deleteUploadKey(key)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存印象图片失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"image_ref": canonical})
}

func characterCardImpressionPendingSubdir(userID uint, kind string) string {
	return path.Join(characterCardPortraitPendingSubdir(userID), kind)
}

func characterCardImpressionCurrentSubdir(userID uint, kind string) string {
	return path.Join("character-cards", strconv.FormatUint(uint64(userID), 10), "impression-"+kind)
}

func (s *Server) characterCardOwnedImpressionStoragePath(c *gin.Context, userID uint, raw, kind string) (string, characterCardPortraitStorageKind, bool) {
	if kind != characterCardImpressionKindIcon && kind != characterCardImpressionKindImage {
		return "", characterCardPortraitStorageInvalid, false
	}
	canonical, ok := s.characterCardInternalUploadPath(c, raw)
	if !ok {
		return "", characterCardPortraitStorageInvalid, false
	}
	policyPath := path.Clean(strings.ToLower(strings.ReplaceAll(canonical, `\`, "/")))
	pendingRoot := "/uploads/" + characterCardImpressionPendingSubdir(userID, kind) + "/"
	currentRoot := "/uploads/" + characterCardImpressionCurrentSubdir(userID, kind) + "/"
	switch {
	case strings.HasPrefix(policyPath, pendingRoot):
		return canonical, characterCardPortraitStoragePending, true
	case strings.HasPrefix(policyPath, currentRoot):
		return canonical, characterCardPortraitStorageCurrent, true
	default:
		return "", characterCardPortraitStorageInvalid, false
	}
}

func (s *Server) prepareCharacterCardImpressionUpdate(
	c *gin.Context,
	userID uint,
	card model.CharacterCard,
	existing []model.CharacterCardImpression,
	requests []characterCardImpressionRequest,
) (characterCardImpressionUpdatePlan, error) {
	validated, err := validateCharacterCardImpressionRequests(requests)
	if err != nil {
		return characterCardImpressionUpdatePlan{}, err
	}
	plan := characterCardImpressionUpdatePlan{Rows: fixedCharacterCardImpressions(card.ID, existing)}
	cleanupGenerated := func() {
		for _, reference := range plan.Generated {
			s.cleanupOwnedCharacterCardImpressionImage(c, userID, reference.Path, reference.Kind, characterCardPortraitStorageCurrent)
		}
	}

	for index, request := range validated {
		row := plan.Rows[index]
		oldIconImage := row.IconImage
		oldImage := row.Image
		oldActive := row.Active

		row.Active = request.Active
		row.Title = request.Title
		row.Text = request.Text
		row.TRP3Icon = request.TRP3Icon

		normalizedIcon, err := s.normalizeCharacterCardImpressionImage(c, userID, row, characterCardImpressionKindIcon, request.IconImageURL)
		if err != nil {
			cleanupGenerated()
			return characterCardImpressionUpdatePlan{}, fmt.Errorf("第 %d 槽 icon_image_url 无效: %w", request.Slot, err)
		}
		if normalizedIcon.Generated {
			plan.Generated = append(plan.Generated, characterCardImpressionFileRef{Kind: characterCardImpressionKindIcon, Path: normalizedIcon.Path})
		}
		if normalizedIcon.PendingSource != "" {
			plan.Pending = append(plan.Pending, characterCardImpressionFileRef{Kind: characterCardImpressionKindIcon, Path: normalizedIcon.PendingSource})
		}
		row.IconImage = normalizedIcon.Path

		normalizedImage, err := s.normalizeCharacterCardImpressionImage(c, userID, row, characterCardImpressionKindImage, request.ImageURL)
		if err != nil {
			cleanupGenerated()
			return characterCardImpressionUpdatePlan{}, fmt.Errorf("第 %d 槽 image_url 无效: %w", request.Slot, err)
		}
		if normalizedImage.Generated {
			plan.Generated = append(plan.Generated, characterCardImpressionFileRef{Kind: characterCardImpressionKindImage, Path: normalizedImage.Path})
		}
		if normalizedImage.PendingSource != "" {
			plan.Pending = append(plan.Pending, characterCardImpressionFileRef{Kind: characterCardImpressionKindImage, Path: normalizedImage.PendingSource})
		}
		row.Image = normalizedImage.Path

		now := time.Now().UTC()
		if row.IconImage != oldIconImage {
			nextVersion := nextCharacterCardImpressionImageVersion(row.IconImageUpdatedAt, now)
			row.IconImageUpdatedAt = &nextVersion
			if oldIconImage != "" {
				plan.Replaced = append(plan.Replaced, characterCardImpressionFileRef{Kind: characterCardImpressionKindIcon, Path: oldIconImage})
			}
		}
		if row.Image != oldImage {
			nextVersion := nextCharacterCardImpressionImageVersion(row.ImageUpdatedAt, now)
			row.ImageUpdatedAt = &nextVersion
			if oldImage != "" {
				plan.Replaced = append(plan.Replaced, characterCardImpressionFileRef{Kind: characterCardImpressionKindImage, Path: oldImage})
			}
		}
		if row.Active != oldActive {
			if row.IconImage != "" && row.IconImage == oldIconImage {
				nextVersion := nextCharacterCardImpressionImageVersion(row.IconImageUpdatedAt, now)
				row.IconImageUpdatedAt = &nextVersion
			}
			if row.Image != "" && row.Image == oldImage {
				nextVersion := nextCharacterCardImpressionImageVersion(row.ImageUpdatedAt, now)
				row.ImageUpdatedAt = &nextVersion
			}
		}
		plan.Rows[index] = row
	}
	return plan, nil
}

func (s *Server) normalizeCharacterCardImpressionImage(
	c *gin.Context,
	userID uint,
	impression model.CharacterCardImpression,
	kind string,
	raw string,
) (normalizedCharacterCardPortrait, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return normalizedCharacterCardPortrait{}, nil
	}
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		return normalizedCharacterCardPortrait{}, errors.New("请先使用人物卡印象图片上传接口")
	}

	currentValue := impression.Image
	if kind == characterCardImpressionKindIcon {
		currentValue = impression.IconImage
	}
	currentProxy := s.isCurrentCharacterCardImpressionImageURL(c, impression.ID, kind, value)
	sourceValue := value
	if currentProxy {
		sourceValue = strings.TrimSpace(currentValue)
		if sourceValue == "" {
			return normalizedCharacterCardPortrait{}, errors.New("印象图片不存在")
		}
	}

	canonical, storageKind, owned := s.characterCardOwnedImpressionStoragePath(c, userID, sourceValue, kind)
	if !owned {
		return normalizedCharacterCardPortrait{}, errors.New("印象图片只接受本人的 pending 或已归档引用")
	}
	if storageKind == characterCardPortraitStorageCurrent {
		if canonical == currentValue {
			return normalizedCharacterCardPortrait{Path: canonical}, nil
		}
		return normalizedCharacterCardPortrait{}, errors.New("不能引用其他印象槽的图片")
	}

	data, contentType, err := s.loadImageBytes(c, canonical)
	if err != nil {
		return normalizedCharacterCardPortrait{}, errors.New("印象图片文件不存在")
	}
	detectedType, err := validateCharacterCardImageBytes(data, contentType, "印象图片")
	if err != nil {
		return normalizedCharacterCardPortrait{}, err
	}
	saved, err := s.saveImageBytes(c, data, detectedType, characterCardImpressionCurrentSubdir(userID, kind))
	if err != nil {
		return normalizedCharacterCardPortrait{}, errors.New("保存印象图片失败")
	}
	protectedPath, protectedKind, ok := s.characterCardOwnedImpressionStoragePath(c, userID, saved, kind)
	if !ok || protectedKind != characterCardPortraitStorageCurrent {
		if key := extractUploadKey(c, saved); key != "" {
			s.deleteUploadKey(key)
		}
		return normalizedCharacterCardPortrait{}, errors.New("保存印象图片失败")
	}
	return normalizedCharacterCardPortrait{
		Path:          protectedPath,
		Generated:     true,
		PendingSource: canonical,
	}, nil
}

func (s *Server) isCurrentCharacterCardImpressionImageURL(c *gin.Context, impressionID uint, kind, raw string) bool {
	if impressionID == 0 {
		return false
	}
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
	imageType := "character-card-impression-image"
	if kind == characterCardImpressionKindIcon {
		imageType = "character-card-impression-icon"
	}
	expectedPath := fmt.Sprintf("/api/v1/images/%s/%d", imageType, impressionID)
	return path.Clean(parsed.Path) == expectedPath
}

func (s *Server) cleanupOwnedCharacterCardImpressionImage(c *gin.Context, userID uint, raw, kind string, expectedKind characterCardPortraitStorageKind) {
	canonical, storageKind, ok := s.characterCardOwnedImpressionStoragePath(c, userID, raw, kind)
	if !ok || storageKind != expectedKind {
		return
	}
	if key := uploadsKeyFromPath(canonical); key != "" {
		s.deleteUploadKey(key)
	}
}

func (s *Server) cleanupGeneratedCharacterCardImpressionImages(c *gin.Context, userID uint, plan characterCardImpressionUpdatePlan) {
	for _, reference := range plan.Generated {
		s.cleanupOwnedCharacterCardImpressionImage(c, userID, reference.Path, reference.Kind, characterCardPortraitStorageCurrent)
	}
}

func (s *Server) finishCharacterCardImpressionImageUpdate(c *gin.Context, userID, cardID uint, plan characterCardImpressionUpdatePlan) {
	for _, reference := range plan.Replaced {
		s.cleanupCharacterCardAssetIfUnreferenced(c, userID, cardID, reference.Path)
	}
	for _, reference := range plan.Pending {
		s.cleanupOwnedCharacterCardImpressionImage(c, userID, reference.Path, reference.Kind, characterCardPortraitStoragePending)
	}
}

func saveCharacterCardImpressions(tx *gorm.DB, rows []model.CharacterCardImpression) error {
	sort.Slice(rows, func(i, j int) bool { return rows[i].Slot < rows[j].Slot })
	for index := range rows {
		if err := tx.Omit("CharacterCard").Save(&rows[index]).Error; err != nil {
			return err
		}
	}
	return nil
}

func rotateCharacterCardImpressionImageVersions(rows []model.CharacterCardImpression, now time.Time) bool {
	changed := false
	for index := range rows {
		if rows[index].ID == 0 {
			continue
		}
		if rows[index].IconImage != "" {
			nextVersion := nextCharacterCardImpressionImageVersion(rows[index].IconImageUpdatedAt, now)
			rows[index].IconImageUpdatedAt = &nextVersion
			changed = true
		}
		if rows[index].Image != "" {
			nextVersion := nextCharacterCardImpressionImageVersion(rows[index].ImageUpdatedAt, now)
			rows[index].ImageUpdatedAt = &nextVersion
			changed = true
		}
	}
	return changed
}

func nextCharacterCardImpressionImageVersion(previous *time.Time, now time.Time) time.Time {
	now = now.UTC()
	if previous != nil {
		// PostgreSQL timestamps have microsecond resolution, so advance by a full
		// microsecond to guarantee a distinct persisted cache version.
		minimum := previous.UTC().Add(time.Microsecond)
		if now.Before(minimum) {
			return minimum
		}
	}
	return now
}

func (s *Server) loadProtectedCharacterCardImpressionImage(imageType, id string, viewerID uint) (string, time.Time, error) {
	idNum, err := strconv.ParseUint(id, 10, 32)
	if err != nil || idNum == 0 {
		return "", time.Time{}, gorm.ErrRecordNotFound
	}
	var impression model.CharacterCardImpression
	if err := database.DB.First(&impression, uint(idNum)).Error; err != nil {
		return "", time.Time{}, err
	}
	var card model.CharacterCard
	if err := database.DB.Select("id", "user_id", "status", "visibility", "updated_at").First(&card, impression.CharacterCardID).Error; err != nil {
		return "", time.Time{}, err
	}
	ownerOrModerator := viewerID != 0 && (viewerID == card.UserID || isCharacterCardModerator(viewerID))
	selected := impression
	if !ownerOrModerator {
		if snapshot, _, publicationErr := loadCharacterCardPublication(database.DB, card.ID); publicationErr == nil {
			found := false
			for _, row := range snapshot.Impressions {
				if row.ID == impression.ID && row.Active {
					selected = model.CharacterCardImpression{
						ID: row.ID, CharacterCardID: card.ID, Slot: row.Slot, Active: row.Active,
						IconImage: row.IconImage, IconImageUpdatedAt: row.IconImageUpdatedAt,
						Image: row.Image, ImageUpdatedAt: row.ImageUpdatedAt,
						UpdatedAt: row.UpdatedAt,
					}
					found = true
					break
				}
			}
			if !found {
				return "", time.Time{}, gorm.ErrRecordNotFound
			}
			card.UpdatedAt = snapshot.Card.UpdatedAt
		} else if !errors.Is(publicationErr, gorm.ErrRecordNotFound) {
			return "", time.Time{}, publicationErr
		} else if !impression.Active || !canViewCharacterCard(card, viewerID) {
			return "", time.Time{}, gorm.ErrRecordNotFound
		}
	}

	value := selected.Image
	version := selected.UpdatedAt
	if imageType == "character-card-impression-icon" {
		value = selected.IconImage
		if selected.IconImageUpdatedAt != nil {
			version = *selected.IconImageUpdatedAt
		}
	} else if imageType == "character-card-impression-image" {
		if selected.ImageUpdatedAt != nil {
			version = *selected.ImageUpdatedAt
		}
	} else {
		return "", time.Time{}, gorm.ErrRecordNotFound
	}
	if strings.TrimSpace(value) == "" {
		return "", time.Time{}, gorm.ErrRecordNotFound
	}
	if card.UpdatedAt.After(version) {
		version = card.UpdatedAt
	}
	return value, version, nil
}
