package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

// CreateCollectionRequest 创建合集请求
type CreateCollectionRequest struct {
	Name        string `json:"name" binding:"required,max=128"`
	Description string `json:"description"`
	ContentType string `json:"content_type"` // post|item|mixed
	IsPublic    bool   `json:"is_public"`
}

// UpdateCollectionRequest 更新合集请求
type UpdateCollectionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ContentType string `json:"content_type"`
	IsPublic    *bool  `json:"is_public"`
}

// AddToCollectionRequest 添加内容到合集请求
type AddToCollectionRequest struct {
	PostID    uint `json:"post_id"`
	ItemID    uint `json:"item_id"`
	SortOrder int  `json:"sort_order"`
}

// collectionListItem 合集列表项
type collectionListItem struct {
	model.Collection
	AuthorName string `json:"author_name"`
}

// collectionDetailResponse 合集详情响应
type collectionDetailResponse struct {
	model.Collection
	AuthorName  string           `json:"author_name"`
	IsFavorited bool             `json:"is_favorited"`
	Posts       []collectionPost `json:"posts,omitempty"`
	Items       []collectionItem `json:"items,omitempty"`
}

type collectionPost struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	SortOrder int    `json:"sort_order"`
}

type collectionItem struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

// listCollections 获取合集列表
func (s *Server) listCollections(c *gin.Context) {
	userID := c.GetUint("userID")
	authorID := c.Query("author_id")
	contentType := c.Query("content_type")

	query := database.DB.Model(&model.Collection{})

	// 筛选条件
	if authorID != "" {
		aid, err := strconv.ParseUint(authorID, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的author_id"})
			return
		}
		query = query.Where("author_id = ?", aid)
		// 如果不是自己的合集，只显示公开的
		if uint(aid) != userID {
			query = query.Where("is_public = ?", true)
		}
	} else {
		// 默认只显示公开合集
		query = query.Where("is_public = ?", true)
	}

	if contentType != "" {
		query = query.Where("content_type = ? OR content_type = 'mixed'", contentType)
	}

	var collections []model.Collection
	if err := query.Order("created_at DESC").Find(&collections).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取合集列表失败"})
		return
	}

	// 获取作者信息
	result := make([]collectionListItem, len(collections))
	for i, col := range collections {
		result[i].Collection = col
		var user model.User
		if err := database.DB.Select("username").First(&user, col.AuthorID).Error; err == nil {
			result[i].AuthorName = user.Username
		}
	}

	c.JSON(http.StatusOK, gin.H{"collections": result})
}

// listUserCollections 获取当前用户的合集列表（编辑器用）
func (s *Server) listUserCollections(c *gin.Context) {
	userID := c.GetUint("userID")
	contentType := c.Query("content_type")

	query := database.DB.Model(&model.Collection{}).Where("author_id = ?", userID)

	if contentType != "" {
		query = query.Where("content_type = ? OR content_type = 'mixed'", contentType)
	}

	var collections []model.Collection
	if err := query.Order("created_at DESC").Find(&collections).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取合集列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"collections": collections})
}

// createCollection 创建合集
func (s *Server) createCollection(c *gin.Context) {
	userID := c.GetUint("userID")

	var req CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 默认内容类型
	if req.ContentType == "" {
		req.ContentType = "mixed"
	}

	collection := model.Collection{
		AuthorID:    userID,
		Name:        req.Name,
		Description: req.Description,
		ContentType: req.ContentType,
		IsPublic:    req.IsPublic,
	}

	if err := database.DB.Create(&collection).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建合集失败"})
		return
	}

	c.JSON(http.StatusOK, collection)
}

// getCollection 获取合集详情
func (s *Server) getCollection(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的合集ID"})
		return
	}

	var collection model.Collection
	if err := database.DB.First(&collection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "合集不存在"})
		return
	}

	// 检查权限
	if !collection.IsPublic && collection.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此合集"})
		return
	}

	// 获取作者信息
	var user model.User
	var authorName string
	if err := database.DB.Select("username").First(&user, collection.AuthorID).Error; err == nil {
		authorName = user.Username
	}

	// 获取合集中的帖子
	var posts []collectionPost
	database.DB.Table("collection_posts").
		Select("collection_posts.sort_order, posts.id, posts.title").
		Joins("JOIN posts ON posts.id = collection_posts.post_id").
		Where("collection_posts.collection_id = ?", id).
		Order("collection_posts.sort_order ASC").
		Scan(&posts)

	// 获取合集中的作品
	var items []collectionItem
	database.DB.Table("collection_items").
		Select("collection_items.sort_order, items.id, items.name").
		Joins("JOIN items ON items.id = collection_items.item_id").
		Where("collection_items.collection_id = ?", id).
		Order("collection_items.sort_order ASC").
		Scan(&items)

	// 检查是否已收藏
	var isFavorited bool
	if userID > 0 {
		var fav model.CollectionFavorite
		if err := database.DB.Where("user_id = ? AND collection_id = ?", userID, id).First(&fav).Error; err == nil {
			isFavorited = true
		}
	}

	response := collectionDetailResponse{
		Collection:  collection,
		AuthorName:  authorName,
		IsFavorited: isFavorited,
		Posts:       posts,
		Items:       items,
	}

	c.JSON(http.StatusOK, response)
}

// updateCollection 更新合集
func (s *Server) updateCollection(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的合集ID"})
		return
	}

	var collection model.Collection
	if err := database.DB.First(&collection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "合集不存在"})
		return
	}

	// 检查权限
	if collection.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改此合集"})
		return
	}

	var req UpdateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.ContentType != "" {
		updates["content_type"] = req.ContentType
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&collection).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新合集失败"})
			return
		}
	}

	database.DB.First(&collection, id)
	c.JSON(http.StatusOK, collection)
}

// deleteCollection 删除合集
func (s *Server) deleteCollection(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的合集ID"})
		return
	}

	var collection model.Collection
	if err := database.DB.First(&collection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "合集不存在"})
		return
	}

	// 检查权限
	if collection.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除此合集"})
		return
	}

	// 删除关联
	database.DB.Where("collection_id = ?", id).Delete(&model.CollectionPost{})
	database.DB.Where("collection_id = ?", id).Delete(&model.CollectionItem{})

	// 删除合集
	if err := database.DB.Delete(&collection).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除合集失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// getCollectionPosts 获取合集中的帖子
func (s *Server) getCollectionPosts(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的合集ID"})
		return
	}

	var collection model.Collection
	if err := database.DB.First(&collection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "合集不存在"})
		return
	}

	// 检查权限
	if !collection.IsPublic && collection.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此合集"})
		return
	}

	var posts []struct {
		model.Post
		SortOrder int `json:"sort_order"`
	}
	database.DB.Table("collection_posts").
		Select("posts.*, collection_posts.sort_order").
		Joins("JOIN posts ON posts.id = collection_posts.post_id").
		Where("collection_posts.collection_id = ?", id).
		Order("collection_posts.sort_order ASC").
		Scan(&posts)

	c.JSON(http.StatusOK, gin.H{"posts": posts})
}

// addPostToCollection 添加帖子到合集
func (s *Server) addPostToCollection(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的合集ID"})
		return
	}

	var collection model.Collection
	if err := database.DB.First(&collection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "合集不存在"})
		return
	}

	// 检查权限
	if collection.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改此合集"})
		return
	}

	var req AddToCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 验证帖子存在且属于当前用户
	var post model.Post
	if err := database.DB.First(&post, req.PostID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}
	if post.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能添加自己的帖子到合集"})
		return
	}

	// 检查是否已存在
	var existing model.CollectionPost
	if err := database.DB.Where("collection_id = ? AND post_id = ?", id, req.PostID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "帖子已在合集中"})
		return
	}

	// 获取最大排序值
	var maxOrder int
	database.DB.Model(&model.CollectionPost{}).
		Where("collection_id = ?", id).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxOrder)

	cp := model.CollectionPost{
		CollectionID: uint(id),
		PostID:       req.PostID,
		SortOrder:    maxOrder + 1,
	}

	if err := database.DB.Create(&cp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加失败"})
		return
	}

	// 更新合集计数
	database.DB.Model(&collection).Update("item_count", collection.ItemCount+1)

	c.JSON(http.StatusOK, gin.H{"message": "添加成功"})
}

// removePostFromCollection 从合集移除帖子
func (s *Server) removePostFromCollection(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的合集ID"})
		return
	}
	postID, err := strconv.ParseUint(c.Param("postId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子ID"})
		return
	}

	var collection model.Collection
	if err := database.DB.First(&collection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "合集不存在"})
		return
	}

	// 检查权限
	if collection.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改此合集"})
		return
	}

	result := database.DB.Where("collection_id = ? AND post_id = ?", id, postID).Delete(&model.CollectionPost{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不在合集中"})
		return
	}

	// 更新合集计数
	if collection.ItemCount > 0 {
		database.DB.Model(&collection).Update("item_count", collection.ItemCount-1)
	}

	c.JSON(http.StatusOK, gin.H{"message": "移除成功"})
}

// getCollectionItems 获取合集中的作品
func (s *Server) getCollectionItems(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的合集ID"})
		return
	}

	var collection model.Collection
	if err := database.DB.First(&collection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "合集不存在"})
		return
	}

	// 检查权限
	if !collection.IsPublic && collection.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此合集"})
		return
	}

	var items []struct {
		model.Item
		SortOrder int `json:"sort_order"`
	}
	database.DB.Table("collection_items").
		Select("items.*, collection_items.sort_order").
		Joins("JOIN items ON items.id = collection_items.item_id").
		Where("collection_items.collection_id = ?", id).
		Order("collection_items.sort_order ASC").
		Scan(&items)

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// addItemToCollection 添加作品到合集
func (s *Server) addItemToCollection(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的合集ID"})
		return
	}

	var collection model.Collection
	if err := database.DB.First(&collection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "合集不存在"})
		return
	}

	// 检查权限
	if collection.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改此合集"})
		return
	}

	var req AddToCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 验证作品存在且属于当前用户
	var item model.Item
	if err := database.DB.First(&item, req.ItemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	if item.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能添加自己的作品到合集"})
		return
	}

	// 检查是否已存在
	var existing model.CollectionItem
	if err := database.DB.Where("collection_id = ? AND item_id = ?", id, req.ItemID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "作品已在合集中"})
		return
	}

	// 获取最大排序值
	var maxOrder int
	database.DB.Model(&model.CollectionItem{}).
		Where("collection_id = ?", id).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxOrder)

	ci := model.CollectionItem{
		CollectionID: uint(id),
		ItemID:       req.ItemID,
		SortOrder:    maxOrder + 1,
	}

	if err := database.DB.Create(&ci).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加失败"})
		return
	}

	// 更新合集计数
	database.DB.Model(&collection).Update("item_count", collection.ItemCount+1)

	c.JSON(http.StatusOK, gin.H{"message": "添加成功"})
}

// removeItemFromCollection 从合集移除作品
func (s *Server) removeItemFromCollection(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的合集ID"})
		return
	}
	itemID, err := strconv.ParseUint(c.Param("itemId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品ID"})
		return
	}

	var collection model.Collection
	if err := database.DB.First(&collection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "合集不存在"})
		return
	}

	// 检查权限
	if collection.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改此合集"})
		return
	}

	result := database.DB.Where("collection_id = ? AND item_id = ?", id, itemID).Delete(&model.CollectionItem{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不在合集中"})
		return
	}

	// 更新合集计数
	if collection.ItemCount > 0 {
		database.DB.Model(&collection).Update("item_count", collection.ItemCount-1)
	}

	c.JSON(http.StatusOK, gin.H{"message": "移除成功"})
}

// getPostCollection 获取帖子所属的合集信息
func (s *Server) getPostCollection(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子ID"})
		return
	}

	var cp model.CollectionPost
	if err := database.DB.Where("post_id = ?", postID).First(&cp).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"collection": nil})
		return
	}

	var collection model.Collection
	if err := database.DB.First(&collection, cp.CollectionID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"collection": nil})
		return
	}

	// 获取合集中的所有帖子ID（用于导航）
	var postIDs []uint
	database.DB.Model(&model.CollectionPost{}).
		Where("collection_id = ?", collection.ID).
		Order("sort_order ASC").
		Pluck("post_id", &postIDs)

	// 找到当前帖子的位置
	currentIndex := -1
	for i, pid := range postIDs {
		if pid == uint(postID) {
			currentIndex = i
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"collection":    collection,
		"post_ids":      postIDs,
		"current_index": currentIndex,
	})
}

// getItemCollection 获取作品所属的合集信息
func (s *Server) getItemCollection(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品ID"})
		return
	}

	var ci model.CollectionItem
	if err := database.DB.Where("item_id = ?", itemID).First(&ci).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"collection": nil})
		return
	}

	var collection model.Collection
	if err := database.DB.First(&collection, ci.CollectionID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"collection": nil})
		return
	}

	// 获取合集中的所有作品ID（用于导航）
	var itemIDs []uint
	database.DB.Model(&model.CollectionItem{}).
		Where("collection_id = ?", collection.ID).
		Order("sort_order ASC").
		Pluck("item_id", &itemIDs)

	// 找到当前作品的位置
	currentIndex := -1
	for i, iid := range itemIDs {
		if iid == uint(itemID) {
			currentIndex = i
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"collection":    collection,
		"item_ids":      itemIDs,
		"current_index": currentIndex,
	})
}

// favoriteCollection 收藏合集
func (s *Server) favoriteCollection(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的合集ID"})
		return
	}

	var collection model.Collection
	if err := database.DB.First(&collection, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "合集不存在"})
		return
	}

	// 检查是否已收藏
	var existing model.CollectionFavorite
	if err := database.DB.Where("user_id = ? AND collection_id = ?", userID, id).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "已收藏此合集"})
		return
	}

	favorite := model.CollectionFavorite{
		UserID:       userID,
		CollectionID: uint(id),
	}

	if err := database.DB.Create(&favorite).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "收藏失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "收藏成功"})
}

// unfavoriteCollection 取消收藏合集
func (s *Server) unfavoriteCollection(c *gin.Context) {
	userID := c.GetUint("userID")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的合集ID"})
		return
	}

	result := database.DB.Where("user_id = ? AND collection_id = ?", userID, id).Delete(&model.CollectionFavorite{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未收藏此合集"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "取消收藏成功"})
}

// listMyCollectionFavorites 获取用户收藏的合集列表
func (s *Server) listMyCollectionFavorites(c *gin.Context) {
	userID := c.GetUint("userID")

	var favorites []model.CollectionFavorite
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&favorites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取收藏列表失败"})
		return
	}

	if len(favorites) == 0 {
		c.JSON(http.StatusOK, gin.H{"collections": []collectionListItem{}})
		return
	}

	// 获取合集ID列表
	collectionIDs := make([]uint, len(favorites))
	for i, f := range favorites {
		collectionIDs[i] = f.CollectionID
	}

	// 获取合集详情
	var collections []model.Collection
	if err := database.DB.Where("id IN ?", collectionIDs).Find(&collections).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取合集详情失败"})
		return
	}

	// 构建结果
	result := make([]collectionListItem, len(collections))
	for i, col := range collections {
		result[i].Collection = col
		var user model.User
		if err := database.DB.Select("username").First(&user, col.AuthorID).Error; err == nil {
			result[i].AuthorName = user.Username
		}
	}

	c.JSON(http.StatusOK, gin.H{"collections": result})
}
