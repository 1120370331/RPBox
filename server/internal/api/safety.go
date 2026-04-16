package api

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/pkg/validator"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	reportTargetPost        = "post"
	reportTargetItem        = "item"
	reportTargetUser        = "user"
	reportTargetComment     = "comment"
	reportTargetItemComment = "item_comment"
)

type createUserBlockRequest struct {
	BlockedUserID uint   `json:"blocked_user_id" binding:"required"`
	Reason        string `json:"reason"`
}

type createContentReportRequest struct {
	TargetType  string `json:"target_type" binding:"required"`
	TargetID    uint   `json:"target_id" binding:"required"`
	Reason      string `json:"reason" binding:"required"`
	Detail      string `json:"detail"`
	HideTarget  bool   `json:"hide_target"`
	BlockAuthor bool   `json:"block_author"`
}

type reviewContentReportRequest struct {
	Action   string `json:"action" binding:"required"` // delete_content|delete_and_mute_user|delete_and_ban_user|mute_user|ban_user|reject
	Duration int    `json:"duration"`
	Comment  string `json:"comment"`
}

type contentReportGroupRow struct {
	ID               uint      `json:"id"`
	TargetType       string    `json:"target_type"`
	TargetID         uint      `json:"target_id"`
	Status           string    `json:"status"`
	ReportCount      int64     `json:"report_count"`
	LatestReportedAt time.Time `json:"latest_reported_at"`
}

var reportHTMLTagPattern = regexp.MustCompile(`<[^>]+>`)

func getBlockedUserIDs(userID uint) ([]uint, error) {
	if userID == 0 {
		return nil, nil
	}

	var blockedIDs []uint
	if err := database.DB.Model(&model.UserBlock{}).
		Where("blocker_id = ?", userID).
		Pluck("blocked_user_id", &blockedIDs).Error; err != nil {
		return nil, err
	}

	return blockedIDs, nil
}

func getHiddenContentIDMap(userID uint, targetTypes ...string) (map[string]map[uint]struct{}, error) {
	if userID == 0 || len(targetTypes) == 0 {
		return map[string]map[uint]struct{}{}, nil
	}

	var rows []model.UserHiddenContent
	if err := database.DB.
		Select("target_type", "target_id").
		Where("user_id = ? AND target_type IN ?", userID, targetTypes).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[string]map[uint]struct{}, len(targetTypes))
	for _, targetType := range targetTypes {
		result[targetType] = make(map[uint]struct{})
	}
	for _, row := range rows {
		if _, ok := result[row.TargetType]; !ok {
			result[row.TargetType] = make(map[uint]struct{})
		}
		result[row.TargetType][row.TargetID] = struct{}{}
	}

	return result, nil
}

func hiddenContentIDs(userID uint, targetType string) ([]uint, error) {
	if userID == 0 || targetType == "" {
		return nil, nil
	}

	hiddenMap, err := getHiddenContentIDMap(userID, targetType)
	if err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(hiddenMap[targetType]))
	for targetID := range hiddenMap[targetType] {
		ids = append(ids, targetID)
	}
	return ids, nil
}

func isUserBlocked(blockerID, blockedUserID uint) bool {
	if blockerID == 0 || blockedUserID == 0 || blockerID == blockedUserID {
		return false
	}

	var count int64
	if err := database.DB.Model(&model.UserBlock{}).
		Where("blocker_id = ? AND blocked_user_id = ?", blockerID, blockedUserID).
		Count(&count).Error; err != nil {
		return false
	}

	return count > 0
}

func isContentHidden(userID uint, targetType string, targetID uint) bool {
	if userID == 0 || targetID == 0 || targetType == "" {
		return false
	}

	var count int64
	if err := database.DB.Model(&model.UserHiddenContent{}).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Count(&count).Error; err != nil {
		return false
	}

	return count > 0
}

func resolveReportTarget(targetType string, targetID uint) (string, uint, error) {
	switch targetType {
	case reportTargetPost:
		var post model.Post
		if err := database.DB.Select("id", "title", "author_id").First(&post, targetID).Error; err != nil {
			return "", 0, err
		}
		return post.Title, post.AuthorID, nil
	case reportTargetItem:
		var item model.Item
		if err := database.DB.Select("id", "name", "author_id").First(&item, targetID).Error; err != nil {
			return "", 0, err
		}
		return item.Name, item.AuthorID, nil
	case reportTargetUser:
		var user model.User
		if err := database.DB.Select("id", "username").First(&user, targetID).Error; err != nil {
			return "", 0, err
		}
		return user.Username, user.ID, nil
	case reportTargetComment:
		var comment model.Comment
		if err := database.DB.Select("id", "post_id", "author_id", "content").First(&comment, targetID).Error; err != nil {
			return "", 0, err
		}
		return buildCommentReportTitle(comment.Content), comment.AuthorID, nil
	case reportTargetItemComment:
		var comment model.ItemComment
		if err := database.DB.Select("id", "item_id", "user_id", "content").First(&comment, targetID).Error; err != nil {
			return "", 0, err
		}
		return buildCommentReportTitle(comment.Content), comment.UserID, nil
	default:
		return "", 0, errors.New("unsupported report target")
	}
}

func buildCommentReportTitle(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "评论"
	}
	runes := []rune(content)
	if len(runes) > 36 {
		return string(runes[:36]) + "..."
	}
	return content
}

func buildReportGroupKey(targetType string, targetID uint, status string) string {
	return fmt.Sprintf("%s:%d:%s", targetType, targetID, status)
}

func normalizeReportPreviewText(content string, limit int) string {
	if limit <= 0 {
		limit = 220
	}
	content = reportHTMLTagPattern.ReplaceAllString(content, " ")
	content = strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	if content == "" {
		return ""
	}
	runes := []rune(content)
	if len(runes) > limit {
		return string(runes[:limit]) + "..."
	}
	return content
}

func buildSafetyActionReason(reason, detail string) string {
	reason = strings.TrimSpace(reason)
	detail = strings.TrimSpace(detail)
	if reason == "" {
		return detail
	}
	if detail == "" {
		return reason
	}

	combined := reason + " | " + detail
	runes := []rune(combined)
	if len(runes) > 500 {
		return string(runes[:500])
	}
	return combined
}

func translateReportReason(reason string) string {
	switch strings.TrimSpace(strings.ToLower(reason)) {
	case "spam":
		return "垃圾信息或刷屏"
	case "abuse":
		return "辱骂、人身攻击"
	case "fraud":
		return "诈骗或恶意引流"
	case "sexual":
		return "色情或不适内容"
	case "illegal":
		return "违法违规内容"
	case "other":
		return "其他问题"
	case "block_user":
		return "用户已执行屏蔽"
	default:
		return strings.TrimSpace(reason)
	}
}

func buildReportModerationReason(reports []model.ContentReport, moderatorComment string) string {
	reasonParts := make([]string, 0, len(reports))
	seen := make(map[string]struct{}, len(reports))
	for _, report := range reports {
		part := translateReportReason(report.Reason)
		if detail := strings.TrimSpace(report.Detail); detail != "" {
			part = buildSafetyActionReason(part, detail)
		}
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		reasonParts = append(reasonParts, part)
	}

	summary := ""
	if len(reasonParts) > 0 {
		summary = "举报处置：" + strings.Join(reasonParts, "；")
	}
	if moderatorComment = strings.TrimSpace(moderatorComment); moderatorComment != "" {
		if summary != "" {
			summary += " | 版主备注：" + moderatorComment
		} else {
			summary = "版主备注：" + moderatorComment
		}
	}
	if summary == "" {
		summary = "举报处置"
	}

	runes := []rune(summary)
	if len(runes) > 500 {
		return string(runes[:500])
	}
	return summary
}

func reportActionRequiresDuration(action string) bool {
	switch action {
	case "delete_and_mute_user", "delete_and_ban_user", "mute_user", "ban_user":
		return true
	default:
		return false
	}
}

func formatReportActionDuration(duration int) string {
	if duration <= 0 {
		return "永久"
	}
	return fmt.Sprintf("%d小时", duration)
}

func buildReportReviewActionLabel(action string, duration int) string {
	switch action {
	case "delete_content":
		return "删除内容"
	case "delete_and_mute_user":
		return "删除并禁言用户（" + formatReportActionDuration(duration) + "）"
	case "delete_and_ban_user":
		return "删除并封禁用户（" + formatReportActionDuration(duration) + "）"
	case "mute_user":
		return "禁言用户（" + formatReportActionDuration(duration) + "）"
	case "ban_user":
		return "封禁用户（" + formatReportActionDuration(duration) + "）"
	case "reject":
		return "驳回举报"
	default:
		return action
	}
}

func validateReportReviewAction(targetType, action string) error {
	switch targetType {
	case reportTargetUser:
		switch action {
		case "mute_user", "ban_user", "reject":
			return nil
		default:
			return errors.New("该举报目标不支持此处理动作")
		}
	case reportTargetPost, reportTargetItem, reportTargetComment, reportTargetItemComment:
		switch action {
		case "delete_content", "delete_and_mute_user", "delete_and_ban_user", "reject":
			return nil
		default:
			return errors.New("该举报目标不支持此处理动作")
		}
	default:
		return errors.New("不支持的举报目标类型")
	}
}

func normalizeReviewComment(actionLabel, comment string, notes []string) string {
	parts := make([]string, 0, 2)
	actionLabel = strings.TrimSpace(actionLabel)
	if actionLabel != "" {
		parts = append(parts, actionLabel)
	}
	if comment = strings.TrimSpace(comment); comment != "" {
		parts = append(parts, "备注："+comment)
	}

	result := strings.Join(parts, " | ")
	if len(notes) > 0 {
		if result != "" {
			result += "（" + strings.Join(notes, "，") + "）"
		} else {
			result = strings.Join(notes, "，")
		}
	}
	runes := []rune(result)
	if len(runes) > 500 {
		return string(runes[:500])
	}
	return result
}

func (s *Server) deleteReportedTarget(c *gin.Context, targetType string, targetID uint) (string, uint, bool, error) {
	switch targetType {
	case reportTargetPost:
		var post model.Post
		if err := database.DB.First(&post, targetID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return buildMissingReportTitle(targetType, targetID), 0, true, nil
			}
			return "", 0, false, err
		}

		targetName := strings.TrimSpace(post.Title)
		if targetName == "" {
			targetName = buildMissingReportTitle(targetType, targetID)
		}

		if err := database.DB.Where(
			"comment_id IN (?)",
			database.DB.Model(&model.Comment{}).Select("id").Where("post_id = ?", targetID),
		).Delete(&model.CommentLike{}).Error; err != nil {
			return "", 0, false, err
		}
		database.DB.Where("post_id = ?", targetID).Delete(&model.PostTag{})
		database.DB.Where("post_id = ?", targetID).Delete(&model.Comment{})
		database.DB.Where("post_id = ?", targetID).Delete(&model.PostLike{})
		database.DB.Where("post_id = ?", targetID).Delete(&model.PostFavorite{})
		s.cleanupPostImages(c, post)
		if err := database.DB.Delete(&post).Error; err != nil {
			return "", 0, false, err
		}
		return targetName, post.AuthorID, false, nil
	case reportTargetItem:
		var item model.Item
		if err := database.DB.First(&item, targetID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return buildMissingReportTitle(targetType, targetID), 0, true, nil
			}
			return "", 0, false, err
		}

		targetName := strings.TrimSpace(item.Name)
		if targetName == "" {
			targetName = buildMissingReportTitle(targetType, targetID)
		}

		var itemImages []model.ItemImage
		_ = database.DB.Where("item_id = ?", targetID).Find(&itemImages).Error
		database.DB.Where("item_id = ?", targetID).Delete(&model.ItemTag{})
		database.DB.Where("item_id = ?", targetID).Delete(&model.ItemComment{})
		database.DB.Where("item_id = ?", targetID).Delete(&model.ItemLike{})
		database.DB.Where("item_id = ?", targetID).Delete(&model.ItemFavorite{})
		database.DB.Where("item_id = ?", targetID).Delete(&model.ItemRating{})
		s.cleanupItemImages(c, item, itemImages)
		database.DB.Where("item_id = ?", targetID).Delete(&model.ItemImage{})
		if err := database.DB.Delete(&item).Error; err != nil {
			return "", 0, false, err
		}
		return targetName, item.AuthorID, false, nil
	case reportTargetComment:
		var comment model.Comment
		if err := database.DB.First(&comment, targetID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return buildMissingReportTitle(targetType, targetID), 0, true, nil
			}
			return "", 0, false, err
		}

		targetName := buildCommentReportTitle(comment.Content)
		if err := database.DB.Where("comment_id = ?", targetID).Delete(&model.CommentLike{}).Error; err != nil {
			return "", 0, false, err
		}
		if err := database.DB.Delete(&comment).Error; err != nil {
			return "", 0, false, err
		}
		_ = database.DB.Model(&model.Post{}).
			Where("id = ? AND comment_count > 0", comment.PostID).
			Update("comment_count", gorm.Expr("comment_count - 1")).Error
		return targetName, comment.AuthorID, false, nil
	case reportTargetItemComment:
		var comment model.ItemComment
		if err := database.DB.First(&comment, targetID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return buildMissingReportTitle(targetType, targetID), 0, true, nil
			}
			return "", 0, false, err
		}

		targetName := buildCommentReportTitle(comment.Content)
		itemID := comment.ItemID
		if err := database.DB.Delete(&comment).Error; err != nil {
			return "", 0, false, err
		}

		var avgRating float64
		var ratingCount int64
		_ = database.DB.Model(&model.ItemComment{}).Where("item_id = ? AND rating > 0", itemID).Count(&ratingCount).Error
		_ = database.DB.Model(&model.ItemComment{}).Where("item_id = ? AND rating > 0", itemID).Select("AVG(rating)").Scan(&avgRating).Error
		_ = database.DB.Model(&model.Item{}).Where("id = ?", itemID).Updates(map[string]interface{}{
			"rating":       avgRating,
			"rating_count": ratingCount,
		}).Error

		return targetName, comment.UserID, false, nil
	default:
		return "", 0, false, errors.New("unsupported report target")
	}
}

func applyReportedUserMute(targetUserID, moderatorID uint, duration int, reason string) (string, bool, error) {
	if targetUserID == 0 {
		return "", true, nil
	}

	var user model.User
	if err := database.DB.First(&user, targetUserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return buildMissingReportTitle(reportTargetUser, targetUserID), true, nil
		}
		return "", false, err
	}
	if user.Role == "admin" || user.Role == "moderator" {
		return user.Username, false, errors.New("不能禁言管理员或版主")
	}

	now := time.Now()
	user.IsMuted = true
	user.MuteReason = reason
	user.BannedBy = &moderatorID
	user.BannedAt = &now
	if duration > 0 {
		until := now.Add(time.Duration(duration) * time.Hour)
		user.MutedUntil = &until
	} else {
		user.MutedUntil = nil
	}

	if err := database.DB.Save(&user).Error; err != nil {
		return "", false, err
	}
	return user.Username, false, nil
}

func applyReportedUserBan(targetUserID, moderatorID uint, duration int, reason string) (string, bool, error) {
	if targetUserID == 0 {
		return "", true, nil
	}

	var user model.User
	if err := database.DB.First(&user, targetUserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return buildMissingReportTitle(reportTargetUser, targetUserID), true, nil
		}
		return "", false, err
	}
	if user.Role == "admin" || user.Role == "moderator" {
		return user.Username, false, errors.New("不能封禁管理员或版主")
	}

	now := time.Now()
	user.IsBanned = true
	user.BanReason = reason
	user.BannedBy = &moderatorID
	user.BannedAt = &now
	if duration > 0 {
		until := now.Add(time.Duration(duration) * time.Hour)
		user.BannedUntil = &until
	} else {
		user.BannedUntil = nil
	}

	if err := database.DB.Save(&user).Error; err != nil {
		return "", false, err
	}
	return user.Username, false, nil
}

func buildMissingReportTitle(targetType string, targetID uint) string {
	switch targetType {
	case reportTargetPost:
		return fmt.Sprintf("帖子 #%d", targetID)
	case reportTargetItem:
		return fmt.Sprintf("作品 #%d", targetID)
	case reportTargetUser:
		return fmt.Sprintf("用户 #%d", targetID)
	case reportTargetComment:
		return fmt.Sprintf("帖子评论 #%d", targetID)
	case reportTargetItemComment:
		return fmt.Sprintf("作品评论 #%d", targetID)
	default:
		return fmt.Sprintf("目标 #%d", targetID)
	}
}

func upsertPendingReport(db *gorm.DB, reporterID uint, req createContentReportRequest, targetUserID uint) (*model.ContentReport, bool, error) {
	var existing model.ContentReport
	err := db.Where(
		"reporter_id = ? AND target_type = ? AND target_id = ? AND status = ?",
		reporterID,
		req.TargetType,
		req.TargetID,
		"pending",
	).First(&existing).Error
	if err == nil {
		existing.Reason = req.Reason
		existing.Detail = req.Detail
		existing.TargetUserID = targetUserID
		if saveErr := db.Save(&existing).Error; saveErr != nil {
			return nil, false, saveErr
		}
		return &existing, true, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	report := model.ContentReport{
		ReporterID:   reporterID,
		TargetType:   req.TargetType,
		TargetID:     req.TargetID,
		TargetUserID: targetUserID,
		Reason:       req.Reason,
		Detail:       req.Detail,
		Status:       "pending",
	}
	if err := db.Create(&report).Error; err != nil {
		return nil, false, err
	}
	return &report, false, nil
}

func upsertUserBlockRecord(db *gorm.DB, blockerID uint, blockedUserID uint, reason string) error {
	block := model.UserBlock{
		BlockerID:     blockerID,
		BlockedUserID: blockedUserID,
		Reason:        reason,
	}

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "blocker_id"}, {Name: "blocked_user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"reason", "updated_at"}),
	}).Create(&block).Error
}

func upsertHiddenContentRecord(db *gorm.DB, userID uint, targetType string, targetID uint, reason string) error {
	if userID == 0 || targetID == 0 {
		return nil
	}
	if targetType != reportTargetPost && targetType != reportTargetItem && targetType != reportTargetComment && targetType != reportTargetItemComment {
		return nil
	}

	record := model.UserHiddenContent{
		UserID:     userID,
		TargetType: targetType,
		TargetID:   targetID,
		Reason:     reason,
	}

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "target_type"}, {Name: "target_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"reason", "updated_at"}),
	}).Create(&record).Error
}

func (s *Server) listUserBlocks(c *gin.Context) {
	userID := c.GetUint("userID")

	var blocks []model.UserBlock
	if err := database.DB.Where("blocker_id = ?", userID).
		Order("created_at DESC").
		Find(&blocks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载屏蔽列表失败"})
		return
	}

	blockedIDs := make([]uint, 0, len(blocks))
	for _, block := range blocks {
		blockedIDs = append(blockedIDs, block.BlockedUserID)
	}

	var users []model.User
	if len(blockedIDs) > 0 {
		if err := database.DB.Select("id", "username", "avatar", "avatar_review_status").
			Where("id IN ?", blockedIDs).
			Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "加载屏蔽用户失败"})
			return
		}
	}

	userMap := make(map[uint]model.User, len(users))
	for _, user := range users {
		userMap[user.ID] = user
	}

	result := make([]gin.H, 0, len(blocks))
	for _, block := range blocks {
		target := userMap[block.BlockedUserID]
		result = append(result, gin.H{
			"id":              block.ID,
			"blocked_user_id": block.BlockedUserID,
			"username":        target.Username,
			"avatar":          userAvatarURL(s.cfg.Server.ApiHost, target),
			"reason":          block.Reason,
			"created_at":      block.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"blocks": result})
}

func (s *Server) createUserBlock(c *gin.Context) {
	userID := c.GetUint("userID")

	var req createUserBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)

	if req.BlockedUserID == 0 || req.BlockedUserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的屏蔽目标"})
		return
	}

	var target model.User
	if err := database.DB.Select("id", "username").First(&target, req.BlockedUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	var existingBlock model.UserBlock
	alreadyBlocked := database.DB.Where("blocker_id = ? AND blocked_user_id = ?", userID, req.BlockedUserID).
		First(&existingBlock).Error == nil

	block := model.UserBlock{
		BlockerID:     userID,
		BlockedUserID: req.BlockedUserID,
		Reason:        req.Reason,
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := upsertUserBlockRecord(tx, block.BlockerID, block.BlockedUserID, block.Reason); err != nil {
			return err
		}

		reportReq := createContentReportRequest{
			TargetType: reportTargetUser,
			TargetID:   req.BlockedUserID,
			Reason:     "block_user",
			Detail:     req.Reason,
		}
		_, _, err := upsertPendingReport(tx, userID, reportReq, req.BlockedUserID)
		return err
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "屏蔽用户失败"})
		return
	}

	message := "已屏蔽该用户"
	if alreadyBlocked {
		message = "该用户已在屏蔽列表中"
	}
	c.JSON(http.StatusOK, gin.H{"message": message})
}

func (s *Server) deleteUserBlock(c *gin.Context) {
	userID := c.GetUint("userID")
	blockedUserID, err := strconv.ParseUint(c.Param("blockedUserId"), 10, 64)
	if err != nil || blockedUserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	if err := database.DB.Where("blocker_id = ? AND blocked_user_id = ?", userID, uint(blockedUserID)).
		Delete(&model.UserBlock{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消屏蔽失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已取消屏蔽"})
}

func (s *Server) createContentReport(c *gin.Context) {
	userID := c.GetUint("userID")

	var req createContentReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	req.TargetType = strings.TrimSpace(strings.ToLower(req.TargetType))
	req.Reason = strings.TrimSpace(req.Reason)
	req.Detail = strings.TrimSpace(req.Detail)
	if req.Reason == "" || req.Detail == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择举报原因并填写备注说明"})
		return
	}

	targetName, targetUserID, err := resolveReportTarget(req.TargetType, req.TargetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "举报目标不存在或不支持"})
		return
	}
	if targetUserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能举报自己"})
		return
	}

	actionReason := buildSafetyActionReason(req.Reason, req.Detail)
	var report *model.ContentReport
	var updated bool
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		var txErr error
		report, updated, txErr = upsertPendingReport(tx, userID, req, targetUserID)
		if txErr != nil {
			return txErr
		}
		if req.HideTarget {
			if err := upsertHiddenContentRecord(tx, userID, req.TargetType, req.TargetID, actionReason); err != nil {
				return err
			}
		}
		if req.BlockAuthor {
			if err := upsertUserBlockRecord(tx, userID, targetUserID, actionReason); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交举报失败"})
		return
	}

	message := "举报已提交"
	if updated {
		message = "已更新你的举报信息"
	}
	c.JSON(http.StatusOK, gin.H{
		"message":      message,
		"report_id":    report.ID,
		"target":       targetName,
		"hide_target":  req.HideTarget,
		"block_author": req.BlockAuthor,
	})
	if req.HideTarget && req.TargetType == reportTargetPost || req.BlockAuthor {
		s.bumpPostListCache(c.Request.Context())
	}
}

func (s *Server) listContentReports(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	statusFilter := strings.TrimSpace(strings.ToLower(c.DefaultQuery("status", "pending")))
	targetScopeFilter := strings.TrimSpace(strings.ToLower(c.Query("target_scope")))
	targetTypeFilter := strings.TrimSpace(strings.ToLower(c.Query("target_type")))
	sortBy := strings.TrimSpace(strings.ToLower(c.DefaultQuery("sort", "report_count")))
	order := strings.TrimSpace(strings.ToLower(c.DefaultQuery("order", "desc")))
	if order != "asc" {
		order = "desc"
	}
	if sortBy != "latest_reported_at" {
		sortBy = "report_count"
	}

	baseQuery := database.DB.Model(&model.ContentReport{}).
		Select("MAX(id) AS id, target_type, target_id, status, COUNT(*) AS report_count, MAX(created_at) AS latest_reported_at").
		Group("target_type, target_id, status")

	if statusFilter != "" && statusFilter != "all" {
		baseQuery = baseQuery.Where("status = ?", statusFilter)
	}
	switch targetScopeFilter {
	case "user":
		baseQuery = baseQuery.Where("target_type = ?", reportTargetUser)
	case "content":
		baseQuery = baseQuery.Where("target_type IN ?", []string{reportTargetPost, reportTargetItem})
	case "comment":
		baseQuery = baseQuery.Where("target_type IN ?", []string{reportTargetComment, reportTargetItemComment})
	}
	if targetTypeFilter != "" {
		baseQuery = baseQuery.Where("target_type = ?", targetTypeFilter)
	}

	var total int64
	if err := database.DB.Table("(?) AS report_groups", baseQuery).
		Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载举报列表失败"})
		return
	}

	orderExpr := sortBy + " " + strings.ToUpper(order)
	var rows []contentReportGroupRow
	if err := baseQuery.Order(orderExpr).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载举报列表失败"})
		return
	}

	reportGroupMap := make(map[string][]model.ContentReport, len(rows))
	postIDs := make([]uint, 0)
	itemIDs := make([]uint, 0)
	commentIDs := make([]uint, 0)
	itemCommentIDs := make([]uint, 0)
	userIDs := make([]uint, 0)
	for _, row := range rows {
		var reports []model.ContentReport
		if err := database.DB.
			Where("target_type = ? AND target_id = ? AND status = ?", row.TargetType, row.TargetID, row.Status).
			Order("created_at DESC").
			Find(&reports).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "加载举报详情失败"})
			return
		}

		reportGroupMap[buildReportGroupKey(row.TargetType, row.TargetID, row.Status)] = reports
		switch row.TargetType {
		case reportTargetPost:
			postIDs = append(postIDs, row.TargetID)
		case reportTargetItem:
			itemIDs = append(itemIDs, row.TargetID)
		case reportTargetUser:
			userIDs = append(userIDs, row.TargetID)
		case reportTargetComment:
			commentIDs = append(commentIDs, row.TargetID)
		case reportTargetItemComment:
			itemCommentIDs = append(itemCommentIDs, row.TargetID)
		}

		for _, report := range reports {
			userIDs = append(userIDs, report.ReporterID)
			if report.TargetUserID != 0 {
				userIDs = append(userIDs, report.TargetUserID)
			}
		}
	}

	var posts []model.Post
	if len(postIDs) > 0 {
		_ = database.DB.Select("id", "title", "author_id", "content", "cover_image").Where("id IN ?", postIDs).Find(&posts).Error
	}
	postMap := make(map[uint]model.Post, len(posts))
	for _, post := range posts {
		postMap[post.ID] = post
	}

	var items []model.Item
	if len(itemIDs) > 0 {
		_ = database.DB.Select("id", "name", "author_id", "description", "detail_content", "preview_image").Where("id IN ?", itemIDs).Find(&items).Error
	}
	itemMap := make(map[uint]model.Item, len(items))
	for _, item := range items {
		itemMap[item.ID] = item
	}

	var comments []model.Comment
	if len(commentIDs) > 0 {
		_ = database.DB.Select("id", "post_id", "author_id", "content", "image_url").Where("id IN ?", commentIDs).Find(&comments).Error
	}
	commentMap := make(map[uint]model.Comment, len(comments))
	for _, comment := range comments {
		commentMap[comment.ID] = comment
		postIDs = append(postIDs, comment.PostID)
		userIDs = append(userIDs, comment.AuthorID)
	}

	var itemComments []model.ItemComment
	if len(itemCommentIDs) > 0 {
		_ = database.DB.Select("id", "item_id", "user_id", "content", "image_url").Where("id IN ?", itemCommentIDs).Find(&itemComments).Error
	}
	itemCommentMap := make(map[uint]model.ItemComment, len(itemComments))
	for _, comment := range itemComments {
		itemCommentMap[comment.ID] = comment
		itemIDs = append(itemIDs, comment.ItemID)
		userIDs = append(userIDs, comment.UserID)
	}

	if len(postIDs) > 0 {
		var extraPosts []model.Post
		_ = database.DB.Select("id", "title", "author_id", "content", "cover_image").Where("id IN ?", postIDs).Find(&extraPosts).Error
		for _, post := range extraPosts {
			postMap[post.ID] = post
		}
	}

	if len(itemIDs) > 0 {
		var extraItems []model.Item
		_ = database.DB.Select("id", "name", "author_id", "description", "detail_content", "preview_image").Where("id IN ?", itemIDs).Find(&extraItems).Error
		for _, item := range extraItems {
			itemMap[item.ID] = item
		}
	}

	var users []model.User
	if len(userIDs) > 0 {
		_ = database.DB.Select("id", "username", "avatar", "avatar_review_status", "role").Where("id IN ?", userIDs).Find(&users).Error
	}
	userMap := make(map[uint]model.User, len(users))
	for _, user := range users {
		userMap[user.ID] = user
	}

	result := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		groupReports := reportGroupMap[buildReportGroupKey(row.TargetType, row.TargetID, row.Status)]
		var latestReport model.ContentReport
		if len(groupReports) > 0 {
			latestReport = groupReports[0]
		}

		targetTitle := buildMissingReportTitle(row.TargetType, row.TargetID)
		targetAuthorName := ""
		targetUserID := latestReport.TargetUserID
		parentTargetID := uint(0)
		parentTargetTitle := ""
		targetPreviewText := ""
		targetPreviewImage := ""

		switch row.TargetType {
		case reportTargetPost:
			post := postMap[row.TargetID]
			if strings.TrimSpace(post.Title) != "" {
				targetTitle = post.Title
			}
			targetPreviewText = normalizeReportPreviewText(post.Content, 220)
			targetPreviewImage = post.CoverImage
			if targetUserID == 0 {
				targetUserID = post.AuthorID
			}
		case reportTargetItem:
			item := itemMap[row.TargetID]
			if strings.TrimSpace(item.Name) != "" {
				targetTitle = item.Name
			}
			targetPreviewText = normalizeReportPreviewText(strings.TrimSpace(item.Description+"\n"+item.DetailContent), 220)
			targetPreviewImage = item.PreviewImage
			if targetUserID == 0 {
				targetUserID = item.AuthorID
			}
		case reportTargetUser:
			targetUser := userMap[row.TargetID]
			if strings.TrimSpace(targetUser.Username) != "" {
				targetTitle = targetUser.Username
			}
			if strings.TrimSpace(targetUser.Role) != "" {
				targetPreviewText = fmt.Sprintf("被举报用户角色：%s", targetUser.Role)
			} else {
				targetPreviewText = "被举报用户资料"
			}
			targetPreviewImage = userAvatarURL(s.cfg.Server.ApiHost, targetUser)
			targetUserID = row.TargetID
		case reportTargetComment:
			comment := commentMap[row.TargetID]
			if comment.ID != 0 {
				targetTitle = buildCommentReportTitle(comment.Content)
			}
			parentTargetID = comment.PostID
			parentTargetTitle = postMap[comment.PostID].Title
			targetPreviewText = normalizeReportPreviewText(comment.Content, 220)
			targetPreviewImage = comment.ImageURL
			if targetUserID == 0 {
				targetUserID = comment.AuthorID
			}
		case reportTargetItemComment:
			comment := itemCommentMap[row.TargetID]
			if comment.ID != 0 {
				targetTitle = buildCommentReportTitle(comment.Content)
			}
			parentTargetID = comment.ItemID
			parentTargetTitle = itemMap[comment.ItemID].Name
			targetPreviewText = normalizeReportPreviewText(comment.Content, 220)
			targetPreviewImage = comment.ImageURL
			if targetUserID == 0 {
				targetUserID = comment.UserID
			}
		}
		if targetUserID != 0 {
			targetAuthorName = userMap[targetUserID].Username
		}

		reasons := make([]gin.H, 0, len(groupReports))
		for _, report := range groupReports {
			reporterName := strings.TrimSpace(userMap[report.ReporterID].Username)
			if reporterName == "" {
				if report.ReporterID > 0 {
					reporterName = fmt.Sprintf("用户#%d", report.ReporterID)
				} else {
					reporterName = "未知用户"
				}
			}
			reasons = append(reasons, gin.H{
				"id":            report.ID,
				"reporter_id":   report.ReporterID,
				"reporter_name": reporterName,
				"reason":        report.Reason,
				"detail":        report.Detail,
				"created_at":    report.CreatedAt,
			})
		}

		result = append(result, gin.H{
			"id":                   row.ID,
			"target_type":          row.TargetType,
			"target_id":            row.TargetID,
			"target_user_id":       targetUserID,
			"target_title":         targetTitle,
			"target_author_name":   targetAuthorName,
			"parent_target_id":     parentTargetID,
			"parent_target_title":  parentTargetTitle,
			"target_preview_text":  targetPreviewText,
			"target_preview_image": targetPreviewImage,
			"status":               row.Status,
			"report_count":         row.ReportCount,
			"latest_reported_at":   row.LatestReportedAt,
			"reasons":              reasons,
			"review_comment":       latestReport.ReviewComment,
			"reviewed_at":          latestReport.ReviewedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"reports": result,
		"total":   total,
	})
}

func (s *Server) reviewContentReport(c *gin.Context) {
	moderatorID := c.GetUint("userID")
	reportID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || reportID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的举报ID"})
		return
	}

	var report model.ContentReport
	if err := database.DB.First(&report, uint(reportID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "举报不存在"})
		return
	}

	var req reviewContentReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}
	req.Action = strings.TrimSpace(strings.ToLower(req.Action))
	req.Comment = strings.TrimSpace(req.Comment)
	if req.Duration < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "处理时长不能为负数"})
		return
	}
	if err := validateReportReviewAction(report.TargetType, req.Action); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var pendingReports []model.ContentReport
	if err := database.DB.
		Where("target_type = ? AND target_id = ? AND status = ?", report.TargetType, report.TargetID, "pending").
		Order("created_at DESC").
		Find(&pendingReports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载举报详情失败"})
		return
	}
	if len(pendingReports) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该目标没有待处理举报"})
		return
	}

	nextStatus := "resolved"
	if req.Action == "reject" {
		nextStatus = "rejected"
	}

	targetName := buildMissingReportTitle(report.TargetType, report.TargetID)
	targetUserID := report.TargetUserID
	if resolvedName, resolvedUserID, err := resolveReportTarget(report.TargetType, report.TargetID); err == nil {
		if strings.TrimSpace(resolvedName) != "" {
			targetName = resolvedName
		}
		if targetUserID == 0 {
			targetUserID = resolvedUserID
		}
	}
	if report.TargetType == reportTargetUser {
		targetUserID = report.TargetID
	}

	actionLabel := buildReportReviewActionLabel(req.Action, req.Duration)
	moderationReason := buildReportModerationReason(pendingReports, req.Comment)
	reviewNotes := make([]string, 0, 2)

	switch req.Action {
	case "delete_content":
		deletedName, deletedUserID, missing, err := s.deleteReportedTarget(c, report.TargetType, report.TargetID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除被举报内容失败"})
			return
		}
		if strings.TrimSpace(deletedName) != "" {
			targetName = deletedName
		}
		if targetUserID == 0 && deletedUserID != 0 {
			targetUserID = deletedUserID
		}
		if missing {
			reviewNotes = append(reviewNotes, "目标内容已不存在")
		}
	case "delete_and_mute_user":
		deletedName, deletedUserID, missing, err := s.deleteReportedTarget(c, report.TargetType, report.TargetID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除被举报内容失败"})
			return
		}
		if strings.TrimSpace(deletedName) != "" {
			targetName = deletedName
		}
		if targetUserID == 0 && deletedUserID != 0 {
			targetUserID = deletedUserID
		}
		if missing {
			reviewNotes = append(reviewNotes, "目标内容已不存在")
		}
		_, missingUser, err := applyReportedUserMute(targetUserID, moderatorID, req.Duration, moderationReason)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if missingUser {
			reviewNotes = append(reviewNotes, "目标用户已不存在")
		}
	case "delete_and_ban_user":
		deletedName, deletedUserID, missing, err := s.deleteReportedTarget(c, report.TargetType, report.TargetID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除被举报内容失败"})
			return
		}
		if strings.TrimSpace(deletedName) != "" {
			targetName = deletedName
		}
		if targetUserID == 0 && deletedUserID != 0 {
			targetUserID = deletedUserID
		}
		if missing {
			reviewNotes = append(reviewNotes, "目标内容已不存在")
		}
		_, missingUser, err := applyReportedUserBan(targetUserID, moderatorID, req.Duration, moderationReason)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if missingUser {
			reviewNotes = append(reviewNotes, "目标用户已不存在")
		}
	case "mute_user":
		userName, missingUser, err := applyReportedUserMute(targetUserID, moderatorID, req.Duration, moderationReason)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.TrimSpace(userName) != "" {
			targetName = userName
		}
		if missingUser {
			reviewNotes = append(reviewNotes, "目标用户已不存在")
		}
	case "ban_user":
		userName, missingUser, err := applyReportedUserBan(targetUserID, moderatorID, req.Duration, moderationReason)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.TrimSpace(userName) != "" {
			targetName = userName
		}
		if missingUser {
			reviewNotes = append(reviewNotes, "目标用户已不存在")
		}
	case "reject":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的处理动作"})
		return
	}

	reviewComment := normalizeReviewComment(actionLabel, req.Comment, reviewNotes)
	now := time.Now()
	result := database.DB.Model(&model.ContentReport{}).
		Where("target_type = ? AND target_id = ? AND status = ?", report.TargetType, report.TargetID, "pending").
		Updates(map[string]interface{}{
			"status":         nextStatus,
			"reviewer_id":    moderatorID,
			"review_comment": reviewComment,
			"reviewed_at":    now,
		})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理举报失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该目标没有待处理举报"})
		return
	}

	logDetails := map[string]interface{}{
		"action":         req.Action,
		"review_comment": reviewComment,
		"report_count":   result.RowsAffected,
	}
	if reportActionRequiresDuration(req.Action) {
		logDetails["duration"] = formatReportActionDuration(req.Duration)
	}
	logAdminAction(c, "review_report", report.TargetType, report.TargetID, targetName, map[string]interface{}{
		"action":         logDetails["action"],
		"review_comment": logDetails["review_comment"],
		"report_count":   logDetails["report_count"],
		"duration":       logDetails["duration"],
	})

	c.JSON(http.StatusOK, gin.H{
		"message":        "举报处理完成",
		"affected_count": result.RowsAffected,
		"status":         nextStatus,
	})
}
