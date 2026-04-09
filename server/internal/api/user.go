package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/cache"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
	"github.com/rpbox/server/pkg/validator"
	"gorm.io/gorm"
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
	activity := buildUserActivityPayload(user, loadUserActivitySnapshot(user.ID, time.Now()))

	// 返回头像 URL 而不是 base64 数据
	avatarURL := userAvatarURL(s.cfg.Server.ApiHost, user)

	response := struct {
		ID                 uint      `json:"id"`
		Username           string    `json:"username"`
		Email              string    `json:"email"`
		Avatar             string    `json:"avatar"`
		AvatarReviewStatus string    `json:"avatar_review_status"`
		Role               string    `json:"role"`
		IsSponsor          bool      `json:"is_sponsor"`
		SponsorLevel       int       `json:"sponsor_level"`
		SponsorColor       string    `json:"sponsor_color"`
		SponsorBold        bool      `json:"sponsor_bold"`
		NameColor          string    `json:"name_color"`
		NameBold           bool      `json:"name_bold"`
		Bio                string    `json:"bio"`
		Location           string    `json:"location"`
		Website            string    `json:"website"`
		PostCount          int64     `json:"post_count"`
		StoryCount         int64     `json:"story_count"`
		ProfileCount       int64     `json:"profile_count"`
		CreatedAt          time.Time `json:"created_at"`
		userActivityPayload
	}{
		ID:                  user.ID,
		Username:            user.Username,
		Email:               user.Email,
		Avatar:              avatarURL,
		AvatarReviewStatus:  user.AvatarReviewStatus,
		Role:                user.Role,
		IsSponsor:           level > sponsorLevelNone,
		SponsorLevel:        level,
		SponsorColor:        user.SponsorColor,
		SponsorBold:         user.SponsorBold,
		NameColor:           nameColor,
		NameBold:            nameBold,
		Bio:                 user.Bio,
		Location:            user.Location,
		Website:             user.Website,
		PostCount:           postCount,
		StoryCount:          storyCount,
		ProfileCount:        profileCount,
		CreatedAt:           user.CreatedAt,
		userActivityPayload: activity,
	}

	c.JSON(http.StatusOK, response)
}

// signInDaily 处理每日签到。
func (s *Server) signInDaily(c *gin.Context) {
	userID := c.GetUint("userID")
	now := time.Now()

	result, err := func() (service.RewardResult, error) {
		var output service.RewardResult
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			reward, txErr := service.ApplyDailySignIn(tx, userID, now)
			if txErr != nil {
				return txErr
			}
			output = reward
			return nil
		})
		return output, err
	}()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签到失败"})
		return
	}

	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	activity := buildUserActivityPayload(user, loadUserActivitySnapshot(user.ID, now))
	c.JSON(http.StatusOK, gin.H{
		"message":                map[bool]string{true: "签到成功", false: "今天已经签到过了"}[result.Granted],
		"granted":                result.Granted,
		"points_delta":           result.PointsDelta,
		"experience_delta":       result.ExperienceDelta,
		"activity_points":        activity.ActivityPoints,
		"activity_experience":    activity.ActivityExperience,
		"forum_level":            activity.ForumLevel,
		"forum_level_name":       activity.ForumLevelName,
		"forum_level_color":      activity.ForumLevelColor,
		"forum_level_bold":       activity.ForumLevelBold,
		"current_level_exp":      activity.CurrentLevelExp,
		"next_level_exp":         activity.NextLevelExp,
		"level_progress_percent": activity.LevelProgressPercent,
		"signed_in_today":        activity.SignedInToday,
	})
}

// updateAvatar 更新用户头像
func (s *Server) updateAvatar(c *gin.Context) {
	userID := c.GetUint("userID")

	var user model.User
	if err := database.DB.Select("id", "avatar_review_status", "avatar_change_count", "activity_points").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if user.AvatarReviewStatus == "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "你最多只能同时有1个待审核头像申请"})
		return
	}

	header, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择头像文件"})
		return
	}

	// 检查文件大小 (最大 20MB)
	if header.Size > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "头像文件不能超过20MB"})
		return
	}

	if user.AvatarChangeCount > 0 && user.ActivityPoints < service.AvatarChangeCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("积分不足，修改头像需要%d积分", service.AvatarChangeCost)})
		return
	}

	avatarURL, err := s.saveUploadedImage(c, header, fmt.Sprintf("users/%d/avatar", userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存头像失败"})
		return
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var current model.User
		if err := tx.Select("id", "avatar_change_count", "activity_points").First(&current, userID).Error; err != nil {
			return err
		}
		if current.AvatarChangeCount > 0 {
			if _, spendErr := service.SpendActivityPoints(tx, userID, "avatar_change_cost", fmt.Sprintf("avatar-change:%d", current.AvatarChangeCount+1), service.AvatarChangeCost); spendErr != nil {
				return spendErr
			}
		}

		updates := map[string]interface{}{
			"avatar":                avatarURL,
			"avatar_review_status":  "pending",
			"avatar_reviewer_id":    nil,
			"avatar_reviewed_at":    nil,
			"avatar_review_comment": "",
			"avatar_change_count":   current.AvatarChangeCount + 1,
		}
		return tx.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
	})
	if err != nil {
		if errors.Is(err, service.ErrInsufficientPoints) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("积分不足，修改头像需要%d积分", service.AvatarChangeCost)})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新头像失败"})
		return
	}
	s.invalidateUserProfileCache(c.Request.Context(), userID)

	var refreshed model.User
	_ = database.DB.First(&refreshed, userID).Error
	activity := buildUserActivityPayload(refreshed, loadUserActivitySnapshot(refreshed.ID, time.Now()))
	c.JSON(http.StatusOK, gin.H{
		"message":                 "头像已提交审核",
		"avatar":                  avatarURL,
		"avatar_review_status":    "pending",
		"activity_points":         activity.ActivityPoints,
		"avatar_change_count":     activity.AvatarChangeCount,
		"next_avatar_change_cost": activity.NextAvatarChangeCost,
	})
}

// updateUserInfo 更新用户信息
func (s *Server) updateUserInfo(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		Username            string  `json:"username"`
		Email               string  `json:"email"`
		Bio                 string  `json:"bio"`
		Location            string  `json:"location"`
		Website             string  `json:"website"`
		SponsorColor        *string `json:"sponsor_color"`
		SponsorBold         *bool   `json:"sponsor_bold"`
		NameStylePreference *string `json:"name_style_preference"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	req.Username = strings.TrimSpace(req.Username)

	var normalizedSponsorColor *string
	if req.SponsorColor != nil {
		if *req.SponsorColor == "" {
			empty := ""
			normalizedSponsorColor = &empty
		} else {
			normalized := normalizeHexValue(*req.SponsorColor)
			if normalized == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "颜色格式无效"})
				return
			}
			normalizedSponsorColor = &normalized
		}
	}

	if req.NameStylePreference != nil {
		preference := strings.ToLower(strings.TrimSpace(*req.NameStylePreference))
		if preference == "level" {
			preference = "default"
		}
		if preference != "default" && preference != "sponsor" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "昵称样式无效"})
			return
		}
		req.NameStylePreference = &preference
	}

	updated := false
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Select("id", "username", "username_change_count", "is_sponsor", "sponsor_level", "sponsor_color", "sponsor_bold").
			First(&user, userID).Error; err != nil {
			return err
		}

		updates := make(map[string]interface{})
		if req.Username != "" && req.Username != user.Username {
			if len([]rune(req.Username)) < 3 || len([]rune(req.Username)) > 50 {
				return &apiError{status: http.StatusBadRequest, message: "用户名长度需在3到50个字符之间"}
			}

			var existingCount int64
			if err := tx.Model(&model.User{}).Where("username = ? AND id != ?", req.Username, userID).Count(&existingCount).Error; err != nil {
				return err
			}
			if existingCount > 0 {
				return &apiError{status: http.StatusConflict, message: "用户名已存在"}
			}

			if user.UsernameChangeCount > 0 {
				if _, spendErr := service.SpendActivityPoints(tx, userID, "username_change_cost", fmt.Sprintf("username-change:%d", user.UsernameChangeCount+1), service.UsernameChangeCost); spendErr != nil {
					return spendErr
				}
			}
			updates["username"] = req.Username
			updates["username_change_count"] = user.UsernameChangeCount + 1
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

		if normalizedSponsorColor != nil || req.SponsorBold != nil || req.NameStylePreference != nil {
			if resolveSponsorLevel(user) < sponsorLevelStyle {
				if normalizedSponsorColor != nil || req.SponsorBold != nil || (req.NameStylePreference != nil && *req.NameStylePreference == "sponsor") {
					return &apiError{status: http.StatusForbidden, message: "无权操作赞助者样式"}
				}
			}
			if normalizedSponsorColor != nil {
				updates["sponsor_color"] = *normalizedSponsorColor
			}
			if req.SponsorBold != nil {
				updates["sponsor_bold"] = *req.SponsorBold
			}
			if req.NameStylePreference != nil {
				updates["name_style_preference"] = *req.NameStylePreference
			}
		} else if req.NameStylePreference != nil {
			updates["name_style_preference"] = *req.NameStylePreference
		}

		if len(updates) == 0 {
			return &apiError{status: http.StatusBadRequest, message: "没有要更新的内容"}
		}

		if err := tx.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
			return err
		}
		updated = true
		return nil
	})
	if err != nil {
		if apiErr, ok := err.(*apiError); ok {
			c.JSON(apiErr.status, gin.H{"error": apiErr.message})
			return
		}
		if errors.Is(err, service.ErrInsufficientPoints) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("积分不足，修改用户名需要%d积分", service.UsernameChangeCost)})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	if !updated {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有要更新的内容"})
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

type publicUserProfileResponse struct {
	ID                   uint      `json:"id"`
	Username             string    `json:"username"`
	Avatar               string    `json:"avatar"`
	Role                 string    `json:"role"`
	IsSponsor            bool      `json:"is_sponsor"`
	SponsorLevel         int       `json:"sponsor_level"`
	NameColor            string    `json:"name_color"`
	NameBold             bool      `json:"name_bold"`
	Bio                  string    `json:"bio"`
	Location             string    `json:"location"`
	Website              string    `json:"website"`
	PostCount            int64     `json:"post_count"`
	StoryCount           int64     `json:"story_count"`
	ProfileCount         int64     `json:"profile_count"`
	CreatedAt            time.Time `json:"created_at"`
	Email                string    `json:"email,omitempty"`
	ForumLevel           int       `json:"forum_level"`
	ForumLevelName       string    `json:"forum_level_name"`
	ForumLevelColor      string    `json:"forum_level_color"`
	ForumLevelBold       bool      `json:"forum_level_bold"`
	CurrentLevelExp      int       `json:"current_level_exp"`
	NextLevelExp         int       `json:"next_level_exp"`
	LevelProgressPercent int       `json:"level_progress_percent"`
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

func buildPublicUserProfile(apiHost string, user model.User, counts userProfileCounts) publicUserProfileResponse {
	nameColor, nameBold := userDisplayStyle(user)
	level := resolveSponsorLevel(user)
	levelInfo := resolveForumLevelInfo(user.ActivityExperience)
	response := publicUserProfileResponse{
		ID:                   user.ID,
		Username:             user.Username,
		Avatar:               userAvatarURL(apiHost, user),
		Role:                 user.Role,
		IsSponsor:            level > sponsorLevelNone,
		SponsorLevel:         level,
		NameColor:            nameColor,
		NameBold:             nameBold,
		Bio:                  user.Bio,
		Location:             user.Location,
		Website:              user.Website,
		PostCount:            counts.PostCount,
		StoryCount:           counts.StoryCount,
		ProfileCount:         counts.ProfileCount,
		CreatedAt:            user.CreatedAt,
		ForumLevel:           levelInfo.Level,
		ForumLevelName:       levelInfo.Name,
		ForumLevelColor:      levelInfo.Color,
		ForumLevelBold:       levelInfo.Bold,
		CurrentLevelExp:      levelInfo.CurrentLevelExp,
		NextLevelExp:         levelInfo.NextLevelExp,
		LevelProgressPercent: levelInfo.ProgressPercent,
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
		var cached publicUserProfileResponse
		cacheKey := s.userProfileCacheKey(userID)
		err := s.cache.Fetch(c.Request.Context(), cacheKey, cache.TTL["user"], &cached, func(ctx context.Context) (interface{}, error) {
			user, counts, err := fetchUserProfileData(ctx, userID)
			if err != nil {
				return nil, err
			}
			return buildPublicUserProfile(s.cfg.Server.ApiHost, user, counts), nil
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

	publicProfile := buildPublicUserProfile(s.cfg.Server.ApiHost, user, counts)

	// 如果是查看自己的资料，返回敏感信息
	if isOwnProfile {
		activity := buildUserActivityPayload(user, loadUserActivitySnapshot(user.ID, time.Now()))
		response := gin.H{
			"id":                        publicProfile.ID,
			"username":                  publicProfile.Username,
			"avatar":                    publicProfile.Avatar,
			"role":                      publicProfile.Role,
			"is_sponsor":                publicProfile.IsSponsor,
			"sponsor_level":             publicProfile.SponsorLevel,
			"name_color":                publicProfile.NameColor,
			"name_bold":                 publicProfile.NameBold,
			"bio":                       publicProfile.Bio,
			"location":                  publicProfile.Location,
			"website":                   publicProfile.Website,
			"post_count":                publicProfile.PostCount,
			"story_count":               publicProfile.StoryCount,
			"profile_count":             publicProfile.ProfileCount,
			"created_at":                publicProfile.CreatedAt,
			"email":                     user.Email,
			"email_verified":            user.EmailVerified,
			"sponsor_color":             user.SponsorColor,
			"sponsor_bold":              user.SponsorBold,
			"activity_points":           activity.ActivityPoints,
			"activity_experience":       activity.ActivityExperience,
			"forum_level":               publicProfile.ForumLevel,
			"forum_level_name":          publicProfile.ForumLevelName,
			"forum_level_color":         publicProfile.ForumLevelColor,
			"forum_level_bold":          publicProfile.ForumLevelBold,
			"current_level_exp":         publicProfile.CurrentLevelExp,
			"next_level_exp":            publicProfile.NextLevelExp,
			"level_progress_percent":    publicProfile.LevelProgressPercent,
			"signed_in_today":           activity.SignedInToday,
			"name_style_preference":     activity.NameStylePreference,
			"avatar_change_count":       activity.AvatarChangeCount,
			"username_change_count":     activity.UsernameChangeCount,
			"next_avatar_change_cost":   activity.NextAvatarChangeCost,
			"next_username_change_cost": activity.NextUsernameChangeCost,
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

type extraSponsor struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Avatar       string `json:"avatar"`
	Role         string `json:"role"`
	IsSponsor    bool   `json:"is_sponsor"`
	SponsorLevel int    `json:"sponsor_level"`
	NameColor    string `json:"name_color"`
	NameBold     bool   `json:"name_bold"`
}

const defaultSponsorRole = "赞助支持"

func sponsorNameKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizeSponsorDisplayRole(role string) string {
	trimmed := strings.TrimSpace(role)
	switch strings.ToLower(trimmed) {
	case "", "user", "admin", "moderator":
		return defaultSponsorRole
	default:
		return trimmed
	}
}

// listSponsors 获取赞助者名单
func (s *Server) listSponsors(c *gin.Context) {
	// 先读取文件维护名单，支持按用户名覆盖默认文案，便于后续直接编辑 JSON。
	extraSponsors := loadExtraSponsors()
	extraRoleByName := make(map[string]string, len(extraSponsors))
	for _, sp := range extraSponsors {
		key := sponsorNameKey(sp.Username)
		if key == "" {
			continue
		}
		if role := strings.TrimSpace(sp.Role); role != "" {
			extraRoleByName[key] = role
		}
	}

	var users []model.User
	if err := database.DB.
		Select("id", "username", "avatar", "avatar_review_status", "role", "is_sponsor", "sponsor_level", "sponsor_color", "sponsor_bold", "name_style_preference", "activity_experience", "created_at").
		Where("sponsor_level > ? OR is_sponsor = ?", 0, true).
		Order("sponsor_level DESC, created_at ASC").
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	type SponsorUser struct {
		ID           uint   `json:"id"`
		Username     string `json:"username"`
		Avatar       string `json:"avatar"`
		Role         string `json:"role"`
		IsSponsor    bool   `json:"is_sponsor"`
		SponsorLevel int    `json:"sponsor_level"`
		NameColor    string `json:"name_color"`
		NameBold     bool   `json:"name_bold"`
	}

	result := make([]SponsorUser, 0, len(users))
	nameSet := make(map[string]struct{}, len(users))
	for _, user := range users {
		nameColor, nameBold := userDisplayStyle(user)
		level := resolveSponsorLevel(user)
		username := strings.TrimSpace(user.Username)
		nameKey := sponsorNameKey(username)
		if nameKey != "" {
			nameSet[nameKey] = struct{}{}
		}
		role := normalizeSponsorDisplayRole(user.Role)
		if override, ok := extraRoleByName[nameKey]; ok {
			role = normalizeSponsorDisplayRole(override)
		}

		result = append(result, SponsorUser{
			ID:           user.ID,
			Username:     user.Username,
			Avatar:       userAvatarURL(s.cfg.Server.ApiHost, user),
			Role:         role,
			IsSponsor:    level > sponsorLevelNone,
			SponsorLevel: level,
			NameColor:    nameColor,
			NameBold:     nameBold,
		})
	}

	// 合并额外鸣谢名单（文件维护，便于 CI 上传）
	for _, sp := range extraSponsors {
		nameKey := sponsorNameKey(sp.Username)
		if nameKey == "" {
			continue
		}
		if _, exists := nameSet[nameKey]; exists {
			continue
		}
		if sp.SponsorLevel <= 0 {
			sp.SponsorLevel = 1
		}
		sp.Role = normalizeSponsorDisplayRole(sp.Role)
		sp.ID = 900000 + uint(len(result)+1)
		result = append(result, SponsorUser{
			ID:           sp.ID,
			Username:     sp.Username,
			Avatar:       sp.Avatar,
			Role:         sp.Role,
			IsSponsor:    sp.IsSponsor,
			SponsorLevel: sp.SponsorLevel,
			NameColor:    sp.NameColor,
			NameBold:     sp.NameBold,
		})
		nameSet[nameKey] = struct{}{}
	}

	c.JSON(http.StatusOK, gin.H{"users": result})
}

func loadExtraSponsors() []extraSponsor {
	path := "storage/sponsors/extra_sponsors.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var sponsors []extraSponsor
	if err := json.Unmarshal(raw, &sponsors); err != nil {
		return nil
	}
	return sponsors
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
