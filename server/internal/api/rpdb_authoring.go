package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
)

type rpdbReferenceInput struct {
	ExternalType      string `json:"external_type"`
	ExternalID        string `json:"external_id"`
	Name              string `json:"name"`
	Icon              string `json:"icon"`
	Quality           string `json:"quality"`
	Description       string `json:"description"`
	AcquisitionMethod string `json:"acquisition_method"`
	Source            string `json:"source"`
	URL               string `json:"url"`
	Locale            string `json:"locale"`
	IsPrimary         bool   `json:"is_primary"`
	SortOrder         int    `json:"sort_order"`
}

type rpdbMediaInput struct {
	Type         string `json:"type"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Caption      string `json:"caption"`
	SortOrder    int    `json:"sort_order"`
}

type rpdbTransmogSlotInput struct {
	ReferenceID *uint  `json:"reference_id"`
	Slot        string `json:"slot"`
	Role        string `json:"role"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	WowheadURL  string `json:"wowhead_url"`
	Variant     string `json:"variant"`
	Note        string `json:"note"`
	SortOrder   int    `json:"sort_order"`
}

type rpdbGuideStepInput struct {
	SortOrder    int     `json:"sort_order"`
	Title        string  `json:"title"`
	Body         string  `json:"body"`
	Zone         string  `json:"zone"`
	MapID        string  `json:"map_id"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Label        string  `json:"label"`
	Prerequisite string  `json:"prerequisite"`
}

type rpdbWorkWriteRequest struct {
	Type               string                  `json:"type"`
	Title              string                  `json:"title"`
	Summary            string                  `json:"summary"`
	Content            string                  `json:"content"`
	ContentType        string                  `json:"content_type"`
	CoverImage         string                  `json:"cover_image"`
	RPUseCases         string                  `json:"rp_use_cases"`
	EffectDescription  string                  `json:"effect_description"`
	Restrictions       json.RawMessage         `json:"restrictions"`
	Extra              json.RawMessage         `json:"extra"`
	GameVersion        string                  `json:"game_version"`
	Expansion          string                  `json:"expansion"`
	AvailabilityStatus string                  `json:"availability_status"`
	BindType           string                  `json:"bind_type"`
	Faction            string                  `json:"faction"`
	ArmorType          string                  `json:"armor_type"`
	Status             string                  `json:"status"`
	IsPublic           bool                    `json:"is_public"`
	Visibility         string                  `json:"visibility"`
	GuildID            *uint                   `json:"guild_id"`
	References         []rpdbReferenceInput    `json:"references"`
	Media              []rpdbMediaInput        `json:"media"`
	TransmogSlots      []rpdbTransmogSlotInput `json:"transmog_slots"`
	GuideSteps         []rpdbGuideStepInput    `json:"guide_steps"`
	TagIDs             []uint                  `json:"tag_ids"`
	TagNames           []string                `json:"tag_names"`
	ChangeSummary      string                  `json:"change_summary"`
	present            map[string]json.RawMessage
}

func (s *Server) createRPDBWork(c *gin.Context) {
	userID := c.GetUint("userID")
	request, err := bindRPDBWorkWriteRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}
	if err := validateRPDBWriteRequest(request, true); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role, err := rpdbUserRole(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}
	status, reviewStatus, _ := rpdbSubmissionState(request.Status, request.IsPublic, role)
	visibility := normalizeRPDBVisibility(request.Visibility, request.IsPublic)
	guildID, err := validateRPDBVisibility(userID, visibility, request.GuildID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	work := model.RPDBWork{
		AuthorID:           userID,
		Type:               request.Type,
		Title:              strings.TrimSpace(request.Title),
		Slug:               fmt.Sprintf("rpdb-%d-%d", userID, time.Now().UnixNano()),
		Summary:            strings.TrimSpace(request.Summary),
		Content:            request.Content,
		ContentType:        defaultString(request.ContentType, "html"),
		CoverImage:         strings.TrimSpace(request.CoverImage),
		RPUseCases:         request.RPUseCases,
		EffectDescription:  request.EffectDescription,
		Restrictions:       rawJSONOrObject(request.Restrictions),
		Extra:              rawJSONOrObject(request.Extra),
		GameVersion:        strings.TrimSpace(request.GameVersion),
		Expansion:          strings.TrimSpace(request.Expansion),
		AvailabilityStatus: strings.TrimSpace(request.AvailabilityStatus),
		BindType:           strings.TrimSpace(request.BindType),
		Faction:            strings.TrimSpace(request.Faction),
		ArmorType:          strings.TrimSpace(request.ArmorType),
		VerificationStatus: model.RPDBVerificationUnverified,
		Status:             status,
		IsPublic:           visibility == model.RPDBVisibilityPublic,
		Visibility:         visibility,
		GuildID:            guildID,
		ReviewStatus:       reviewStatus,
		Version:            1,
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&work).Error; err != nil {
			return err
		}
		return replaceRPDBWorkChildren(tx, work.ID, userID, role, request, true)
	}); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "创建作品失败，可能存在重复的关联物品"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"work": work})
}

func (s *Server) updateRPDBWork(c *gin.Context) {
	userID := c.GetUint("userID")
	workID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || workID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品 ID"})
		return
	}

	request, err := bindRPDBWorkWriteRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}

	var work model.RPDBWork
	if err := database.DB.First(&work, uint(workID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	role, err := rpdbUserRole(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}
	canModerate := role == "moderator" || role == "admin"
	if work.AuthorID != userID && !canModerate {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权编辑该作品"})
		return
	}

	if work.Status == model.RPDBStatusPublished && !canModerate {
		payload, err := json.Marshal(request.present)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无法保存修订内容"})
			return
		}
		revision := model.RPDBRevision{
			WorkID:        work.ID,
			ProposerID:    userID,
			BaseVersion:   work.Version,
			Payload:       string(payload),
			ChangeSummary: strings.TrimSpace(request.ChangeSummary),
			Status:        model.RPDBReviewPending,
		}
		if err := database.DB.Create(&revision).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建修订申请失败"})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"revision": revision})
		return
	}

	if err := validateRPDBWriteRequest(request, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	applyRPDBWorkRequest(&work, request)
	if request.has("visibility") || request.has("guild_id") || request.has("is_public") {
		visibility := request.Visibility
		if !request.has("visibility") {
			visibility = work.Visibility
		}
		if strings.TrimSpace(visibility) == "" {
			visibility = normalizeRPDBVisibility("", request.IsPublic)
		}
		guildID, visibilityErr := validateRPDBVisibility(userID, visibility, request.GuildID)
		if visibilityErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": visibilityErr.Error()})
			return
		}
		work.Visibility = visibility
		work.GuildID = guildID
		work.IsPublic = visibility == model.RPDBVisibilityPublic
	}
	if work.Status != model.RPDBStatusPublished && (request.has("status") || request.has("is_public")) {
		work.Status, work.ReviewStatus, _ = rpdbSubmissionState(request.Status, request.IsPublic, role)
		work.IsPublic = work.Visibility == model.RPDBVisibilityPublic
	}
	work.Version++

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&work).Error; err != nil {
			return err
		}
		return replaceRPDBWorkChildren(tx, work.ID, userID, role, request, false)
	}); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "更新作品失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"work": work})
}

func (s *Server) deleteRPDBWork(c *gin.Context) {
	userID := c.GetUint("userID")
	workID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || workID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品 ID"})
		return
	}

	var work model.RPDBWork
	if err := database.DB.First(&work, uint(workID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	role, _ := rpdbUserRole(userID)
	if work.AuthorID != userID && role != "moderator" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除该作品"})
		return
	}

	if work.Status == model.RPDBStatusDraft {
		if err := database.DB.Transaction(func(tx *gorm.DB) error {
			if err := deleteRPDBWorkChildren(tx, work.ID); err != nil {
				return err
			}
			return tx.Delete(&work).Error
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除作品失败"})
			return
		}
	} else {
		if err := database.DB.Model(&work).Updates(map[string]interface{}{
			"status":     model.RPDBStatusArchived,
			"is_public":  false,
			"visibility": model.RPDBVisibilityPrivate,
			"guild_id":   nil,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "归档作品失败"})
			return
		}
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) listMyRPDBWorks(c *gin.Context) {
	userID := c.GetUint("userID")
	var works []model.RPDBWork
	if err := database.DB.Where("author_id = ? AND status <> ?", userID, model.RPDBStatusArchived).Order("updated_at DESC").Find(&works).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载作品失败"})
		return
	}
	cards, err := buildRPDBWorkCards(works, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载作品信息失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"works": cards})
}

type rpdbVisibilityRequest struct {
	Visibility string `json:"visibility" binding:"required"`
	GuildID    *uint  `json:"guild_id"`
}

func (s *Server) updateRPDBWorkVisibility(c *gin.Context) {
	userID := c.GetUint("userID")
	workID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || workID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品 ID"})
		return
	}

	var request rpdbVisibilityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择可见范围"})
		return
	}

	var work model.RPDBWork
	if err := database.DB.First(&work, uint(workID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	if work.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权管理该作品"})
		return
	}

	if !isValidRPDBVisibility(request.Visibility) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的可见范围"})
		return
	}
	visibility := normalizeRPDBVisibility(request.Visibility, false)
	guildID, err := validateRPDBVisibility(userID, visibility, request.GuildID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := database.DB.Model(&work).Updates(map[string]interface{}{
		"visibility": visibility,
		"guild_id":   guildID,
		"is_public":  visibility == model.RPDBVisibilityPublic,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新可见范围失败"})
		return
	}
	work.Visibility = visibility
	work.GuildID = guildID
	work.IsPublic = visibility == model.RPDBVisibilityPublic
	c.JSON(http.StatusOK, gin.H{"work": work})
}

func normalizeRPDBVisibility(value string, isPublic bool) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case model.RPDBVisibilityPublic:
		return model.RPDBVisibilityPublic
	case model.RPDBVisibilityGuild:
		return model.RPDBVisibilityGuild
	case model.RPDBVisibilityPrivate:
		return model.RPDBVisibilityPrivate
	default:
		if isPublic {
			return model.RPDBVisibilityPublic
		}
		return model.RPDBVisibilityPrivate
	}
}

func isValidRPDBVisibility(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case model.RPDBVisibilityPublic, model.RPDBVisibilityGuild, model.RPDBVisibilityPrivate:
		return true
	default:
		return false
	}
}

func validateRPDBVisibility(userID uint, visibility string, guildID *uint) (*uint, error) {
	if visibility != model.RPDBVisibilityGuild {
		return nil, nil
	}
	if guildID == nil || *guildID == 0 {
		return nil, fmt.Errorf("请选择允许查看的公会")
	}
	var membershipCount int64
	if err := database.DB.Model(&model.GuildMember{}).
		Where("guild_id = ? AND user_id = ?", *guildID, userID).
		Count(&membershipCount).Error; err != nil {
		return nil, fmt.Errorf("校验公会成员身份失败")
	}
	if membershipCount == 0 {
		return nil, fmt.Errorf("只能选择你已加入的公会")
	}
	return guildID, nil
}

func validateRPDBWriteRequest(request rpdbWorkWriteRequest, creating bool) error {
	if creating || request.has("type") {
		switch request.Type {
		case model.RPDBWorkTypeItemShowcase, model.RPDBWorkTypeTransmog, model.RPDBWorkTypeHomeShowcase:
		default:
			return fmt.Errorf("不支持的作品类型")
		}
	}
	if creating || request.has("title") {
		title := strings.TrimSpace(request.Title)
		if title == "" || len([]rune(title)) > 256 {
			return fmt.Errorf("标题长度必须为 1 到 256 个字符")
		}
	}
	if len([]rune(request.Summary)) > 512 {
		return fmt.Errorf("摘要不能超过 512 个字符")
	}
	if request.has("visibility") {
		if !isValidRPDBVisibility(request.Visibility) {
			return fmt.Errorf("不支持的可见范围")
		}
	}
	for _, reference := range request.References {
		if strings.TrimSpace(reference.ExternalType) == "" || strings.TrimSpace(reference.Name) == "" {
			return fmt.Errorf("内容清单必须包含类型和名称")
		}
		if err := validateRPDBURL(reference.URL); err != nil {
			return err
		}
	}
	for _, media := range request.Media {
		if media.Type != "image" && media.Type != "gif" && media.Type != "video" && media.Type != "embed" {
			return fmt.Errorf("不支持的媒体类型")
		}
		if err := validateRPDBURL(media.URL); err != nil {
			return err
		}
		if err := validateRPDBURL(media.ThumbnailURL); err != nil {
			return err
		}
	}
	for _, step := range request.GuideSteps {
		if step.X < 0 || step.X > 100 || step.Y < 0 || step.Y > 100 {
			return fmt.Errorf("攻略坐标必须位于 0 到 100 之间")
		}
	}
	return nil
}

func validateRPDBURL(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "/uploads/") || strings.HasPrefix(value, "uploads/") {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("链接仅支持 http、https 或 uploads 路径")
	}
	return nil
}

func rpdbSubmissionState(requestedStatus string, requestedPublic bool, role string) (string, string, bool) {
	if requestedStatus == model.RPDBStatusDraft || requestedStatus == "" {
		return model.RPDBStatusDraft, model.RPDBReviewNone, requestedPublic
	}
	if role == "moderator" || role == "admin" {
		return model.RPDBStatusPublished, model.RPDBReviewApproved, true
	}
	return model.RPDBStatusPending, model.RPDBReviewPending, true
}

func rpdbUserRole(userID uint) (string, error) {
	var user model.User
	if err := database.DB.Select("id", "role").First(&user, userID).Error; err != nil {
		return "", err
	}
	return user.Role, nil
}

func applyRPDBWorkRequest(work *model.RPDBWork, request rpdbWorkWriteRequest) {
	if request.has("type") {
		work.Type = request.Type
	}
	if request.has("title") {
		work.Title = strings.TrimSpace(request.Title)
	}
	if request.has("summary") {
		work.Summary = strings.TrimSpace(request.Summary)
	}
	if request.has("content") {
		work.Content = request.Content
	}
	if request.has("content_type") {
		work.ContentType = defaultString(request.ContentType, work.ContentType)
	}
	if request.has("cover_image") {
		work.CoverImage = strings.TrimSpace(request.CoverImage)
	}
	if request.has("rp_use_cases") {
		work.RPUseCases = request.RPUseCases
	}
	if request.has("effect_description") {
		work.EffectDescription = request.EffectDescription
	}
	if request.has("restrictions") {
		work.Restrictions = rawJSONOrExisting(request.Restrictions, work.Restrictions)
	}
	if request.has("extra") {
		work.Extra = rawJSONOrExisting(request.Extra, work.Extra)
	}
	if request.has("game_version") {
		work.GameVersion = strings.TrimSpace(request.GameVersion)
	}
	if request.has("expansion") {
		work.Expansion = strings.TrimSpace(request.Expansion)
	}
	if request.has("availability_status") {
		work.AvailabilityStatus = strings.TrimSpace(request.AvailabilityStatus)
	}
	if request.has("bind_type") {
		work.BindType = strings.TrimSpace(request.BindType)
	}
	if request.has("faction") {
		work.Faction = strings.TrimSpace(request.Faction)
	}
	if request.has("armor_type") {
		work.ArmorType = strings.TrimSpace(request.ArmorType)
	}
}

func replaceRPDBWorkChildren(tx *gorm.DB, workID uint, userID uint, role string, request rpdbWorkWriteRequest, replaceAll bool) error {
	if replaceAll || request.has("references") {
		if err := tx.Where("work_id = ?", workID).Delete(&model.RPDBReference{}).Error; err != nil {
			return err
		}
		for _, input := range request.References {
			reference := model.RPDBReference{
				WorkID:            workID,
				ExternalType:      strings.TrimSpace(input.ExternalType),
				ExternalID:        strings.TrimSpace(input.ExternalID),
				Name:              strings.TrimSpace(input.Name),
				Icon:              strings.TrimSpace(input.Icon),
				Quality:           strings.TrimSpace(input.Quality),
				Description:       strings.TrimSpace(input.Description),
				AcquisitionMethod: strings.TrimSpace(input.AcquisitionMethod),
				Source:            strings.TrimSpace(input.Source),
				URL:               strings.TrimSpace(input.URL),
				Locale:            strings.TrimSpace(input.Locale),
				IsPrimary:         input.IsPrimary,
				SortOrder:         input.SortOrder,
			}
			if err := tx.Create(&reference).Error; err != nil {
				return err
			}
		}
	}

	if replaceAll || request.has("media") {
		if err := tx.Where("work_id = ?", workID).Delete(&model.RPDBMedia{}).Error; err != nil {
			return err
		}
		mediaReview := model.RPDBReviewPending
		if role == "moderator" || role == "admin" {
			mediaReview = model.RPDBReviewApproved
		}
		for _, input := range request.Media {
			authorID := userID
			media := model.RPDBMedia{
				WorkID:       workID,
				AuthorID:     &authorID,
				Type:         input.Type,
				URL:          strings.TrimSpace(input.URL),
				ThumbnailURL: strings.TrimSpace(input.ThumbnailURL),
				Caption:      strings.TrimSpace(input.Caption),
				SortOrder:    input.SortOrder,
				Meta:         "{}",
				ReviewStatus: mediaReview,
			}
			if err := tx.Create(&media).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&model.RPDBWork{}).Where("id = ?", workID).Update("media_count", len(request.Media)).Error; err != nil {
			return err
		}
	}

	if replaceAll || request.has("transmog_slots") {
		if err := tx.Where("work_id = ?", workID).Delete(&model.RPDBTransmogSlot{}).Error; err != nil {
			return err
		}
		for _, input := range request.TransmogSlots {
			slot := model.RPDBTransmogSlot{
				WorkID:      workID,
				ReferenceID: input.ReferenceID,
				Slot:        strings.TrimSpace(input.Slot),
				Role:        defaultString(input.Role, "required"),
				Name:        strings.TrimSpace(input.Name),
				Description: strings.TrimSpace(input.Description),
				Source:      strings.TrimSpace(input.Source),
				WowheadURL:  strings.TrimSpace(input.WowheadURL),
				Variant:     strings.TrimSpace(input.Variant),
				Note:        strings.TrimSpace(input.Note),
				SortOrder:   input.SortOrder,
			}
			if err := tx.Create(&slot).Error; err != nil {
				return err
			}
		}
	}

	if replaceAll || request.has("guide_steps") {
		if err := tx.Where("work_id = ?", workID).Delete(&model.RPDBGuideStep{}).Error; err != nil {
			return err
		}
		for _, input := range request.GuideSteps {
			step := model.RPDBGuideStep{
				WorkID:       workID,
				SortOrder:    input.SortOrder,
				Title:        strings.TrimSpace(input.Title),
				Body:         input.Body,
				Zone:         strings.TrimSpace(input.Zone),
				MapID:        strings.TrimSpace(input.MapID),
				X:            input.X,
				Y:            input.Y,
				Label:        strings.TrimSpace(input.Label),
				Prerequisite: input.Prerequisite,
				Meta:         "{}",
			}
			if err := tx.Create(&step).Error; err != nil {
				return err
			}
		}
	}

	if replaceAll || request.has("tag_ids") || request.has("tag_names") {
		if err := tx.Where("work_id = ?", workID).Delete(&model.RPDBTag{}).Error; err != nil {
			return err
		}
		seenTagIDs := map[uint]struct{}{}
		for _, tagID := range request.TagIDs {
			if tagID == 0 {
				continue
			}
			if _, exists := seenTagIDs[tagID]; exists {
				continue
			}
			seenTagIDs[tagID] = struct{}{}
			tag := model.RPDBTag{WorkID: workID, TagID: tagID, AddedBy: userID}
			if err := tx.Create(&tag).Error; err != nil {
				return err
			}
		}
		tagNames := normalizeRPDBCustomTagNames(request.TagNames)
		customTagsPublic := role == "moderator" || role == "admin"
		for _, name := range tagNames {
			tag, err := findOrCreateRPDBCustomTag(tx, userID, name, customTagsPublic)
			if err != nil {
				return err
			}
			if _, exists := seenTagIDs[tag.ID]; exists {
				continue
			}
			seenTagIDs[tag.ID] = struct{}{}
			workTag := model.RPDBTag{WorkID: workID, TagID: tag.ID, AddedBy: userID}
			if err := tx.Create(&workTag).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeRPDBCustomTagNames(names []string) []string {
	cleaned := make([]string, 0, len(names))
	seen := map[string]struct{}{}
	for _, name := range names {
		normalized := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(name), "#"))
		if normalized == "" {
			continue
		}
		key := strings.ToLower(normalized)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, normalized)
	}
	return cleaned
}

func findOrCreateRPDBCustomTag(tx *gorm.DB, userID uint, name string, isPublic bool) (model.Tag, error) {
	var tag model.Tag
	err := tx.Where("LOWER(name) = ? AND category = ?", strings.ToLower(name), "rpdb").First(&tag).Error
	if err == nil {
		if isPublic && !tag.IsPublic {
			tag.IsPublic = true
			if err := tx.Model(&tag).Update("is_public", true).Error; err != nil {
				return tag, err
			}
		}
		return tag, nil
	}
	if err != gorm.ErrRecordNotFound {
		return tag, err
	}
	tag = model.Tag{
		Name:      name,
		Color:     "B87333",
		Category:  "rpdb",
		Type:      "custom",
		CreatorID: userID,
		IsPublic:  isPublic,
	}
	return tag, tx.Create(&tag).Error
}

func publishRPDBWorkCustomTags(tx *gorm.DB, workID uint) error {
	return tx.Model(&model.Tag{}).
		Where("category = ? AND type = ? AND id IN (?)",
			"rpdb",
			"custom",
			tx.Model(&model.RPDBTag{}).Select("tag_id").Where("work_id = ?", workID),
		).
		Update("is_public", true).Error
}

func deleteRPDBWorkChildren(tx *gorm.DB, workID uint) error {
	for _, target := range []interface{}{
		&model.RPDBReference{},
		&model.RPDBMedia{},
		&model.RPDBTransmogSlot{},
		&model.RPDBGuideStep{},
		&model.RPDBTag{},
	} {
		if err := tx.Where("work_id = ?", workID).Delete(target).Error; err != nil {
			return err
		}
	}
	return nil
}

func rawJSONOrObject(value json.RawMessage) string {
	if len(value) == 0 {
		return "{}"
	}
	return string(value)
}

func rawJSONOrExisting(value json.RawMessage, existing string) string {
	if len(value) == 0 {
		return existing
	}
	return string(value)
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func bindRPDBWorkWriteRequest(c *gin.Context) (rpdbWorkWriteRequest, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		return rpdbWorkWriteRequest{}, fmt.Errorf("read request body")
	}
	return decodeRPDBWorkWriteRequest(body)
}

func decodeRPDBWorkWriteRequest(body []byte) (rpdbWorkWriteRequest, error) {
	var request rpdbWorkWriteRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return request, err
	}
	if err := json.Unmarshal(body, &request.present); err != nil {
		return request, err
	}
	return request, nil
}

func normalizeLegacyRPDBRevisionRequest(request rpdbWorkWriteRequest) rpdbWorkWriteRequest {
	legacyFields := []string{
		"type", "title", "summary", "content", "content_type", "cover_image",
		"rp_use_cases", "effect_description", "restrictions", "extra",
		"game_version", "expansion", "availability_status", "bind_type", "faction", "armor_type",
		"status", "is_public", "references", "media", "transmog_slots", "guide_steps", "tag_ids",
		"change_summary",
	}
	for _, field := range legacyFields {
		if !request.has(field) {
			return request
		}
	}

	collectionFields := []string{"references", "media", "transmog_slots", "guide_steps", "tag_ids", "tag_names"}
	hasLegacyNullCollection := false
	for _, field := range collectionFields {
		if strings.TrimSpace(string(request.present[field])) == "null" {
			hasLegacyNullCollection = true
			break
		}
	}
	if !hasLegacyNullCollection {
		return request
	}

	stringFields := map[string]string{
		"type":                request.Type,
		"title":               request.Title,
		"summary":             request.Summary,
		"content":             request.Content,
		"content_type":        request.ContentType,
		"cover_image":         request.CoverImage,
		"rp_use_cases":        request.RPUseCases,
		"effect_description":  request.EffectDescription,
		"game_version":        request.GameVersion,
		"expansion":           request.Expansion,
		"availability_status": request.AvailabilityStatus,
		"bind_type":           request.BindType,
		"faction":             request.Faction,
		"armor_type":          request.ArmorType,
		"status":              request.Status,
		"change_summary":      request.ChangeSummary,
	}
	for field, value := range stringFields {
		if value == "" {
			delete(request.present, field)
		}
	}
	for _, field := range []string{"restrictions", "extra"} {
		if strings.TrimSpace(string(request.present[field])) == "null" {
			delete(request.present, field)
		}
	}
	for _, field := range collectionFields {
		if strings.TrimSpace(string(request.present[field])) == "null" {
			delete(request.present, field)
		}
	}
	if !request.IsPublic {
		delete(request.present, "is_public")
	}
	return request
}

func (request rpdbWorkWriteRequest) has(field string) bool {
	_, ok := request.present[field]
	return ok
}
