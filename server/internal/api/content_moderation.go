package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	moderationActionWarning       = "warning"
	moderationActionSevereWarning = "severe_warning"
	sensitiveViolationDecayWindow = 7 * 24 * time.Hour
	legacySensitiveBanReason      = "敏感关键词违规累计达到3次，永久封禁"
)

type sensitiveDecision struct {
	Action         string
	ViolationCount int
	Message        string
}

// enforcePostCommentHardRules applies hard moderation rules for post/comment publishing.
// Returns true when request has been handled and should stop.
func (s *Server) enforcePostCommentHardRules(c *gin.Context, userID uint, contentType string, contentID *uint, texts ...string) bool {
	if _, ok := s.ensureUserCanPublish(c, userID); !ok {
		return true
	}

	hitKeywords := service.DetectSensitiveKeywords(texts...)
	if len(hitKeywords) == 0 {
		return false
	}

	decision, deletedPost, err := s.applySensitiveViolation(userID, contentType, contentID, hitKeywords)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "敏感词审核处理失败"})
		return true
	}

	if deletedPost != nil {
		s.cleanupPostImages(c, *deletedPost)
		s.bumpPostListCache(c.Request.Context())
	}

	_ = service.CreateNotification(&model.Notification{
		UserID:     userID,
		Type:       "system",
		TargetType: "user",
		TargetID:   userID,
		Content:    decision.Message,
	})

	c.JSON(http.StatusForbidden, gin.H{
		"error":           decision.Message,
		"action":          decision.Action,
		"violation_count": decision.ViolationCount,
	})
	return true
}

func (s *Server) ensureUserCanPublish(c *gin.Context, userID uint) (*model.User, bool) {
	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return nil, false
	}

	now := time.Now()
	if applySensitiveViolationDecay(&user, now) {
		updates := map[string]interface{}{
			"sensitive_violation_count": user.SensitiveViolationCount,
		}
		if user.SensitiveLastViolationAt == nil {
			updates["sensitive_last_violation_at"] = nil
		} else {
			updates["sensitive_last_violation_at"] = *user.SensitiveLastViolationAt
		}
		database.DB.Model(&user).Updates(updates)
	}

	if user.IsBanned {
		if clearLegacySensitiveBan(&user) {
			database.DB.Model(&user).Updates(map[string]interface{}{
				"is_banned":    false,
				"banned_until": nil,
				"ban_reason":   "",
				"banned_by":    nil,
				"banned_at":    nil,
			})
		}
	}

	if user.IsBanned {
		if user.BannedUntil != nil && user.BannedUntil.Before(now) {
			user.IsBanned = false
			user.BannedUntil = nil
			user.BanReason = ""
			database.DB.Model(&user).Updates(map[string]interface{}{
				"is_banned":    false,
				"banned_until": nil,
				"ban_reason":   "",
			})
		} else {
			msg := "账号已被封禁"
			if user.BanReason != "" {
				msg += "，原因：" + user.BanReason
			}
			if user.BannedUntil != nil {
				msg += "，解封时间：" + user.BannedUntil.Format("2006-01-02 15:04")
			} else {
				msg += "（永久）"
			}
			c.JSON(http.StatusForbidden, gin.H{"error": msg})
			return nil, false
		}
	}

	if user.IsMuted {
		if user.MutedUntil != nil && user.MutedUntil.Before(now) {
			user.IsMuted = false
			user.MutedUntil = nil
			user.MuteReason = ""
			database.DB.Model(&user).Updates(map[string]interface{}{
				"is_muted":    false,
				"muted_until": nil,
				"mute_reason": "",
			})
		} else {
			msg := "账号已被禁言"
			if user.MuteReason != "" {
				msg += "，原因：" + user.MuteReason
			}
			if user.MutedUntil != nil {
				msg += "，解禁时间：" + user.MutedUntil.Format("2006-01-02 15:04")
			} else {
				msg += "（永久）"
			}
			c.JSON(http.StatusForbidden, gin.H{"error": msg})
			return nil, false
		}
	}

	return &user, true
}

func (s *Server) applySensitiveViolation(userID uint, contentType string, contentID *uint, hitKeywords []string) (sensitiveDecision, *model.Post, error) {
	now := time.Now()
	decision := sensitiveDecision{}
	var deletedPost *model.Post

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return err
		}
		applySensitiveViolationDecay(&user, now)

		// 删除违规内容（存在内容ID时）
		if contentID != nil {
			switch contentType {
			case "post":
				post, err := deletePostForModeration(tx, *contentID)
				if err != nil {
					return err
				}
				deletedPost = post
			case "comment":
				if err := deleteCommentForModeration(tx, *contentID); err != nil {
					return err
				}
			case "item_comment":
				if err := deleteItemCommentForModeration(tx, *contentID); err != nil {
					return err
				}
			}
		}

		user.SensitiveViolationCount++
		user.SensitiveLastViolationAt = &now

		if user.SensitiveViolationCount == 1 {
			decision.Action = moderationActionWarning
			decision.Message = "检测到敏感关键词，已删除内容并执行第1次处罚：警告。"
		} else if user.SensitiveViolationCount == 2 {
			decision.Action = moderationActionSevereWarning
			decision.Message = "检测到敏感关键词，已删除内容并执行第2次处罚：严重警告。"
		} else {
			decision.Action = moderationActionSevereWarning
			decision.Message = fmt.Sprintf("检测到敏感关键词，已删除内容并记录第%d次违规：严重警告。当前不作封号处理，请勿继续违规。", user.SensitiveViolationCount)
		}
		decision.ViolationCount = user.SensitiveViolationCount

		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		violation := model.ContentModerationViolation{
			UserID:         userID,
			ContentType:    contentType,
			ContentID:      contentID,
			HitKeywords:    strings.Join(hitKeywords, ","),
			Action:         decision.Action,
			ViolationCount: user.SensitiveViolationCount,
		}
		if err := tx.Create(&violation).Error; err != nil {
			return err
		}
		return nil
	})

	return decision, deletedPost, err
}

func applySensitiveViolationDecay(user *model.User, now time.Time) bool {
	if user.SensitiveViolationCount <= 0 || user.SensitiveLastViolationAt == nil {
		return false
	}

	elapsed := now.Sub(*user.SensitiveLastViolationAt)
	if elapsed < sensitiveViolationDecayWindow {
		return false
	}

	steps := int(elapsed / sensitiveViolationDecayWindow)
	if steps <= 0 {
		return false
	}

	dec := steps
	if dec > user.SensitiveViolationCount {
		dec = user.SensitiveViolationCount
	}
	user.SensitiveViolationCount -= dec

	if user.SensitiveViolationCount == 0 {
		user.SensitiveLastViolationAt = nil
		return true
	}

	nextTime := user.SensitiveLastViolationAt.Add(time.Duration(steps) * sensitiveViolationDecayWindow)
	user.SensitiveLastViolationAt = &nextTime
	return true
}

func clearLegacySensitiveBan(user *model.User) bool {
	if user == nil || !user.IsBanned {
		return false
	}
	if strings.TrimSpace(user.BanReason) != legacySensitiveBanReason {
		return false
	}
	user.IsBanned = false
	user.BannedUntil = nil
	user.BanReason = ""
	user.BannedBy = nil
	user.BannedAt = nil
	return true
}

func deletePostForModeration(tx *gorm.DB, postID uint) (*model.Post, error) {
	var post model.Post
	if err := tx.First(&post, postID).Error; err != nil {
		return nil, err
	}

	var commentIDs []uint
	if err := tx.Model(&model.Comment{}).Where("post_id = ?", postID).Pluck("id", &commentIDs).Error; err != nil {
		return nil, err
	}
	if len(commentIDs) > 0 {
		if err := tx.Where("comment_id IN ?", commentIDs).Delete(&model.CommentLike{}).Error; err != nil {
			return nil, err
		}
	}

	if err := tx.Where("post_id = ?", postID).Delete(&model.PostTag{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("post_id = ?", postID).Delete(&model.PostEditRequest{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("post_id = ?", postID).Delete(&model.Comment{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("post_id = ?", postID).Delete(&model.PostLike{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("post_id = ?", postID).Delete(&model.PostFavorite{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("post_id = ?", postID).Delete(&model.PostView{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Where("post_id = ?", postID).Delete(&model.CollectionPost{}).Error; err != nil {
		return nil, err
	}
	if err := tx.Delete(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func deleteCommentForModeration(tx *gorm.DB, commentID uint) error {
	var comment model.Comment
	if err := tx.First(&comment, commentID).Error; err != nil {
		return err
	}

	if err := tx.Where("comment_id = ?", commentID).Delete(&model.CommentLike{}).Error; err != nil {
		return err
	}
	if err := tx.Delete(&comment).Error; err != nil {
		return err
	}
	return tx.Model(&model.Post{}).
		Where("id = ?", comment.PostID).
		Update("comment_count", gorm.Expr("CASE WHEN comment_count > 0 THEN comment_count - 1 ELSE 0 END")).Error
}

func deleteItemCommentForModeration(tx *gorm.DB, commentID uint) error {
	var comment model.ItemComment
	if err := tx.First(&comment, commentID).Error; err != nil {
		return err
	}
	if err := tx.Delete(&comment).Error; err != nil {
		return err
	}

	var avgRating float64
	var count int64
	if err := tx.Model(&model.ItemComment{}).Where("item_id = ? AND rating > 0", comment.ItemID).Count(&count).Error; err != nil {
		return err
	}
	if err := tx.Model(&model.ItemComment{}).Where("item_id = ? AND rating > 0", comment.ItemID).Select("AVG(rating)").Scan(&avgRating).Error; err != nil {
		return err
	}

	return tx.Model(&model.Item{}).
		Where("id = ?", comment.ItemID).
		Updates(map[string]interface{}{
			"rating":       avgRating,
			"rating_count": count,
		}).Error
}
