package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

// CreateStoryRequest 创建剧情请求
type CreateStoryRequest struct {
	Title        string   `json:"title" binding:"required"`
	Description  string   `json:"description"`
	Participants []string `json:"participants"`
	Tags         []string `json:"tags"`
	StartTime    string   `json:"start_time"`
	EndTime      string   `json:"end_time"`
}

// CreateStoryEntryRequest 创建剧情条目请求
type CreateStoryEntryRequest struct {
	SourceID  string `json:"source_id"`
	Type      string `json:"type"`
	Speaker   string `json:"speaker"`
	Content   string `json:"content" binding:"required"`
	Channel   string `json:"channel"`
	Timestamp string `json:"timestamp"`
	// 角色信息
	RefID    string `json:"ref_id"`    // TRP3 ref ID
	GameID   string `json:"game_id"`   // 游戏内ID
	TRP3Data string `json:"trp3_data"` // 完整TRP3 profile JSON
	IsNPC    bool   `json:"is_npc"`    // 是否NPC
}

func (s *Server) listStories(c *gin.Context) {
	userID := c.GetUint("userID")

	// 构建查询
	// 如果指定了guild_id，则查询公会剧情（需要检查访问权限）
	// 否则只查询当前用户的剧情（私有访问）
	query := database.DB.Model(&model.Story{})
	if c.Query("guild_id") == "" {
		query = query.Where("user_id = ?", userID)
	}

	// 标签筛选
	if tagIDs := c.Query("tag_ids"); tagIDs != "" {
		ids := strings.Split(tagIDs, ",")
		var storyIDs []uint
		database.DB.Model(&model.StoryTag{}).
			Where("tag_id IN ?", ids).
			Distinct("story_id").
			Pluck("story_id", &storyIDs)
		if len(storyIDs) > 0 {
			query = query.Where("id IN ?", storyIDs)
		} else {
			// 没有匹配的剧情
			c.JSON(http.StatusOK, gin.H{"stories": []model.Story{}})
			return
		}
	}

	// 公会筛选
	if guildID := c.Query("guild_id"); guildID != "" {
		guildIDNum, _ := strconv.ParseUint(guildID, 10, 32)

		// 检查公会内容访问权限
		canAccess, _ := checkGuildContentAccess(uint(guildIDNum), userID)
		if !canAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权查看公会内容"})
			return
		}

		var storyIDs []uint
		database.DB.Model(&model.StoryGuild{}).
			Where("guild_id = ?", guildID).
			Pluck("story_id", &storyIDs)
		if len(storyIDs) > 0 {
			query = query.Where("id IN ?", storyIDs)
		} else {
			c.JSON(http.StatusOK, gin.H{"stories": []model.Story{}})
			return
		}
	}

	// 搜索关键词
	if search := c.Query("search"); search != "" {
		query = query.Where("title LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 日期范围筛选
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			query = query.Where("created_at <= ?", t.Add(24*time.Hour))
		}
	}

	// 排序
	sortBy := c.DefaultQuery("sort", "created_at")
	sortOrder := c.DefaultQuery("order", "desc")
	orderClause := sortBy + " " + strings.ToUpper(sortOrder)
	query = query.Order(orderClause)

	var stories []model.Story
	if err := query.Find(&stories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stories": stories})
}

func (s *Server) createStory(c *gin.Context) {
	userID := c.GetUint("userID")

	var req CreateStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	story := model.Story{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Status:      "draft",
	}

	// 处理参与者
	if len(req.Participants) > 0 {
		data, _ := json.Marshal(req.Participants)
		story.Participants = string(data)
	}

	// 处理标签
	if len(req.Tags) > 0 {
		story.Tags = strings.Join(req.Tags, ",")
	}

	// 处理时间
	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			story.StartTime = t
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			story.EndTime = t
		}
	}

	if err := database.DB.Create(&story).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, story)
}

func (s *Server) getStory(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var story model.Story
	if err := database.DB.Where("id = ?", id).First(&story).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}

	// 权限检查：公会剧情公开访问，个人剧情只有作者可见
	var guildCount int64
	database.DB.Model(&model.StoryGuild{}).Where("story_id = ?", id).Count(&guildCount)

	isGuildStory := guildCount > 0
	isOwner := story.UserID == userID

	if !isGuildStory && !isOwner {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}

	// 获取剧情条目
	var entries []model.StoryEntry
	database.DB.Where("story_id = ?", id).Order("sort_order, timestamp").Find(&entries)

	c.JSON(http.StatusOK, gin.H{
		"story":   story,
		"entries": entries,
	})
}

func (s *Server) updateStory(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var story model.Story
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).
		First(&story).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}

	var req CreateStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	story.Title = req.Title
	story.Description = req.Description

	if len(req.Participants) > 0 {
		data, _ := json.Marshal(req.Participants)
		story.Participants = string(data)
	}
	if len(req.Tags) > 0 {
		story.Tags = strings.Join(req.Tags, ",")
	}

	if err := database.DB.Save(&story).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, story)
}

func (s *Server) deleteStory(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var story model.Story
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).
		First(&story).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}

	// 删除剧情条目
	database.DB.Where("story_id = ?", id).Delete(&model.StoryEntry{})
	// 删除剧情
	database.DB.Delete(&story)

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (s *Server) addStoryEntries(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var story model.Story
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).
		First(&story).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}

	var entries []CreateStoryEntryRequest
	if err := c.ShouldBindJSON(&entries); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取当前最大排序号
	var maxOrder int
	database.DB.Model(&model.StoryEntry{}).
		Where("story_id = ?", id).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxOrder)

	for i, req := range entries {
		var characterID *uint

		// 如果有角色信息，查找或创建角色
		if req.RefID != "" || req.GameID != "" {
			character := findOrCreateCharacter(userID, req.RefID, req.GameID, req.TRP3Data, req.IsNPC)
			if character != nil {
				characterID = &character.ID
			}
		}

		entry := model.StoryEntry{
			StoryID:     uint(id),
			SourceID:    req.SourceID,
			Type:        req.Type,
			CharacterID: characterID,
			Speaker:     req.Speaker,
			Content:     req.Content,
			Channel:     req.Channel,
			SortOrder:   maxOrder + i + 1,
		}
		if req.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
				entry.Timestamp = t
			}
		}
		database.DB.Create(&entry)
	}

	// 更新剧情的更新时间
	database.DB.Model(&story).Update("updated_at", time.Now())

	c.JSON(http.StatusCreated, gin.H{"message": "添加成功", "count": len(entries)})
}

// publishStory 发布/取消发布剧情
func (s *Server) publishStory(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var story model.Story
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).
		First(&story).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}

	var req struct {
		IsPublic bool `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	story.IsPublic = req.IsPublic
	if req.IsPublic && story.ShareCode == "" {
		story.ShareCode = generateShareCode()
	}

	database.DB.Save(&story)
	c.JSON(http.StatusOK, story)
}

// getPublicStory 获取公开剧情（无需登录）
func (s *Server) getPublicStory(c *gin.Context) {
	code := c.Param("code")

	var story model.Story
	if err := database.DB.Where("share_code = ? AND is_public = ?", code, true).
		First(&story).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在或未公开"})
		return
	}

	// 增加浏览次数
	database.DB.Model(&story).Update("view_count", story.ViewCount+1)

	// 获取条目
	var entries []model.StoryEntry
	database.DB.Where("story_id = ?", story.ID).Order("sort_order, timestamp").Find(&entries)

	// 收集所有角色ID
	characterIDs := make([]uint, 0)
	for _, entry := range entries {
		if entry.CharacterID != nil {
			characterIDs = append(characterIDs, *entry.CharacterID)
		}
	}

	// 获取角色信息
	charactersMap := make(map[uint]model.Character)
	if len(characterIDs) > 0 {
		var characters []model.Character
		database.DB.Where("id IN ?", characterIDs).Find(&characters)
		for _, char := range characters {
			charactersMap[char.ID] = char
		}
	}

	// 获取作者信息
	var user model.User
	database.DB.First(&user, story.UserID)

	c.JSON(http.StatusOK, gin.H{
		"story":      story,
		"entries":    entries,
		"characters": charactersMap,
		"author":     user.Username,
	})
}

// generateShareCode 生成分享码
func generateShareCode() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 8)
	for i := range code {
		code[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Nanosecond)
	}
	return string(code)
}

// UpdateStoryEntryRequest 更新剧情条目请求
type UpdateStoryEntryRequest struct {
	Content     string     `json:"content"`
	Speaker     string     `json:"speaker"`
	Channel     string     `json:"channel"`
	Type        string     `json:"type"`
	CharacterID *uint      `json:"character_id"`
	Timestamp   *time.Time `json:"timestamp"`
}

// updateStoryEntry 更新剧情条目
func (s *Server) updateStoryEntry(c *gin.Context) {
	userID := c.GetUint("userID")
	storyID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	entryID, _ := strconv.ParseUint(c.Param("entryId"), 10, 32)

	// 验证剧情所有权
	var story model.Story
	if err := database.DB.Where("id = ? AND user_id = ?", storyID, userID).
		First(&story).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}

	// 查找条目
	var entry model.StoryEntry
	if err := database.DB.Where("id = ? AND story_id = ?", entryID, storyID).
		First(&entry).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在"})
		return
	}

	var req UpdateStoryEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新字段
	if req.Content != "" {
		entry.Content = req.Content
	}
	if req.Speaker != "" {
		entry.Speaker = req.Speaker
	}
	if req.Channel != "" {
		entry.Channel = req.Channel
	}
	if req.Type != "" {
		entry.Type = req.Type
	}
	if req.CharacterID != nil {
		entry.CharacterID = req.CharacterID
	}
	if req.Timestamp != nil {
		entry.Timestamp = *req.Timestamp
	}

	if err := database.DB.Save(&entry).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	// 更新剧情的更新时间
	database.DB.Model(&story).Update("updated_at", time.Now())

	c.JSON(http.StatusOK, entry)
}

// deleteStoryEntry 删除剧情条目
func (s *Server) deleteStoryEntry(c *gin.Context) {
	userID := c.GetUint("userID")
	storyID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	entryID, _ := strconv.ParseUint(c.Param("entryId"), 10, 32)

	// 验证剧情所有权
	var story model.Story
	if err := database.DB.Where("id = ? AND user_id = ?", storyID, userID).
		First(&story).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}

	// 删除条目
	result := database.DB.Where("id = ? AND story_id = ?", entryID, storyID).
		Delete(&model.StoryEntry{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "条目不存在"})
		return
	}

	// 更新剧情的更新时间
	database.DB.Model(&story).Update("updated_at", time.Now())

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
