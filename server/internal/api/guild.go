package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

// CreateGuildRequest 创建公会请求
type CreateGuildRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
}

// UpdateGuildRequest 更新公会请求
type UpdateGuildRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
}

// JoinGuildRequest 加入公会请求
type JoinGuildRequest struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

// UpdateMemberRoleRequest 更新成员角色请求
type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// generateInviteCode 生成邀请码
func generateInviteCode() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// listGuilds 获取我的公会列表
func (s *Server) listGuilds(c *gin.Context) {
	userID := c.GetUint("userID")

	var members []model.GuildMember
	database.DB.Where("user_id = ?", userID).Find(&members)

	guildIDs := make([]uint, len(members))
	roleMap := make(map[uint]string)
	for i, m := range members {
		guildIDs[i] = m.GuildID
		roleMap[m.GuildID] = m.Role
	}

	var guilds []model.Guild
	if len(guildIDs) > 0 {
		database.DB.Where("id IN ?", guildIDs).Find(&guilds)
	}

	// 添加用户角色信息
	type GuildWithRole struct {
		model.Guild
		MyRole string `json:"my_role"`
	}
	result := make([]GuildWithRole, len(guilds))
	for i, g := range guilds {
		result[i] = GuildWithRole{Guild: g, MyRole: roleMap[g.ID]}
	}

	c.JSON(http.StatusOK, gin.H{"guilds": result})
}

// createGuild 创建公会
func (s *Server) createGuild(c *gin.Context) {
	userID := c.GetUint("userID")

	var req CreateGuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	guild := model.Guild{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Color:       req.Color,
		OwnerID:     userID,
		InviteCode:  generateInviteCode(),
		MemberCount: 1,
	}

	if err := database.DB.Create(&guild).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	// 创建会长成员记录
	member := model.GuildMember{
		GuildID:  guild.ID,
		UserID:   userID,
		Role:     "owner",
		JoinedAt: time.Now(),
	}
	database.DB.Create(&member)

	c.JSON(http.StatusCreated, guild)
}

// getGuild 获取公会详情
func (s *Server) getGuild(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var guild model.Guild
	if err := database.DB.First(&guild, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "公会不存在"})
		return
	}

	// 检查是否是成员
	var member model.GuildMember
	if err := database.DB.Where("guild_id = ? AND user_id = ?", id, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "非公会成员"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"guild": guild, "my_role": member.Role})
}

// updateGuild 更新公会信息
func (s *Server) updateGuild(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查权限
	if !checkGuildAdmin(uint(id), userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		return
	}

	var guild model.Guild
	if err := database.DB.First(&guild, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "公会不存在"})
		return
	}

	var req UpdateGuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		guild.Name = req.Name
	}
	if req.Description != "" {
		guild.Description = req.Description
	}
	if req.Icon != "" {
		guild.Icon = req.Icon
	}
	if req.Color != "" {
		guild.Color = req.Color
	}

	database.DB.Save(&guild)
	c.JSON(http.StatusOK, guild)
}

// deleteGuild 解散公会
func (s *Server) deleteGuild(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var guild model.Guild
	if err := database.DB.First(&guild, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "公会不存在"})
		return
	}

	// 只有会长可以解散
	if guild.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有会长可以解散公会"})
		return
	}

	// 删除成员记录
	database.DB.Where("guild_id = ?", id).Delete(&model.GuildMember{})
	// 删除公会标签
	database.DB.Where("guild_id = ?", id).Delete(&model.Tag{})
	// 删除剧情归档
	database.DB.Where("guild_id = ?", id).Delete(&model.StoryGuild{})
	// 删除公会
	database.DB.Delete(&guild)

	c.JSON(http.StatusOK, gin.H{"message": "公会已解散"})
}

// checkGuildAdmin 检查是否有管理权限
func checkGuildAdmin(guildID, userID uint) bool {
	var member model.GuildMember
	if err := database.DB.Where("guild_id = ? AND user_id = ?", guildID, userID).First(&member).Error; err != nil {
		return false
	}
	return member.Role == "owner" || member.Role == "admin"
}

// joinGuild 加入公会
func (s *Server) joinGuild(c *gin.Context) {
	userID := c.GetUint("userID")

	var req JoinGuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var guild model.Guild
	if err := database.DB.Where("invite_code = ?", req.InviteCode).First(&guild).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "邀请码无效"})
		return
	}

	// 检查是否已是成员
	var existing model.GuildMember
	if err := database.DB.Where("guild_id = ? AND user_id = ?", guild.ID, userID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已是公会成员"})
		return
	}

	member := model.GuildMember{
		GuildID:  guild.ID,
		UserID:   userID,
		Role:     "member",
		JoinedAt: time.Now(),
	}
	database.DB.Create(&member)

	// 更新成员数
	database.DB.Model(&guild).Update("member_count", guild.MemberCount+1)

	c.JSON(http.StatusOK, gin.H{"message": "加入成功", "guild": guild})
}

// leaveGuild 退出公会
func (s *Server) leaveGuild(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var member model.GuildMember
	if err := database.DB.Where("guild_id = ? AND user_id = ?", id, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "非公会成员"})
		return
	}

	if member.Role == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会长不能退出，请先转让或解散公会"})
		return
	}

	database.DB.Delete(&member)
	database.DB.Model(&model.Guild{}).Where("id = ?", id).Update("member_count", database.DB.Raw("member_count - 1"))

	c.JSON(http.StatusOK, gin.H{"message": "已退出公会"})
}

// listGuildMembers 获取成员列表
func (s *Server) listGuildMembers(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查是否是成员
	var self model.GuildMember
	if err := database.DB.Where("guild_id = ? AND user_id = ?", id, userID).First(&self).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "非公会成员"})
		return
	}

	var members []model.GuildMember
	database.DB.Where("guild_id = ?", id).Order("role ASC, joined_at ASC").Find(&members)

	// 获取用户名
	userIDs := make([]uint, len(members))
	for i, m := range members {
		userIDs[i] = m.UserID
	}

	var users []model.User
	database.DB.Where("id IN ?", userIDs).Find(&users)
	userMap := make(map[uint]string)
	for _, u := range users {
		userMap[u.ID] = u.Username
	}

	type MemberInfo struct {
		model.GuildMember
		Username string `json:"username"`
	}
	result := make([]MemberInfo, len(members))
	for i, m := range members {
		result[i] = MemberInfo{GuildMember: m, Username: userMap[m.UserID]}
	}

	c.JSON(http.StatusOK, gin.H{"members": result})
}

// updateMemberRole 更新成员角色
func (s *Server) updateMemberRole(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	memberUID, _ := strconv.ParseUint(c.Param("uid"), 10, 32)

	if !checkGuildAdmin(uint(guildID), userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		return
	}

	var req UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 只能设置 admin 或 member
	if req.Role != "admin" && req.Role != "member" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效角色"})
		return
	}

	var member model.GuildMember
	if err := database.DB.Where("guild_id = ? AND user_id = ?", guildID, memberUID).First(&member).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "成员不存在"})
		return
	}

	if member.Role == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能修改会长角色"})
		return
	}

	member.Role = req.Role
	database.DB.Save(&member)
	c.JSON(http.StatusOK, gin.H{"message": "角色已更新"})
}

// removeMember 移除成员
func (s *Server) removeMember(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	memberUID, _ := strconv.ParseUint(c.Param("uid"), 10, 32)

	if !checkGuildAdmin(uint(guildID), userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		return
	}

	var member model.GuildMember
	if err := database.DB.Where("guild_id = ? AND user_id = ?", guildID, memberUID).First(&member).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "成员不存在"})
		return
	}

	if member.Role == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能移除会长"})
		return
	}

	database.DB.Delete(&member)
	database.DB.Model(&model.Guild{}).Where("id = ?", guildID).Update("member_count", database.DB.Raw("member_count - 1"))

	c.JSON(http.StatusOK, gin.H{"message": "成员已移除"})
}

// ========== 剧情归档到公会 ==========

// archiveStoryToGuild 将剧情归档到公会
func (s *Server) archiveStoryToGuild(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	storyID, _ := strconv.ParseUint(c.Param("storyId"), 10, 32)

	// 检查是否是公会成员
	var member model.GuildMember
	if err := database.DB.Where("guild_id = ? AND user_id = ?", guildID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "非公会成员"})
		return
	}

	// 检查剧情所有权
	var story model.Story
	if err := database.DB.First(&story, storyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}
	if story.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能归档自己的剧情"})
		return
	}

	// 创建归档关联
	storyGuild := model.StoryGuild{
		StoryID: uint(storyID),
		GuildID: uint(guildID),
		AddedBy: userID,
	}

	if err := database.DB.Create(&storyGuild).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已归档到此公会"})
		return
	}

	// 更新公会剧情数
	database.DB.Model(&model.Guild{}).Where("id = ?", guildID).Update("story_count", database.DB.Raw("story_count + 1"))

	c.JSON(http.StatusOK, gin.H{"message": "归档成功"})
}

// removeStoryFromGuild 从公会移除剧情归档
func (s *Server) removeStoryFromGuild(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	storyID, _ := strconv.ParseUint(c.Param("storyId"), 10, 32)

	// 检查归档记录
	var storyGuild model.StoryGuild
	if err := database.DB.Where("guild_id = ? AND story_id = ?", guildID, storyID).First(&storyGuild).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "归档记录不存在"})
		return
	}

	// 只有归档者或管理员可以移除
	if storyGuild.AddedBy != userID && !checkGuildAdmin(uint(guildID), userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	database.DB.Delete(&storyGuild)
	database.DB.Model(&model.Guild{}).Where("id = ?", guildID).Update("story_count", database.DB.Raw("story_count - 1"))

	c.JSON(http.StatusOK, gin.H{"message": "已移除归档"})
}

// listGuildStories 获取公会归档的剧情列表
func (s *Server) listGuildStories(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查是否是公会成员
	var member model.GuildMember
	if err := database.DB.Where("guild_id = ? AND user_id = ?", guildID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "非公会成员"})
		return
	}

	// 获取归档的剧情ID
	var storyGuilds []model.StoryGuild
	database.DB.Where("guild_id = ?", guildID).Order("created_at DESC").Find(&storyGuilds)

	storyIDs := make([]uint, len(storyGuilds))
	for i, sg := range storyGuilds {
		storyIDs[i] = sg.StoryID
	}

	var stories []model.Story
	if len(storyIDs) > 0 {
		database.DB.Where("id IN ?", storyIDs).Find(&stories)
	}

	c.JSON(http.StatusOK, gin.H{"stories": stories})
}

// getStoryGuilds 获取剧情归档的公会列表
func (s *Server) getStoryGuilds(c *gin.Context) {
	userID := c.GetUint("userID")
	storyID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查剧情所有权
	var story model.Story
	if err := database.DB.First(&story, storyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}
	if story.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权查看"})
		return
	}

	var storyGuilds []model.StoryGuild
	database.DB.Where("story_id = ?", storyID).Find(&storyGuilds)

	guildIDs := make([]uint, len(storyGuilds))
	for i, sg := range storyGuilds {
		guildIDs[i] = sg.GuildID
	}

	var guilds []model.Guild
	if len(guildIDs) > 0 {
		database.DB.Where("id IN ?", guildIDs).Find(&guilds)
	}

	c.JSON(http.StatusOK, gin.H{"guilds": guilds})
}
