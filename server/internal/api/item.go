package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
	"github.com/rpbox/server/pkg/validator"
	"gorm.io/gorm/clause"
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

	// 按作者名称筛选
	if authorName := c.Query("author_name"); authorName != "" {
		query = query.Joins("JOIN users ON users.id = items.author_id").
			Where("users.username LIKE ?", "%"+authorName+"%")
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
		query = query.Order("items.downloads " + order)
	case "rating":
		query = query.Order("items.rating " + order)
	case "created_at":
		query = query.Order("items.created_at " + order)
	default:
		query = query.Order("items.created_at desc")
	}

	// 分页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize

	var total int64
	query.Count(&total)

	// 列表查询排除大字段（import_code, raw_data, detail_content, preview_image）以提高性能
	// preview_image 通过独立的图片 API 访问
	if err := query.Select("items.id, items.author_id, items.name, items.type, items.icon, items.description, items.downloads, items.rating, items.rating_count, items.like_count, items.favorite_count, items.requires_permission, items.status, items.review_status, items.preview_image_updated_at, items.created_at, items.updated_at").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取有预览图的 item ID 列表
	var itemIDs []uint
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	var itemsWithPreview []uint
	if len(itemIDs) > 0 {
		database.DB.Model(&model.Item{}).
			Select("id").
			Where("id IN ? AND preview_image IS NOT NULL AND preview_image != ''", itemIDs).
			Pluck("id", &itemsWithPreview)
	}
	hasPreviewMap := make(map[uint]bool)
	for _, id := range itemsWithPreview {
		hasPreviewMap[id] = true
	}

	// 批量获取作者信息
	var authorIDs []uint
	for _, item := range items {
		authorIDs = append(authorIDs, item.AuthorID)
	}

	var authors []model.User
	if len(authorIDs) > 0 {
		database.DB.Select("id", "username", "avatar", "role").Where("id IN ?", authorIDs).Find(&authors)
	}

	// 创建作者ID到用户信息的映射
	authorMap := make(map[uint]model.User)
	for _, author := range authors {
		authorMap[author.ID] = author
	}

	// 构建包含作者信息的响应
	type ItemWithAuthor struct {
		model.Item
		AuthorUsername  string `json:"author_username"`
		AuthorAvatar    string `json:"author_avatar"`
		AuthorRole      string `json:"author_role"`
		PreviewImageURL string `json:"preview_image_url"`
	}

	var result []ItemWithAuthor
	for _, item := range items {
		author := authorMap[item.AuthorID]
		// 构造缩略图 URL：宽度 400，质量 80
		// 只有确认有预览图或是画作类型才返回 URL
		previewURL := ""
		if hasPreviewMap[item.ID] || item.Type == "artwork" {
			previewURL = fmt.Sprintf("/api/v1/images/item-preview/%d?w=400&q=80", item.ID)
		}
		if item.PreviewImageUpdatedAt == nil && hasPreviewMap[item.ID] {
			t := item.UpdatedAt
			item.PreviewImageUpdatedAt = &t
		}
		result = append(result, ItemWithAuthor{
			Item:            item,
			AuthorUsername:  author.Username,
			AuthorAvatar:    author.Avatar,
			AuthorRole:      author.Role,
			PreviewImageURL: previewURL,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items": result,
			"total": total,
			"page":  page,
		},
	})
}

func listUserItemsByRelation(c *gin.Context, joinTable, orderColumn string) {
	userID := c.GetUint("user_id")
	var items []model.Item

	query := database.DB.Model(&model.Item{}).
		Joins("JOIN "+joinTable+" ON "+joinTable+".item_id = items.id").
		Where(joinTable+".user_id = ?", userID).
		Order(joinTable + "." + orderColumn + " DESC")

	if err := query.Select("items.id, items.author_id, items.name, items.type, items.icon, items.description, items.downloads, items.rating, items.rating_count, items.like_count, items.favorite_count, items.requires_permission, items.status, items.review_status, items.preview_image_updated_at, items.created_at, items.updated_at").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 权限过滤：作者可见所有状态，其他人只看已发布且审核通过
	filtered := make([]model.Item, 0, len(items))
	for _, item := range items {
		if item.AuthorID == userID || (item.Status == "published" && item.ReviewStatus == "approved") {
			filtered = append(filtered, item)
		}
	}

	// 获取有预览图的 item ID 列表
	var itemIDs []uint
	for _, item := range filtered {
		itemIDs = append(itemIDs, item.ID)
	}
	var itemsWithPreview []uint
	if len(itemIDs) > 0 {
		database.DB.Model(&model.Item{}).
			Select("id").
			Where("id IN ? AND preview_image IS NOT NULL AND preview_image != ''", itemIDs).
			Pluck("id", &itemsWithPreview)
	}
	hasPreviewMap := make(map[uint]bool)
	for _, id := range itemsWithPreview {
		hasPreviewMap[id] = true
	}

	// 批量获取作者信息
	var authorIDs []uint
	for _, item := range filtered {
		authorIDs = append(authorIDs, item.AuthorID)
	}

	var authors []model.User
	if len(authorIDs) > 0 {
		database.DB.Select("id", "username", "avatar", "role").Where("id IN ?", authorIDs).Find(&authors)
	}

	authorMap := make(map[uint]model.User)
	for _, author := range authors {
		authorMap[author.ID] = author
	}

	type ItemWithAuthor struct {
		model.Item
		AuthorUsername  string `json:"author_username"`
		AuthorAvatar    string `json:"author_avatar"`
		AuthorRole      string `json:"author_role"`
		PreviewImageURL string `json:"preview_image_url"`
	}

	result := make([]ItemWithAuthor, 0, len(filtered))
	for _, item := range filtered {
		author := authorMap[item.AuthorID]
		previewURL := ""
		if hasPreviewMap[item.ID] || item.Type == "artwork" {
			previewURL = fmt.Sprintf("/api/v1/images/item-preview/%d?w=400&q=80", item.ID)
		}
		if item.PreviewImageUpdatedAt == nil && hasPreviewMap[item.ID] {
			t := item.UpdatedAt
			item.PreviewImageUpdatedAt = &t
		}
		result = append(result, ItemWithAuthor{
			Item:            item,
			AuthorUsername:  author.Username,
			AuthorAvatar:    author.Avatar,
			AuthorRole:      author.Role,
			PreviewImageURL: previewURL,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items": result,
			"total": len(result),
		},
	})
}

// listMyItemFavorites 获取我收藏的道具
func (s *Server) listMyItemFavorites(c *gin.Context) {
	listUserItemsByRelation(c, "item_favorites", "created_at")
}

// listMyItemLikes 获取我点赞的道具
func (s *Server) listMyItemLikes(c *gin.Context) {
	listUserItemsByRelation(c, "item_likes", "created_at")
}

// listMyItemViews 获取我浏览的道具
func (s *Server) listMyItemViews(c *gin.Context) {
	listUserItemsByRelation(c, "item_views", "updated_at")
}

// createItem 创建道具
func (s *Server) createItem(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Name               string `json:"name" binding:"required"`
		Type               string `json:"type" binding:"required"`
		Icon               string `json:"icon"`
		PreviewImage       string `json:"preview_image"`
		Description        string `json:"description"`
		DetailContent      string `json:"detail_content"`
		ImportCode         string `json:"import_code"`
		RawData            string `json:"raw_data"`
		RequiresPermission bool   `json:"requires_permission"`
		EnableWatermark    bool   `json:"enable_watermark"`
		TagIDs             []uint `json:"tag_ids"`
		Status             string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	// 验证 ImportCode 大小（最大 10MB）
	if len(req.ImportCode) > 10<<20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Import code too large, max 10MB"})
		return
	}

	// 验证类型：item（道具）、document（文档）、campaign（剧本）、artwork（画作）
	validTypes := map[string]bool{"item": true, "document": true, "campaign": true, "artwork": true}
	if !validTypes[req.Type] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item type"})
		return
	}

	// 对于非画作类型，import_code是必填的
	if req.Type != "artwork" && req.ImportCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "导入代码是必填项"})
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
		EnableWatermark:    req.EnableWatermark,
		Status:             req.Status,
	}
	if req.PreviewImage != "" {
		now := time.Now()
		item.PreviewImageUpdatedAt = &now
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

	// 记录浏览历史
	if userID != 0 {
		view := model.ItemView{
			ItemID: item.ID,
			UserID: userID,
		}
		database.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "item_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
		}).Create(&view)
	}

	// 检查当前用户是否点赞和收藏
	var liked, favorited bool
	var existingLike model.ItemLike
	if err := database.DB.Where("item_id = ? AND user_id = ?", item.ID, userID).First(&existingLike).Error; err == nil {
		liked = true
	}
	var existingFavorite model.ItemFavorite
	if err := database.DB.Where("item_id = ? AND user_id = ?", item.ID, userID).First(&existingFavorite).Error; err == nil {
		favorited = true
	}

	response := gin.H{
		"item":      item,
		"author":    author,
		"tags":      tags,
		"liked":     liked,
		"favorited": favorited,
	}

	// 如果是画作类型，获取图片列表
	if item.Type == "artwork" {
		var images []model.ItemImage
		database.DB.Where("item_id = ?", item.ID).Order("sort_order asc").Find(&images)

		type ImageResponse struct {
			ID        uint   `json:"id"`
			ImageURL  string `json:"image_url"`
			SortOrder int    `json:"sort_order"`
		}

		var imageList []ImageResponse
		for _, img := range images {
			imageList = append(imageList, ImageResponse{
				ID:        img.ID,
				ImageURL:  buildPublicURL(c, "/api/v1/items/"+id+"/images/"+strconv.Itoa(int(img.ID))),
				SortOrder: img.SortOrder,
			})
		}
		response["images"] = imageList
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
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
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
		now := time.Now()
		item.PreviewImageUpdatedAt = &now
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
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	// 验证评论长度：带评分时至少10字
	if req.Rating > 0 && len([]rune(req.Content)) < 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "带评分的评价至少需要10个字符"})
		return
	}

	// 检查道具是否存在
	var item model.Item
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// 检查用户是否已发表过评分评论（只在提交评分时检查）
	if req.Rating > 0 {
		var existingRatingComment model.ItemComment
		if err := database.DB.Where("item_id = ? AND user_id = ? AND rating > 0", id, userID).First(&existingRatingComment).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "您已经对此道具发表过评分评价，每个用户只能发表一次评分"})
			return
		}
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

	// 重新计算平均评分（只计算带评分的评论，rating > 0）
	var avgRating float64
	var count int64
	database.DB.Model(&model.ItemComment{}).Where("item_id = ? AND rating > 0", id).Count(&count)
	database.DB.Model(&model.ItemComment{}).Where("item_id = ? AND rating > 0", id).Select("AVG(rating)").Scan(&avgRating)

	database.DB.Model(&item).Updates(map[string]interface{}{
		"rating":       avgRating,
		"rating_count": count,
	})

	// 创建通知（不给自己发通知）
	if item.AuthorID != userID {
		// 构建通知内容：包含道具名称和评论片段
		commentPreview := req.Content
		if len([]rune(commentPreview)) > 50 {
			commentPreview = string([]rune(commentPreview)[:50]) + "..."
		}
		content := "评论了你的道具《" + item.Name + "》：" + commentPreview

		notification := model.Notification{
			UserID:     item.AuthorID,
			Type:       "item_comment",
			ActorID:    &userID,
			TargetType: "item",
			TargetID:   uint(itemID),
			Content:    content,
		}
		service.CreateNotification(&notification)
	}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
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
		"code":    0,
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

	// 创建通知（不给自己发通知）
	if item.AuthorID != userID {
		content := "点赞了你的道具《" + item.Name + "》"

		notification := model.Notification{
			UserID:     item.AuthorID,
			Type:       "item_like",
			ActorID:    &userID,
			TargetType: "item",
			TargetID:   uint(itemID),
			Content:    content,
		}
		service.CreateNotification(&notification)
	}

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
