package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
)

// userContribution is a small, public-only card shared by the three content types.
// Cover data and rich-text bodies must never be included in this list response.
type userContribution struct {
	ID             uint       `json:"id"`
	Kind           string     `json:"kind"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	HasCover       bool       `json:"has_cover"`
	CoverURL       string     `json:"cover_url"`
	CoverUpdatedAt *time.Time `json:"cover_updated_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CreatedAt      time.Time  `json:"created_at"`
	LikeCount      int        `json:"like_count"`
	ViewCount      int        `json:"view_count"`
}

type userContributionResponse struct {
	Entries  []userContribution `json:"entries"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

func (s *Server) listUserContributions(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户 ID"})
		return
	}
	kind := c.DefaultQuery("kind", "all")
	if kind != "all" && kind != "post" && kind != "rpdb" && kind != "item" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的内容类型"})
		return
	}
	sort := c.DefaultQuery("sort", "created_at")
	if sort != "created_at" && sort != "popular" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的排序方式"})
		return
	}
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 6)
	if pageSize > 24 {
		pageSize = 24
	}
	db := database.DB.WithContext(c.Request.Context())
	viewerID := c.GetUint("userID")
	blockedIDs, err := getBlockedUserIDs(viewerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载帖子失败"})
		return
	}
	hidden, err := getHiddenContentIDMap(viewerID, reportTargetPost, reportTargetItem, reportTargetRPDBWork)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载帖子失败"})
		return
	}
	publicQuery := func(value interface{}, targetType string) *gorm.DB {
		query := db.Model(value).Where("author_id = ? AND status = ? AND review_status = ? AND is_public = ?", userID, "published", "approved", true)
		if len(blockedIDs) > 0 {
			query = query.Where("author_id NOT IN ?", blockedIDs)
		}
		if len(hidden[targetType]) > 0 {
			ids := make([]uint, 0, len(hidden[targetType]))
			for id := range hidden[targetType] {
				ids = append(ids, id)
			}
			query = query.Where("id NOT IN ?", ids)
		}
		return query
	}
	posts := publicQuery(&model.Post{}, reportTargetPost).Select("id, 'post' AS kind, title, '' AS summary, (COALESCE(cover_image, '') <> '') AS has_cover, '' AS cover_url, cover_image_updated_at AS cover_updated_at, updated_at, created_at, COALESCE(like_count, 0) AS like_count, COALESCE(view_count, 0) AS view_count")
	items := publicQuery(&model.Item{}, reportTargetItem).Select("id, 'item' AS kind, name AS title, description AS summary, (COALESCE(preview_image, '') <> '') AS has_cover, '' AS cover_url, preview_image_updated_at AS cover_updated_at, updated_at, created_at, COALESCE(like_count, 0) AS like_count, COALESCE(downloads, 0) AS view_count")
	// RPDB media is stored as a URL. Legacy inline images are excluded from cards.
	works := publicQuery(&model.RPDBWork{}, reportTargetRPDBWork).Select("id, 'rpdb' AS kind, title, summary, (COALESCE(cover_image, '') <> '') AS has_cover, CASE WHEN cover_image LIKE 'https://%' OR cover_image LIKE 'http://%' OR cover_image LIKE '/uploads/%' OR cover_image LIKE 'uploads/%' THEN cover_image ELSE '' END AS cover_url, cover_image_updated_at AS cover_updated_at, updated_at, created_at, COALESCE(like_count, 0) AS like_count, COALESCE(view_count, 0) AS view_count")
	query := db.Table("(? UNION ALL ? UNION ALL ?) AS contributions", posts, works, items)
	if kind != "all" {
		query = query.Where("kind = ?", kind)
	}
	response := userContributionResponse{Entries: []userContribution{}, Page: page, PageSize: pageSize}
	if err := query.Count(&response.Total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载帖子数量失败"})
		return
	}
	// Stable tie breakers keep equal timestamps/popularity from moving between pages.
	if sort == "popular" {
		query = query.Order("like_count DESC, view_count DESC")
	}
	if err := query.Order("created_at DESC, kind ASC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&response.Entries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载帖子失败"})
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, response)
}
