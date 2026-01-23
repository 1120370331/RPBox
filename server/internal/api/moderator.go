package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/pkg/validator"
)

// logAdminAction 记录管理员操作日志
func logAdminAction(c *gin.Context, actionType, targetType string, targetID uint, targetName string, details map[string]interface{}) {
	userID := c.GetUint("userID")

	var user model.User
	database.DB.Select("username, role").First(&user, userID)

	detailsJSON := ""
	if details != nil {
		if jsonBytes, err := json.Marshal(details); err == nil {
			detailsJSON = string(jsonBytes)
		}
	}

	log := model.AdminActionLog{
		OperatorID:   userID,
		OperatorName: user.Username,
		OperatorRole: user.Role,
		ActionType:   actionType,
		TargetType:   targetType,
		TargetID:     targetID,
		TargetName:   targetName,
		Details:      detailsJSON,
		IPAddress:    c.ClientIP(),
	}
	database.DB.Create(&log)
}

// ReviewRequest 审核请求
type ReviewRequest struct {
	Action  string `json:"action" binding:"required"` // approve|reject
	Comment string `json:"comment"`                   // 审核意见
}

// ========== 帖子审核 ==========

// listPendingPosts 获取待审核帖子列表
func (s *Server) listPendingPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	category := c.Query("category")

	query := database.DB.Model(&model.Post{}).
		Where("review_status = ?", "pending")

	if category != "" {
		query = query.Where("category = ?", category)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * pageSize
	var posts []model.Post
	query.Order("created_at ASC").Offset(offset).Limit(pageSize).Find(&posts)

	// 获取作者信息
	authorIDs := make([]uint, len(posts))
	for i, p := range posts {
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

	type PostWithAuthor struct {
		model.Post
		AuthorName      string `json:"author_name"`
		AuthorNameColor string `json:"author_name_color"`
		AuthorNameBold  bool   `json:"author_name_bold"`
	}
	result := make([]PostWithAuthor, len(posts))
	for i, p := range posts {
		author := userMap[p.AuthorID]
		nameColor, nameBold := userDisplayStyle(author)
		result[i] = PostWithAuthor{
			Post:            p,
			AuthorName:      author.Username,
			AuthorNameColor: nameColor,
			AuthorNameBold:  nameBold,
		}
	}

	c.JSON(http.StatusOK, gin.H{"posts": result, "total": total})
}

// reviewPost 审核帖子
func (s *Server) reviewPost(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	now := time.Now()
	post.ReviewerID = &userID
	post.ReviewComment = req.Comment
	post.ReviewedAt = &now

	if req.Action == "approve" {
		post.ReviewStatus = "approved"
		post.Status = "published"
	} else if req.Action == "reject" {
		post.ReviewStatus = "rejected"
		post.Status = "draft"
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的审核操作"})
		return
	}

	database.DB.Save(&post)

	// 记录日志
	logAdminAction(c, "review_post", "post", uint(id), post.Title, map[string]interface{}{
		"action":  req.Action,
		"comment": req.Comment,
	})

	c.JSON(http.StatusOK, gin.H{"message": "审核完成", "post": post})
}

// listPendingEdits 获取待审核的帖子编辑列表
func (s *Server) listPendingEdits(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	query := database.DB.Model(&model.PostEditRequest{}).
		Where("status = ?", "pending")

	var total int64
	query.Count(&total)

	offset := (page - 1) * pageSize
	var edits []model.PostEditRequest
	query.Order("created_at ASC").Offset(offset).Limit(pageSize).Find(&edits)

	// 获取原帖子和作者信息
	type EditWithInfo struct {
		model.PostEditRequest
		AuthorName      string `json:"author_name"`
		AuthorNameColor string `json:"author_name_color"`
		AuthorNameBold  bool   `json:"author_name_bold"`
		OriginalTitle   string `json:"original_title"`
	}

	result := make([]EditWithInfo, len(edits))
	for i, edit := range edits {
		var user model.User
		database.DB.First(&user, edit.AuthorID)
		nameColor, nameBold := userDisplayStyle(user)
		var post model.Post
		database.DB.Select("title").First(&post, edit.PostID)
		result[i] = EditWithInfo{
			PostEditRequest: edit,
			AuthorName:      user.Username,
			AuthorNameColor: nameColor,
			AuthorNameBold:  nameBold,
			OriginalTitle:   post.Title,
		}
	}

	c.JSON(http.StatusOK, gin.H{"edits": result, "total": total})
}

// reviewPostEdit 审核帖子编辑
func (s *Server) reviewPostEdit(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var edit model.PostEditRequest
	if err := database.DB.First(&edit, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "编辑记录不存在"})
		return
	}

	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	now := time.Now()
	edit.ReviewerID = &userID
	edit.ReviewedAt = &now

	if req.Action == "approve" {
		// 审核通过：将编辑内容应用到原帖子
		var post model.Post
		if err := database.DB.First(&post, edit.PostID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "原帖子不存在"})
			return
		}

		post.Title = edit.Title
		post.Content = edit.Content
		post.ContentType = edit.ContentType
		post.Category = edit.Category
		database.DB.Save(&post)

		// 删除待审核记录
		database.DB.Delete(&edit)

		c.JSON(http.StatusOK, gin.H{"message": "编辑已通过并应用", "post": post})
	} else if req.Action == "reject" {
		edit.Status = "rejected"
		database.DB.Save(&edit)
		c.JSON(http.StatusOK, gin.H{"message": "编辑已拒绝", "edit": edit})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的审核操作"})
	}
}

// ========== 道具审核 ==========

// listPendingItems 获取待审核道具列表
func (s *Server) listPendingItems(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	itemType := c.Query("type")

	query := database.DB.Model(&model.Item{}).
		Where("review_status = ?", "pending")

	if itemType != "" {
		query = query.Where("type = ?", itemType)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * pageSize
	var items []model.Item
	query.Order("created_at ASC").Offset(offset).Limit(pageSize).Find(&items)

	// 获取作者信息
	authorIDs := make([]uint, len(items))
	for i, item := range items {
		authorIDs[i] = item.AuthorID
	}

	var users []model.User
	if len(authorIDs) > 0 {
		database.DB.Where("id IN ?", authorIDs).Find(&users)
	}
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	type ItemWithAuthor struct {
		model.Item
		AuthorName      string `json:"author_name"`
		AuthorNameColor string `json:"author_name_color"`
		AuthorNameBold  bool   `json:"author_name_bold"`
	}
	result := make([]ItemWithAuthor, len(items))
	for i, item := range items {
		author := userMap[item.AuthorID]
		nameColor, nameBold := userDisplayStyle(author)
		result[i] = ItemWithAuthor{
			Item:            item,
			AuthorName:      author.Username,
			AuthorNameColor: nameColor,
			AuthorNameBold:  nameBold,
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": result, "total": total})
}

// reviewItem 审核道具
func (s *Server) reviewItem(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "道具不存在"})
		return
	}

	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	now := time.Now()
	item.ReviewerID = &userID
	item.ReviewComment = req.Comment
	item.ReviewedAt = &now

	if req.Action == "approve" {
		item.ReviewStatus = "approved"
		item.Status = "published"
	} else if req.Action == "reject" {
		item.ReviewStatus = "rejected"
		item.Status = "draft"
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的审核操作"})
		return
	}

	database.DB.Save(&item)

	// 记录日志
	logAdminAction(c, "review_item", "item", uint(id), item.Name, map[string]interface{}{
		"action":  req.Action,
		"comment": req.Comment,
	})

	c.JSON(http.StatusOK, gin.H{"message": "审核完成", "item": item})
}

// listPendingItemEdits 获取待审核的道具编辑列表
func (s *Server) listPendingItemEdits(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	query := database.DB.Model(&model.ItemPendingEdit{}).
		Where("review_status = ?", "pending")

	var total int64
	query.Count(&total)

	offset := (page - 1) * pageSize
	var edits []model.ItemPendingEdit
	query.Order("created_at ASC").Offset(offset).Limit(pageSize).Find(&edits)

	// 获取原道具和作者信息
	type EditWithInfo struct {
		model.ItemPendingEdit
		AuthorName      string `json:"author_name"`
		AuthorNameColor string `json:"author_name_color"`
		AuthorNameBold  bool   `json:"author_name_bold"`
		OriginalName    string `json:"original_name"`
	}

	result := make([]EditWithInfo, len(edits))
	for i, edit := range edits {
		var user model.User
		database.DB.First(&user, edit.AuthorID)
		nameColor, nameBold := userDisplayStyle(user)
		var item model.Item
		database.DB.Select("name").First(&item, edit.ItemID)
		result[i] = EditWithInfo{
			ItemPendingEdit: edit,
			AuthorName:      user.Username,
			AuthorNameColor: nameColor,
			AuthorNameBold:  nameBold,
			OriginalName:    item.Name,
		}
	}

	c.JSON(http.StatusOK, gin.H{"edits": result, "total": total})
}

// reviewItemEdit 审核道具编辑
func (s *Server) reviewItemEdit(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var edit model.ItemPendingEdit
	if err := database.DB.First(&edit, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "编辑记录不存在"})
		return
	}

	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	now := time.Now()
	edit.ReviewerID = &userID
	edit.ReviewComment = req.Comment
	edit.ReviewedAt = &now

	if req.Action == "approve" {
		// 审核通过：将编辑内容应用到原道具
		var item model.Item
		if err := database.DB.First(&item, edit.ItemID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "原道具不存在"})
			return
		}

		item.Name = edit.Name
		item.Icon = edit.Icon
		item.Description = edit.Description
		item.ImportCode = edit.ImportCode
		database.DB.Save(&item)

		// 删除待审核记录
		database.DB.Delete(&edit)

		c.JSON(http.StatusOK, gin.H{"message": "编辑已通过并应用", "item": item})
	} else if req.Action == "reject" {
		edit.ReviewStatus = "rejected"
		database.DB.Save(&edit)
		c.JSON(http.StatusOK, gin.H{"message": "编辑已拒绝", "edit": edit})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的审核操作"})
	}
}

// ========== 管理中心 ==========

// listAllPosts 获取所有帖子（管理用）
func (s *Server) listAllPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	reviewStatus := c.Query("review_status")
	category := c.Query("category")
	keyword := c.Query("keyword")
	isPinned := c.Query("is_pinned")
	isFeatured := c.Query("is_featured")

	query := database.DB.Model(&model.Post{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if reviewStatus != "" {
		query = query.Where("review_status = ?", reviewStatus)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if keyword != "" {
		query = query.Where("title LIKE ?", "%"+keyword+"%")
	}
	if isPinned != "" {
		value, err := strconv.ParseBool(isPinned)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的is_pinned参数"})
			return
		}
		query = query.Where("is_pinned = ?", value)
	}
	if isFeatured != "" {
		value, err := strconv.ParseBool(isFeatured)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的is_featured参数"})
			return
		}
		query = query.Where("is_featured = ?", value)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * pageSize
	var posts []model.Post
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&posts)

	// 获取作者信息
	authorIDs := make([]uint, len(posts))
	for i, p := range posts {
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

	type PostWithAuthor struct {
		model.Post
		AuthorName      string `json:"author_name"`
		AuthorNameColor string `json:"author_name_color"`
		AuthorNameBold  bool   `json:"author_name_bold"`
	}
	result := make([]PostWithAuthor, len(posts))
	for i, p := range posts {
		author := userMap[p.AuthorID]
		nameColor, nameBold := userDisplayStyle(author)
		result[i] = PostWithAuthor{
			Post:            p,
			AuthorName:      author.Username,
			AuthorNameColor: nameColor,
			AuthorNameBold:  nameBold,
		}
	}

	c.JSON(http.StatusOK, gin.H{"posts": result, "total": total})
}

// deletePostByMod 版主删除帖子
func (s *Server) deletePostByMod(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	postTitle := post.Title // 保存标题用于日志

	// 删除关联数据
	database.DB.Where("post_id = ?", id).Delete(&model.PostTag{})
	database.DB.Where("post_id = ?", id).Delete(&model.Comment{})
	database.DB.Where("post_id = ?", id).Delete(&model.PostLike{})
	database.DB.Where("post_id = ?", id).Delete(&model.PostFavorite{})

	s.cleanupPostImages(c, post)
	database.DB.Delete(&post)

	// 记录日志
	logAdminAction(c, "delete_post", "post", uint(id), postTitle, nil)

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// hidePostByMod 版主屏蔽帖子（打回待审核）
func (s *Server) hidePostByMod(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	post.ReviewStatus = "pending"
	post.Status = "pending"
	database.DB.Save(&post)

	// 记录日志
	logAdminAction(c, "hide_post", "post", uint(id), post.Title, nil)

	c.JSON(http.StatusOK, gin.H{"message": "已屏蔽，帖子已打回待审核"})
}

// pinPost 置顶/取消置顶帖子
func (s *Server) pinPost(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	post.IsPinned = !post.IsPinned
	database.DB.Save(&post)

	// 记录日志
	logAdminAction(c, "pin_post", "post", uint(id), post.Title, map[string]interface{}{
		"is_pinned": post.IsPinned,
	})

	msg := "已置顶"
	if !post.IsPinned {
		msg = "已取消置顶"
	}
	c.JSON(http.StatusOK, gin.H{"message": msg, "is_pinned": post.IsPinned})
}

// featurePost 设为精华/取消精华
func (s *Server) featurePost(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	post.IsFeatured = !post.IsFeatured
	database.DB.Save(&post)

	// 记录日志
	logAdminAction(c, "feature_post", "post", uint(id), post.Title, map[string]interface{}{
		"is_featured": post.IsFeatured,
	})

	msg := "已设为精华"
	if !post.IsFeatured {
		msg = "已取消精华"
	}
	c.JSON(http.StatusOK, gin.H{"message": msg, "is_featured": post.IsFeatured})
}

// listAllItems 获取所有道具（管理用）
func (s *Server) listAllItems(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	reviewStatus := c.Query("review_status")
	itemType := c.Query("type")
	keyword := c.Query("keyword")

	query := database.DB.Model(&model.Item{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if reviewStatus != "" {
		query = query.Where("review_status = ?", reviewStatus)
	}
	if itemType != "" {
		query = query.Where("type = ?", itemType)
	}
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * pageSize
	var items []model.Item
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items)

	// 获取作者信息
	authorIDs := make([]uint, len(items))
	for i, item := range items {
		authorIDs[i] = item.AuthorID
	}

	var users []model.User
	if len(authorIDs) > 0 {
		database.DB.Where("id IN ?", authorIDs).Find(&users)
	}
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	type ItemWithAuthor struct {
		model.Item
		AuthorName      string `json:"author_name"`
		AuthorNameColor string `json:"author_name_color"`
		AuthorNameBold  bool   `json:"author_name_bold"`
	}
	result := make([]ItemWithAuthor, len(items))
	for i, item := range items {
		author := userMap[item.AuthorID]
		nameColor, nameBold := userDisplayStyle(author)
		result[i] = ItemWithAuthor{
			Item:            item,
			AuthorName:      author.Username,
			AuthorNameColor: nameColor,
			AuthorNameBold:  nameBold,
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": result, "total": total})
}

// deleteItemByMod 版主删除道具
func (s *Server) deleteItemByMod(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "道具不存在"})
		return
	}

	itemName := item.Name // 保存名称用于日志

	// 删除关联数据
	database.DB.Where("item_id = ?", id).Delete(&model.ItemTag{})
	database.DB.Where("item_id = ?", id).Delete(&model.ItemComment{})
	database.DB.Where("item_id = ?", id).Delete(&model.ItemLike{})
	database.DB.Where("item_id = ?", id).Delete(&model.ItemFavorite{})
	database.DB.Where("item_id = ?", id).Delete(&model.ItemRating{})

	var itemImages []model.ItemImage
	database.DB.Where("item_id = ?", id).Find(&itemImages)
	s.cleanupItemImages(c, item, itemImages)
	database.DB.Where("item_id = ?", id).Delete(&model.ItemImage{})
	database.DB.Delete(&item)

	// 记录日志
	logAdminAction(c, "delete_item", "item", uint(id), itemName, nil)

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// hideItemByMod 版主屏蔽道具（打回待审核）
func (s *Server) hideItemByMod(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "道具不存在"})
		return
	}

	item.ReviewStatus = "pending"
	item.Status = "pending"
	database.DB.Save(&item)

	// 记录日志
	logAdminAction(c, "hide_item", "item", uint(id), item.Name, nil)

	c.JSON(http.StatusOK, gin.H{"message": "已屏蔽，道具已打回待审核"})
}

// getModeratorStats 获取版主统计数据
func (s *Server) getModeratorStats(c *gin.Context) {
	var pendingPosts, pendingItems int64
	var totalPosts, totalItems int64
	var todayPosts, todayItems int64
	var pendingGuilds, totalGuilds int64
	var totalUsers, todayUsers int64

	// 待审核数量
	database.DB.Model(&model.Post{}).Where("review_status = ?", "pending").Count(&pendingPosts)
	database.DB.Model(&model.Item{}).Where("review_status = ?", "pending").Count(&pendingItems)
	database.DB.Model(&model.Guild{}).Where("status = ?", "pending").Count(&pendingGuilds)

	// 总数量
	database.DB.Model(&model.Post{}).Count(&totalPosts)
	database.DB.Model(&model.Item{}).Count(&totalItems)
	database.DB.Model(&model.Guild{}).Where("status = ?", "approved").Count(&totalGuilds)
	database.DB.Model(&model.User{}).Count(&totalUsers)

	// 今日新增
	today := time.Now().Format("2006-01-02")
	database.DB.Model(&model.Post{}).Where("DATE(created_at) = ?", today).Count(&todayPosts)
	database.DB.Model(&model.Item{}).Where("DATE(created_at) = ?", today).Count(&todayItems)
	database.DB.Model(&model.User{}).Where("DATE(created_at) = ?", today).Count(&todayUsers)

	c.JSON(http.StatusOK, gin.H{
		"pending_posts":  pendingPosts,
		"pending_items":  pendingItems,
		"pending_guilds": pendingGuilds,
		"total_posts":    totalPosts,
		"total_items":    totalItems,
		"total_guilds":   totalGuilds,
		"total_users":    totalUsers,
		"today_posts":    todayPosts,
		"today_items":    todayItems,
		"today_users":    todayUsers,
	})
}

// ========== 公会管理（版主可用） ==========

// listPendingGuilds 获取待审核公会列表
func (s *Server) listPendingGuilds(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	query := database.DB.Model(&model.Guild{}).Where("status = ?", "pending")

	var total int64
	query.Count(&total)

	offset := (page - 1) * pageSize
	var guilds []model.Guild
	query.Order("created_at ASC").Offset(offset).Limit(pageSize).Find(&guilds)

	// 获取创建者信息
	ownerIDs := make([]uint, len(guilds))
	for i, g := range guilds {
		ownerIDs[i] = g.OwnerID
	}

	var users []model.User
	if len(ownerIDs) > 0 {
		database.DB.Where("id IN ?", ownerIDs).Find(&users)
	}
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	type GuildWithOwner struct {
		model.Guild
		OwnerName      string `json:"owner_name"`
		OwnerNameColor string `json:"owner_name_color"`
		OwnerNameBold  bool   `json:"owner_name_bold"`
	}
	result := make([]GuildWithOwner, len(guilds))
	for i, g := range guilds {
		owner := userMap[g.OwnerID]
		nameColor, nameBold := userDisplayStyle(owner)
		result[i] = GuildWithOwner{
			Guild:          g,
			OwnerName:      owner.Username,
			OwnerNameColor: nameColor,
			OwnerNameBold:  nameBold,
		}
	}

	c.JSON(http.StatusOK, gin.H{"guilds": result, "total": total})
}

// reviewGuild 审核公会
func (s *Server) reviewGuild(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var guild model.Guild
	if err := database.DB.First(&guild, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "公会不存在"})
		return
	}

	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	now := time.Now()
	guild.ReviewerID = &userID
	guild.ReviewComment = req.Comment
	guild.ReviewedAt = &now

	if req.Action == "approve" {
		guild.Status = "approved"
	} else if req.Action == "reject" {
		guild.Status = "rejected"
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的审核操作"})
		return
	}

	database.DB.Save(&guild)

	// 记录日志
	logAdminAction(c, "review_guild", "guild", uint(id), guild.Name, map[string]interface{}{
		"action":  req.Action,
		"comment": req.Comment,
	})

	c.JSON(http.StatusOK, gin.H{"message": "审核完成", "guild": guild})
}

// listAllGuilds 获取所有公会（管理用）
func (s *Server) listAllGuilds(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	keyword := c.Query("keyword")

	query := database.DB.Model(&model.Guild{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * pageSize
	var guilds []model.Guild
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&guilds)

	// 获取创建者信息
	ownerIDs := make([]uint, len(guilds))
	for i, g := range guilds {
		ownerIDs[i] = g.OwnerID
	}

	var users []model.User
	if len(ownerIDs) > 0 {
		database.DB.Where("id IN ?", ownerIDs).Find(&users)
	}
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	type GuildWithOwner struct {
		model.Guild
		OwnerName      string `json:"owner_name"`
		OwnerNameColor string `json:"owner_name_color"`
		OwnerNameBold  bool   `json:"owner_name_bold"`
	}
	result := make([]GuildWithOwner, len(guilds))
	for i, g := range guilds {
		owner := userMap[g.OwnerID]
		nameColor, nameBold := userDisplayStyle(owner)
		result[i] = GuildWithOwner{
			Guild:          g,
			OwnerName:      owner.Username,
			OwnerNameColor: nameColor,
			OwnerNameBold:  nameBold,
		}
	}

	c.JSON(http.StatusOK, gin.H{"guilds": result, "total": total})
}

// ChangeGuildOwnerRequest 更改公会会长请求
type ChangeGuildOwnerRequest struct {
	NewOwnerID   *uint  `json:"new_owner_id"`   // 可选：用户ID
	NewOwnerName string `json:"new_owner_name"` // 可选：用户名或邮箱
}

// changeGuildOwner 更改公会会长
func (s *Server) changeGuildOwner(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var guild model.Guild
	if err := database.DB.First(&guild, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "公会不存在"})
		return
	}

	var req ChangeGuildOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	// 查找新会长
	var newOwner model.User
	if req.NewOwnerID != nil && *req.NewOwnerID > 0 {
		// 通过ID查找
		if err := database.DB.First(&newOwner, *req.NewOwnerID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
	} else if req.NewOwnerName != "" {
		// 通过用户名或邮箱查找
		if err := database.DB.Where("username = ? OR email = ?", req.NewOwnerName, req.NewOwnerName).First(&newOwner).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "未找到该用户，请检查用户名或邮箱"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供新会长的ID或用户名/邮箱"})
		return
	}

	oldOwnerID := guild.OwnerID
	guild.OwnerID = newOwner.ID
	database.DB.Save(&guild)

	// 更新成员角色
	database.DB.Model(&model.GuildMember{}).
		Where("guild_id = ? AND user_id = ?", id, oldOwnerID).
		Update("role", "admin")

	// 检查新会长是否已是成员
	var member model.GuildMember
	err := database.DB.Where("guild_id = ? AND user_id = ?", id, newOwner.ID).First(&member).Error
	if err != nil {
		database.DB.Create(&model.GuildMember{
			GuildID:  uint(id),
			UserID:   newOwner.ID,
			Role:     "owner",
			JoinedAt: time.Now(),
		})
		database.DB.Model(&guild).Update("member_count", guild.MemberCount+1)
	} else {
		database.DB.Model(&member).Update("role", "owner")
	}

	// 记录日志
	logAdminAction(c, "change_guild_owner", "guild", uint(id), guild.Name, map[string]interface{}{
		"old_owner_id":   oldOwnerID,
		"new_owner_id":   newOwner.ID,
		"new_owner_name": newOwner.Username,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "会长已更换",
		"guild":   guild,
		"new_owner": gin.H{
			"id":       newOwner.ID,
			"username": newOwner.Username,
		},
	})
}

// deleteGuildByMod 版主删除公会
func (s *Server) deleteGuildByMod(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var guild model.Guild
	if err := database.DB.First(&guild, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "公会不存在"})
		return
	}

	guildName := guild.Name // 保存名称用于日志

	// 删除关联数据
	database.DB.Where("guild_id = ?", id).Delete(&model.GuildMember{})
	database.DB.Where("guild_id = ?", id).Delete(&model.StoryGuild{})
	database.DB.Where("guild_id = ?", id).Delete(&model.Tag{})
	database.DB.Delete(&guild)

	// 记录日志
	logAdminAction(c, "delete_guild", "guild", uint(id), guildName, nil)

	c.JSON(http.StatusOK, gin.H{"message": "公会已删除"})
}

// ========== 管理日志 ==========

// listActionLogs 获取管理操作日志
func (s *Server) listActionLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	operatorID := c.Query("operator_id")
	actionType := c.Query("action_type")
	targetType := c.Query("target_type")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	query := database.DB.Model(&model.AdminActionLog{})

	if operatorID != "" {
		if id, err := strconv.ParseUint(operatorID, 10, 32); err == nil {
			query = query.Where("operator_id = ?", id)
		}
	}
	if actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}
	if targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	if startDate != "" {
		query = query.Where("DATE(created_at) >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("DATE(created_at) <= ?", endDate)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * pageSize
	var logs []model.AdminActionLog
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs)

	userIDs := make(map[uint]struct{})
	for _, log := range logs {
		if log.OperatorID != 0 {
			userIDs[log.OperatorID] = struct{}{}
		}
		if log.TargetType == "user" && log.TargetID != 0 {
			userIDs[log.TargetID] = struct{}{}
		}
	}

	ids := make([]uint, 0, len(userIDs))
	for id := range userIDs {
		ids = append(ids, id)
	}

	userMap := make(map[uint]model.User)
	if len(ids) > 0 {
		var users []model.User
		database.DB.Where("id IN ?", ids).Find(&users)
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	type LogWithStyle struct {
		model.AdminActionLog
		OperatorNameColor string `json:"operator_name_color"`
		OperatorNameBold  bool   `json:"operator_name_bold"`
		TargetNameColor   string `json:"target_name_color,omitempty"`
		TargetNameBold    bool   `json:"target_name_bold,omitempty"`
	}

	result := make([]LogWithStyle, len(logs))
	for i, log := range logs {
		operator := userMap[log.OperatorID]
		operatorColor, operatorBold := userDisplayStyle(operator)
		var targetColor string
		var targetBold bool
		if log.TargetType == "user" {
			if target, ok := userMap[log.TargetID]; ok {
				targetColor, targetBold = userDisplayStyle(target)
			}
		}
		result[i] = LogWithStyle{
			AdminActionLog:    log,
			OperatorNameColor: operatorColor,
			OperatorNameBold:  operatorBold,
			TargetNameColor:   targetColor,
			TargetNameBold:    targetBold,
		}
	}

	c.JSON(http.StatusOK, gin.H{"logs": result, "total": total})
}

// ========== Metrics 统计 ==========

// getMetricsHistory 获取历史统计数据
func (s *Server) getMetricsHistory(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days > 365 {
		days = 365
	}
	if days < 1 {
		days = 30
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days+1)

	type DailyData struct {
		Date        string `json:"date"`
		TotalUsers  int64  `json:"total_users"`
		TotalPosts  int64  `json:"total_posts"`
		TotalItems  int64  `json:"total_items"`
		TotalGuilds int64  `json:"total_guilds"`
		NewUsers    int64  `json:"new_users"`
		NewPosts    int64  `json:"new_posts"`
		NewItems    int64  `json:"new_items"`
		NewGuilds   int64  `json:"new_guilds"`
	}

	var result []DailyData

	// 遍历每一天计算数据
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		dateEnd := d.Add(24 * time.Hour)

		var data DailyData
		data.Date = dateStr

		// 计算截止到当天的累计数量
		database.DB.Model(&model.User{}).Where("created_at < ?", dateEnd).Count(&data.TotalUsers)
		database.DB.Model(&model.Post{}).Where("created_at < ?", dateEnd).Count(&data.TotalPosts)
		database.DB.Model(&model.Item{}).Where("created_at < ?", dateEnd).Count(&data.TotalItems)
		database.DB.Model(&model.Guild{}).Where("status = ? AND created_at < ?", "approved", dateEnd).Count(&data.TotalGuilds)

		// 计算当天新增数量
		database.DB.Model(&model.User{}).Where("DATE(created_at) = ?", dateStr).Count(&data.NewUsers)
		database.DB.Model(&model.Post{}).Where("DATE(created_at) = ?", dateStr).Count(&data.NewPosts)
		database.DB.Model(&model.Item{}).Where("DATE(created_at) = ?", dateStr).Count(&data.NewItems)
		database.DB.Model(&model.Guild{}).Where("DATE(created_at) = ? AND status = ?", dateStr, "approved").Count(&data.NewGuilds)

		result = append(result, data)
	}

	c.JSON(http.StatusOK, gin.H{"metrics": result, "days": days})
}

// getMetricsSummary 获取统计摘要（今日/周/月对比）
func (s *Server) getMetricsSummary(c *gin.Context) {
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	monthAgo := time.Now().AddDate(0, -1, 0).Format("2006-01-02")

	type PeriodStats struct {
		Users  int64 `json:"users"`
		Posts  int64 `json:"posts"`
		Items  int64 `json:"items"`
		Guilds int64 `json:"guilds"`
	}

	var todayStats, yesterdayStats, weekStats, monthStats PeriodStats

	// 今日新增
	database.DB.Model(&model.User{}).Where("DATE(created_at) = ?", today).Count(&todayStats.Users)
	database.DB.Model(&model.Post{}).Where("DATE(created_at) = ?", today).Count(&todayStats.Posts)
	database.DB.Model(&model.Item{}).Where("DATE(created_at) = ?", today).Count(&todayStats.Items)
	database.DB.Model(&model.Guild{}).Where("DATE(created_at) = ? AND status = ?", today, "approved").Count(&todayStats.Guilds)

	// 昨日新增
	database.DB.Model(&model.User{}).Where("DATE(created_at) = ?", yesterday).Count(&yesterdayStats.Users)
	database.DB.Model(&model.Post{}).Where("DATE(created_at) = ?", yesterday).Count(&yesterdayStats.Posts)
	database.DB.Model(&model.Item{}).Where("DATE(created_at) = ?", yesterday).Count(&yesterdayStats.Items)
	database.DB.Model(&model.Guild{}).Where("DATE(created_at) = ? AND status = ?", yesterday, "approved").Count(&yesterdayStats.Guilds)

	// 本周新增（过去7天）
	database.DB.Model(&model.User{}).Where("DATE(created_at) >= ?", weekAgo).Count(&weekStats.Users)
	database.DB.Model(&model.Post{}).Where("DATE(created_at) >= ?", weekAgo).Count(&weekStats.Posts)
	database.DB.Model(&model.Item{}).Where("DATE(created_at) >= ?", weekAgo).Count(&weekStats.Items)
	database.DB.Model(&model.Guild{}).Where("DATE(created_at) >= ? AND status = ?", weekAgo, "approved").Count(&weekStats.Guilds)

	// 本月新增（过去30天）
	database.DB.Model(&model.User{}).Where("DATE(created_at) >= ?", monthAgo).Count(&monthStats.Users)
	database.DB.Model(&model.Post{}).Where("DATE(created_at) >= ?", monthAgo).Count(&monthStats.Posts)
	database.DB.Model(&model.Item{}).Where("DATE(created_at) >= ?", monthAgo).Count(&monthStats.Items)
	database.DB.Model(&model.Guild{}).Where("DATE(created_at) >= ? AND status = ?", monthAgo, "approved").Count(&monthStats.Guilds)

	c.JSON(http.StatusOK, gin.H{
		"today":     todayStats,
		"yesterday": yesterdayStats,
		"week":      weekStats,
		"month":     monthStats,
	})
}

// getMetricsBasic 获取基础功能监控数据
func (s *Server) getMetricsBasic(c *gin.Context) {
	var storyArchives int64
	var storyEntries int64
	var profileBackups int64

	database.DB.Model(&model.Story{}).Count(&storyArchives)
	database.DB.Model(&model.StoryEntry{}).Count(&storyEntries)
	database.DB.Model(&model.AccountBackup{}).
		Select("COALESCE(SUM(profiles_count), 0)").
		Scan(&profileBackups)

	c.JSON(http.StatusOK, gin.H{
		"story_archives":  storyArchives,
		"story_entries":   storyEntries,
		"profile_backups": profileBackups,
	})
}

// getMetricsBasicHistory 获取基础监控历史趋势数据
func (s *Server) getMetricsBasicHistory(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days > 365 {
		days = 365
	}
	if days < 1 {
		days = 30
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days+1)

	type DailyData struct {
		Date              string `json:"date"`
		NewStoryArchives  int64  `json:"new_story_archives"`
		NewStoryEntries   int64  `json:"new_story_entries"`
		NewProfileBackups int64  `json:"new_profile_backups"`
	}

	var result []DailyData

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")

		var data DailyData
		data.Date = dateStr

		database.DB.Model(&model.Story{}).Where("DATE(created_at) = ?", dateStr).Count(&data.NewStoryArchives)
		database.DB.Model(&model.StoryEntry{}).Where("DATE(created_at) = ?", dateStr).Count(&data.NewStoryEntries)
		database.DB.Model(&model.AccountBackup{}).
			Select("COALESCE(SUM(profiles_count), 0)").
			Where("DATE(created_at) = ?", dateStr).
			Scan(&data.NewProfileBackups)

		result = append(result, data)
	}

	c.JSON(http.StatusOK, gin.H{"metrics": result, "days": days})
}
