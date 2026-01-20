package api

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/pkg/validator"
)

// getUserInfo 获取当前用户信息
func (s *Server) getUserInfo(c *gin.Context) {
	userID := c.GetUint("userID")

	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 动态计算统计数据
	var postCount int64
	database.DB.Model(&model.Post{}).Where("author_id = ? AND status = ? AND review_status = ?", userID, "published", "approved").Count(&postCount)

	var storyCount int64
	database.DB.Model(&model.Story{}).Where("user_id = ?", userID).Count(&storyCount)

	var profileCount int64
	database.DB.Model(&model.Profile{}).Where("user_id = ?", userID).Count(&profileCount)

	c.JSON(http.StatusOK, gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"avatar":        user.Avatar,
		"role":          user.Role,
		"bio":           user.Bio,
		"location":      user.Location,
		"website":       user.Website,
		"post_count":    postCount,
		"story_count":   storyCount,
		"profile_count": profileCount,
		"created_at":    user.CreatedAt,
	})
}

// updateAvatar 更新用户头像
func (s *Server) updateAvatar(c *gin.Context) {
	userID := c.GetUint("userID")

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择头像文件"})
		return
	}
	defer file.Close()

	// 检查文件大小 (最大 20MB)
	if header.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "头像文件不能超过20MB"})
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
	avatarURL := "data:" + contentType + ";base64," + base64Data

	// 更新数据库
	if err := database.DB.Model(&model.User{}).Where("id = ?", userID).Update("avatar", avatarURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新头像失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "头像更新成功",
		"avatar":  avatarURL,
	})
}

// updateUserInfo 更新用户信息
func (s *Server) updateUserInfo(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Bio      string `json:"bio"`
		Location string `json:"location"`
		Website  string `json:"website"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	updates := make(map[string]interface{})
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Bio != "" {
		updates["bio"] = req.Bio
	}
	if req.Location != "" {
		updates["location"] = req.Location
	}
	if req.Website != "" {
		updates["website"] = req.Website
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有要更新的内容"})
		return
	}

	if err := database.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// bindEmail 绑定/更新邮箱（需要验证码）
func (s *Server) bindEmail(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		Email            string `json:"email" binding:"required,email"`
		VerificationCode string `json:"verification_code" binding:"required,len=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	ctx := context.Background()

	// 验证邮箱验证码
	valid, err := s.verificationService.VerifyCode(ctx, req.Email, req.VerificationCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证码校验失败"})
		return
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误或已过期"})
		return
	}

	// 检查邮箱是否已被其他用户使用
	var existing model.User
	if err := database.DB.Where("email = ? AND id != ?", req.Email, userID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被其他用户使用"})
		return
	}

	// 更新邮箱并标记为已验证
	updates := map[string]interface{}{
		"email":          req.Email,
		"email_verified": true,
	}
	if err := database.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新邮箱失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "邮箱绑定成功"})
}

// getUserProfile 获取指定用户的公开信息
func (s *Server) getUserProfile(c *gin.Context) {
	userID := c.Param("id")

	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 动态计算统计数据
	var postCount int64
	database.DB.Model(&model.Post{}).Where("author_id = ? AND status = ? AND review_status = ?", userID, "published", "approved").Count(&postCount)

	var storyCount int64
	database.DB.Model(&model.Story{}).Where("user_id = ?", userID).Count(&storyCount)

	var profileCount int64
	database.DB.Model(&model.Profile{}).Where("user_id = ?", userID).Count(&profileCount)

	// 检查是否是查看自己的资料
	currentUserID, exists := c.Get("userID")
	isOwnProfile := exists && currentUserID.(uint) == user.ID

	response := gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"avatar":        user.Avatar,
		"role":          user.Role,
		"bio":           user.Bio,
		"location":      user.Location,
		"website":       user.Website,
		"post_count":    postCount,
		"story_count":   storyCount,
		"profile_count": profileCount,
		"created_at":    user.CreatedAt,
	}

	// 如果是查看自己的资料，返回敏感信息
	if isOwnProfile {
		response["email"] = user.Email
		response["email_verified"] = user.EmailVerified
	}

	c.JSON(http.StatusOK, response)
}

// getUserGuilds 获取用户加入的公会列表
func (s *Server) getUserGuilds(c *gin.Context) {
	userID := c.Param("id")

	var memberships []model.GuildMember
	if err := database.DB.Where("user_id = ?", userID).Find(&memberships).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 获取公会详情
	var guilds []gin.H
	for _, membership := range memberships {
		var guild model.Guild
		if err := database.DB.First(&guild, membership.GuildID).Error; err != nil {
			continue
		}
		// 只显示已通过审核的公会，或者用户是会长的待审核公会
		if guild.Status != "approved" && membership.Role != "owner" {
			continue
		}
		guilds = append(guilds, gin.H{
			"id":           guild.ID,
			"name":         guild.Name,
			"icon":         guild.Icon,
			"color":        guild.Color,
			"member_count": guild.MemberCount,
			"status":       guild.Status,
			"role":         membership.Role,
			"joined_at":    membership.JoinedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"guilds": guilds})
}

// uploadImage 通用图片上传（返回base64）
func (s *Server) uploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择图片文件"})
		return
	}
	defer file.Close()

	// 检查文件大小 (最大 20MB)
	if header.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片文件不能超过20MB"})
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
	imageURL := "data:" + contentType + ";base64," + base64Data

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"url": imageURL,
		},
	})
}
