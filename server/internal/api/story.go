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
	"github.com/rpbox/server/pkg/validator"
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
		canAccess, _ := checkGuildContentAccess(uint(guildIDNum), userID, "story")
		if !canAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权查看公会内容"})
			return
		}

		// 构建公会剧情查询
		sgQuery := database.DB.Model(&model.StoryGuild{}).Where("guild_id = ?", guildID)

		// 上传者筛选
		if addedBy := c.Query("added_by"); addedBy != "" {
			sgQuery = sgQuery.Where("added_by = ?", addedBy)
		}

		var storyIDs []uint
		sgQuery.Pluck("story_id", &storyIDs)
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

	storyIDs := make([]uint, len(stories))
	for i, story := range stories {
		storyIDs[i] = story.ID
	}

	if len(storyIDs) > 0 {
		type storyEntryStats struct {
			StoryID    uint       `gorm:"column:story_id"`
			EntryCount int64      `gorm:"column:entry_count"`
			MinTime    *time.Time `gorm:"column:min_time"`
			MaxTime    *time.Time `gorm:"column:max_time"`
		}

		var entryStats []storyEntryStats
		zeroTime := time.Time{}
		database.DB.Model(&model.StoryEntry{}).
			Select("story_id, COUNT(*) AS entry_count, MIN(CASE WHEN timestamp > ? THEN timestamp ELSE created_at END) AS min_time, MAX(CASE WHEN timestamp > ? THEN timestamp ELSE created_at END) AS max_time", zeroTime, zeroTime).
			Where("story_id IN ?", storyIDs).
			Group("story_id").
			Scan(&entryStats)

		entryStatsMap := make(map[uint]storyEntryStats, len(entryStats))
		for _, stat := range entryStats {
			entryStatsMap[stat.StoryID] = stat
		}

		for i := range stories {
			stories[i].EntryCount = 0
			if stat, ok := entryStatsMap[stories[i].ID]; ok {
				stories[i].EntryCount = int(stat.EntryCount)
				if stories[i].StartTime.IsZero() && stat.MinTime != nil {
					stories[i].StartTime = *stat.MinTime
				}
				if stories[i].EndTime.IsZero() && stat.MaxTime != nil {
					stories[i].EndTime = *stat.MaxTime
				}
			}
		}

		var storyTags []model.StoryTag
		database.DB.Where("story_id IN ?", storyIDs).Order("created_at ASC").Find(&storyTags)
		if len(storyTags) > 0 {
			tagIDSet := make(map[uint]struct{})
			tagIDs := make([]uint, 0, len(storyTags))
			for _, st := range storyTags {
				if _, ok := tagIDSet[st.TagID]; !ok {
					tagIDSet[st.TagID] = struct{}{}
					tagIDs = append(tagIDs, st.TagID)
				}
			}

			var tags []model.Tag
			database.DB.Where("id IN ?", tagIDs).Find(&tags)
			tagNameMap := make(map[uint]string, len(tags))
			tagInfoMap := make(map[uint]model.StoryTagInfo, len(tags))
			for _, tag := range tags {
				tagNameMap[tag.ID] = tag.Name
				tagInfoMap[tag.ID] = model.StoryTagInfo{
					Name:  tag.Name,
					Color: tag.Color,
				}
			}

			storyTagMap := make(map[uint][]string)
			storyTagInfoMap := make(map[uint][]model.StoryTagInfo)
			for _, st := range storyTags {
				if name := tagNameMap[st.TagID]; name != "" {
					storyTagMap[st.StoryID] = append(storyTagMap[st.StoryID], name)
				}
				if info, ok := tagInfoMap[st.TagID]; ok {
					storyTagInfoMap[st.StoryID] = append(storyTagInfoMap[st.StoryID], info)
				}
			}

			for i := range stories {
				if names, ok := storyTagMap[stories[i].ID]; ok {
					stories[i].Tags = strings.Join(names, ",")
				}
				if infos, ok := storyTagInfoMap[stories[i].ID]; ok {
					stories[i].TagList = infos
				}
			}
		}
	}

	// 如果是公会剧情查询，添加上传者信息
	if guildID := c.Query("guild_id"); guildID != "" {
		// 查询上传者信息
		var storyGuilds []model.StoryGuild
		database.DB.Where("story_id IN ? AND guild_id = ?", storyIDs, guildID).Find(&storyGuilds)

		// 构建 storyID -> addedBy 映射
		addedByMap := make(map[uint]uint)
		addedByIDs := make([]uint, 0)
		for _, sg := range storyGuilds {
			addedByMap[sg.StoryID] = sg.AddedBy
			addedByIDs = append(addedByIDs, sg.AddedBy)
		}

		// 查询用户信息
		var users []model.User
		if len(addedByIDs) > 0 {
			database.DB.Where("id IN ?", addedByIDs).Find(&users)
		}
		userMap := make(map[uint]model.User)
		for _, u := range users {
			userMap[u.ID] = u
		}

		// 组装结果
		type StoryWithUploader struct {
			model.Story
			AddedBy          uint   `json:"added_by"`
			AddedByUsername  string `json:"added_by_username"`
			AddedByAvatar    string `json:"added_by_avatar"`
			AddedByNameColor string `json:"added_by_name_color"`
			AddedByNameBold  bool   `json:"added_by_name_bold"`
		}

		result := make([]StoryWithUploader, len(stories))
		for i, story := range stories {
			addedBy := addedByMap[story.ID]
			uploader := userMap[addedBy]
			nameColor, nameBold := userDisplayStyle(uploader)
			result[i] = StoryWithUploader{
				Story:            story,
				AddedBy:          addedBy,
				AddedByUsername:  uploader.Username,
				AddedByAvatar:    uploader.Avatar,
				AddedByNameColor: nameColor,
				AddedByNameBold:  nameBold,
			}
		}

		c.JSON(http.StatusOK, gin.H{"stories": result})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stories": stories})
}

func (s *Server) createStory(c *gin.Context) {
	userID := c.GetUint("userID")

	var req CreateStoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
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
	database.DB.Where("story_id = ?", id).Order("timestamp, sort_order").Find(&entries)

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
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
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
	// 删除剧情标签关联
	database.DB.Where("story_id = ?", id).Delete(&model.StoryTag{})
	// 删除剧情
	database.DB.Delete(&story)

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// BatchDeleteStoriesRequest 批量删除剧情请求
type BatchDeleteStoriesRequest struct {
	IDs []uint `json:"ids" binding:"required"`
}

// batchDeleteStories 批量删除剧情
func (s *Server) batchDeleteStories(c *gin.Context) {
	userID := c.GetUint("userID")

	var req BatchDeleteStoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要删除的剧情"})
		return
	}

	// 验证所有剧情都属于当前用户
	var count int64
	database.DB.Model(&model.Story{}).
		Where("id IN ? AND user_id = ?", req.IDs, userID).
		Count(&count)

	if count != int64(len(req.IDs)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "部分剧情不存在或无权删除"})
		return
	}

	// 删除剧情条目
	database.DB.Where("story_id IN ?", req.IDs).Delete(&model.StoryEntry{})
	// 删除剧情标签关联
	database.DB.Where("story_id IN ?", req.IDs).Delete(&model.StoryTag{})
	// 删除剧情
	database.DB.Where("id IN ? AND user_id = ?", req.IDs, userID).Delete(&model.Story{})

	c.JSON(http.StatusOK, gin.H{"message": "删除成功", "count": len(req.IDs)})
}

// BatchMoveStoriesRequest 批量移动剧情请求
type BatchMoveStoriesRequest struct {
	SourceIDs []uint `json:"source_ids" binding:"required"`
	TargetID  uint   `json:"target_id" binding:"required"`
}

// batchMoveStories 批量移动剧情条目到目标剧情
func (s *Server) batchMoveStories(c *gin.Context) {
	userID := c.GetUint("userID")

	var req BatchMoveStoriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	if len(req.SourceIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要移动的剧情"})
		return
	}

	// 验证目标剧情属于当前用户
	var targetStory model.Story
	if err := database.DB.Where("id = ? AND user_id = ?", req.TargetID, userID).
		First(&targetStory).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "目标剧情不存在"})
		return
	}

	// 验证所有源剧情都属于当前用户
	var count int64
	database.DB.Model(&model.Story{}).
		Where("id IN ? AND user_id = ?", req.SourceIDs, userID).
		Count(&count)

	if count != int64(len(req.SourceIDs)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "部分剧情不存在或无权操作"})
		return
	}

	// 获取目标剧情当前最大排序号
	var maxOrder int
	database.DB.Model(&model.StoryEntry{}).
		Where("story_id = ?", req.TargetID).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxOrder)

	// 移动所有条目到目标剧情
	database.DB.Model(&model.StoryEntry{}).
		Where("story_id IN ?", req.SourceIDs).
		Update("story_id", req.TargetID)

	// 更新排序号（简单处理：按原顺序追加）
	var entries []model.StoryEntry
	database.DB.Where("story_id = ? AND sort_order <= ?", req.TargetID, maxOrder).
		Order("sort_order").Find(&entries)

	// 删除源剧情的标签关联
	database.DB.Where("story_id IN ?", req.SourceIDs).Delete(&model.StoryTag{})
	// 删除源剧情
	database.DB.Where("id IN ? AND user_id = ?", req.SourceIDs, userID).Delete(&model.Story{})

	// 更新目标剧情的时间范围
	type storyEntryStats struct {
		MinTime *time.Time `gorm:"column:min_time"`
		MaxTime *time.Time `gorm:"column:max_time"`
	}
	var stat storyEntryStats
	zeroTime := time.Time{}
	database.DB.Model(&model.StoryEntry{}).
		Select("MIN(CASE WHEN timestamp > ? THEN timestamp ELSE created_at END) AS min_time, MAX(CASE WHEN timestamp > ? THEN timestamp ELSE created_at END) AS max_time", zeroTime, zeroTime).
		Where("story_id = ?", req.TargetID).
		Scan(&stat)

	updates := map[string]interface{}{"updated_at": time.Now()}
	if stat.MinTime != nil {
		updates["start_time"] = *stat.MinTime
	}
	if stat.MaxTime != nil {
		updates["end_time"] = *stat.MaxTime
	}
	database.DB.Model(&targetStory).Updates(updates)

	c.JSON(http.StatusOK, gin.H{"message": "移动成功", "count": len(req.SourceIDs)})
}

// BatchUpdateBackgroundColorRequest 批量更新背景色请求
type BatchUpdateBackgroundColorRequest struct {
	IDs             []uint `json:"ids" binding:"required"`
	BackgroundColor string `json:"background_color"` // 空字符串表示清除背景色
}

// batchUpdateBackgroundColor 批量更新剧情背景色
func (s *Server) batchUpdateBackgroundColor(c *gin.Context) {
	userID := c.GetUint("userID")

	var req BatchUpdateBackgroundColorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要更新的剧情"})
		return
	}

	// 验证所有剧情都属于当前用户
	var count int64
	database.DB.Model(&model.Story{}).
		Where("id IN ? AND user_id = ?", req.IDs, userID).
		Count(&count)

	if count != int64(len(req.IDs)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "部分剧情不存在或无权操作"})
		return
	}

	// 更新背景色
	database.DB.Model(&model.Story{}).
		Where("id IN ? AND user_id = ?", req.IDs, userID).
		Update("background_color", req.BackgroundColor)

	c.JSON(http.StatusOK, gin.H{"message": "更新成功", "count": len(req.IDs)})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	// 获取当前最大排序号
	var maxOrder int
	database.DB.Model(&model.StoryEntry{}).
		Where("story_id = ?", id).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxOrder)

	var minTimestamp *time.Time
	var maxTimestamp *time.Time
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
				if minTimestamp == nil || t.Before(*minTimestamp) {
					ts := t
					minTimestamp = &ts
				}
				if maxTimestamp == nil || t.After(*maxTimestamp) {
					ts := t
					maxTimestamp = &ts
				}
			}
		}
		database.DB.Create(&entry)
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if story.StartTime.IsZero() || story.EndTime.IsZero() {
		type storyEntryStats struct {
			MinTime *time.Time `gorm:"column:min_time"`
			MaxTime *time.Time `gorm:"column:max_time"`
		}
		var stat storyEntryStats
		zeroTime := time.Time{}
		database.DB.Model(&model.StoryEntry{}).
			Select("MIN(CASE WHEN timestamp > ? THEN timestamp ELSE created_at END) AS min_time, MAX(CASE WHEN timestamp > ? THEN timestamp ELSE created_at END) AS max_time", zeroTime, zeroTime).
			Where("story_id = ?", id).
			Scan(&stat)
		if story.StartTime.IsZero() && stat.MinTime != nil {
			updates["start_time"] = *stat.MinTime
		}
		if story.EndTime.IsZero() && stat.MaxTime != nil {
			updates["end_time"] = *stat.MaxTime
		}
	} else {
		if minTimestamp != nil && minTimestamp.Before(story.StartTime) {
			updates["start_time"] = *minTimestamp
		}
		if maxTimestamp != nil && maxTimestamp.After(story.EndTime) {
			updates["end_time"] = *maxTimestamp
		}
	}

	database.DB.Model(&story).Updates(updates)

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
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
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
	database.DB.Where("story_id = ?", story.ID).Order("timestamp, sort_order").Find(&entries)

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
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
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
