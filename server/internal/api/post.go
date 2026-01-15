package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

// CreatePostRequest 创建帖子请求
type CreatePostRequest struct {
	Title       string  `json:"title" binding:"required"`
	Content     string  `json:"content" binding:"required"`
	ContentType string  `json:"content_type"`
	Category    string  `json:"category"` // profile|guild|report|novel|item|event|other
	GuildID     *uint   `json:"guild_id"`
	StoryID     *uint   `json:"story_id"`
	TagIDs      []uint  `json:"tag_ids"`
	Status      string  `json:"status"` // draft|published
}

// UpdatePostRequest 更新帖子请求
type UpdatePostRequest struct {
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	ContentType string  `json:"content_type"`
	Category    string  `json:"category"`
	GuildID     *uint   `json:"guild_id"`
	StoryID     *uint   `json:"story_id"`
	Status      string  `json:"status"`
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
		// 查看他人帖子：只能看到已发布的公开帖子
		query = query.Where("is_public = ?", true)
		query = query.Where("status = ?", "published")
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

	// 更新字段
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

	database.DB.Save(&post)
	c.JSON(http.StatusOK, post)
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
