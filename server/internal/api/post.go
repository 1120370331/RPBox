package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

// CreatePostRequest 创建帖子请求
type CreatePostRequest struct {
	Title          string  `json:"title" binding:"required"`
	Content        string  `json:"content" binding:"required"`
	ContentType    string  `json:"content_type"`
	Category       string  `json:"category"` // profile|guild|report|novel|item|event|other
	GuildID        *uint   `json:"guild_id"`
	StoryID        *uint   `json:"story_id"`
	TagIDs         []uint  `json:"tag_ids"`
	Status         string  `json:"status"`           // draft|published
	EventType      string  `json:"event_type"`       // server|guild
	EventStartTime *string `json:"event_start_time"` // ISO8601格式
	EventEndTime   *string `json:"event_end_time"`
}

// UpdatePostRequest 更新帖子请求
type UpdatePostRequest struct {
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	ContentType    string  `json:"content_type"`
	Category       string  `json:"category"`
	GuildID        *uint   `json:"guild_id"`
	StoryID        *uint   `json:"story_id"`
	Status         string  `json:"status"`
	EventType      string  `json:"event_type"`
	EventStartTime *string `json:"event_start_time"`
	EventEndTime   *string `json:"event_end_time"`
}

// listPosts 获取帖子列表
func (s *Server) listPosts(c *gin.Context) {
	userID := c.GetUint("userID")

	// 查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	sortBy := c.DefaultQuery("sort", "created_at") // created_at|view_count|like_count
	order := c.DefaultQuery("order", "desc")
	guildID := c.Query("guild_id")
	tagID := c.Query("tag_id")
	authorID := c.Query("author_id")
	status := c.DefaultQuery("status", "published")
	category := c.Query("category") // 分区筛选

	query := database.DB.Model(&model.Post{})

	// 只显示公开的帖子，除非是查看自己的
	if authorID != "" && authorID == strconv.Itoa(int(userID)) {
		// 查看自己的帖子：可以看到所有状态（包括草稿）
		query = query.Where("author_id = ?", authorID)
		// 如果指定了状态，则过滤
		if status != "" && status != "all" {
			query = query.Where("status = ?", status)
		}
	} else {
		// 查看他人帖子：只能看到已发布且审核通过的公开帖子
		query = query.Where("is_public = ?", true)
		query = query.Where("status = ?", "published")
		query = query.Where("review_status = ?", "approved")
		if authorID != "" {
			query = query.Where("author_id = ?", authorID)
		}
	}

	if guildID != "" {
		query = query.Where("guild_id = ?", guildID)
	}

	// 分区筛选
	if category != "" {
		query = query.Where("category = ?", category)
	}

	if tagID != "" {
		// 通过标签筛选
		var postTags []model.PostTag
		database.DB.Where("tag_id = ?", tagID).Find(&postTags)
		postIDs := make([]uint, len(postTags))
		for i, pt := range postTags {
			postIDs[i] = pt.PostID
		}
		if len(postIDs) > 0 {
			query = query.Where("id IN ?", postIDs)
		} else {
			c.JSON(http.StatusOK, gin.H{"posts": []model.Post{}, "total": 0})
			return
		}
	}

	// 统计总数
	var total int64
	query.Count(&total)

	// 排序和分页
	offset := (page - 1) * pageSize
	query = query.Order(sortBy + " " + order).Offset(offset).Limit(pageSize)

	var posts []model.Post
	if err := query.Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 获取作者信息
	authorIDs := make([]uint, len(posts))
	for i, p := range posts {
		authorIDs[i] = p.AuthorID
	}

	var users []model.User
	database.DB.Where("id IN ?", authorIDs).Find(&users)
	userMap := make(map[uint]string)
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	// 组装响应
	type PostWithAuthor struct {
		model.Post
		AuthorName string `json:"author_name"`
	}
	result := make([]PostWithAuthor, len(posts))
	for i, p := range posts {
		result[i] = PostWithAuthor{Post: p, AuthorName: userMap[p.AuthorID]}
	}

	c.JSON(http.StatusOK, gin.H{"posts": result, "total": total})
}

// createPost 创建帖子
func (s *Server) createPost(c *gin.Context) {
	userID := c.GetUint("userID")

	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// 活动分区权限验证
	if req.Category == "event" && req.EventType != "" {
		if req.EventType == "server" {
			// 服务器活动需要版主权限
			if !checkModerator(userID) {
				c.JSON(http.StatusForbidden, gin.H{"error": "发布服务器活动需要版主权限"})
				return
			}
		} else if req.EventType == "guild" {
			// 公会活动需要公会管理员权限
			if req.GuildID == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "公会活动需要选择公会"})
				return
			}
			if !checkGuildAdmin(*req.GuildID, userID) {
				c.JSON(http.StatusForbidden, gin.H{"error": "发布公会活动需要公会管理员权限"})
				return
			}
		}
	}

	post := model.Post{
		AuthorID:    userID,
		Title:       req.Title,
		Content:     req.Content,
		ContentType: req.ContentType,
		Category:    req.Category,
		GuildID:     req.GuildID,
		StoryID:     req.StoryID,
		Status:      req.Status,
		IsPublic:    true,
		EventType:   req.EventType,
	}

	// 设置审核状态：版主/管理员自动通过，普通用户需要审核
	// 草稿状态不需要审核
	isModerator := checkModerator(userID)
	if req.Status == "published" {
		if isModerator {
			post.Status = "published"
			post.ReviewStatus = "approved" // 版主/管理员自动通过
		} else {
			post.Status = "pending"        // 改为待发布状态
			post.ReviewStatus = "pending"  // 待审核
		}
	} else {
		// 草稿状态，不需要审核
		post.ReviewStatus = ""
	}

	// 解析活动时间
	if req.EventStartTime != nil {
		t, err := time.Parse(time.RFC3339, *req.EventStartTime)
		if err == nil {
			post.EventStartTime = &t
		}
	}
	if req.EventEndTime != nil {
		t, err := time.Parse(time.RFC3339, *req.EventEndTime)
		if err == nil {
			post.EventEndTime = &t
		}
	}

	if err := database.DB.Create(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

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

	c.JSON(http.StatusCreated, post)
}

// getPost 获取帖子详情
func (s *Server) getPost(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	// 权限检查：非公开帖子只有作者可见
	if !post.IsPublic && post.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权查看"})
		return
	}

	// 增加浏览次数
	database.DB.Model(&post).Update("view_count", post.ViewCount+1)

	// 获取作者信息
	var author model.User
	database.DB.First(&author, post.AuthorID)

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

	c.JSON(http.StatusOK, gin.H{
		"post":        post,
		"author_name": author.Username,
		"tags":        tags,
		"liked":       liked,
		"favorited":   favorited,
	})
}

// updatePost 更新帖子
func (s *Server) updatePost(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	// 只有作者可以更新
	if post.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isModerator := checkModerator(userID)

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

		database.DB.Save(&editReq)
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
	if req.Category != "" {
		post.Category = req.Category
	}
	if req.Status != "" {
		post.Status = req.Status
	}
	post.GuildID = req.GuildID
	post.StoryID = req.StoryID

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

	database.DB.Save(&post)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": post})
}

// deletePost 删除帖子
func (s *Server) deletePost(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	// 只有作者可以删除
	if post.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	// 删除关联数据
	database.DB.Where("post_id = ?", id).Delete(&model.PostTag{})
	database.DB.Where("post_id = ?", id).Delete(&model.Comment{})
	database.DB.Where("post_id = ?", id).Delete(&model.PostLike{})
	database.DB.Where("post_id = ?", id).Delete(&model.PostFavorite{})
	database.DB.Delete(&post)

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

	// 检查帖子所有权
	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}
	if post.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	var req AddPostTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// 检查帖子所有权
	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}
	if post.AuthorID != userID {
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
	userID := c.GetUint("userID")

	// 获取收藏的帖子ID
	var favorites []model.PostFavorite
	database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&favorites)

	postIDs := make([]uint, len(favorites))
	for i, fav := range favorites {
		postIDs[i] = fav.PostID
	}

	var posts []model.Post
	if len(postIDs) > 0 {
		database.DB.Where("id IN ?", postIDs).Find(&posts)
	}

	// 获取作者信息
	authorIDs := make([]uint, len(posts))
	for i, p := range posts {
		authorIDs[i] = p.AuthorID
	}

	var users []model.User
	if len(authorIDs) > 0 {
		database.DB.Where("id IN ?", authorIDs).Find(&users)
	}
	userMap := make(map[uint]string)
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	// 组装响应
	type PostWithAuthor struct {
		model.Post
		AuthorName string `json:"author_name"`
	}
	result := make([]PostWithAuthor, len(posts))
	for i, p := range posts {
		result[i] = PostWithAuthor{Post: p, AuthorName: userMap[p.AuthorID]}
	}

	c.JSON(http.StatusOK, gin.H{"posts": result})
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
		AuthorName string `json:"author_name"`
		GuildName  string `json:"guild_name,omitempty"`
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
	userMap := make(map[uint]string)
	if len(authorIDs) > 0 {
		var users []model.User
		database.DB.Where("id IN ?", authorIDs).Find(&users)
		for _, u := range users {
			userMap[u.ID] = u.Username
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
		item := EventItem{Post: p, AuthorName: userMap[p.AuthorID]}
		if p.GuildID != nil {
			item.GuildName = guildMap[*p.GuildID]
		}
		result[i] = item
	}

	c.JSON(http.StatusOK, gin.H{"events": result})
}
