package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

// listItems 获取道具列表
func (s *Server) listItems(c *gin.Context) {
	userID := c.GetUint("user_id")
	var items []model.Item
	query := database.DB.Model(&model.Item{})

	// 类型筛选
	if itemType := c.Query("type"); itemType != "" {
		query = query.Where("type = ?", itemType)
	}

	// 作者筛选
	authorID := c.Query("author_id")

	// 权限控制：查看自己的道具 vs 查看他人的道具
	if authorID != "" && authorID == strconv.Itoa(int(userID)) {
		// 查看自己的道具：可以看到所有状态
		query = query.Where("author_id = ?", authorID)
		// 如果指定了状态，则过滤
		if status := c.Query("status"); status != "" && status != "all" {
			query = query.Where("status = ?", status)
		}
	} else {
		// 查看他人道具：只能看到已发布且审核通过的
		query = query.Where("status = ?", "published")
		query = query.Where("review_status = ?", "approved")
		if authorID != "" {
			query = query.Where("author_id = ?", authorID)
		}
	}

	// 搜索
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 标签筛选
	if tagID := c.Query("tag_id"); tagID != "" {
		query = query.Joins("JOIN item_tags ON item_tags.item_id = items.id").
			Where("item_tags.tag_id = ?", tagID)
	}

	// 排序
	sortBy := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")
	switch sortBy {
	case "downloads":
		query = query.Order("downloads " + order)
	case "rating":
		query = query.Order("rating " + order)
	case "created_at":
		query = query.Order("created_at " + order)
	default:
		query = query.Order("created_at desc")
	}

	// 分页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize

	var total int64
	query.Count(&total)

	// 列表查询排除大字段（import_code, raw_data, detail_content）以提高性能
	if err := query.Select("id, author_id, name, type, icon, preview_image, description, downloads, rating, rating_count, like_count, favorite_count, requires_permission, status, review_status, created_at, updated_at").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items": items,
			"total": total,
			"page":  page,
		},
	})
}

// createItem 创建道具
func (s *Server) createItem(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Name               string   `json:"name" binding:"required"`
		Type               string   `json:"type" binding:"required"`
		Icon               string   `json:"icon"`
		PreviewImage       string   `json:"preview_image"`
		Description        string   `json:"description"`
		DetailContent      string   `json:"detail_content"`
		ImportCode         string   `json:"import_code" binding:"required"`
		RawData            string   `json:"raw_data"`
		RequiresPermission bool     `json:"requires_permission"`
		TagIDs             []uint   `json:"tag_ids"`
		Status             string   `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 ImportCode 大小（最大 10MB）
	if len(req.ImportCode) > 10<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Import code too large, max 10MB"})
		return
	}

	// 验证类型：item（道具）、script（剧本）
	if req.Type != "item" && req.Type != "script" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item type, must be 'item' or 'script'"})
		return
	}

	// 默认状态为草稿
	if req.Status == "" {
		req.Status = "draft"
	}

	item := model.Item{
		AuthorID:           userID,
		Name:               req.Name,
		Type:               req.Type,
		Icon:               req.Icon,
		PreviewImage:       req.PreviewImage,
		Description:        req.Description,
		DetailContent:      req.DetailContent,
		ImportCode:         req.ImportCode,
		RawData:            req.RawData,
		RequiresPermission: req.RequiresPermission,
		Status:             req.Status,
	}

	// 设置审核状态：版主/管理员自动通过，普通用户需要审核
	// 草稿状态不需要审核
	isModerator := checkModerator(userID)
	if req.Status == "published" {
		if isModerator {
			item.Status = "published"
			item.ReviewStatus = "approved"
		} else {
			item.Status = "pending"
			item.ReviewStatus = "pending"
		}
	} else {
		// 草稿状态，不需要审核
		item.ReviewStatus = ""
	}

	if err := database.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 添加标签
	for _, tagID := range req.TagIDs {
		itemTag := model.ItemTag{
			ItemID:  item.ID,
			TagID:   tagID,
			AddedBy: userID,
		}
		database.DB.Create(&itemTag)
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"data": item,
	})
}

// getItem 获取道具详情
func (s *Server) getItem(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var item model.Item

	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "道具不存在"})
		return
	}

	// 获取作者信息
	var author model.User
	database.DB.Select("id", "username").First(&author, item.AuthorID)

	// 获取标签
	var tags []model.Tag
	database.DB.Joins("JOIN item_tags ON item_tags.tag_id = tags.id").
		Where("item_tags.item_id = ?", item.ID).
		Find(&tags)

	response := gin.H{
		"item":   item,
		"author": author,
		"tags":   tags,
	}

	// 如果是作者，返回待审核编辑信息
	if item.AuthorID == userID {
		var pendingEdit model.ItemPendingEdit
		if err := database.DB.Where("item_id = ?", item.ID).First(&pendingEdit).Error; err == nil {
			response["pending_edit"] = pendingEdit
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": response,
	})
}

// updateItem 更新道具
func (s *Server) updateItem(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "道具不存在"})
		return
	}

	// 验证权限：只有作者可以编辑
	if item.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权编辑此道具"})
		return
	}

	var req struct {
		Name               string `json:"name"`
		Description        string `json:"description"`
		DetailContent      string `json:"detail_content"`
		Icon               string `json:"icon"`
		PreviewImage       string `json:"preview_image"`
		ImportCode         string `json:"import_code"`
		RawData            string `json:"raw_data"`
		RequiresPermission *bool  `json:"requires_permission"`
		TagIDs             []uint `json:"tag_ids"`
		Status             string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 ImportCode 大小（最大 10MB）
	if len(req.ImportCode) > 10<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Import code too large, max 10MB"})
		return
	}

	isModerator := checkModerator(userID)

	// 如果道具已发布且审核通过，普通用户编辑需要创建待审核记录
	if item.Status == "published" && item.ReviewStatus == "approved" && !isModerator {
		// 创建或更新待审核编辑记录
		var pendingEdit model.ItemPendingEdit
		err := database.DB.Where("item_id = ?", item.ID).First(&pendingEdit).Error

		if err != nil {
			// 创建新的待审核编辑
			pendingEdit = model.ItemPendingEdit{
				ItemID:       item.ID,
				AuthorID:     userID,
				Name:         item.Name,
				Icon:         item.Icon,
				Description:  item.Description,
				ImportCode:   item.ImportCode,
				ReviewStatus: "pending",
			}
		}

		// 更新待审核编辑的字段
		if req.Name != "" {
			pendingEdit.Name = req.Name
		}
		if req.Description != "" {
			pendingEdit.Description = req.Description
		}
		if req.Icon != "" {
			pendingEdit.Icon = req.Icon
		}
		if req.ImportCode != "" {
			pendingEdit.ImportCode = req.ImportCode
		}
		pendingEdit.ReviewStatus = "pending"
		pendingEdit.ReviewerID = nil
		pendingEdit.ReviewComment = ""
		pendingEdit.ReviewedAt = nil

		if err != nil {
			database.DB.Create(&pendingEdit)
		} else {
			database.DB.Save(&pendingEdit)
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "编辑已提交审核，原道具保持不变",
			"data":    item,
			"pending": pendingEdit,
		})
		return
	}

	// 版主/管理员或未发布道具：直接修改
	if req.Name != "" {
		item.Name = req.Name
	}
	if req.Description != "" {
		item.Description = req.Description
	}
	if req.DetailContent != "" {
		item.DetailContent = req.DetailContent
	}
	if req.Icon != "" {
		item.Icon = req.Icon
	}
	if req.PreviewImage != "" {
		item.PreviewImage = req.PreviewImage
	}
	if req.ImportCode != "" {
		item.ImportCode = req.ImportCode
	}
	if req.RawData != "" {
		item.RawData = req.RawData
	}
	if req.RequiresPermission != nil {
		item.RequiresPermission = *req.RequiresPermission
	}

	// 处理标签更新
	if req.TagIDs != nil {
		// 删除旧标签
		database.DB.Where("item_id = ?", item.ID).Delete(&model.ItemTag{})
		// 添加新标签
		for _, tagID := range req.TagIDs {
			itemTag := model.ItemTag{
				ItemID:  item.ID,
				TagID:   tagID,
				AddedBy: userID,
			}
			database.DB.Create(&itemTag)
		}
	}

	// 处理状态变更
	if req.Status == "published" {
		if isModerator {
			item.Status = "published"
			item.ReviewStatus = "approved"
		} else {
			item.Status = "pending"
			item.ReviewStatus = "pending"
		}
	} else if req.Status == "draft" {
		item.Status = "draft"
		item.ReviewStatus = ""
	}

	if err := database.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": item,
	})
}

// deleteItem 删除道具
func (s *Server) deleteItem(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "道具不存在"})
		return
	}

	// 验证权限：只有作者可以删除
	if item.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除此道具"})
		return
	}

	// 删除关联数据
	database.DB.Where("item_id = ?", id).Delete(&model.ItemTag{})
	database.DB.Where("item_id = ?", id).Delete(&model.ItemComment{})
	database.DB.Where("item_id = ?", id).Delete(&model.ItemLike{})
	database.DB.Where("item_id = ?", id).Delete(&model.ItemFavorite{})
	database.DB.Where("item_id = ?", id).Delete(&model.ItemRating{})

	// 删除道具
	if err := database.DB.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// downloadItem 记录下载
func (s *Server) downloadItem(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")
	var item model.Item

	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// 检查用户是否已经下载过（每用户每道具最多贡献1次下载量）
	var existingDownload model.ItemDownload
	err := database.DB.Where("item_id = ? AND user_id = ?", item.ID, userID).First(&existingDownload).Error

	if err != nil {
		// 用户首次下载，记录并增加下载次数
		download := model.ItemDownload{
			ItemID: item.ID,
			UserID: userID,
		}
		database.DB.Create(&download)
		database.DB.Model(&item).Update("downloads", item.Downloads+1)
	}
	// 如果已下载过，不增加下载次数，但仍返回导入代码

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"import_code": item.ImportCode,
		},
	})
}

// rateItem 评分
func (s *Server) rateItem(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var req struct {
		Rating int `json:"rating" binding:"required,min=1,max=5"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查道具是否存在
	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// 创建或更新评分
	var rating model.ItemRating
	result := database.DB.Where("item_id = ? AND user_id = ?", id, userID).First(&rating)

	if result.Error != nil {
		// 创建新评分
		rating = model.ItemRating{
			ItemID: item.ID,
			UserID: userID,
			Rating: req.Rating,
		}
		database.DB.Create(&rating)
	} else {
		// 更新评分
		database.DB.Model(&rating).Update("rating", req.Rating)
	}

	// 重新计算平均评分
	var avgRating float64
	var count int64
	database.DB.Model(&model.ItemRating{}).Where("item_id = ?", id).Count(&count)
	database.DB.Model(&model.ItemRating{}).Where("item_id = ?", id).Select("AVG(rating)").Scan(&avgRating)

	database.DB.Model(&item).Updates(map[string]interface{}{
		"rating":       avgRating,
		"rating_count": count,
	})

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"rating":       avgRating,
			"rating_count": count,
		},
	})
}

// getItemComments 获取道具评论
func (s *Server) getItemComments(c *gin.Context) {
	id := c.Param("id")
	var comments []model.ItemComment

	if err := database.DB.Where("item_id = ?", id).Order("created_at desc").Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取评论者信息
	type CommentWithUser struct {
		model.ItemComment
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
	}

	var result []CommentWithUser
	for _, comment := range comments {
		var user model.User
		database.DB.Select("username, avatar").First(&user, comment.UserID)
		result = append(result, CommentWithUser{
			ItemComment: comment,
			Username:    user.Username,
			Avatar:      user.Avatar,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}

// addItemComment 添加评论（带评分）
func (s *Server) addItemComment(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var req struct {
		Rating  int    `json:"rating" binding:"min=0,max=5"`
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证评论长度：带评分时至少30字
	if req.Rating > 0 && len([]rune(req.Content)) < 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "带评分的评价至少需要30个字符"})
		return
	}

	// 检查道具是否存在
	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// 检查用户是否已评论过
	var existingComment model.ItemComment
	if err := database.DB.Where("item_id = ? AND user_id = ?", id, userID).First(&existingComment).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "您已经评论过此道具"})
		return
	}

	itemID, _ := strconv.ParseUint(id, 10, 32)
	comment := model.ItemComment{
		ItemID:  uint(itemID),
		UserID:  userID,
		Rating:  req.Rating,
		Content: req.Content,
	}

	if err := database.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新计算平均评分（基于评论中的评分）
	var avgRating float64
	var count int64
	database.DB.Model(&model.ItemComment{}).Where("item_id = ?", id).Count(&count)
	database.DB.Model(&model.ItemComment{}).Where("item_id = ?", id).Select("AVG(rating)").Scan(&avgRating)

	database.DB.Model(&item).Updates(map[string]interface{}{
		"rating":       avgRating,
		"rating_count": count,
	})

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"data": comment,
	})
}

// getItemTags 获取道具标签
func (s *Server) getItemTags(c *gin.Context) {
	id := c.Param("id")
	var tags []model.Tag

	if err := database.DB.Joins("JOIN item_tags ON item_tags.tag_id = tags.id").
		Where("item_tags.item_id = ?", id).
		Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": tags,
	})
}

// addItemTag 添加道具标签
func (s *Server) addItemTag(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var req struct {
		TagID uint `json:"tag_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查道具是否存在
	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// 检查是否已存在
	var existing model.ItemTag
	if err := database.DB.Where("item_id = ? AND tag_id = ?", id, req.TagID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tag already exists"})
		return
	}

	itemID, _ := strconv.ParseUint(id, 10, 32)
	itemTag := model.ItemTag{
		ItemID:  uint(itemID),
		TagID:   req.TagID,
		AddedBy: userID,
	}

	if err := database.DB.Create(&itemTag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 0,
		"data": itemTag,
	})
}

// removeItemTag 移除道具标签
func (s *Server) removeItemTag(c *gin.Context) {
	id := c.Param("id")
	tagID := c.Param("tagId")

	if err := database.DB.Where("item_id = ? AND tag_id = ?", id, tagID).Delete(&model.ItemTag{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "Tag removed",
	})
}

// likeItem 点赞道具
func (s *Server) likeItem(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	// 检查道具是否存在
	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// 检查是否已点赞
	var existing model.ItemLike
	if err := database.DB.Where("item_id = ? AND user_id = ?", id, userID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Already liked"})
		return
	}

	itemID, _ := strconv.ParseUint(id, 10, 32)
	like := model.ItemLike{
		ItemID: uint(itemID),
		UserID: userID,
	}

	if err := database.DB.Create(&like).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新点赞数
	database.DB.Model(&item).Update("like_count", item.LikeCount+1)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"like_count": item.LikeCount + 1},
	})
}

// unlikeItem 取消点赞
func (s *Server) unlikeItem(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	// 检查道具是否存在
	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	result := database.DB.Where("item_id = ? AND user_id = ?", id, userID).Delete(&model.ItemLike{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not liked yet"})
		return
	}

	// 更新点赞数
	newCount := item.LikeCount - 1
	if newCount < 0 {
		newCount = 0
	}
	database.DB.Model(&item).Update("like_count", newCount)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"like_count": newCount},
	})
}

// favoriteItem 收藏道具
func (s *Server) favoriteItem(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	var existing model.ItemFavorite
	if err := database.DB.Where("item_id = ? AND user_id = ?", id, userID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Already favorited"})
		return
	}

	itemID, _ := strconv.ParseUint(id, 10, 32)
	fav := model.ItemFavorite{
		ItemID: uint(itemID),
		UserID: userID,
	}

	if err := database.DB.Create(&fav).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	database.DB.Model(&item).Update("favorite_count", item.FavoriteCount+1)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"favorite_count": item.FavoriteCount + 1},
	})
}

// unfavoriteItem 取消收藏
func (s *Server) unfavoriteItem(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	result := database.DB.Where("item_id = ? AND user_id = ?", id, userID).Delete(&model.ItemFavorite{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not favorited yet"})
		return
	}

	newCount := item.FavoriteCount - 1
	if newCount < 0 {
		newCount = 0
	}
	database.DB.Model(&item).Update("favorite_count", newCount)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"favorite_count": newCount},
	})
}
