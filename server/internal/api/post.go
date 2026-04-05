package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/cache"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
	"github.com/rpbox/server/pkg/validator"
	"gorm.io/gorm/clause"
)

// CreatePostRequest 创建帖子请求
type CreatePostRequest struct {
	Title          string  `json:"title" binding:"required"`
	Content        string  `json:"content" binding:"required"`
	ContentType    string  `json:"content_type"`
	CoverImage     string  `json:"cover_image"`
	Category       string  `json:"category"` // profile|guild|report|novel|item|event|other
	GuildID        *uint   `json:"guild_id"`
	StoryID        *uint   `json:"story_id"`
	TagIDs         []uint  `json:"tag_ids"`
	Status         string  `json:"status"` // draft|published
	IsPublic       *bool   `json:"is_public"`
	EventType      string  `json:"event_type"`       // server|guild
	EventStartTime *string `json:"event_start_time"` // ISO8601格式
	EventEndTime   *string `json:"event_end_time"`
	EventColor     string  `json:"event_color"` // 活动标记颜色（十六进制）
}

// UpdatePostRequest 更新帖子请求
type UpdatePostRequest struct {
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	ContentType    string  `json:"content_type"`
	CoverImage     string  `json:"cover_image"`
	Category       string  `json:"category"`
	GuildID        *uint   `json:"guild_id"`
	StoryID        *uint   `json:"story_id"`
	Status         string  `json:"status"`
	IsPublic       *bool   `json:"is_public"`
	EventType      string  `json:"event_type"`
	EventStartTime *string `json:"event_start_time"`
	EventEndTime   *string `json:"event_end_time"`
	EventColor     string  `json:"event_color"`
}

type postListParams struct {
	UserID   uint
	Page     int
	PageSize int
	SortBy   string
	Order    string
	Search   string
	GuildID  string
	TagID    string
	AuthorID string
	Status   string
	Category string
	IsPinned *bool
}

type postListResponse struct {
	Posts []postListItem `json:"posts"`
	Total int64          `json:"total"`
}

type postListItem struct {
	model.Post
	AuthorName      string `json:"author_name"`
	AuthorAvatar    string `json:"author_avatar"`
	AuthorRole      string `json:"author_role"`
	AuthorNameColor string `json:"author_name_color"`
	AuthorNameBold  bool   `json:"author_name_bold"`
	CoverImageURL   string `json:"cover_image_url"`
}

type apiError struct {
	status  int
	message string
}

func (e *apiError) Error() string {
	return e.message
}

func parseEventDateTime(raw string) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}

	// 兼容客户端/外部编辑器的常见时间格式
	withZoneLayouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
	}
	withoutZoneLayouts := []string{
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}

	for _, layout := range withZoneLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return &t, nil
		}
	}
	for _, layout := range withoutZoneLayouts {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("invalid event datetime: %s", value)
}

// listPosts 获取帖子列表
func (s *Server) listPosts(c *gin.Context) {
	userID := c.GetUint("userID")

	// 查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	sortBy := c.DefaultQuery("sort", "created_at") // created_at|view_count|like_count
	order := c.DefaultQuery("order", "desc")
	search := strings.TrimSpace(c.Query("search"))
	if search == "" {
		// 兼容历史参数名 keyword
		search = strings.TrimSpace(c.Query("keyword"))
	}
	guildID := c.Query("guild_id")
	tagID := c.Query("tag_id")
	authorID := c.Query("author_id")
	status := c.DefaultQuery("status", "published")
	category := c.Query("category") // 分区筛选
	isPinned := c.Query("is_pinned")

	params := postListParams{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
		SortBy:   sortBy,
		Order:    order,
		Search:   search,
		GuildID:  guildID,
		TagID:    tagID,
		AuthorID: authorID,
		Status:   status,
		Category: category,
	}
	if isPinned != "" {
		pinnedValue, err := strconv.ParseBool(isPinned)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的is_pinned参数"})
			return
		}
		params.IsPinned = &pinnedValue
	}

	isSelfView := authorID != "" && authorID == strconv.Itoa(int(userID))
	if s.cache != nil && params.GuildID == "" && !isSelfView {
		startTime := time.Now()
		pinnedValue := "any"
		if params.IsPinned != nil {
			pinnedValue = strconv.FormatBool(*params.IsPinned)
		}
		filterKey := fmt.Sprintf("page=%d|size=%d|sort=%s|order=%s|search=%s|tag=%s|author=%s|category=%s|pinned=%s|status=%s",
			params.Page, params.PageSize, params.SortBy, params.Order, params.Search, params.TagID, params.AuthorID, params.Category, pinnedValue, params.Status)
		version, err := s.cache.Version(c.Request.Context(), postListCacheName)
		if err != nil {
			log.Printf("[Cache] Version error: %v", err)
		} else {
			cacheKey := cache.VersionedKey(postListCacheName, version, cache.HashKey(filterKey))
			var cached postListResponse
			cacheHit := true
			if err := s.cache.Fetch(c.Request.Context(), cacheKey, cache.TTL["post:list"], &cached, func(ctx context.Context) (interface{}, error) {
				cacheHit = false
				return s.loadPostList(ctx, params)
			}); err != nil {
				log.Printf("[Cache] Fetch error for key %s: %v", cacheKey, err)
			} else {
				log.Printf("[Cache] %s posts=%d time=%v", map[bool]string{true: "HIT", false: "MISS"}[cacheHit], len(cached.Posts), time.Since(startTime))
				c.JSON(http.StatusOK, cached)
				return
			}
		}
	}

	response, err := s.loadPostList(c.Request.Context(), params)
	if err != nil {
		if apiErr, ok := err.(*apiError); ok {
			c.JSON(apiErr.status, gin.H{"error": apiErr.message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) loadPostList(ctx context.Context, params postListParams) (postListResponse, error) {
	db := database.DB.WithContext(ctx)
	query := db.Model(&model.Post{})
	isSelfView := params.AuthorID != "" && params.AuthorID == strconv.Itoa(int(params.UserID))
	guildRole := ""

	// 只显示公开的帖子，除非是查看自己的
	if isSelfView {
		// 查看自己的帖子：可以看到所有状态（包括草稿）
		query = query.Where("author_id = ?", params.AuthorID)
		// 如果指定了状态，则过滤
		if params.Status != "" && params.Status != "all" {
			query = query.Where("status = ?", params.Status)
		}
	} else {
		// 查看他人帖子：只能看到已发布且审核通过的公开帖子
		query = query.Where("status = ?", "published")
		query = query.Where("review_status = ?", "approved")
		if params.AuthorID != "" {
			query = query.Where("author_id = ?", params.AuthorID)
		}
	}

	if params.GuildID != "" {
		// 检查公会内容访问权限
		guildIDUint, _ := strconv.ParseUint(params.GuildID, 10, 32)
		canAccess, role := checkGuildContentAccess(uint(guildIDUint), params.UserID, "post")
		if !canAccess {
			return postListResponse{}, &apiError{status: http.StatusForbidden, message: "无权查看公会内容"}
		}
		guildRole = role
		query = query.Where("guild_id = ?", params.GuildID)
		// 访客查看公会帖子时，仅返回公开帖子
		if !isSelfView && guildRole == "" {
			query = query.Where("is_public = ?", true)
		}
	} else if !isSelfView {
		// 社区广场：只显示公开帖子
		query = query.Where("is_public = ?", true)
	}

	// 分区筛选
	if params.Category != "" {
		query = query.Where("category = ?", params.Category)
	}

	// 关键字搜索
	if params.Search != "" {
		likeKeyword := "%" + params.Search + "%"
		query = query.Where("(title LIKE ? OR content LIKE ?)", likeKeyword, likeKeyword)
	}

	if params.IsPinned != nil {
		query = query.Where("is_pinned = ?", *params.IsPinned)
	}

	if params.TagID != "" {
		// 通过标签筛选
		var postTags []model.PostTag
		if err := db.Where("tag_id = ?", params.TagID).Find(&postTags).Error; err != nil {
			return postListResponse{}, err
		}
		postIDs := make([]uint, len(postTags))
		for i, pt := range postTags {
			postIDs[i] = pt.PostID
		}
		if len(postIDs) > 0 {
			query = query.Where("id IN ?", postIDs)
		} else {
			return postListResponse{Posts: []postListItem{}, Total: 0}, nil
		}
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return postListResponse{}, err
	}

	// 排序和分页
	offset := (params.Page - 1) * params.PageSize
	query = query.Order(params.SortBy + " " + params.Order).Offset(offset).Limit(params.PageSize)

	var posts []model.Post
	// 列表查询排除大字段（content, cover_image）以提高性能
	// cover_image 通过独立的图片 API 访问
	if err := query.Select("id, author_id, title, content_type, category, guild_id, story_id, status, is_public, is_pinned, is_featured, view_count, like_count, comment_count, favorite_count, review_status, event_type, event_start_time, event_end_time, event_color, cover_image_updated_at, created_at, updated_at").Find(&posts).Error; err != nil {
		return postListResponse{}, err
	}

	if len(posts) == 0 {
		return postListResponse{Posts: []postListItem{}, Total: total}, nil
	}

	// 获取有封面图的帖子 ID 列表
	var postIDs []uint
	for _, p := range posts {
		postIDs = append(postIDs, p.ID)
	}
	var postsWithCover []uint
	if len(postIDs) > 0 {
		db.Model(&model.Post{}).
			Select("id").
			Where("id IN ? AND cover_image IS NOT NULL AND cover_image != ''", postIDs).
			Pluck("id", &postsWithCover)
	}
	hasCoverMap := make(map[uint]bool)
	for _, id := range postsWithCover {
		hasCoverMap[id] = true
	}

	// 优先返回已落盘(URL)的封面地址，避免额外图片代理开销
	directCoverURLMap := make(map[uint]string)
	if len(postIDs) > 0 {
		var coverLinks []struct {
			ID         uint
			CoverImage string
		}
		db.Model(&model.Post{}).
			Select("id, cover_image").
			Where("id IN ?", postIDs).
			Where("(cover_image LIKE ? OR cover_image LIKE ? OR cover_image LIKE ? OR cover_image LIKE ?)",
				"http://%", "https://%", "/uploads/%", "uploads/%").
			Find(&coverLinks)
		for _, row := range coverLinks {
			url := strings.TrimSpace(row.CoverImage)
			if url != "" {
				directCoverURLMap[row.ID] = buildAPIURL(s.cfg.Server.ApiHost, url)
			}
		}
	}

	// 获取作者信息
	authorIDs := make([]uint, len(posts))
	for i, p := range posts {
		authorIDs[i] = p.AuthorID
	}

	var users []model.User
	if len(authorIDs) > 0 {
		db.Where("id IN ?", authorIDs).Find(&users)
	}
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 组装响应
	result := make([]postListItem, len(posts))
	for i, p := range posts {
		author := userMap[p.AuthorID]
		nameColor, nameBold := userDisplayStyle(author)
		coverURL := ""
		if hasCoverMap[p.ID] {
			if directURL := directCoverURLMap[p.ID]; directURL != "" {
				coverURL = directURL
			} else {
				ensurePostCoverUpdatedAt(&p)
				coverURL = postCoverURLFromMeta(p.ID, p.UpdatedAt, p.CoverImageUpdatedAt)
			}
		}
		result[i] = postListItem{
			Post:            p,
			AuthorName:      author.Username,
			AuthorAvatar:    userAvatarURL(s.cfg.Server.ApiHost, author),
			AuthorRole:      author.Role,
			AuthorNameColor: nameColor,
			AuthorNameBold:  nameBold,
			CoverImageURL:   coverURL,
		}
	}

	return postListResponse{Posts: result, Total: total}, nil
}

// createPost 创建帖子
func (s *Server) createPost(c *gin.Context) {
	userID := c.GetUint("userID")

	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)

	if s.enforcePostCommentHardRules(c, userID, "post", nil, req.Title, req.Content) {
		return
	}

	// 默认值
	if req.ContentType == "" {
		req.ContentType = "markdown"
	}
	if req.Status == "" {
		req.Status = "draft"
	}
	if req.Category == "" {
		req.Category = "other"
	}
	if req.CoverImage != "" {
		normalizedCover, err := s.normalizeAndStoreImageValue(c, req.CoverImage, fmt.Sprintf("posts/%d/cover", userID))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "封面图格式无效"})
			return
		}
		req.CoverImage = normalizedCover
	}
	if req.Content != "" {
		req.Content = s.normalizeAndStoreContentImages(c, req.Content, fmt.Sprintf("posts/%d/content", userID))
	}

	// 活动分区基础校验
	if req.Category == "event" {
		if req.EventType != "" && req.EventType != "server" && req.EventType != "guild" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "活动类型无效"})
			return
		}
		if req.EventType == "guild" && req.GuildID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "公会活动需要选择公会"})
			return
		}
	} else {
		req.EventType = ""
		req.EventStartTime = nil
		req.EventEndTime = nil
		req.EventColor = ""
	}

	post := model.Post{
		AuthorID:    userID,
		Title:       req.Title,
		Content:     req.Content,
		ContentType: req.ContentType,
		CoverImage:  req.CoverImage,
		Category:    req.Category,
		GuildID:     req.GuildID,
		StoryID:     req.StoryID,
		Status:      req.Status,
		IsPublic:    true,
		EventType:   req.EventType,
		EventColor:  req.EventColor,
	}
	if req.GuildID != nil && req.IsPublic != nil {
		post.IsPublic = *req.IsPublic
	}
	if req.CoverImage != "" {
		now := time.Now()
		post.CoverImageUpdatedAt = &now
	}

	// 设置审核状态：版主/管理员自动通过，普通用户需要审核
	// 草稿状态不需要审核
	isModerator := checkModerator(userID)
	if req.Status == "published" {
		if isModerator {
			post.Status = "published"
			post.ReviewStatus = "approved" // 版主/管理员自动通过
		} else {
			post.Status = "pending"       // 改为待发布状态
			post.ReviewStatus = "pending" // 待审核
		}
	} else {
		// 草稿状态，不需要审核
		post.ReviewStatus = ""
	}

	// 解析活动时间
	if req.EventStartTime != nil {
		parsed, err := parseEventDateTime(*req.EventStartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "活动开始时间格式无效"})
			return
		}
		post.EventStartTime = parsed
	}
	if req.EventEndTime != nil {
		parsed, err := parseEventDateTime(*req.EventEndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "活动结束时间格式无效"})
			return
		}
		post.EventEndTime = parsed
	}

	if err := database.DB.Create(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	ensurePostCoverUpdatedAt(&post)
	post.CoverImage = postCoverURL(post)

	// 添加标签
	if len(req.TagIDs) > 0 {
		for _, tagID := range req.TagIDs {
			postTag := model.PostTag{
				PostID:  post.ID,
				TagID:   tagID,
				AddedBy: userID,
			}
			database.DB.Create(&postTag)
			// 更新标签使用次数
			database.DB.Model(&model.Tag{}).Where("id = ?", tagID).Update("usage_count", database.DB.Raw("usage_count + 1"))
		}
	}

	// @提及通知（非草稿）
	if post.Status != "draft" {
		mentionMessage := "在帖子《" + post.Title + "》中提到了你"
		service.CreateMentionNotifications(userID, "post", post.ID, mentionMessage, post.Content)
	}
	s.bumpPostListCache(c.Request.Context())

	ensurePostCoverUpdatedAt(&post)
	post.CoverImage = postCoverURL(post)
	c.JSON(http.StatusCreated, post)
}

// getPost 获取帖子详情
func (s *Server) getPost(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	isModerator := checkModerator(userID)

	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}
	if normalizedContent := s.normalizeAndStoreContentImages(c, post.Content, fmt.Sprintf("posts/%d/content", post.AuthorID)); normalizedContent != post.Content {
		post.Content = normalizedContent
		_ = database.DB.Model(&model.Post{}).Where("id = ?", post.ID).Update("content", normalizedContent).Error
	}

	// 权限检查：非公开帖子仅公会成员可见
	if post.AuthorID != userID && !isModerator {
		if !post.IsPublic {
			if post.GuildID != nil {
				canAccess, role := checkGuildContentAccess(*post.GuildID, userID, "post")
				if !canAccess || role == "" {
					c.JSON(http.StatusForbidden, gin.H{"error": "无权查看"})
					return
				}
			} else {
				c.JSON(http.StatusForbidden, gin.H{"error": "无权查看"})
				return
			}
		}
	}

	// 增加浏览次数
	database.DB.Model(&post).Update("view_count", post.ViewCount+1)

	// 记录浏览历史
	if userID != 0 {
		view := model.PostView{
			PostID: post.ID,
			UserID: userID,
		}
		database.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "post_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
		}).Create(&view)
	}

	// 获取作者信息
	var author model.User
	database.DB.First(&author, post.AuthorID)
	nameColor, nameBold := userDisplayStyle(author)

	// 获取标签
	var postTags []model.PostTag
	database.DB.Where("post_id = ?", id).Find(&postTags)
	tagIDs := make([]uint, len(postTags))
	for i, pt := range postTags {
		tagIDs[i] = pt.TagID
	}
	var tags []model.Tag
	if len(tagIDs) > 0 {
		database.DB.Where("id IN ?", tagIDs).Find(&tags)
	}

	// 检查当前用户是否点赞和收藏
	var liked, favorited bool
	var postLike model.PostLike
	if err := database.DB.Where("post_id = ? AND user_id = ?", id, userID).First(&postLike).Error; err == nil {
		liked = true
	}
	var postFav model.PostFavorite
	if err := database.DB.Where("post_id = ? AND user_id = ?", id, userID).First(&postFav).Error; err == nil {
		favorited = true
	}

	ensurePostCoverUpdatedAt(&post)
	post.CoverImage = postCoverURL(post)

	c.JSON(http.StatusOK, gin.H{
		"post":              post,
		"author_name":       author.Username,
		"author_avatar":     userAvatarURL(s.cfg.Server.ApiHost, author),
		"author_name_color": nameColor,
		"author_name_bold":  nameBold,
		"tags":              tags,
		"liked":             liked,
		"favorited":         favorited,
	})
}

// updatePost 更新帖子
func (s *Server) updatePost(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	isModerator := checkModerator(userID)

	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	// 只有作者或管理员/版主可以更新
	if post.AuthorID != userID && !isModerator {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)

	nextTitle := post.Title
	if req.Title != "" {
		nextTitle = req.Title
	}
	nextContent := post.Content
	if req.Content != "" {
		nextContent = req.Content
	}
	if s.enforcePostCommentHardRules(c, userID, "post", &post.ID, nextTitle, nextContent) {
		return
	}

	if req.CoverImage != "" {
		normalizedCover, err := s.normalizeAndStoreImageValue(c, req.CoverImage, fmt.Sprintf("posts/%d/cover", userID))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "封面图格式无效"})
			return
		}
		req.CoverImage = normalizedCover
	}
	if req.Content != "" {
		req.Content = s.normalizeAndStoreContentImages(c, req.Content, fmt.Sprintf("posts/%d/content", userID))
	}

	effectiveCategory := post.Category
	if req.Category != "" {
		effectiveCategory = req.Category
	}
	effectiveEventType := post.EventType
	if req.EventType != "" {
		effectiveEventType = req.EventType
	}
	effectiveGuildID := post.GuildID
	if req.GuildID != nil {
		effectiveGuildID = req.GuildID
	}

	if effectiveCategory == "event" {
		if effectiveEventType != "" && effectiveEventType != "server" && effectiveEventType != "guild" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "活动类型无效"})
			return
		}
		if effectiveEventType == "guild" && effectiveGuildID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "公会活动需要选择公会"})
			return
		}
	}

	eventStartProvided := req.EventStartTime != nil
	var parsedEventStart *time.Time
	if eventStartProvided {
		parsed, err := parseEventDateTime(*req.EventStartTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "活动开始时间格式无效"})
			return
		}
		parsedEventStart = parsed
	}

	eventEndProvided := req.EventEndTime != nil
	var parsedEventEnd *time.Time
	if eventEndProvided {
		parsed, err := parseEventDateTime(*req.EventEndTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "活动结束时间格式无效"})
			return
		}
		parsedEventEnd = parsed
	}

	// 已发布帖子的编辑：普通用户需要审核，版主直接生效
	if post.Status == "published" && post.ReviewStatus == "approved" && !isModerator {
		// 创建或更新编辑请求
		var editReq model.PostEditRequest
		database.DB.Where("post_id = ?", post.ID).First(&editReq)

		editReq.PostID = post.ID
		editReq.AuthorID = userID
		editReq.Status = "pending"
		if req.Title != "" {
			editReq.Title = req.Title
		} else {
			editReq.Title = post.Title
		}
		if req.Content != "" {
			editReq.Content = req.Content
		} else {
			editReq.Content = post.Content
		}
		if req.ContentType != "" {
			editReq.ContentType = req.ContentType
		} else {
			editReq.ContentType = post.ContentType
		}
		if req.Category != "" {
			editReq.Category = req.Category
		} else {
			editReq.Category = post.Category
		}
		if editReq.Category == "event" {
			if req.EventType != "" {
				editReq.EventType = req.EventType
			} else {
				editReq.EventType = post.EventType
			}
			if eventStartProvided {
				editReq.EventStartTime = parsedEventStart
			} else {
				editReq.EventStartTime = post.EventStartTime
			}
			if eventEndProvided {
				editReq.EventEndTime = parsedEventEnd
			} else {
				editReq.EventEndTime = post.EventEndTime
			}
			if req.EventColor != "" {
				editReq.EventColor = req.EventColor
			} else {
				editReq.EventColor = post.EventColor
			}
		} else {
			editReq.EventType = ""
			editReq.EventStartTime = nil
			editReq.EventEndTime = nil
			editReq.EventColor = ""
		}

		if req.IsPublic != nil {
			newPublic := true
			if post.GuildID != nil {
				newPublic = *req.IsPublic
			}
			if post.IsPublic != newPublic {
				database.DB.Model(&post).Update("is_public", newPublic)
				post.IsPublic = newPublic
			}
		}

		database.DB.Save(&editReq)
		s.bumpPostListCache(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "编辑请求已提交，等待审核",
			"data":    editReq,
		})
		return
	}

	// 版主或草稿状态：直接修改
	if req.Title != "" {
		post.Title = req.Title
	}
	if req.Content != "" {
		post.Content = req.Content
	}
	if req.ContentType != "" {
		post.ContentType = req.ContentType
	}
	if req.CoverImage != "" {
		post.CoverImage = req.CoverImage
		now := time.Now()
		post.CoverImageUpdatedAt = &now
	}
	if req.Category != "" {
		post.Category = req.Category
	}
	if req.Status != "" {
		post.Status = req.Status
	}
	post.GuildID = req.GuildID
	post.StoryID = req.StoryID
	if post.GuildID == nil {
		post.IsPublic = true
	} else if req.IsPublic != nil {
		post.IsPublic = *req.IsPublic
	}

	// 处理发布时的审核逻辑
	if req.Status == "published" && post.ReviewStatus != "approved" {
		if isModerator {
			post.Status = "published"
			post.ReviewStatus = "approved"
		} else {
			post.Status = "pending"
			post.ReviewStatus = "pending"
		}
	} else if req.Status == "draft" {
		post.ReviewStatus = ""
	}

	if post.Category == "event" {
		if req.EventType != "" {
			post.EventType = req.EventType
		}
		if eventStartProvided {
			post.EventStartTime = parsedEventStart
		}
		if eventEndProvided {
			post.EventEndTime = parsedEventEnd
		}
		if req.EventColor != "" {
			post.EventColor = req.EventColor
		}
	} else {
		post.EventType = ""
		post.EventStartTime = nil
		post.EventEndTime = nil
		post.EventColor = ""
	}

	database.DB.Save(&post)

	if post.Status != "draft" {
		mentionMessage := "在帖子《" + post.Title + "》中提到了你"
		service.CreateMentionNotifications(userID, "post", post.ID, mentionMessage, post.Content)
	}
	s.bumpPostListCache(c.Request.Context())
	ensurePostCoverUpdatedAt(&post)
	post.CoverImage = postCoverURL(post)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": post})
}

// deletePost 删除帖子
func (s *Server) deletePost(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	isModerator := checkModerator(userID)

	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	// 只有作者或管理员/版主可以删除
	if post.AuthorID != userID && !isModerator {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	// 删除关联数据
	database.DB.Where("post_id = ?", id).Delete(&model.PostTag{})
	database.DB.Where("post_id = ?", id).Delete(&model.Comment{})
	database.DB.Where("post_id = ?", id).Delete(&model.PostLike{})
	database.DB.Where("post_id = ?", id).Delete(&model.PostFavorite{})

	s.cleanupPostImages(c, post)
	database.DB.Delete(&post)
	s.bumpPostListCache(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// likePost 点赞帖子
func (s *Server) likePost(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查帖子是否存在
	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	// 检查是否已点赞
	var existing model.PostLike
	if err := database.DB.Where("post_id = ? AND user_id = ?", id, userID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已点赞"})
		return
	}

	// 创建点赞记录
	postLike := model.PostLike{
		PostID: uint(id),
		UserID: userID,
	}
	database.DB.Create(&postLike)

	// 更新点赞数
	database.DB.Model(&post).Update("like_count", post.LikeCount+1)

	// 创建通知（不给自己发通知）
	if post.AuthorID != userID {
		content := "点赞了你的帖子《" + post.Title + "》"

		notification := model.Notification{
			UserID:     post.AuthorID,
			Type:       "post_like",
			ActorID:    &userID,
			TargetType: "post",
			TargetID:   uint(id),
			Content:    content,
		}
		service.CreateNotification(&notification)
	}

	c.JSON(http.StatusOK, gin.H{"message": "点赞成功"})
}

// unlikePost 取消点赞
func (s *Server) unlikePost(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	result := database.DB.Where("post_id = ? AND user_id = ?", id, userID).Delete(&model.PostLike{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未点赞"})
		return
	}

	// 更新点赞数
	database.DB.Model(&model.Post{}).Where("id = ?", id).Update("like_count", database.DB.Raw("like_count - 1"))

	c.JSON(http.StatusOK, gin.H{"message": "取消点赞成功"})
}

// favoritePost 收藏帖子
func (s *Server) favoritePost(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查帖子是否存在
	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	// 检查是否已收藏
	var existing model.PostFavorite
	if err := database.DB.Where("post_id = ? AND user_id = ?", id, userID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已收藏"})
		return
	}

	// 创建收藏记录
	postFav := model.PostFavorite{
		PostID: uint(id),
		UserID: userID,
	}
	database.DB.Create(&postFav)

	// 更新收藏数
	database.DB.Model(&post).Update("favorite_count", post.FavoriteCount+1)

	c.JSON(http.StatusOK, gin.H{"message": "收藏成功"})
}

// unfavoritePost 取消收藏
func (s *Server) unfavoritePost(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	result := database.DB.Where("post_id = ? AND user_id = ?", id, userID).Delete(&model.PostFavorite{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未收藏"})
		return
	}

	// 更新收藏数
	database.DB.Model(&model.Post{}).Where("id = ?", id).Update("favorite_count", database.DB.Raw("favorite_count - 1"))

	c.JSON(http.StatusOK, gin.H{"message": "取消收藏成功"})
}

// ========== 帖子标签管理 ==========

// getPostTags 获取帖子的标签列表
func (s *Server) getPostTags(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var postTags []model.PostTag
	database.DB.Where("post_id = ?", id).Find(&postTags)

	tagIDs := make([]uint, len(postTags))
	for i, pt := range postTags {
		tagIDs[i] = pt.TagID
	}

	var tags []model.Tag
	if len(tagIDs) > 0 {
		database.DB.Where("id IN ?", tagIDs).Find(&tags)
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// AddPostTagRequest 添加帖子标签请求
type AddPostTagRequest struct {
	TagID uint `json:"tag_id" binding:"required"`
}

// addPostTag 为帖子添加标签
func (s *Server) addPostTag(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	isModerator := checkModerator(userID)

	// 检查帖子所有权
	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}
	if post.AuthorID != userID && !isModerator {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	var req AddPostTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	// 检查标签是否存在
	var tag model.Tag
	if err := database.DB.First(&tag, req.TagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "标签不存在"})
		return
	}

	// 创建关联
	postTag := model.PostTag{
		PostID:  uint(id),
		TagID:   req.TagID,
		AddedBy: userID,
	}

	if err := database.DB.Create(&postTag).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标签已存在"})
		return
	}

	// 更新标签使用次数
	database.DB.Model(&tag).Update("usage_count", tag.UsageCount+1)

	c.JSON(http.StatusOK, gin.H{"message": "添加成功"})
}

// removePostTag 移除帖子标签
func (s *Server) removePostTag(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	tagID, _ := strconv.ParseUint(c.Param("tagId"), 10, 32)
	isModerator := checkModerator(userID)

	// 检查帖子所有权
	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}
	if post.AuthorID != userID && !isModerator {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	result := database.DB.Where("post_id = ? AND tag_id = ?", id, tagID).Delete(&model.PostTag{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "标签关联不存在"})
		return
	}

	// 更新标签使用次数
	database.DB.Model(&model.Tag{}).Where("id = ?", tagID).Update("usage_count", database.DB.Raw("usage_count - 1"))

	c.JSON(http.StatusOK, gin.H{"message": "移除成功"})
}

// listMyFavorites 获取我的收藏列表
func (s *Server) listMyFavorites(c *gin.Context) {
	s.listUserPostsByRelation(c, "post_favorites", "created_at")
}

// listMyPostLikes 获取我点赞的帖子
func (s *Server) listMyPostLikes(c *gin.Context) {
	s.listUserPostsByRelation(c, "post_likes", "created_at")
}

// listMyPostViews 获取我浏览的帖子
func (s *Server) listMyPostViews(c *gin.Context) {
	s.listUserPostsByRelation(c, "post_views", "updated_at")
}

func canAccessPost(userID uint, post model.Post) bool {
	if post.AuthorID == userID {
		return true
	}
	if post.Status != "published" || post.ReviewStatus != "approved" {
		return false
	}
	if post.IsPublic {
		return true
	}
	if post.GuildID != nil {
		canAccess, role := checkGuildContentAccess(*post.GuildID, userID, "post")
		return canAccess && role != ""
	}
	return false
}

func (s *Server) listUserPostsByRelation(c *gin.Context, joinTable, orderColumn string) {
	userID := c.GetUint("userID")

	var posts []model.Post
	query := database.DB.Model(&model.Post{}).
		Joins("JOIN "+joinTable+" ON "+joinTable+".post_id = posts.id").
		Where(joinTable+".user_id = ?", userID).
		Order(joinTable + "." + orderColumn + " DESC")

	if err := query.Select("posts.id, posts.author_id, posts.title, posts.content_type, posts.category, posts.guild_id, posts.story_id, posts.status, posts.is_public, posts.is_pinned, posts.is_featured, posts.view_count, posts.like_count, posts.comment_count, posts.favorite_count, posts.review_status, posts.event_type, posts.event_start_time, posts.event_end_time, posts.event_color, posts.cover_image_updated_at, posts.created_at, posts.updated_at").Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	filtered := make([]model.Post, 0, len(posts))
	for _, post := range posts {
		if canAccessPost(userID, post) {
			filtered = append(filtered, post)
		}
	}

	// 获取有封面图的帖子 ID 列表
	var postIDs []uint
	for _, p := range filtered {
		postIDs = append(postIDs, p.ID)
	}
	var postsWithCover []uint
	if len(postIDs) > 0 {
		database.DB.Model(&model.Post{}).
			Select("id").
			Where("id IN ? AND cover_image IS NOT NULL AND cover_image != ''", postIDs).
			Pluck("id", &postsWithCover)
	}
	hasCoverMap := make(map[uint]bool)
	for _, id := range postsWithCover {
		hasCoverMap[id] = true
	}

	// 优先返回已落盘(URL)的封面地址，避免额外图片代理开销
	directCoverURLMap := make(map[uint]string)
	if len(postIDs) > 0 {
		var coverLinks []struct {
			ID         uint
			CoverImage string
		}
		database.DB.Model(&model.Post{}).
			Select("id, cover_image").
			Where("id IN ?", postIDs).
			Where("(cover_image LIKE ? OR cover_image LIKE ? OR cover_image LIKE ? OR cover_image LIKE ?)",
				"http://%", "https://%", "/uploads/%", "uploads/%").
			Find(&coverLinks)
		for _, row := range coverLinks {
			url := strings.TrimSpace(row.CoverImage)
			if url != "" {
				directCoverURLMap[row.ID] = buildAPIURL(s.cfg.Server.ApiHost, url)
			}
		}
	}

	// 获取作者信息
	authorIDs := make([]uint, len(filtered))
	for i, p := range filtered {
		authorIDs[i] = p.AuthorID
	}

	var users []model.User
	if len(authorIDs) > 0 {
		database.DB.Where("id IN ?", authorIDs).Find(&users)
	}
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	result := make([]postListItem, 0, len(filtered))
	for _, p := range filtered {
		author := userMap[p.AuthorID]
		nameColor, nameBold := userDisplayStyle(author)
		coverURL := ""
		if hasCoverMap[p.ID] {
			if directURL := directCoverURLMap[p.ID]; directURL != "" {
				coverURL = directURL
			} else {
				ensurePostCoverUpdatedAt(&p)
				coverURL = postCoverURLFromMeta(p.ID, p.UpdatedAt, p.CoverImageUpdatedAt)
			}
		}
		result = append(result, postListItem{
			Post:            p,
			AuthorName:      author.Username,
			AuthorAvatar:    userAvatarURL(s.cfg.Server.ApiHost, author),
			AuthorRole:      author.Role,
			AuthorNameColor: nameColor,
			AuthorNameBold:  nameBold,
			CoverImageURL:   coverURL,
		})
	}

	c.JSON(http.StatusOK, gin.H{"posts": result, "total": len(result)})
}

// checkModerator 检查用户是否为社区版主
func checkModerator(userID uint) bool {
	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return false
	}
	// 版主或管理员都有权限
	return user.Role == "moderator" || user.Role == "admin"
}

// listEvents 获取活动列表（用于日历）
func (s *Server) listEvents(c *gin.Context) {
	userID := c.GetUint("userID")
	startDate := c.Query("start") // YYYY-MM-DD
	endDate := c.Query("end")     // YYYY-MM-DD

	query := database.DB.Model(&model.Post{}).
		Where("category = ? AND status = ?", "event", "published").
		Where("review_status = ?", "approved").
		Where("is_public = ?", true).
		Where("event_start_time IS NOT NULL")

	// 时间范围过滤
	if startDate != "" {
		query = query.Where("event_start_time >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("event_start_time <= ?", endDate+" 23:59:59")
	}

	// 获取用户所在公会
	var memberGuildIDs []uint
	var members []model.GuildMember
	database.DB.Where("user_id = ?", userID).Find(&members)
	for _, m := range members {
		memberGuildIDs = append(memberGuildIDs, m.GuildID)
	}

	// 只显示：服务器活动 或 用户所在公会的活动
	if len(memberGuildIDs) > 0 {
		query = query.Where("event_type = ? OR (event_type = ? AND guild_id IN ?)",
			"server", "guild", memberGuildIDs)
	} else {
		query = query.Where("event_type = ?", "server")
	}

	var posts []model.Post
	query.Order("event_start_time ASC").Find(&posts)

	// 获取作者和公会信息
	type EventItem struct {
		model.Post
		AuthorName      string `json:"author_name"`
		AuthorNameColor string `json:"author_name_color"`
		AuthorNameBold  bool   `json:"author_name_bold"`
		GuildName       string `json:"guild_name,omitempty"`
	}

	// 收集ID
	authorIDs := make([]uint, len(posts))
	guildIDs := []uint{}
	for i, p := range posts {
		authorIDs[i] = p.AuthorID
		if p.GuildID != nil {
			guildIDs = append(guildIDs, *p.GuildID)
		}
	}

	// 查询用户名
	userMap := make(map[uint]model.User)
	if len(authorIDs) > 0 {
		var users []model.User
		database.DB.Where("id IN ?", authorIDs).Find(&users)
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	// 查询公会名
	guildMap := make(map[uint]string)
	if len(guildIDs) > 0 {
		var guilds []model.Guild
		database.DB.Where("id IN ?", guildIDs).Find(&guilds)
		for _, g := range guilds {
			guildMap[g.ID] = g.Name
		}
	}

	// 组装结果
	result := make([]EventItem, len(posts))
	for i, p := range posts {
		author := userMap[p.AuthorID]
		nameColor, nameBold := userDisplayStyle(author)
		item := EventItem{
			Post:            p,
			AuthorName:      author.Username,
			AuthorNameColor: nameColor,
			AuthorNameBold:  nameBold,
		}
		if p.GuildID != nil {
			item.GuildName = guildMap[*p.GuildID]
		}
		result[i] = item
	}

	c.JSON(http.StatusOK, gin.H{"events": result})
}
