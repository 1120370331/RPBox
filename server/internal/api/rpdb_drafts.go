package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errRPDBDraftBaseVersionConflict = errors.New("rpdb draft base version conflict")
var errRPDBDraftWorkTypeConflict = errors.New("rpdb draft work type conflict")

type rpdbDraftWriteRequest struct {
	WorkID  *uint           `json:"work_id"`
	Payload json.RawMessage `json:"payload"`
}

type rpdbDraftResponse struct {
	model.RPDBDraft
	Payload json.RawMessage `json:"payload"`
}

func (s *Server) listRPDBDrafts(c *gin.Context) {
	userID := c.GetUint("userID")
	if err := migrateLegacyRPDBDrafts(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "迁移旧草稿失败"})
		return
	}

	query := database.DB.Where("author_id = ? AND status = ?", userID, model.RPDBDraftStatusActive)
	if workID := strings.TrimSpace(c.Query("work_id")); workID != "" {
		parsed, err := strconv.ParseUint(workID, 10, 64)
		if err != nil || parsed == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品 ID"})
			return
		}
		query = query.Where("work_id = ?", uint(parsed))
	}

	var drafts []model.RPDBDraft
	if err := query.Order("updated_at DESC, id DESC").Find(&drafts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载草稿箱失败"})
		return
	}
	responses := make([]rpdbDraftResponse, 0, len(drafts))
	for index := range drafts {
		if err := repairRPDBDraftTypeMismatch(&drafts[index]); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "修复草稿类型失败"})
			return
		}
		draft := drafts[index]
		responses = append(responses, makeRPDBDraftResponse(draft))
	}
	c.JSON(http.StatusOK, gin.H{"drafts": responses})
}

func (s *Server) createRPDBDraft(c *gin.Context) {
	userID := c.GetUint("userID")
	var input rpdbDraftWriteRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}

	payload := input.Payload
	baseVersion := 0
	var linkedWork *model.RPDBWork
	if input.WorkID != nil {
		var work model.RPDBWork
		if err := database.DB.First(&work, *input.WorkID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "正式作品不存在"})
			return
		}
		if work.AuthorID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权为该作品创建草稿"})
			return
		}
		if work.Status == model.RPDBStatusArchived || work.Status == model.RPDBStatusRemoved {
			c.JSON(http.StatusConflict, gin.H{"error": "该作品当前不能编辑"})
			return
		}
		linkedWork = &work
		baseVersion = work.Version
		if len(payload) == 0 || string(payload) == "null" {
			var err error
			payload, err = buildRPDBDraftPayload(database.DB, work)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "生成作品草稿失败"})
				return
			}
		}
	}

	request, normalizedPayload, err := normalizeRPDBDraftPayload(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if linkedWork != nil && request.Type != linkedWork.Type {
		c.JSON(http.StatusConflict, gin.H{"error": "关联草稿不能更改正式作品类型"})
		return
	}
	draft := model.RPDBDraft{
		AuthorID:    userID,
		WorkID:      input.WorkID,
		Type:        request.Type,
		Title:       strings.TrimSpace(request.Title),
		CoverImage:  strings.TrimSpace(request.CoverImage),
		Payload:     string(normalizedPayload),
		BaseVersion: baseVersion,
		Status:      model.RPDBDraftStatusActive,
	}
	if err := database.DB.Create(&draft).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建草稿失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"draft": makeRPDBDraftResponse(draft)})
}

func (s *Server) getRPDBDraft(c *gin.Context) {
	draft, ok := loadOwnedRPDBDraft(c)
	if !ok {
		return
	}
	if err := repairRPDBDraftTypeMismatch(&draft); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "修复草稿类型失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"draft": makeRPDBDraftResponse(draft)})
}

func (s *Server) updateRPDBDraft(c *gin.Context) {
	draft, ok := loadOwnedRPDBDraft(c)
	if !ok {
		return
	}
	var input rpdbDraftWriteRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}
	request, payload, err := normalizeRPDBDraftPayload(input.Payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if draft.WorkID != nil {
		var work model.RPDBWork
		if err := database.DB.First(&work, *draft.WorkID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "关联的正式作品不存在"})
			return
		}
		if request.Type != work.Type {
			c.JSON(http.StatusConflict, gin.H{"error": "关联草稿不能更改正式作品类型"})
			return
		}
	}
	draft.Type = request.Type
	draft.Title = strings.TrimSpace(request.Title)
	draft.CoverImage = strings.TrimSpace(request.CoverImage)
	draft.Payload = string(payload)
	if err := database.DB.Select("type", "title", "cover_image", "payload", "updated_at").Save(&draft).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存草稿失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"draft": makeRPDBDraftResponse(draft)})
}

func (s *Server) deleteRPDBDraft(c *gin.Context) {
	draft, ok := loadOwnedRPDBDraft(c)
	if !ok {
		return
	}
	if err := database.DB.Delete(&draft).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除草稿失败"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) publishRPDBDraft(c *gin.Context) {
	userID := c.GetUint("userID")
	draftID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || draftID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的草稿 ID"})
		return
	}
	role, err := rpdbUserRole(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	var publishedWork model.RPDBWork
	var revision model.RPDBRevision
	createdRevision := false
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var draft model.RPDBDraft
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND author_id = ? AND status = ?", uint(draftID), userID, model.RPDBDraftStatusActive).
			First(&draft).Error; err != nil {
			return err
		}

		request, _, err := normalizeRPDBDraftPayload([]byte(draft.Payload))
		if err != nil {
			return err
		}
		request.Status = model.RPDBStatusPublished
		if request.present == nil {
			request.present = map[string]json.RawMessage{}
		}
		request.present["status"] = json.RawMessage(`"published"`)
		if err := validateRPDBWriteRequest(request, true); err != nil {
			return err
		}
		visibility := normalizeRPDBVisibility(request.Visibility, request.IsPublic)
		guildIDs, err := validateRPDBVisibility(userID, visibility, request.GuildIDs, request.GuildID)
		if err != nil {
			return err
		}

		if draft.WorkID == nil {
			status, reviewStatus, _ := rpdbSubmissionState(model.RPDBStatusPublished, request.IsPublic, role)
			publishedWork = model.RPDBWork{
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
				GuildID:            firstRPDBGuildID(guildIDs),
				GuildIDs:           guildIDs,
				ReviewStatus:       reviewStatus,
				Version:            1,
			}
			if publishedWork.CoverImage != "" {
				now := time.Now()
				publishedWork.CoverImageUpdatedAt = &now
			}
			if err := tx.Create(&publishedWork).Error; err != nil {
				return err
			}
			if err := replaceRPDBWorkChildren(tx, publishedWork.ID, userID, role, request, true); err != nil {
				return err
			}
		} else {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&publishedWork, *draft.WorkID).Error; err != nil {
				return err
			}
			if publishedWork.AuthorID != userID {
				return gorm.ErrRecordNotFound
			}
			if request.Type != publishedWork.Type {
				return errRPDBDraftWorkTypeConflict
			}
			if draft.BaseVersion != 0 && draft.BaseVersion != publishedWork.Version {
				return errRPDBDraftBaseVersionConflict
			}
			canModerate := role == "moderator" || role == "admin"
			if publishedWork.Status == model.RPDBStatusPublished && !canModerate {
				payload, err := json.Marshal(request.present)
				if err != nil {
					return err
				}
				revision = model.RPDBRevision{
					WorkID:        publishedWork.ID,
					ProposerID:    userID,
					BaseVersion:   publishedWork.Version,
					Payload:       string(payload),
					ChangeSummary: strings.TrimSpace(request.ChangeSummary),
					Status:        model.RPDBReviewPending,
				}
				if err := tx.Create(&revision).Error; err != nil {
					return err
				}
				createdRevision = true
			} else {
				applyRPDBWorkRequest(&publishedWork, request)
				publishedWork.Visibility = visibility
				publishedWork.GuildID = firstRPDBGuildID(guildIDs)
				publishedWork.GuildIDs = guildIDs
				publishedWork.IsPublic = visibility == model.RPDBVisibilityPublic
				publishedWork.Status, publishedWork.ReviewStatus, _ = rpdbSubmissionState(model.RPDBStatusPublished, request.IsPublic, role)
				publishedWork.Version++
				if err := tx.Save(&publishedWork).Error; err != nil {
					return err
				}
				if err := replaceRPDBWorkChildren(tx, publishedWork.ID, userID, role, request, true); err != nil {
					return err
				}
			}
		}
		return tx.Delete(&draft).Error
	})

	if errors.Is(err, errRPDBDraftBaseVersionConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": "正式作品已发生更新，请基于最新版本重新创建草稿"})
		return
	}
	if errors.Is(err, errRPDBDraftWorkTypeConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": "关联草稿不能更改正式作品类型"})
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "草稿不存在或已发布"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.bumpRPDBListCache(c.Request.Context())
	if createdRevision {
		c.JSON(http.StatusAccepted, gin.H{"work": publishedWork, "revision": revision})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"work": publishedWork})
}

func loadOwnedRPDBDraft(c *gin.Context) (model.RPDBDraft, bool) {
	draftID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || draftID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的草稿 ID"})
		return model.RPDBDraft{}, false
	}
	var draft model.RPDBDraft
	if err := database.DB.Where("id = ? AND author_id = ? AND status = ?", uint(draftID), c.GetUint("userID"), model.RPDBDraftStatusActive).First(&draft).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "草稿不存在"})
		return model.RPDBDraft{}, false
	}
	return draft, true
}

func normalizeRPDBDraftPayload(payload []byte) (rpdbWorkWriteRequest, []byte, error) {
	if len(payload) == 0 || strings.TrimSpace(string(payload)) == "" || string(payload) == "null" {
		payload = []byte(`{}`)
	}
	request, err := decodeRPDBWorkWriteRequest(payload)
	if err != nil {
		return request, nil, fmt.Errorf("草稿内容格式无效")
	}
	if request.has("type") && request.Type != "" {
		switch request.Type {
		case model.RPDBWorkTypeItemShowcase, model.RPDBWorkTypeTransmog, model.RPDBWorkTypeHomeShowcase:
		default:
			return request, nil, fmt.Errorf("不支持的作品类型")
		}
	}
	if len([]rune(request.Title)) > 256 {
		return request, nil, fmt.Errorf("标题不能超过 256 个字符")
	}
	if len([]rune(request.Summary)) > 512 {
		return request, nil, fmt.Errorf("摘要不能超过 512 个字符")
	}
	if request.has("visibility") && request.Visibility != "" && !isValidRPDBVisibility(request.Visibility) {
		return request, nil, fmt.Errorf("不支持的可见范围")
	}
	for _, reference := range request.References {
		if err := validateRPDBURL(reference.URL); err != nil {
			return request, nil, err
		}
	}
	for _, media := range request.Media {
		if media.Type != "" && media.Type != "image" && media.Type != "gif" && media.Type != "video" && media.Type != "embed" {
			return request, nil, fmt.Errorf("不支持的媒体类型")
		}
		if err := validateRPDBURL(media.URL); err != nil {
			return request, nil, err
		}
		if err := validateRPDBURL(media.ThumbnailURL); err != nil {
			return request, nil, err
		}
	}
	for _, step := range request.GuideSteps {
		if step.X < 0 || step.X > 100 || step.Y < 0 || step.Y > 100 {
			return request, nil, fmt.Errorf("攻略坐标必须位于 0 到 100 之间")
		}
	}
	normalized, err := json.Marshal(request.present)
	if err != nil {
		return request, nil, fmt.Errorf("草稿内容格式无效")
	}
	return request, normalized, nil
}

func makeRPDBDraftResponse(draft model.RPDBDraft) rpdbDraftResponse {
	payload := json.RawMessage(draft.Payload)
	if !json.Valid(payload) {
		payload = json.RawMessage(`{}`)
	}
	return rpdbDraftResponse{RPDBDraft: draft, Payload: payload}
}

func repairRPDBDraftTypeMismatch(draft *model.RPDBDraft) error {
	if draft.WorkID == nil {
		return nil
	}
	var work model.RPDBWork
	if err := database.DB.First(&work, *draft.WorkID).Error; err != nil {
		return err
	}
	request, _, err := normalizeRPDBDraftPayload([]byte(draft.Payload))
	if err == nil && draft.Type == work.Type && request.Type == work.Type {
		return nil
	}
	payload, err := buildRPDBDraftPayload(database.DB, work)
	if err != nil {
		return err
	}
	draft.Type = work.Type
	draft.Title = work.Title
	draft.CoverImage = work.CoverImage
	draft.Payload = string(payload)
	draft.BaseVersion = work.Version
	return database.DB.Model(draft).Updates(map[string]interface{}{
		"type":         draft.Type,
		"title":        draft.Title,
		"cover_image":  draft.CoverImage,
		"payload":      draft.Payload,
		"base_version": draft.BaseVersion,
	}).Error
}

func buildRPDBDraftPayload(tx *gorm.DB, work model.RPDBWork) ([]byte, error) {
	request := rpdbWorkWriteRequest{
		Type:               work.Type,
		Title:              work.Title,
		Summary:            work.Summary,
		Content:            work.Content,
		ContentType:        work.ContentType,
		CoverImage:         work.CoverImage,
		RPUseCases:         work.RPUseCases,
		EffectDescription:  work.EffectDescription,
		Restrictions:       json.RawMessage(defaultString(work.Restrictions, "{}")),
		Extra:              json.RawMessage(defaultString(work.Extra, "{}")),
		GameVersion:        work.GameVersion,
		Expansion:          work.Expansion,
		AvailabilityStatus: work.AvailabilityStatus,
		BindType:           work.BindType,
		Faction:            work.Faction,
		ArmorType:          work.ArmorType,
		IsPublic:           work.IsPublic,
		Visibility:         normalizeRPDBVisibility(work.Visibility, work.IsPublic),
		GuildID:            work.GuildID,
		GuildIDs:           work.GuildIDs,
	}
	if err := tx.Model(&model.RPDBReference{}).Where("work_id = ?", work.ID).Order("sort_order ASC, id ASC").Scan(&request.References).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&model.RPDBMedia{}).Where("work_id = ?", work.ID).Order("sort_order ASC, id ASC").Scan(&request.Media).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&model.RPDBTransmogSlot{}).Where("work_id = ?", work.ID).Order("sort_order ASC, id ASC").Scan(&request.TransmogSlots).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&model.RPDBGuideStep{}).Where("work_id = ?", work.ID).Order("sort_order ASC, id ASC").Scan(&request.GuideSteps).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&model.RPDBTag{}).Where("work_id = ?", work.ID).Pluck("tag_id", &request.TagIDs).Error; err != nil {
		return nil, err
	}
	return json.Marshal(request)
}

func migrateLegacyRPDBDrafts(userID uint) error {
	var works []model.RPDBWork
	if err := database.DB.
		Where("author_id = ? AND status = ? AND COALESCE(review_status, '') IN ?",
			userID, model.RPDBStatusDraft, []string{"", model.RPDBReviewNone}).
		Order("updated_at ASC").
		Find(&works).Error; err != nil {
		return err
	}
	for _, legacy := range works {
		if err := database.DB.Transaction(func(tx *gorm.DB) error {
			var locked model.RPDBWork
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, legacy.ID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			if locked.Status != model.RPDBStatusDraft {
				return nil
			}
			payload, err := buildRPDBDraftPayload(tx, locked)
			if err != nil {
				return err
			}
			draft := model.RPDBDraft{
				AuthorID:    locked.AuthorID,
				Type:        locked.Type,
				Title:       locked.Title,
				CoverImage:  locked.CoverImage,
				Payload:     string(payload),
				BaseVersion: 0,
				Status:      model.RPDBDraftStatusActive,
				CreatedAt:   locked.CreatedAt,
				UpdatedAt:   locked.UpdatedAt,
			}
			if err := tx.Create(&draft).Error; err != nil {
				return err
			}
			if err := deleteRPDBWorkChildren(tx, locked.ID); err != nil {
				return err
			}
			return tx.Delete(&locked).Error
		}); err != nil {
			return err
		}
	}
	return nil
}
