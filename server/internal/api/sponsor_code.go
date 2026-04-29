package api

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
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
	sponsorCodeAlphabet    = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	sponsorCodeRandomChars = 12
	maxSponsorCodeBatch    = 100
	maxSponsorCodeMonths   = 120
)

type createSponsorRedeemCodesRequest struct {
	Count          int `json:"count" binding:"required"`
	SponsorLevel   int `json:"sponsor_level" binding:"required"`
	DurationMonths int `json:"duration_months"`
	ExpiresMonths  int `json:"expires_months"`
}

func generateSponsorRedeemCode() (string, error) {
	var builder strings.Builder
	builder.Grow(sponsorCodeRandomChars)
	limit := big.NewInt(int64(len(sponsorCodeAlphabet)))

	for i := 0; i < sponsorCodeRandomChars; i++ {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		builder.WriteByte(sponsorCodeAlphabet[n.Int64()])
	}

	raw := builder.String()
	return fmt.Sprintf("RPB-%s-%s-%s", raw[:4], raw[4:8], raw[8:]), nil
}

func generateUniqueSponsorRedeemCode(tx *gorm.DB, seen map[string]struct{}) (string, error) {
	for attempt := 0; attempt < 24; attempt++ {
		code, err := generateSponsorRedeemCode()
		if err != nil {
			return "", err
		}
		if _, ok := seen[code]; ok {
			continue
		}

		var count int64
		if err := tx.Model(&model.SponsorRedeemCode{}).Where("code = ?", code).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			seen[code] = struct{}{}
			return code, nil
		}
	}

	return "", errors.New("failed to generate unique sponsor code")
}

func normalizeSponsorRedeemCode(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	normalized = strings.NewReplacer(
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
		"－", "-",
	).Replace(normalized)

	compact := strings.ReplaceAll(normalized, "-", "")
	if strings.HasPrefix(compact, "RPB") && len(compact) == 15 {
		return fmt.Sprintf("RPB-%s-%s-%s", compact[3:7], compact[7:11], compact[11:15])
	}
	return normalized
}

func validateSponsorCodeMonths(value int, field string, c *gin.Context) bool {
	if value < 0 || value > maxSponsorCodeMonths {
		c.JSON(http.StatusBadRequest, gin.H{"error": field + "必须在0到120个月之间，0表示永久"})
		return false
	}
	return true
}

// createSponsorRedeemCodes 批量生成赞助兑换码。
func (s *Server) createSponsorRedeemCodes(c *gin.Context) {
	var req createSponsorRedeemCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	if req.Count < 1 || req.Count > maxSponsorCodeBatch {
		c.JSON(http.StatusBadRequest, gin.H{"error": "生成数量必须在1到100之间"})
		return
	}
	if req.SponsorLevel < 1 || req.SponsorLevel > sponsorLevelPremium {
		c.JSON(http.StatusBadRequest, gin.H{"error": "赞助类型无效"})
		return
	}
	if !validateSponsorCodeMonths(req.DurationMonths, "赞助持续时间", c) ||
		!validateSponsorCodeMonths(req.ExpiresMonths, "兑换码过期时间", c) {
		return
	}

	now := time.Now()
	var expiresAt *time.Time
	if req.ExpiresMonths > 0 {
		value := now.AddDate(0, req.ExpiresMonths, 0)
		expiresAt = &value
	}

	createdBy := c.GetUint("userID")
	codes := make([]model.SponsorRedeemCode, 0, req.Count)
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		seen := make(map[string]struct{}, req.Count)
		for i := 0; i < req.Count; i++ {
			code, err := generateUniqueSponsorRedeemCode(tx, seen)
			if err != nil {
				return err
			}
			codes = append(codes, model.SponsorRedeemCode{
				Code:           code,
				SponsorLevel:   req.SponsorLevel,
				DurationMonths: req.DurationMonths,
				ExpiresAt:      expiresAt,
				CreatedBy:      createdBy,
			})
		}
		return tx.Create(&codes).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成兑换码失败"})
		return
	}

	logAdminAction(c, "create_sponsor_codes", "sponsor_code", 0, "sponsor_code", map[string]interface{}{
		"count":           req.Count,
		"sponsor_level":   req.SponsorLevel,
		"duration_months": req.DurationMonths,
		"expires_months":  req.ExpiresMonths,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "赞助兑换码已生成",
		"codes":   codes,
	})
}

// listSponsorRedeemCodes 获取最近生成的赞助兑换码。
func (s *Server) listSponsorRedeemCodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.DefaultQuery("status", "all")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	now := time.Now()
	query := database.DB.Model(&model.SponsorRedeemCode{})
	switch status {
	case "active":
		query = query.Where("used_at IS NULL").Where("expires_at IS NULL OR expires_at > ?", now)
	case "used":
		query = query.Where("used_at IS NOT NULL")
	case "expired":
		query = query.Where("used_at IS NULL").Where("expires_at IS NOT NULL AND expires_at <= ?", now)
	case "all", "":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "兑换码状态无效"})
		return
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取兑换码失败"})
		return
	}

	var codes []model.SponsorRedeemCode
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&codes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取兑换码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"codes": codes, "total": total})
}

// redeemSponsorCode 兑换赞助码。
func (s *Server) redeemSponsorCode(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	codeValue := normalizeSponsorRedeemCode(req.Code)
	if codeValue == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入兑换码"})
		return
	}

	userID := c.GetUint("userID")
	now := time.Now()
	var updatedUser model.User
	var redeemedCode model.SponsorRedeemCode

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code = ?", codeValue).
			First(&redeemedCode).Error; err != nil {
			return err
		}
		if redeemedCode.UsedAt != nil {
			return errors.New("兑换码已被使用")
		}
		if redeemedCode.ExpiresAt != nil && !redeemedCode.ExpiresAt.After(now) {
			return errors.New("兑换码已过期")
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&updatedUser, userID).Error; err != nil {
			return err
		}

		activeLevel := resolveSponsorLevel(updatedUser)
		nextLevel := redeemedCode.SponsorLevel
		if activeLevel > nextLevel {
			nextLevel = activeLevel
		}

		var sponsorExpiresAt *time.Time
		if redeemedCode.DurationMonths > 0 {
			if activeLevel > sponsorLevelNone && updatedUser.SponsorExpiresAt == nil {
				sponsorExpiresAt = nil
			} else {
				base := now
				if updatedUser.SponsorExpiresAt != nil && updatedUser.SponsorExpiresAt.After(now) {
					base = *updatedUser.SponsorExpiresAt
				}
				value := base.AddDate(0, redeemedCode.DurationMonths, 0)
				sponsorExpiresAt = &value
			}
		}

		updates := map[string]interface{}{
			"is_sponsor":         true,
			"sponsor_level":      nextLevel,
			"sponsor_expires_at": sponsorExpiresAt,
		}
		if nextLevel < sponsorLevelStyle {
			updates["sponsor_color"] = ""
			updates["sponsor_bold"] = false
		}
		if err := tx.Model(&updatedUser).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Model(&redeemedCode).Updates(map[string]interface{}{
			"used_by": userID,
			"used_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.First(&updatedUser, userID).Error
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "兑换码不存在"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.invalidateUserProfileCache(c.Request.Context(), updatedUser.ID)
	nameColor, nameBold := userDisplayStyle(updatedUser)
	level := resolveSponsorLevel(updatedUser)
	c.JSON(http.StatusOK, gin.H{
		"message": "赞助兑换成功",
		"user": gin.H{
			"id":                 updatedUser.ID,
			"username":           updatedUser.Username,
			"is_sponsor":         level > sponsorLevelNone,
			"sponsor_level":      level,
			"sponsor_expires_at": updatedUser.SponsorExpiresAt,
			"name_color":         nameColor,
			"name_bold":          nameBold,
		},
		"code": redeemedCode,
	})
}
