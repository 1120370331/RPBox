package api

import (
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
)

const characterCardPortraitGalleryMax = 10

type characterCardPortraitDTO struct {
	ID             uint       `json:"id"`
	ImageURL       string     `json:"image_url"`
	ImageUpdatedAt *time.Time `json:"image_updated_at"`
	SortOrder      int        `json:"sort_order"`
	IsCover        bool       `json:"is_cover"`
}

func loadCharacterCardPortraits(tx *gorm.DB, cardIDs []uint) (map[uint][]model.CharacterCardPortrait, error) {
	result := make(map[uint][]model.CharacterCardPortrait, len(cardIDs))
	if len(cardIDs) == 0 {
		return result, nil
	}
	var rows []model.CharacterCardPortrait
	if err := tx.Where("character_card_id IN ?", cardIDs).
		Order("character_card_id ASC, sort_order ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.CharacterCardID] = append(result[row.CharacterCardID], row)
	}
	return result, nil
}

func loadCharacterCardPortraitRows(tx *gorm.DB, cardID uint) ([]model.CharacterCardPortrait, error) {
	var rows []model.CharacterCardPortrait
	if err := tx.Where("character_card_id = ?", cardID).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ensureCharacterCardPortraitRows lazily promotes a legacy single portrait to
// the gallery without changing its protected storage path or cache version.
func ensureCharacterCardPortraitRows(tx *gorm.DB, card model.CharacterCard) ([]model.CharacterCardPortrait, error) {
	rows, err := loadCharacterCardPortraitRows(tx, card.ID)
	if err != nil || len(rows) > 0 || strings.TrimSpace(card.PortraitImage) == "" {
		return rows, err
	}
	version := card.PortraitImageUpdatedAt
	if version == nil {
		fallback := card.UpdatedAt.UTC()
		version = &fallback
	}
	row := model.CharacterCardPortrait{
		CharacterCardID: card.ID,
		SortOrder:       0,
		Image:           card.PortraitImage,
		ImageUpdatedAt:  version,
	}
	if err := tx.Create(&row).Error; err != nil {
		// A concurrent promotion may have won; reload the canonical rows.
		rows, reloadErr := loadCharacterCardPortraitRows(tx, card.ID)
		if reloadErr == nil && len(rows) > 0 {
			return rows, nil
		}
		return nil, err
	}
	return []model.CharacterCardPortrait{row}, nil
}

func (s *Server) buildCharacterCardPortraitDTOs(card model.CharacterCard, rows []model.CharacterCardPortrait) []characterCardPortraitDTO {
	if len(rows) == 0 {
		if strings.TrimSpace(card.PortraitImage) == "" {
			return []characterCardPortraitDTO{}
		}
		return []characterCardPortraitDTO{{
			ID:             0,
			ImageURL:       s.characterCardPortraitURL(card),
			ImageUpdatedAt: card.PortraitImageUpdatedAt,
			SortOrder:      0,
			IsCover:        true,
		}}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].SortOrder == rows[j].SortOrder {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].SortOrder < rows[j].SortOrder
	})
	result := make([]characterCardPortraitDTO, 0, len(rows))
	for index, row := range rows {
		result = append(result, characterCardPortraitDTO{
			ID:             row.ID,
			ImageURL:       s.characterCardPortraitGalleryURL(card, row),
			ImageUpdatedAt: row.ImageUpdatedAt,
			SortOrder:      index,
			IsCover:        index == 0,
		})
	}
	return result
}

func (s *Server) characterCardPortraitGalleryURL(card model.CharacterCard, portrait model.CharacterCardPortrait) string {
	if portrait.ID == 0 || strings.TrimSpace(portrait.Image) == "" {
		return ""
	}
	versionTime := portrait.UpdatedAt
	if portrait.ImageUpdatedAt != nil {
		versionTime = *portrait.ImageUpdatedAt
	}
	imagePath := fmt.Sprintf("/api/v1/images/character-card-portrait-gallery/%d?portrait_id=%d&v=%d", card.ID, portrait.ID, versionTime.UnixNano())
	return buildAPIURL(s.cfg.Server.ApiHost, imagePath)
}

func parseCharacterCardPortraitID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("portraitId"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色大图 ID 无效"})
		return 0, false
	}
	return uint(id), true
}

func (s *Server) addCharacterCardPortrait(c *gin.Context) {
	cardID, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	userID := c.GetUint("userID")
	var req struct {
		ImageRef string `json:"image_ref" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供 image_ref"})
		return
	}
	var card model.CharacterCard
	if err := database.DB.Where("id = ? AND user_id = ?", cardID, userID).First(&card).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		return
	}
	rows, err := ensureCharacterCardPortraitRows(database.DB, card)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取角色大图失败"})
		return
	}
	if len(rows) >= characterCardPortraitGalleryMax {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色大图最多 10 张"})
		return
	}
	normalized, err := s.normalizeCharacterCardPortrait(c, userID, card, strings.TrimSpace(req.ImageRef))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !normalized.Generated || strings.TrimSpace(normalized.Path) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image_ref 必须是尚未使用的本人 pending 引用"})
		return
	}
	now := time.Now().UTC()
	newRow := model.CharacterCardPortrait{
		CharacterCardID: card.ID,
		SortOrder:       len(rows),
		Image:           normalized.Path,
		ImageUpdatedAt:  &now,
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureCharacterCardApprovedSnapshotBeforeMutation(tx, card); err != nil {
			return err
		}
		if err := tx.Create(&newRow).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{"updated_at": now}
		if len(rows) == 0 {
			updates["portrait_image"] = newRow.Image
			updates["portrait_image_updated_at"] = now
		}
		return tx.Model(&model.CharacterCard{}).Where("id = ? AND user_id = ?", card.ID, userID).Updates(updates).Error
	})
	if err != nil {
		s.cleanupOwnedCharacterCardPortrait(c, userID, normalized.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加角色大图失败"})
		return
	}
	if normalized.PendingSource != "" {
		s.cleanupOwnedCharacterCardPendingPortrait(c, userID, normalized.PendingSource)
	}
	dto, err := s.loadOwnerCharacterCardDTO(card.ID, userID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
		return
	}
	var added characterCardPortraitDTO
	for _, portrait := range dto.Portraits {
		if portrait.ID == newRow.ID {
			added = portrait
			break
		}
	}
	c.JSON(http.StatusCreated, gin.H{"portrait": added, "character_card": dto})
}

func (s *Server) reorderCharacterCardPortraits(c *gin.Context) {
	cardID, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	userID := c.GetUint("userID")
	var req struct {
		PortraitIDs []uint `json:"portrait_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "portrait_ids 无效"})
		return
	}
	var card model.CharacterCard
	if err := database.DB.Where("id = ? AND user_id = ?", cardID, userID).First(&card).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		return
	}
	rows, err := ensureCharacterCardPortraitRows(database.DB, card)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取角色大图失败"})
		return
	}
	ordered, err := validateCharacterCardPortraitOrder(rows, req.PortraitIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.persistCharacterCardPortraitOrder(card, ordered, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "排序角色大图失败"})
		return
	}
	dto, err := s.loadOwnerCharacterCardDTO(card.ID, userID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"character_card": dto})
}

func (s *Server) setCharacterCardPortraitCover(c *gin.Context) {
	cardID, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	portraitID, ok := parseCharacterCardPortraitID(c)
	if !ok {
		return
	}
	userID := c.GetUint("userID")
	var card model.CharacterCard
	if err := database.DB.Where("id = ? AND user_id = ?", cardID, userID).First(&card).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		return
	}
	rows, err := ensureCharacterCardPortraitRows(database.DB, card)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取角色大图失败"})
		return
	}
	ordered := make([]model.CharacterCardPortrait, 0, len(rows))
	found := false
	for _, row := range rows {
		if row.ID == portraitID {
			ordered = append(ordered, row)
			found = true
		}
	}
	for _, row := range rows {
		if row.ID != portraitID {
			ordered = append(ordered, row)
		}
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色大图不存在"})
		return
	}
	if err := s.persistCharacterCardPortraitOrder(card, ordered, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "设置封面失败"})
		return
	}
	dto, err := s.loadOwnerCharacterCardDTO(card.ID, userID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"character_card": dto})
}

func (s *Server) deleteCharacterCardPortrait(c *gin.Context) {
	cardID, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	portraitID, ok := parseCharacterCardPortraitID(c)
	if !ok {
		return
	}
	userID := c.GetUint("userID")
	var card model.CharacterCard
	if err := database.DB.Where("id = ? AND user_id = ?", cardID, userID).First(&card).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		return
	}
	rows, err := ensureCharacterCardPortraitRows(database.DB, card)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取角色大图失败"})
		return
	}
	remaining := make([]model.CharacterCardPortrait, 0, len(rows))
	var removed *model.CharacterCardPortrait
	for index := range rows {
		if rows[index].ID == portraitID {
			copy := rows[index]
			removed = &copy
			continue
		}
		remaining = append(remaining, rows[index])
	}
	if removed == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色大图不存在"})
		return
	}
	now := time.Now().UTC()
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureCharacterCardApprovedSnapshotBeforeMutation(tx, card); err != nil {
			return err
		}
		if err := tx.Delete(&model.CharacterCardPortrait{}, removed.ID).Error; err != nil {
			return err
		}
		if err := persistCharacterCardPortraitSortRows(tx, remaining); err != nil {
			return err
		}
		updates := map[string]interface{}{"updated_at": now}
		if len(remaining) == 0 {
			updates["portrait_image"] = ""
			updates["portrait_image_updated_at"] = now
		} else {
			updates["portrait_image"] = remaining[0].Image
			updates["portrait_image_updated_at"] = now
		}
		return tx.Model(&model.CharacterCard{}).Where("id = ? AND user_id = ?", card.ID, userID).Updates(updates).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除角色大图失败"})
		return
	}
	s.cleanupCharacterCardAssetIfUnreferenced(c, userID, card.ID, removed.Image)
	dto, err := s.loadOwnerCharacterCardDTO(card.ID, userID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功", "character_card": dto})
}

func validateCharacterCardPortraitOrder(rows []model.CharacterCardPortrait, ids []uint) ([]model.CharacterCardPortrait, error) {
	if len(rows) != len(ids) {
		return nil, errors.New("portrait_ids 必须完整包含当前全部角色大图")
	}
	byID := make(map[uint]model.CharacterCardPortrait, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	result := make([]model.CharacterCardPortrait, 0, len(ids))
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		row, exists := byID[id]
		if !exists || id == 0 {
			return nil, errors.New("portrait_ids 包含不属于当前人物卡的图片")
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, errors.New("portrait_ids 不能重复")
		}
		seen[id] = struct{}{}
		result = append(result, row)
	}
	return result, nil
}

func (s *Server) persistCharacterCardPortraitOrder(card model.CharacterCard, rows []model.CharacterCardPortrait, userID uint) error {
	now := time.Now().UTC()
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureCharacterCardApprovedSnapshotBeforeMutation(tx, card); err != nil {
			return err
		}
		if err := persistCharacterCardPortraitSortRows(tx, rows); err != nil {
			return err
		}
		updates := map[string]interface{}{"updated_at": now}
		if len(rows) > 0 {
			updates["portrait_image"] = rows[0].Image
			updates["portrait_image_updated_at"] = now
		}
		return tx.Model(&model.CharacterCard{}).Where("id = ? AND user_id = ?", card.ID, userID).Updates(updates).Error
	})
}

func persistCharacterCardPortraitSortRows(tx *gorm.DB, rows []model.CharacterCardPortrait) error {
	// Move rows outside the valid order range first so the unique card/order
	// index never observes a transient collision while swapping two entries.
	for index, row := range rows {
		if err := tx.Model(&model.CharacterCardPortrait{}).Where("id = ?", row.ID).Update("sort_order", 1000+index).Error; err != nil {
			return err
		}
	}
	for index, row := range rows {
		if err := tx.Model(&model.CharacterCardPortrait{}).Where("id = ?", row.ID).Update("sort_order", index).Error; err != nil {
			return err
		}
	}
	return nil
}
