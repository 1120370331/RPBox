package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/cache"
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

	nameColor, nameBold := userDisplayStyle(user)
	level := resolveSponsorLevel(user)
	c.JSON(http.StatusOK, gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"avatar":        user.Avatar,
		"role":          user.Role,
		"is_sponsor":    level > sponsorLevelNone,
		"sponsor_level": level,
		"sponsor_color": user.SponsorColor,
		"sponsor_bold":  user.SponsorBold,
		"name_color":    nameColor,
		"name_bold":     nameBold,
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
	s.invalidateUserProfileCache(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{
		"message": "头像更新成功",
		"avatar":  avatarURL,
	})
}

// updateUserInfo 更新用户信息
func (s *Server) updateUserInfo(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		Username     string  `json:"username"`
		Email        string  `json:"email"`
		Bio          string  `json:"bio"`
		Location     string  `json:"location"`
		Website      string  `json:"website"`
		SponsorColor *string `json:"sponsor_color"`
		SponsorBold  *bool   `json:"sponsor_bold"`
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
	if req.SponsorColor != nil || req.SponsorBold != nil {
		var user model.User
		if err := database.DB.Select("id", "is_sponsor", "sponsor_level").First(&user, userID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
			return
		}
		if resolveSponsorLevel(user) < sponsorLevelStyle {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作赞助者样式"})
			return
		}
		if req.SponsorColor != nil {
			if *req.SponsorColor == "" {
				updates["sponsor_color"] = ""
			} else {
				normalized := normalizeHexValue(*req.SponsorColor)
				if normalized == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "颜色格式无效"})
					return
				}
				updates["sponsor_color"] = normalized
			}
		}
		if req.SponsorBold != nil {
			updates["sponsor_bold"] = *req.SponsorBold
		}
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有要更新的内容"})
		return
	}

	if err := database.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	s.invalidateUserProfileCache(c.Request.Context(), userID)

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
	s.invalidateUserProfileCache(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{"message": "邮箱绑定成功"})
}

type userProfileCounts struct {
	PostCount    int64
	StoryCount   int64
	ProfileCount int64
}

type userProfileResponse struct {
	ID           uint      `json:"id"`
	Username     string    `json:"username"`
	Avatar       string    `json:"avatar"`
	Role         string    `json:"role"`
	IsSponsor    bool      `json:"is_sponsor"`
	SponsorLevel int       `json:"sponsor_level"`
	NameColor    string    `json:"name_color"`
	NameBold     bool      `json:"name_bold"`
	Bio          string    `json:"bio"`
	Location     string    `json:"location"`
	Website      string    `json:"website"`
	PostCount    int64     `json:"post_count"`
	StoryCount   int64     `json:"story_count"`
	ProfileCount int64     `json:"profile_count"`
	CreatedAt    time.Time `json:"created_at"`
	Email        string    `json:"email,omitempty"`
}

func fetchUserProfileData(ctx context.Context, userID string) (model.User, userProfileCounts, error) {
	var user model.User
	db := database.DB.WithContext(ctx)
	if err := db.First(&user, userID).Error; err != nil {
		return user, userProfileCounts{}, err
	}

	var counts userProfileCounts
	db.Model(&model.Post{}).Where("author_id = ? AND status = ? AND review_status = ?", userID, "published", "approved").Count(&counts.PostCount)
	db.Model(&model.Story{}).Where("user_id = ?", userID).Count(&counts.StoryCount)
	db.Model(&model.Profile{}).Where("user_id = ?", userID).Count(&counts.ProfileCount)

	return user, counts, nil
}

func buildPublicUserProfile(user model.User, counts userProfileCounts) userProfileResponse {
	nameColor, nameBold := userDisplayStyle(user)
	level := resolveSponsorLevel(user)
	response := userProfileResponse{
		ID:           user.ID,
		Username:     user.Username,
		Avatar:       user.Avatar,
		Role:         user.Role,
		IsSponsor:    level > sponsorLevelNone,
		SponsorLevel: level,
		NameColor:    nameColor,
		NameBold:     nameBold,
		Bio:          user.Bio,
		Location:     user.Location,
		Website:      user.Website,
		PostCount:    counts.PostCount,
		StoryCount:   counts.StoryCount,
		ProfileCount: counts.ProfileCount,
		CreatedAt:    user.CreatedAt,
	}

	if maskedEmail := user.MaskedEmail(); maskedEmail != "" {
		response.Email = maskedEmail
	}

	return response
}

// getUserProfile 获取指定用户的公开信息
func (s *Server) getUserProfile(c *gin.Context) {
	userID := c.Param("id")
	currentUserID, exists := c.Get("userID")
	isOwnProfile := exists && userID == fmt.Sprint(currentUserID.(uint))

	if !isOwnProfile && s.cache != nil {
		var cached userProfileResponse
		cacheKey := s.userProfileCacheKey(userID)
		err := s.cache.Fetch(c.Request.Context(), cacheKey, cache.TTL["user"], &cached, func(ctx context.Context) (interface{}, error) {
			user, counts, err := fetchUserProfileData(ctx, userID)
			if err != nil {
				return nil, err
			}
			return buildPublicUserProfile(user, counts), nil
		})
		if err == nil {
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	user, counts, err := fetchUserProfileData(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	publicProfile := buildPublicUserProfile(user, counts)

	// 如果是查看自己的资料，返回敏感信息
	if isOwnProfile {
		response := gin.H{
			"id":             publicProfile.ID,
			"username":       publicProfile.Username,
			"avatar":         publicProfile.Avatar,
			"role":           publicProfile.Role,
			"is_sponsor":     publicProfile.IsSponsor,
			"sponsor_level":  publicProfile.SponsorLevel,
			"name_color":     publicProfile.NameColor,
			"name_bold":      publicProfile.NameBold,
			"bio":            publicProfile.Bio,
			"location":       publicProfile.Location,
			"website":        publicProfile.Website,
			"post_count":     publicProfile.PostCount,
			"story_count":    publicProfile.StoryCount,
			"profile_count":  publicProfile.ProfileCount,
			"created_at":     publicProfile.CreatedAt,
			"email":          user.Email,
			"email_verified": user.EmailVerified,
			"sponsor_color":  user.SponsorColor,
			"sponsor_bold":   user.SponsorBold,
		}
		c.JSON(http.StatusOK, response)
		return
	}

	c.JSON(http.StatusOK, publicProfile)
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

// uploadImage 通用图片上传（返回URL）
func (s *Server) uploadImage(c *gin.Context) {
	header, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择图片文件"})
		return
	}

	// 检查文件大小 (最大 20MB)
	if header.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片文件不能超过20MB"})
		return
	}

	// 检查文件类型
	contentType := header.Header.Get("Content-Type")
	if contentType != "" && !strings.HasPrefix(contentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只支持图片格式"})
		return
	}

	imageURL, err := s.saveUploadedImage(c, header, "images")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存图片失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"url": imageURL,
		},
	})
}
