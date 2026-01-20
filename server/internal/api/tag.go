package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/pkg/validator"
)

// CreateTagRequest 创建标签请求
type CreateTagRequest struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color"`
}

// UpdateTagRequest 更新标签请求
type UpdateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// listTags 获取用户的标签列表（含预设和自定义）
func (s *Server) listTags(c *gin.Context) {
	userID := c.GetUint("userID")
	category := c.Query("category") // story|item|post

	query := database.DB.Where("is_public = ? OR creator_id = ?", true, userID)

	// 按类别过滤
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var tags []model.Tag
	if err := query.Order("type ASC, usage_count DESC").Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// getPresetTags 获取预设标签（公开，支持按category过滤）
func (s *Server) getPresetTags(c *gin.Context) {
	category := c.Query("category") // story|item|post

	query := database.DB.Where("type = ? AND is_public = ?", "preset", true)
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var tags []model.Tag
	if err := query.Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// createTag 创建自定义标签
func (s *Server) createTag(c *gin.Context) {
	userID := c.GetUint("userID")

	var req CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	tag := model.Tag{
		Name:      req.Name,
		Color:     req.Color,
		Type:      "custom",
		CreatorID: userID,
		IsPublic:  false,
	}

	if err := database.DB.Create(&tag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, tag)
}

// updateTag 更新标签
func (s *Server) updateTag(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var tag model.Tag
	if err := database.DB.First(&tag, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "标签不存在"})
		return
	}

	// 只能修改自己创建的自定义标签
	if tag.CreatorID != userID || tag.Type == "preset" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改此标签"})
		return
	}

	var req UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	if req.Name != "" {
		tag.Name = req.Name
	}
	if req.Color != "" {
		tag.Color = req.Color
	}

	database.DB.Save(&tag)
	c.JSON(http.StatusOK, tag)
}

// deleteTag 删除标签
func (s *Server) deleteTag(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var tag model.Tag
	if err := database.DB.First(&tag, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "标签不存在"})
		return
	}

	// 只能删除自己创建的自定义标签
	if tag.CreatorID != userID || tag.Type == "preset" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除此标签"})
		return
	}

	// 删除关联
	database.DB.Where("tag_id = ?", id).Delete(&model.StoryTag{})
	database.DB.Delete(&tag)

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ========== 公会标签 ==========

// listGuildTags 获取公会标签
func (s *Server) listGuildTags(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查是否是公会成员
	var member model.GuildMember
	if err := database.DB.Where("guild_id = ? AND user_id = ?", guildID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "非公会成员"})
		return
	}

	var tags []model.Tag
	database.DB.Where("guild_id = ?", guildID).Order("usage_count DESC").Find(&tags)

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// createGuildTag 创建公会标签
func (s *Server) createGuildTag(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查是否有管理权限
	if !checkGuildAdmin(uint(guildID), userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	var req CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	gid := uint(guildID)
	tag := model.Tag{
		Name:      req.Name,
		Color:     req.Color,
		Type:      "guild",
		GuildID:   &gid,
		CreatorID: userID,
		IsPublic:  false,
	}

	if err := database.DB.Create(&tag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, tag)
}

// deleteGuildTag 删除公会标签
func (s *Server) deleteGuildTag(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	tagID, _ := strconv.ParseUint(c.Param("tagId"), 10, 32)

	// 检查是否有管理权限
	if !checkGuildAdmin(uint(guildID), userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	var tag model.Tag
	if err := database.DB.Where("id = ? AND guild_id = ?", tagID, guildID).First(&tag).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "标签不存在"})
		return
	}

	database.DB.Where("tag_id = ?", tagID).Delete(&model.StoryTag{})
	database.DB.Delete(&tag)

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ========== 剧情标签管理 ==========

// AddStoryTagRequest 添加剧情标签请求
type AddStoryTagRequest struct {
	TagID uint `json:"tag_id" binding:"required"`
}

// addStoryTag 为剧情添加标签
func (s *Server) addStoryTag(c *gin.Context) {
	userID := c.GetUint("userID")
	storyID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查剧情所有权
	var story model.Story
	if err := database.DB.First(&story, storyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}
	if story.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	var req AddStoryTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	// 检查标签是否存在且用户有权使用
	var tag model.Tag
	if err := database.DB.First(&tag, req.TagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "标签不存在"})
		return
	}

	// 检查标签使用权限
	if !canUseTag(userID, &tag) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权使用此标签"})
		return
	}

	// 创建关联
	storyTag := model.StoryTag{
		StoryID: uint(storyID),
		TagID:   req.TagID,
		AddedBy: userID,
	}

	if err := database.DB.Create(&storyTag).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标签已存在"})
		return
	}

	// 更新标签使用次数
	database.DB.Model(&tag).Update("usage_count", tag.UsageCount+1)

	c.JSON(http.StatusOK, gin.H{"message": "添加成功"})
}

// removeStoryTag 移除剧情标签
func (s *Server) removeStoryTag(c *gin.Context) {
	userID := c.GetUint("userID")
	storyID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	tagID, _ := strconv.ParseUint(c.Param("tagId"), 10, 32)

	// 检查剧情所有权
	var story model.Story
	if err := database.DB.First(&story, storyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}
	if story.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	result := database.DB.Where("story_id = ? AND tag_id = ?", storyID, tagID).Delete(&model.StoryTag{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "标签关联不存在"})
		return
	}

	// 更新标签使用次数
	database.DB.Model(&model.Tag{}).Where("id = ?", tagID).Update("usage_count", database.DB.Raw("usage_count - 1"))

	c.JSON(http.StatusOK, gin.H{"message": "移除成功"})
}

// getStoryTags 获取剧情的所有标签
func (s *Server) getStoryTags(c *gin.Context) {
	storyID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var storyTags []model.StoryTag
	database.DB.Where("story_id = ?", storyID).Find(&storyTags)

	tagIDs := make([]uint, len(storyTags))
	for i, st := range storyTags {
		tagIDs[i] = st.TagID
	}

	var tags []model.Tag
	if len(tagIDs) > 0 {
		database.DB.Where("id IN ?", tagIDs).Find(&tags)
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// canUseTag 检查用户是否有权使用标签
func canUseTag(userID uint, tag *model.Tag) bool {
	// 预设标签和公开标签任何人可用
	if tag.Type == "preset" || tag.IsPublic {
		return true
	}
	// 自定义标签只有创建者可用
	if tag.Type == "custom" && tag.CreatorID == userID {
		return true
	}
	// 公会标签需要是公会成员
	if tag.Type == "guild" && tag.GuildID != nil {
		var member model.GuildMember
		if err := database.DB.Where("guild_id = ? AND user_id = ?", *tag.GuildID, userID).First(&member).Error; err == nil {
			return true
		}
	}
	return false
}
