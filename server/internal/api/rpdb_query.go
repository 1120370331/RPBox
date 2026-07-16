package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/cache"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/pkg/auth"
	"gorm.io/gorm"
)

type rpdbWorkCard struct {
	model.RPDBWork
	AuthorName       string `json:"author_name"`
	AuthorAvatar     string `json:"author_avatar"`
	AuthorNameColor  string `json:"author_name_color"`
	AuthorNameBold   bool   `json:"author_name_bold"`
	ItemType         string `json:"item_type,omitempty"`
	IsLiked          bool   `json:"is_liked"`
	IsFavorited      bool   `json:"is_favorited"`
	InCollectionList bool   `json:"in_collection_list"`
}

type rpdbWorkDetail struct {
	rpdbWorkCard
	References    []model.RPDBReference    `json:"references"`
	Media         []model.RPDBMedia        `json:"media"`
	TransmogSlots []model.RPDBTransmogSlot `json:"transmog_slots"`
	GuideSteps    []model.RPDBGuideStep    `json:"guide_steps"`
	Tags          []model.Tag              `json:"tags"`
}

type rpdbWorkListResponse struct {
	Works    []rpdbWorkCard `json:"works"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

func (s *Server) listRPDBWorks(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 12)
	if pageSize > 12 {
		pageSize = 12
	}

	base := database.DB.Model(&model.RPDBWork{}).
		Where("status = ? AND review_status = ? AND is_public = ?",
			model.RPDBStatusPublished,
			model.RPDBReviewApproved,
			true,
		)
	viewerID := optionalRPDBUserID(c)
	var cacheKey string
	if viewerID == 0 && s.cache != nil {
		if version, err := s.cache.Version(c.Request.Context(), rpdbListCacheName); err == nil {
			filterKey := "page=" + strconv.Itoa(page) + "&page_size=" + strconv.Itoa(pageSize) + "&" + c.Request.URL.RawQuery
			cacheKey = cache.VersionedKey(rpdbListCacheName, version, cache.HashKey(filterKey))
			var cached rpdbWorkListResponse
			if err := s.cache.Get(c.Request.Context(), cacheKey, &cached); err == nil {
				c.JSON(http.StatusOK, cached)
				return
			}
		}
	}
	if hiddenIDs, err := hiddenContentIDs(viewerID, reportTargetRPDBWork); err == nil && len(hiddenIDs) > 0 {
		base = base.Where("id NOT IN ?", hiddenIDs)
	}

	base = applyRPDBWorkFilters(base, c)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询作品数量失败"})
		return
	}

	sortOrder := rpdbSortOrder(c.Query("sort"))
	var works []model.RPDBWork
	if err := base.
		Order(sortOrder).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&works).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询作品失败"})
		return
	}

	cards, err := buildRPDBWorkCards(works, viewerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载作品信息失败"})
		return
	}

	response := rpdbWorkListResponse{Works: cards, Total: total, Page: page, PageSize: pageSize}
	if cacheKey != "" {
		_ = s.cache.Set(c.Request.Context(), cacheKey, response, cache.TTL["rpdb:list"])
	}
	c.JSON(http.StatusOK, response)
}

func (s *Server) getRPDBWork(c *gin.Context) {
	workID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || workID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品 ID"})
		return
	}

	viewerID := optionalRPDBUserID(c)
	var work model.RPDBWork
	if err := database.DB.First(&work, uint(workID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询作品失败"})
		return
	}
	if !canViewRPDBWork(work, viewerID) || isContentHidden(viewerID, reportTargetRPDBWork, work.ID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}

	cards, err := buildRPDBWorkCards([]model.RPDBWork{work}, viewerID)
	if err != nil || len(cards) != 1 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载作品信息失败"})
		return
	}

	detail := rpdbWorkDetail{rpdbWorkCard: cards[0]}
	if err := database.DB.Where("work_id = ?", work.ID).Order("sort_order ASC, id ASC").Find(&detail.References).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载关联物品失败"})
		return
	}
	mediaQuery := database.DB.Where("work_id = ?", work.ID)
	if work.AuthorID != viewerID {
		mediaQuery = mediaQuery.Where("review_status = ?", model.RPDBReviewApproved)
	}
	if err := mediaQuery.
		Order("sort_order ASC, id ASC").
		Find(&detail.Media).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载媒体失败"})
		return
	}
	if err := database.DB.Where("work_id = ?", work.ID).Order("sort_order ASC, id ASC").Find(&detail.TransmogSlots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载幻化部件失败"})
		return
	}
	if err := database.DB.Where("work_id = ?", work.ID).Order("sort_order ASC, id ASC").Find(&detail.GuideSteps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载攻略步骤失败"})
		return
	}
	if err := database.DB.
		Table("tags").
		Select("tags.*").
		Joins("JOIN rpdb_tags ON rpdb_tags.tag_id = tags.id").
		Where("rpdb_tags.work_id = ?", work.ID).
		Order("tags.usage_count DESC, tags.id ASC").
		Scan(&detail.Tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载标签失败"})
		return
	}

	if work.Status == model.RPDBStatusPublished && work.ReviewStatus == model.RPDBReviewApproved {
		// 登录用户：同一作品每日最多计 1 次；未登录不计浏览
		if recordRPDBView(work.ID, viewerID) {
			database.DB.Model(&model.RPDBWork{}).Where("id = ?", work.ID).
				UpdateColumn("view_count", gorm.Expr("view_count + 1"))
			detail.ViewCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{"work": detail})
}

// listRPDBHotWorks returns rolling 7-day heat TopN works.
// Only optional type filter is applied; search/tag/etc. are ignored.
func (s *Server) listRPDBHotWorks(c *gin.Context) {
	limit := parsePositiveInt(c.Query("limit"), 3)
	if limit > 6 {
		limit = 6
	}

	viewerID := optionalRPDBUserID(c)
	since := time.Now().Add(-7 * 24 * time.Hour)

	base := database.DB.Model(&model.RPDBWork{}).
		Where("status = ? AND review_status = ? AND is_public = ?",
			model.RPDBStatusPublished,
			model.RPDBReviewApproved,
			true,
		)
	if workType := strings.TrimSpace(c.Query("type")); workType != "" {
		base = base.Where("type = ?", workType)
	}
	if hiddenIDs, err := hiddenContentIDs(viewerID, reportTargetRPDBWork); err == nil && len(hiddenIDs) > 0 {
		base = base.Where("id NOT IN ?", hiddenIDs)
	}

	type rankedWork struct {
		ID          uint
		RecentViews int64
	}
	var ranked []rankedWork
	if err := base.
		Select("rpdb_works.id, COALESCE(COUNT(rpdb_view_events.id), 0) AS recent_views").
		Joins("LEFT JOIN rpdb_view_events ON rpdb_view_events.work_id = rpdb_works.id AND rpdb_view_events.created_at >= ?", since).
		Group("rpdb_works.id").
		Order("recent_views DESC, rpdb_works.view_count DESC, rpdb_works.updated_at DESC").
		Limit(limit).
		Scan(&ranked).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询热度榜失败"})
		return
	}

	if len(ranked) == 0 {
		c.JSON(http.StatusOK, gin.H{"works": []rpdbWorkCard{}, "window_days": 7})
		return
	}

	ids := make([]uint, 0, len(ranked))
	rankMap := make(map[uint]int64, len(ranked))
	for _, item := range ranked {
		ids = append(ids, item.ID)
		rankMap[item.ID] = item.RecentViews
	}

	var works []model.RPDBWork
	if err := database.DB.Where("id IN ?", ids).Find(&works).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载热度作品失败"})
		return
	}
	workByID := make(map[uint]model.RPDBWork, len(works))
	for _, work := range works {
		workByID[work.ID] = work
	}
	ordered := make([]model.RPDBWork, 0, len(ids))
	for _, id := range ids {
		if work, ok := workByID[id]; ok {
			ordered = append(ordered, work)
		}
	}

	cards, err := buildRPDBWorkCards(ordered, viewerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载热度作品失败"})
		return
	}
	// Top3 指标展示近 7 日热度，而不是累计浏览
	for i := range cards {
		cards[i].ViewCount = int(rankMap[cards[i].ID])
	}

	c.JSON(http.StatusOK, gin.H{
		"works":       cards,
		"window_days": 7,
		"limit":       limit,
	})
}

// recordRPDBView records a countable view for logged-in users only.
// Each user contributes at most once per work per calendar day.
// Anonymous visitors do not increase view_count or heat ranking.
func recordRPDBView(workID, viewerID uint) bool {
	if workID == 0 || viewerID == 0 {
		return false
	}

	viewDate := time.Now().Format("2006-01-02")
	var existing model.RPDBViewEvent
	err := database.DB.
		Where("work_id = ? AND user_id = ? AND view_date = ?", workID, viewerID, viewDate).
		First(&existing).Error
	if err == nil {
		return false // 今日已计过
	}
	if err != gorm.ErrRecordNotFound {
		return false
	}

	event := model.RPDBViewEvent{
		WorkID:   workID,
		UserID:   viewerID,
		ViewDate: viewDate,
	}
	if err := database.DB.Create(&event).Error; err != nil {
		// 并发下唯一约束冲突视为今日已计
		return false
	}
	return true
}

// loadRPDBEventViewCounts returns total unique daily view events per work.
// This is the canonical display metric for cards/detail/hot ranking base counts.
func loadRPDBEventViewCounts(workIDs []uint) (map[uint]int64, error) {
	counts := make(map[uint]int64, len(workIDs))
	if len(workIDs) == 0 {
		return counts, nil
	}
	type row struct {
		WorkID uint
		Count  int64
	}
	var rows []row
	if err := database.DB.Model(&model.RPDBViewEvent{}).
		Select("work_id, COUNT(*) AS count").
		Where("work_id IN ?", workIDs).
		Group("work_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		counts[item.WorkID] = item.Count
	}
	return counts, nil
}

func (s *Server) getRPDBWorkPreview(c *gin.Context) {
	workID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || workID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品 ID"})
		return
	}

	viewerID := optionalRPDBUserID(c)
	var work model.RPDBWork
	if err := database.DB.First(&work, uint(workID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询作品失败"})
		return
	}
	if !canViewRPDBWork(work, viewerID) || isContentHidden(viewerID, reportTargetRPDBWork, work.ID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}

	cards, err := buildRPDBWorkCards([]model.RPDBWork{work}, viewerID)
	if err != nil || len(cards) != 1 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载作品预览失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"work": cards[0]})
}

func canViewRPDBWork(work model.RPDBWork, viewerID uint) bool {
	if viewerID != 0 && work.AuthorID == viewerID {
		return work.Status != model.RPDBStatusArchived && work.Status != model.RPDBStatusRemoved
	}
	if work.Status != model.RPDBStatusPublished || work.ReviewStatus != model.RPDBReviewApproved {
		return false
	}
	visibility := normalizeRPDBVisibility(work.Visibility, work.IsPublic)
	if visibility == model.RPDBVisibilityPublic {
		return true
	}
	if visibility != model.RPDBVisibilityGuild || viewerID == 0 {
		return false
	}
	guildIDs := normalizeRPDBGuildIDs(work.GuildIDs, work.GuildID)
	if len(guildIDs) == 0 {
		return false
	}
	var count int64
	if err := database.DB.Model(&model.GuildMember{}).
		Where("guild_id IN ? AND user_id = ?", guildIDs, viewerID).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func applyRPDBWorkFilters(query *gorm.DB, c *gin.Context) *gorm.DB {
	if workType := strings.TrimSpace(c.Query("type")); workType != "" {
		query = query.Where("type = ?", workType)
	}
	if availability := strings.TrimSpace(c.Query("availability_status")); availability != "" {
		query = query.Where("availability_status = ?", availability)
	}
	if verification := strings.TrimSpace(c.Query("verification_status")); verification != "" {
		query = query.Where("verification_status = ?", verification)
	}
	if expansion := strings.TrimSpace(c.Query("expansion")); expansion != "" {
		query = query.Where("expansion = ?", expansion)
	}
	if faction := strings.TrimSpace(c.Query("faction")); faction != "" {
		query = query.Where("faction = ?", faction)
	}
	if armorType := strings.TrimSpace(c.Query("armor_type")); armorType != "" {
		query = query.Where("armor_type = ?", armorType)
	}
	if bindType := strings.TrimSpace(c.Query("bind_type")); bindType != "" {
		if bindType == "yes" {
			query = query.Where(
				"type = ? AND bind_type IN ?",
				model.RPDBWorkTypeItemShowcase,
				[]string{"yes", "account", "pickup", "use"},
			)
		} else {
			query = query.Where("type = ? AND bind_type = ?", model.RPDBWorkTypeItemShowcase, bindType)
		}
	}
	if authorID := strings.TrimSpace(c.Query("author_id")); authorID != "" {
		query = query.Where("author_id = ?", authorID)
	}
	if tagID := strings.TrimSpace(c.Query("tag_id")); tagID != "" {
		query = query.Where("id IN (?)",
			database.DB.Model(&model.RPDBTag{}).Select("work_id").Where("tag_id = ?", tagID),
		)
	}
	if tagSearch := normalizeRPDBTagSearch(c.Query("tag_search")); tagSearch != "" {
		query = query.Where("id IN (?)", rpdbTagNameWorkSubquery(tagSearch))
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		normalizedSearch := normalizeRPDBTagSearch(search)
		term := "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
		query = query.Where(
			"LOWER(title) LIKE ? OR LOWER(summary) LIKE ? OR LOWER(effect_description) LIKE ? OR LOWER(content) LIKE ? OR id IN (?)",
			term,
			term,
			term,
			term,
			rpdbTagNameWorkSubquery(normalizedSearch),
		)
	}
	return query
}

func normalizeRPDBTagSearch(value string) string {
	return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(value), "#"))
}

func rpdbTagNameWorkSubquery(value string) *gorm.DB {
	term := "%" + strings.ToLower(value) + "%"
	return database.DB.
		Table("rpdb_tags").
		Select("rpdb_tags.work_id").
		Joins("JOIN tags ON tags.id = rpdb_tags.tag_id").
		Where("tags.category = ? AND tags.is_public = ? AND LOWER(tags.name) LIKE ?", "rpdb", true, term)
}

func rpdbSortOrder(sort string) string {
	switch sort {
	case "popular":
		return "view_count DESC, updated_at DESC"
	case "favorite":
		return "favorite_count DESC, updated_at DESC"
	case "comments":
		return "comment_count DESC, updated_at DESC"
	case "verified":
		return "last_verified_at DESC, updated_at DESC"
	case "created_at":
		return "created_at DESC"
	default:
		return "updated_at DESC"
	}
}

func buildRPDBWorkCards(works []model.RPDBWork, viewerID uint) ([]rpdbWorkCard, error) {
	cards := make([]rpdbWorkCard, 0, len(works))
	if len(works) == 0 {
		return cards, nil
	}

	authorIDs := make([]uint, 0, len(works))
	workIDs := make([]uint, 0, len(works))
	itemWorkIDs := make([]uint, 0, len(works))
	seenAuthors := map[uint]struct{}{}
	for _, work := range works {
		workIDs = append(workIDs, work.ID)
		if work.Type == model.RPDBWorkTypeItemShowcase {
			itemWorkIDs = append(itemWorkIDs, work.ID)
		}
		if _, exists := seenAuthors[work.AuthorID]; !exists {
			authorIDs = append(authorIDs, work.AuthorID)
			seenAuthors[work.AuthorID] = struct{}{}
		}
	}

	var authors []model.User
	if err := database.DB.
		Select("id", "username", "avatar", "sponsor_color", "sponsor_bold").
		Where("id IN ?", authorIDs).
		Find(&authors).Error; err != nil {
		return nil, err
	}
	authorMap := make(map[uint]model.User, len(authors))
	for _, author := range authors {
		authorMap[author.ID] = author
	}

	itemTypeByWorkID := make(map[uint]string, len(itemWorkIDs))
	if len(itemWorkIDs) > 0 {
		var references []model.RPDBReference
		if err := database.DB.
			Select("id", "work_id", "external_type", "is_primary", "sort_order").
			Where("work_id IN ?", itemWorkIDs).
			Order("work_id ASC, is_primary DESC, sort_order ASC, id ASC").
			Find(&references).Error; err != nil {
			return nil, err
		}
		for _, reference := range references {
			if _, exists := itemTypeByWorkID[reference.WorkID]; !exists {
				itemTypeByWorkID[reference.WorkID] = reference.ExternalType
			}
		}
	}

	liked := map[uint]bool{}
	favorited := map[uint]bool{}
	listed := map[uint]bool{}
	if viewerID != 0 {
		var likes []model.RPDBLike
		if err := database.DB.Where("user_id = ? AND work_id IN ?", viewerID, workIDs).Find(&likes).Error; err != nil {
			return nil, err
		}
		for _, like := range likes {
			liked[like.WorkID] = true
		}

		var favorites []model.RPDBFavorite
		if err := database.DB.Where("user_id = ? AND work_id IN ?", viewerID, workIDs).Find(&favorites).Error; err != nil {
			return nil, err
		}
		for _, favorite := range favorites {
			favorited[favorite.WorkID] = true
		}

		var listEntries []model.RPDBListEntry
		if err := database.DB.
			Table("rpdb_list_entries").
			Select("rpdb_list_entries.*").
			Joins("JOIN rpdb_lists ON rpdb_lists.id = rpdb_list_entries.list_id").
			Where("rpdb_lists.user_id = ? AND rpdb_list_entries.work_id IN ?", viewerID, workIDs).
			Find(&listEntries).Error; err != nil {
			return nil, err
		}
		for _, entry := range listEntries {
			listed[entry.WorkID] = true
		}
	}

	eventViews, err := loadRPDBEventViewCounts(workIDs)
	if err != nil {
		return nil, err
	}

	for _, work := range works {
		author := authorMap[work.AuthorID]
		// 列表/详情统一展示「登录用户每日最多 1 次」口径的累计浏览
		work.ViewCount = int(eventViews[work.ID])
		cards = append(cards, rpdbWorkCard{
			RPDBWork:         work,
			AuthorName:       author.Username,
			AuthorAvatar:     author.Avatar,
			AuthorNameColor:  author.SponsorColor,
			AuthorNameBold:   author.SponsorBold,
			ItemType:         itemTypeByWorkID[work.ID],
			IsLiked:          liked[work.ID],
			IsFavorited:      favorited[work.ID],
			InCollectionList: listed[work.ID],
		})
	}

	return cards, nil
}

func optionalRPDBUserID(c *gin.Context) uint {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return 0
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return 0
	}
	claims, err := auth.ParseToken(parts[1])
	if err != nil {
		return 0
	}
	return claims.UserID
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
