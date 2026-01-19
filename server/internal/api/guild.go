package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
)

// CreateGuildRequest 创建公会请求
type CreateGuildRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
	Banner      string `json:"banner"`
	Slogan      string `json:"slogan"`
	Lore        string `json:"lore"`
	Faction     string `json:"faction"`
	Layout      int    `json:"layout"`
}

// UpdateGuildRequest 更新公会请求
type UpdateGuildRequest struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	Icon                 string `json:"icon"`
	Color                string `json:"color"`
	Banner               string `json:"banner"`
	Slogan               string `json:"slogan"`
	Lore                 string `json:"lore"`
	Faction              string `json:"faction"`
	Layout               int    `json:"layout"`
	VisitorCanViewStories *bool `json:"visitor_can_view_stories"` // 访客可查看剧情
	VisitorCanViewPosts   *bool `json:"visitor_can_view_posts"`   // 访客可查看帖子
	MemberCanViewStories  *bool `json:"member_can_view_stories"`  // 成员可查看剧情
	MemberCanViewPosts    *bool `json:"member_can_view_posts"`    // 成员可查看帖子
	AutoApprove          *bool  `json:"auto_approve"`              // 自动审核（无需审核直接加入）
}

// JoinGuildRequest 加入公会请求
type JoinGuildRequest struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

// UpdateMemberRoleRequest 更新成员角色请求
type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// ApplyGuildRequest 申请加入公会请求
type ApplyGuildRequest struct {
	Message string `json:"message"`
}

// ReviewApplicationRequest 审批申请请求
type ReviewApplicationRequest struct {
	Action  string `json:"action" binding:"required"` // approve|reject
	Comment string `json:"comment"`
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
		// 列表查询排除大字段（banner）以提高性能
		// banner 通过独立的图片 API 访问
		database.DB.Select("id, name, description, icon, color, slogan, lore, faction, layout, owner_id, invite_code, member_count, story_count, server, status, visitor_can_view_stories, visitor_can_view_posts, member_can_view_stories, member_can_view_posts, auto_approve, created_at, updated_at").Where("id IN ?", guildIDs).Find(&guilds)
	}

	// 获取有 banner 的公会 ID 列表
	var guildsWithBanner []uint
	if len(guildIDs) > 0 {
		database.DB.Model(&model.Guild{}).
			Select("id").
			Where("id IN ? AND banner IS NOT NULL AND banner != ''", guildIDs).
			Pluck("id", &guildsWithBanner)
	}
	hasBannerMap := make(map[uint]bool)
	for _, id := range guildsWithBanner {
		hasBannerMap[id] = true
	}

	// 添加用户角色信息和 banner URL
	type GuildWithRole struct {
		model.Guild
		MyRole    string `json:"my_role"`
		BannerURL string `json:"banner_url"`
	}
	result := make([]GuildWithRole, len(guilds))
	for i, g := range guilds {
		// 构造 banner URL：宽度 600，质量 80
		bannerURL := ""
		if hasBannerMap[g.ID] {
			bannerURL = fmt.Sprintf("/api/v1/images/guild-banner/%d?w=600&q=80", g.ID)
		}
		result[i] = GuildWithRole{Guild: g, MyRole: roleMap[g.ID], BannerURL: bannerURL}
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

	layout := req.Layout
	if layout < 1 || layout > 4 {
		layout = 3 // 默认布局3
	}

	guild := model.Guild{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Color:       req.Color,
		Banner:      req.Banner,
		Slogan:      req.Slogan,
		Lore:        req.Lore,
		Faction:     req.Faction,
		Layout:      layout,
		OwnerID:     userID,
		InviteCode:  generateInviteCode(),
		MemberCount: 1,
		Status:      "pending", // 需要版主审核
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

	// 查询用户角色（如果是成员）
	var member model.GuildMember
	myRole := ""
	if err := database.DB.Where("guild_id = ? AND user_id = ?", id, userID).First(&member).Error; err == nil {
		myRole = member.Role
	}

	c.JSON(http.StatusOK, gin.H{"guild": guild, "my_role": myRole})
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
	if req.Banner != "" {
		guild.Banner = req.Banner
	}
	if req.Slogan != "" {
		guild.Slogan = req.Slogan
	}
	if req.Lore != "" {
		guild.Lore = req.Lore
	}
	if req.Faction != "" {
		guild.Faction = req.Faction
	}
	if req.Layout >= 1 && req.Layout <= 4 {
		guild.Layout = req.Layout
	}
	if req.VisitorCanViewStories != nil {
		guild.VisitorCanViewStories = *req.VisitorCanViewStories
	}
	if req.VisitorCanViewPosts != nil {
		guild.VisitorCanViewPosts = *req.VisitorCanViewPosts
	}
	if req.MemberCanViewStories != nil {
		guild.MemberCanViewStories = *req.MemberCanViewStories
	}
	if req.MemberCanViewPosts != nil {
		guild.MemberCanViewPosts = *req.MemberCanViewPosts
	}
	if req.AutoApprove != nil {
		guild.AutoApprove = *req.AutoApprove
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

// checkGuildContentAccess 检查用户是否有权限查看公会内容
// contentType: "story" 或 "post"
// 返回: canAccess（是否可访问）, memberRole（成员角色，非成员为空字符串）
func checkGuildContentAccess(guildID, userID uint, contentType string) (bool, string) {
	// 1. 获取公会设置
	var guild model.Guild
	if err := database.DB.First(&guild, guildID).Error; err != nil {
		return false, ""
	}

	// 2. 检查用户是否为成员
	var member model.GuildMember
	err := database.DB.Where("guild_id = ? AND user_id = ?", guildID, userID).First(&member).Error

	if err != nil {
		// 非成员 - 检查访客权限
		if contentType == "story" {
			return guild.VisitorCanViewStories, ""
		}
		return guild.VisitorCanViewPosts, ""
	}

	// 是成员
	role := member.Role

	// owner 和 admin 始终可访问
	if role == "owner" || role == "admin" {
		return true, role
	}

	// 普通成员 - 检查成员权限设置
	if contentType == "story" {
		return guild.MemberCanViewStories, role
	}
	return guild.MemberCanViewPosts, role
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
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	type MemberInfo struct {
		model.GuildMember
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
	}
	result := make([]MemberInfo, len(members))
	for i, m := range members {
		user := userMap[m.UserID]
		result[i] = MemberInfo{GuildMember: m, Username: user.Username, Avatar: user.Avatar}
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

	// 检查内容访问权限
	canAccess, _ := checkGuildContentAccess(uint(guildID), userID, "story")
	if !canAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权查看公会内容"})
		return
	}

	// 获取归档的剧情ID（支持按上传者筛选）
	query := database.DB.Where("guild_id = ?", guildID)
	if addedBy := c.Query("added_by"); addedBy != "" {
		addedByID, _ := strconv.ParseUint(addedBy, 10, 32)
		query = query.Where("added_by = ?", addedByID)
	}

	var storyGuilds []model.StoryGuild
	query.Order("created_at DESC").Find(&storyGuilds)

	storyIDs := make([]uint, len(storyGuilds))
	addedByMap := make(map[uint]uint) // storyID -> addedBy
	for i, sg := range storyGuilds {
		storyIDs[i] = sg.StoryID
		addedByMap[sg.StoryID] = sg.AddedBy
	}

	var stories []model.Story
	if len(storyIDs) > 0 {
		database.DB.Where("id IN ?", storyIDs).Find(&stories)
	}

	// 获取上传者信息
	addedByIDs := make([]uint, 0, len(addedByMap))
	for _, addedBy := range addedByMap {
		addedByIDs = append(addedByIDs, addedBy)
	}

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
		AddedBy         uint   `json:"added_by"`
		AddedByUsername string `json:"added_by_username"`
		AddedByAvatar   string `json:"added_by_avatar"`
	}

	result := make([]StoryWithUploader, len(stories))
	for i, story := range stories {
		addedBy := addedByMap[story.ID]
		uploader := userMap[addedBy]
		result[i] = StoryWithUploader{
			Story:           story,
			AddedBy:         addedBy,
			AddedByUsername: uploader.Username,
			AddedByAvatar:   uploader.Avatar,
		}
	}

	c.JSON(http.StatusOK, gin.H{"stories": result})
}

// getStoryGuilds 获取剧情归档的公会列表
func (s *Server) getStoryGuilds(c *gin.Context) {
	storyID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查剧情是否存在
	var story model.Story
	if err := database.DB.First(&story, storyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "剧情不存在"})
		return
	}

	// 公会剧情的关联信息公开可见，无需权限检查

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

// listPublicGuilds 获取公开公会列表（社区广场）
func (s *Server) listPublicGuilds(c *gin.Context) {
	var guilds []model.Guild
	query := database.DB.Where("status = ? AND is_public = ?", "approved", true)

	// 支持搜索
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 支持阵营筛选
	if faction := c.Query("faction"); faction != "" {
		query = query.Where("faction = ?", faction)
	}

	// 支持服务器筛选
	if server := c.Query("server"); server != "" {
		query = query.Where("server = ?", server)
	}

	// 列表查询排除大字段（banner）以提高性能
	// banner 通过独立的图片 API 访问
	query.Select("id, name, description, icon, color, slogan, lore, faction, layout, owner_id, invite_code, member_count, story_count, server, status, visitor_can_view_stories, visitor_can_view_posts, member_can_view_stories, member_can_view_posts, auto_approve, created_at, updated_at").Order("member_count DESC, created_at DESC").Find(&guilds)

	// 获取有 banner 的公会 ID 列表
	guildIDs := make([]uint, len(guilds))
	for i, g := range guilds {
		guildIDs[i] = g.ID
	}

	var guildsWithBanner []uint
	if len(guildIDs) > 0 {
		database.DB.Model(&model.Guild{}).
			Select("id").
			Where("id IN ? AND banner IS NOT NULL AND banner != ''", guildIDs).
			Pluck("id", &guildsWithBanner)
	}
	hasBannerMap := make(map[uint]bool)
	for _, id := range guildsWithBanner {
		hasBannerMap[id] = true
	}

	// 添加 banner URL
	type GuildWithBanner struct {
		model.Guild
		BannerURL string `json:"banner_url"`
	}
	result := make([]GuildWithBanner, len(guilds))
	for i, g := range guilds {
		// 构造 banner URL：宽度 600，质量 80
		bannerURL := ""
		if hasBannerMap[g.ID] {
			bannerURL = fmt.Sprintf("/api/v1/images/guild-banner/%d?w=600&q=80", g.ID)
		}
		result[i] = GuildWithBanner{Guild: g, BannerURL: bannerURL}
	}

	c.JSON(http.StatusOK, gin.H{"guilds": result})
}

// uploadGuildBanner 上传公会头图
func (s *Server) uploadGuildBanner(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查权限
	if !checkGuildAdmin(uint(guildID), userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		return
	}

	file, header, err := c.Request.FormFile("banner")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择头图文件"})
		return
	}
	defer file.Close()

	// 检查文件大小 (最大 20MB)
	if header.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "头图文件不能超过20MB"})
		return
	}

	// 检查文件类型
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只支持图片格式"})
		return
	}

	// 读取文件内容
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

	// 转换为 base64
	base64Data := base64.StdEncoding.EncodeToString(data)
	bannerURL := "data:" + contentType + ";base64," + base64Data

	// 更新数据库
	if err := database.DB.Model(&model.Guild{}).Where("id = ?", guildID).Update("banner", bannerURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新头图失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "头图更新成功",
		"banner":  bannerURL,
	})
}

// ========== 公会申请系统 ==========

// applyGuild 申请加入公会
func (s *Server) applyGuild(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req ApplyGuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查公会是否存在
	var guild model.Guild
	if err := database.DB.First(&guild, guildID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "公会不存在"})
		return
	}

	// 检查是否已是成员
	var existingMember model.GuildMember
	if err := database.DB.Where("guild_id = ? AND user_id = ?", guildID, userID).First(&existingMember).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已是公会成员"})
		return
	}

	// 如果开启自动审核，直接加入公会
	if guild.AutoApprove {
		member := model.GuildMember{
			GuildID:  uint(guildID),
			UserID:   userID,
			Role:     "member",
			JoinedAt: time.Now(),
		}
		if err := database.DB.Create(&member).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "加入失败"})
			return
		}
		// 更新公会成员数
		database.DB.Model(&model.Guild{}).Where("id = ?", guildID).Update("member_count", database.DB.Raw("member_count + 1"))
		c.JSON(http.StatusCreated, gin.H{"message": "已加入公会", "auto_approved": true})
		return
	}

	// 检查是否已有申请记录（任何状态）
	var existingApp model.GuildApplication
	err := database.DB.Where("guild_id = ? AND user_id = ?", guildID, userID).First(&existingApp).Error

	if err == nil {
		// 已有申请记录
		if existingApp.Status == "pending" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "已有待处理的申请"})
			return
		}
		// 如果是已拒绝或已批准的申请，更新为新的待处理申请
		existingApp.Message = req.Message
		existingApp.Status = "pending"
		existingApp.ReviewerID = nil
		existingApp.ReviewComment = ""
		existingApp.ReviewedAt = nil
		if err := database.DB.Save(&existingApp).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "申请失败"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "申请已提交", "application": existingApp})
		return
	}

	// 没有申请记录，创建新的
	application := model.GuildApplication{
		GuildID: uint(guildID),
		UserID:  userID,
		Message: req.Message,
		Status:  "pending",
	}

	if err := database.DB.Create(&application).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "申请失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "申请已提交", "application": application})
}

// listGuildApplications 获取公会申请列表（管理员）
func (s *Server) listGuildApplications(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查管理员权限
	if !checkGuildAdmin(uint(guildID), userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		return
	}

	// 支持状态筛选
	status := c.Query("status") // pending|approved|rejected|all
	query := database.DB.Where("guild_id = ?", guildID)
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var applications []model.GuildApplication
	query.Order("created_at DESC").Find(&applications)

	// 获取申请人信息
	userIDs := make([]uint, len(applications))
	for i, app := range applications {
		userIDs[i] = app.UserID
	}

	var users []model.User
	if len(userIDs) > 0 {
		database.DB.Where("id IN ?", userIDs).Find(&users)
	}
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 组装结果
	type ApplicationInfo struct {
		model.GuildApplication
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
	}
	result := make([]ApplicationInfo, len(applications))
	for i, app := range applications {
		user := userMap[app.UserID]
		result[i] = ApplicationInfo{
			GuildApplication: app,
			Username:         user.Username,
			Avatar:           user.Avatar,
		}
	}

	c.JSON(http.StatusOK, gin.H{"applications": result})
}

// reviewGuildApplication 审批公会申请（管理员）
func (s *Server) reviewGuildApplication(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	appID, _ := strconv.ParseUint(c.Param("appId"), 10, 32)

	// 检查管理员权限
	if !checkGuildAdmin(uint(guildID), userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		return
	}

	var req ReviewApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Action != "approve" && req.Action != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效操作"})
		return
	}

	// 获取申请记录
	var application model.GuildApplication
	if err := database.DB.First(&application, appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "申请不存在"})
		return
	}

	if application.GuildID != uint(guildID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "申请不属于此公会"})
		return
	}

	if application.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "申请已处理"})
		return
	}

	// 更新申请状态
	now := time.Now()
	application.ReviewerID = &userID
	application.ReviewComment = req.Comment
	application.ReviewedAt = &now

	if req.Action == "approve" {
		application.Status = "approved"

		// 创建成员记录
		member := model.GuildMember{
			GuildID:  uint(guildID),
			UserID:   application.UserID,
			Role:     "member",
			JoinedAt: time.Now(),
		}
		if err := database.DB.Create(&member).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建成员记录失败"})
			return
		}

		// 更新公会成员数
		database.DB.Model(&model.Guild{}).Where("id = ?", guildID).Update("member_count", database.DB.Raw("member_count + 1"))
	} else {
		application.Status = "rejected"
	}

	database.DB.Save(&application)

	// 创建通知
	var notifContent string
	if req.Action == "approve" {
		notifContent = "你的公会申请已通过"
	} else {
		notifContent = "你的公会申请已被拒绝"
	}
	notification := model.Notification{
		UserID:     application.UserID,
		Type:       "guild_application",
		ActorID:    &userID,
		TargetType: "guild",
		TargetID:   uint(guildID),
		Content:    notifContent,
	}
	service.CreateNotification(&notification)

	c.JSON(http.StatusOK, gin.H{"message": "审批完成", "application": application})
}

// listMyApplications 获取我的申请记录
func (s *Server) listMyApplications(c *gin.Context) {
	userID := c.GetUint("userID")

	var applications []model.GuildApplication
	database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&applications)

	// 获取公会信息
	guildIDs := make([]uint, len(applications))
	for i, app := range applications {
		guildIDs[i] = app.GuildID
	}

	var guilds []model.Guild
	if len(guildIDs) > 0 {
		database.DB.Where("id IN ?", guildIDs).Find(&guilds)
	}
	guildMap := make(map[uint]model.Guild)
	for _, g := range guilds {
		guildMap[g.ID] = g
	}

	// 组装结果
	type ApplicationWithGuild struct {
		model.GuildApplication
		GuildName string `json:"guild_name"`
		GuildIcon string `json:"guild_icon"`
	}
	result := make([]ApplicationWithGuild, len(applications))
	for i, app := range applications {
		guild := guildMap[app.GuildID]
		result[i] = ApplicationWithGuild{
			GuildApplication: app,
			GuildName:        guild.Name,
			GuildIcon:        guild.Icon,
		}
	}

	c.JSON(http.StatusOK, gin.H{"applications": result})
}

// cancelApplication 撤销申请
func (s *Server) cancelApplication(c *gin.Context) {
	userID := c.GetUint("userID")
	guildID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	appID, _ := strconv.ParseUint(c.Param("appId"), 10, 32)

	var application model.GuildApplication
	if err := database.DB.First(&application, appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "申请不存在"})
		return
	}

	if application.GuildID != uint(guildID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "申请不属于此公会"})
		return
	}

	if application.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能撤销自己的申请"})
		return
	}

	if application.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能撤销待处理的申请"})
		return
	}

	database.DB.Delete(&application)
	c.JSON(http.StatusOK, gin.H{"message": "申请已撤销"})
}
