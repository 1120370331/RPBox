package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errRPDBStaleRevision = errors.New("rpdb revision base version conflict")

func (s *Server) listPendingRPDBWorks(c *gin.Context) {
	page, pageSize := rpdbModerationPage(c)
	var total int64
	query := database.DB.Model(&model.RPDBWork{}).Where("review_status = ?", model.RPDBReviewPending)
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载待审核作品失败"})
		return
	}
	var works []model.RPDBWork
	if err := query.Order("created_at ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&works).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载待审核作品失败"})
		return
	}
	cards, err := buildRPDBWorkCards(works, c.GetUint("userID"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载作品信息失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"works": cards, "total": total})
}

func (s *Server) reviewRPDBWork(c *gin.Context) {
	workID, ok := parseRPDBModerationID(c, "id", "作品")
	if !ok {
		return
	}
	request, ok := bindRPDBReviewRequest(c)
	if !ok {
		return
	}
	var work model.RPDBWork
	if err := database.DB.Where("id = ? AND review_status = ?", workID, model.RPDBReviewPending).First(&work).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "待审核作品不存在"})
		return
	}
	now := time.Now()
	updates := map[string]interface{}{
		"review_status":  map[bool]string{true: model.RPDBReviewApproved, false: model.RPDBReviewRejected}[request.Action == "approve"],
		"reviewer_id":    c.GetUint("userID"),
		"review_comment": strings.TrimSpace(request.Comment),
		"reviewed_at":    &now,
	}
	if request.Action == "approve" {
		updates["status"] = model.RPDBStatusPublished
		updates["is_public"] = normalizeRPDBVisibility(work.Visibility, work.IsPublic) == model.RPDBVisibilityPublic
	} else {
		// 正式作品与草稿分表。审核拒绝只改变正式作品的审核状态，
		// 不能把正式作品降级成 draft 行。
		updates["status"] = model.RPDBStatusPending
		updates["is_public"] = false
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&work).Updates(updates).Error; err != nil {
			return err
		}
		if request.Action == "approve" {
			if err := publishRPDBWorkCustomTags(tx, work.ID); err != nil {
				return err
			}
			_, err := service.AwardActivityReward(
				tx,
				work.AuthorID,
				"rpdb_publish",
				fmt.Sprintf("rpdb:%d", work.ID),
				0,
				service.RPDBPublishExperience,
			)
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存审核结果失败"})
		return
	}
	s.bumpRPDBListCache(c.Request.Context())
	logAdminAction(c, "review_rpdb_work", "rpdb_work", work.ID, work.Title, map[string]interface{}{"action": request.Action, "comment": request.Comment})
	notifyModerationResult(work.AuthorID, "rpdb_work", work.ID, "RP 数据库作品《"+work.Title+"》", request.Action, request.Comment)
	c.JSON(http.StatusOK, gin.H{"message": "审核完成"})
}

func (s *Server) listPendingRPDBMedia(c *gin.Context) {
	page, pageSize := rpdbModerationPage(c)
	var total int64
	query := database.DB.Model(&model.RPDBMedia{}).Where("review_status = ?", model.RPDBReviewPending)
	_ = query.Count(&total).Error
	var media []model.RPDBMedia
	if err := query.Order("created_at ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&media).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载待审核媒体失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"media": media, "total": total})
}

func (s *Server) reviewRPDBMedia(c *gin.Context) {
	mediaID, ok := parseRPDBModerationID(c, "id", "媒体")
	if !ok {
		return
	}
	request, ok := bindRPDBReviewRequest(c)
	if !ok {
		return
	}
	var media model.RPDBMedia
	if err := database.DB.Where("id = ? AND review_status = ?", mediaID, model.RPDBReviewPending).First(&media).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "待审核媒体不存在"})
		return
	}
	now := time.Now()
	status := model.RPDBReviewRejected
	if request.Action == "approve" {
		status = model.RPDBReviewApproved
	}
	if err := database.DB.Model(&media).Updates(map[string]interface{}{
		"review_status": status, "reviewer_id": c.GetUint("userID"), "review_comment": strings.TrimSpace(request.Comment), "reviewed_at": &now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存媒体审核结果失败"})
		return
	}
	s.bumpRPDBListCache(c.Request.Context())
	logAdminAction(c, "review_rpdb_media", "rpdb_media", media.ID, media.URL, map[string]interface{}{"action": request.Action})
	if media.AuthorID != nil {
		notifyModerationResult(*media.AuthorID, "rpdb_work", media.WorkID, "RP 数据库媒体", request.Action, request.Comment)
	}
	c.JSON(http.StatusOK, gin.H{"message": "审核完成"})
}

func (s *Server) listPendingRPDBRevisions(c *gin.Context) {
	page, pageSize := rpdbModerationPage(c)
	var total int64
	query := database.DB.Model(&model.RPDBRevision{}).Where("status = ?", model.RPDBReviewPending)
	_ = query.Count(&total).Error
	var revisions []model.RPDBRevision
	if err := query.Order("created_at ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&revisions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载待审核修订失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"revisions": revisions, "total": total})
}

func (s *Server) reviewRPDBRevision(c *gin.Context) {
	revisionID, ok := parseRPDBModerationID(c, "id", "修订")
	if !ok {
		return
	}
	request, ok := bindRPDBReviewRequest(c)
	if !ok {
		return
	}
	moderatorID := c.GetUint("userID")
	var revision model.RPDBRevision
	var work model.RPDBWork
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ?", revisionID, model.RPDBReviewPending).
			First(&revision).Error; err != nil {
			return err
		}
		now := time.Now()
		if request.Action == "reject" {
			return tx.Model(&revision).Updates(map[string]interface{}{
				"status": model.RPDBReviewRejected, "reviewer_id": moderatorID, "review_comment": strings.TrimSpace(request.Comment), "reviewed_at": &now,
			}).Error
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&work, revision.WorkID).Error; err != nil {
			return err
		}
		if work.Version != revision.BaseVersion {
			return errRPDBStaleRevision
		}
		payload, err := decodeRPDBWorkWriteRequest([]byte(revision.Payload))
		if err != nil {
			return err
		}
		payload = normalizeLegacyRPDBRevisionRequest(payload)
		if err := validateRPDBWriteRequest(payload, false); err != nil {
			return err
		}
		applyRPDBWorkRequest(&work, payload)
		if payload.has("visibility") || payload.has("guild_ids") || payload.has("guild_id") || payload.has("is_public") {
			visibility := payload.Visibility
			if !payload.has("visibility") {
				visibility = work.Visibility
			}
			if strings.TrimSpace(visibility) == "" {
				visibility = normalizeRPDBVisibility("", payload.IsPublic)
			}
			requestedGuildIDs := payload.GuildIDs
			legacyGuildID := payload.GuildID
			if !payload.has("guild_ids") && !payload.has("guild_id") {
				requestedGuildIDs = work.GuildIDs
				legacyGuildID = work.GuildID
			}
			guildIDs, visibilityErr := validateRPDBVisibility(revision.ProposerID, visibility, requestedGuildIDs, legacyGuildID)
			if visibilityErr != nil {
				return visibilityErr
			}
			work.Visibility = visibility
			work.GuildID = firstRPDBGuildID(guildIDs)
			work.GuildIDs = guildIDs
			work.IsPublic = visibility == model.RPDBVisibilityPublic
		}
		work.Version++
		if err := tx.Save(&work).Error; err != nil {
			return err
		}
		if err := replaceRPDBWorkChildren(tx, work.ID, revision.ProposerID, "moderator", payload, false); err != nil {
			return err
		}
		return tx.Model(&revision).Updates(map[string]interface{}{
			"status": model.RPDBReviewApproved, "reviewer_id": moderatorID, "review_comment": strings.TrimSpace(request.Comment), "reviewed_at": &now, "applied_at": &now,
		}).Error
	})
	if errors.Is(err, errRPDBStaleRevision) {
		c.JSON(http.StatusConflict, gin.H{"error": "作品已更新，该修订需要作者基于最新版本重新提交"})
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "待审核修订不存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "应用修订失败"})
		return
	}
	s.bumpRPDBListCache(c.Request.Context())
	logAdminAction(c, "review_rpdb_revision", "rpdb_revision", revision.ID, revision.ChangeSummary, map[string]interface{}{"action": request.Action})
	notifyModerationResult(revision.ProposerID, "rpdb_work", revision.WorkID, "RP 数据库修订", request.Action, request.Comment)
	c.JSON(http.StatusOK, gin.H{"message": "审核完成"})
}

func (s *Server) hideRPDBWorkByMod(c *gin.Context) {
	workID, ok := parseRPDBModerationID(c, "id", "作品")
	if !ok {
		return
	}
	var work model.RPDBWork
	if err := database.DB.First(&work, workID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	if err := database.DB.Model(&work).Updates(map[string]interface{}{"status": model.RPDBStatusPending, "review_status": model.RPDBReviewPending, "is_public": false}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "隐藏作品失败"})
		return
	}
	s.bumpRPDBListCache(c.Request.Context())
	logAdminAction(c, "hide_rpdb_work", "rpdb_work", work.ID, work.Title, nil)
	c.JSON(http.StatusOK, gin.H{"message": "作品已隐藏并打回审核"})
}

func (s *Server) deleteRPDBWorkByMod(c *gin.Context) {
	workID, ok := parseRPDBModerationID(c, "id", "作品")
	if !ok {
		return
	}
	var work model.RPDBWork
	if err := database.DB.First(&work, workID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	if err := database.DB.Model(&work).Updates(map[string]interface{}{"status": model.RPDBStatusRemoved, "is_public": false}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除作品失败"})
		return
	}
	s.bumpRPDBListCache(c.Request.Context())
	logAdminAction(c, "delete_rpdb_work", "rpdb_work", work.ID, work.Title, nil)
	c.Status(http.StatusNoContent)
}

func rpdbModerationPage(c *gin.Context) (int, int) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func parseRPDBModerationID(c *gin.Context, param, subject string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的" + subject + " ID"})
		return 0, false
	}
	return uint(value), true
}

func bindRPDBReviewRequest(c *gin.Context) (ReviewRequest, bool) {
	var request ReviewRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return request, false
	}
	request.Action = strings.TrimSpace(strings.ToLower(request.Action))
	if request.Action != "approve" && request.Action != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "审核动作必须为 approve 或 reject"})
		return request, false
	}
	return request, true
}
