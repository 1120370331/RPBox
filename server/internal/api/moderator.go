package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

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
	userMap := make(map[uint]string)
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		AuthorName    string `json:"author_name"`
		OriginalTitle string `json:"original_title"`
	}

	result := make([]EditWithInfo, len(edits))
	for i, edit := range edits {
		var user model.User
		database.DB.Select("username").First(&user, edit.AuthorID)
		var post model.Post
		database.DB.Select("title").First(&post, edit.PostID)
		result[i] = EditWithInfo{
			PostEditRequest: edit,
			AuthorName:      user.Username,
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	userMap := make(map[uint]string)
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	type ItemWithAuthor struct {
		model.Item
		AuthorName string `json:"author_name"`
	}
	result := make([]ItemWithAuthor, len(items))
	for i, item := range items {
		result[i] = ItemWithAuthor{Item: item, AuthorName: userMap[item.AuthorID]}
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		AuthorName   string `json:"author_name"`
		OriginalName string `json:"original_name"`
	}

	result := make([]EditWithInfo, len(edits))
	for i, edit := range edits {
		var user model.User
		database.DB.Select("username").First(&user, edit.AuthorID)
		var item model.Item
		database.DB.Select("name").First(&item, edit.ItemID)
		result[i] = EditWithInfo{
			ItemPendingEdit: edit,
			AuthorName:      user.Username,
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	userMap := make(map[uint]string)
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

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

// deletePostByMod 版主删除帖子
func (s *Server) deletePostByMod(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var post model.Post
	if err := database.DB.First(&post, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
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

	c.JSON(http.StatusOK, gin.H{"message": "已屏蔽，帖子已打回待审核"})
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
	userMap := make(map[uint]string)
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	type ItemWithAuthor struct {
		model.Item
		AuthorName string `json:"author_name"`
	}
	result := make([]ItemWithAuthor, len(items))
	for i, item := range items {
		result[i] = ItemWithAuthor{Item: item, AuthorName: userMap[item.AuthorID]}
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

	// 删除关联数据
	database.DB.Where("item_id = ?", id).Delete(&model.ItemTag{})
	database.DB.Where("item_id = ?", id).Delete(&model.ItemComment{})
	database.DB.Where("item_id = ?", id).Delete(&model.ItemLike{})
	database.DB.Where("item_id = ?", id).Delete(&model.ItemFavorite{})
	database.DB.Where("item_id = ?", id).Delete(&model.ItemRating{})
	database.DB.Delete(&item)

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

	c.JSON(http.StatusOK, gin.H{"message": "已屏蔽，道具已打回待审核"})
}

// getModeratorStats 获取版主统计数据
func (s *Server) getModeratorStats(c *gin.Context) {
	var pendingPosts, pendingItems int64
	var totalPosts, totalItems int64
	var todayPosts, todayItems int64
	var pendingGuilds, totalGuilds int64

	// 待审核数量
	database.DB.Model(&model.Post{}).Where("review_status = ?", "pending").Count(&pendingPosts)
	database.DB.Model(&model.Item{}).Where("review_status = ?", "pending").Count(&pendingItems)
	database.DB.Model(&model.Guild{}).Where("status = ?", "pending").Count(&pendingGuilds)

	// 总数量
	database.DB.Model(&model.Post{}).Count(&totalPosts)
	database.DB.Model(&model.Item{}).Count(&totalItems)
	database.DB.Model(&model.Guild{}).Where("status = ?", "approved").Count(&totalGuilds)

	// 今日新增
	today := time.Now().Format("2006-01-02")
	database.DB.Model(&model.Post{}).Where("DATE(created_at) = ?", today).Count(&todayPosts)
	database.DB.Model(&model.Item{}).Where("DATE(created_at) = ?", today).Count(&todayItems)

	c.JSON(http.StatusOK, gin.H{
		"pending_posts":  pendingPosts,
		"pending_items":  pendingItems,
		"pending_guilds": pendingGuilds,
		"total_posts":    totalPosts,
		"total_items":    totalItems,
		"total_guilds":   totalGuilds,
		"today_posts":    todayPosts,
		"today_items":    todayItems,
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
	userMap := make(map[uint]string)
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	type GuildWithOwner struct {
		model.Guild
		OwnerName string `json:"owner_name"`
	}
	result := make([]GuildWithOwner, len(guilds))
	for i, g := range guilds {
		result[i] = GuildWithOwner{Guild: g, OwnerName: userMap[g.OwnerID]}
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	userMap := make(map[uint]string)
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	type GuildWithOwner struct {
		model.Guild
		OwnerName string `json:"owner_name"`
	}
	result := make([]GuildWithOwner, len(guilds))
	for i, g := range guilds {
		result[i] = GuildWithOwner{Guild: g, OwnerName: userMap[g.OwnerID]}
	}

	c.JSON(http.StatusOK, gin.H{"guilds": result, "total": total})
}

// ChangeGuildOwnerRequest 更改公会会长请求
type ChangeGuildOwnerRequest struct {
	NewOwnerID uint `json:"new_owner_id" binding:"required"`
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查新会长是否存在
	var newOwner model.User
	if err := database.DB.First(&newOwner, req.NewOwnerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	oldOwnerID := guild.OwnerID
	guild.OwnerID = req.NewOwnerID
	database.DB.Save(&guild)

	// 更新成员角色
	database.DB.Model(&model.GuildMember{}).
		Where("guild_id = ? AND user_id = ?", id, oldOwnerID).
		Update("role", "admin")

	// 检查新会长是否已是成员
	var member model.GuildMember
	err := database.DB.Where("guild_id = ? AND user_id = ?", id, req.NewOwnerID).First(&member).Error
	if err != nil {
		database.DB.Create(&model.GuildMember{
			GuildID:  uint(id),
			UserID:   req.NewOwnerID,
			Role:     "owner",
			JoinedAt: time.Now(),
		})
		database.DB.Model(&guild).Update("member_count", guild.MemberCount+1)
	} else {
		database.DB.Model(&member).Update("role", "owner")
	}

	c.JSON(http.StatusOK, gin.H{"message": "会长已更换", "guild": guild})
}

// deleteGuildByMod 版主删除公会
func (s *Server) deleteGuildByMod(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var guild model.Guild
	if err := database.DB.First(&guild, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "公会不存在"})
		return
	}

	// 删除关联数据
	database.DB.Where("guild_id = ?", id).Delete(&model.GuildMember{})
	database.DB.Where("guild_id = ?", id).Delete(&model.StoryGuild{})
	database.DB.Where("guild_id = ?", id).Delete(&model.Tag{})
	database.DB.Delete(&guild)

	c.JSON(http.StatusOK, gin.H{"message": "公会已删除"})
}
