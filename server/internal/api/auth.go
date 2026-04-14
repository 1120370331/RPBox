package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/pkg/auth"
	"github.com/rpbox/server/pkg/email"
	"github.com/rpbox/server/pkg/validator"
)

type RegisterRequest struct {
	Username         string `json:"username" binding:"required,min=3,max=50"`
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required,min=6"`
	VerificationCode string `json:"verification_code" binding:"required,len=6"`
	AcceptTerms      bool   `json:"accept_terms"`
	AcceptPrivacy    bool   `json:"accept_privacy"`
	AgreementVersion string `json:"agreement_version"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 支持用户名或邮箱
	Password string `json:"password" binding:"required"`
}

type SendCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Email            string `json:"email" binding:"required,email"`
	VerificationCode string `json:"verification_code" binding:"required,len=6"`
	NewPassword      string `json:"new_password" binding:"required,min=6"`
}

// sendVerificationCode 发送邮箱验证码
func (s *Server) sendVerificationCode(c *gin.Context) {
	var req SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	// 验证邮箱格式
	if !email.ValidateEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式不正确"})
		return
	}

	// 检查邮箱是否已被其他用户注册（且已验证）
	var existing model.User
	if err := database.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		// 如果邮箱已存在且已验证，不允许再注册
		if existing.EmailVerified {
			c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被注册"})
			return
		}
		// 如果邮箱存在但未验证，允许发送验证码（用于验证现有邮箱）
	}

	ctx := context.Background()

	// 检查发送频率限制
	canSend, err := s.verificationService.CheckRateLimit(ctx, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查发送频率失败"})
		return
	}
	if !canSend {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "请等待1分钟后再试"})
		return
	}

	// 生成验证码
	code, err := s.verificationService.GenerateCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成验证码失败"})
		return
	}

	// 保存验证码到Redis
	if err := s.verificationService.SaveCode(ctx, req.Email, code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存验证码失败"})
		return
	}

	// 发送邮件
	if err := s.emailClient.SendVerificationCode(req.Email, code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送邮件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "验证码已发送到您的邮箱",
	})
}

func (s *Server) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	if !req.AcceptTerms || !req.AcceptPrivacy {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先阅读并同意服务条款和隐私政策"})
		return
	}

	ctx := context.Background()
	valid, err := s.verificationService.VerifyCode(ctx, req.Email, req.VerificationCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证码校验失败"})
		return
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误或已过期"})
		return
	}

	// 检查用户名是否存在
	var existing model.User
	if err := database.DB.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	// 检查邮箱是否已注册
	if err := database.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "邮箱已被注册"})
		return
	}

	agreementVersion := req.AgreementVersion
	if agreementVersion == "" {
		agreementVersion = time.Now().Format("2006-01-02")
	}
	acceptedAt := time.Now()

	// 哈希密码
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	user := model.User{
		Username:          req.Username,
		Email:             req.Email,
		PassHash:          hash,
		EmailVerified:     true,
		TermsAcceptedAt:   &acceptedAt,
		PrivacyAcceptedAt: &acceptedAt,
		AgreementVersion:  agreementVersion,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	log.Printf("[Auth] register success user_id=%d username=%s ip=%s", user.ID, user.Username, c.ClientIP())

	c.JSON(http.StatusCreated, gin.H{
		"message": "注册成功",
		"user_id": user.ID,
	})
}

func (s *Server) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	ip := c.ClientIP()
	var user model.User
	// 支持用户名或邮箱登录
	if err := database.DB.Where("username = ? OR email = ?", req.Username, req.Username).First(&user).Error; err != nil {
		log.Printf("[Auth] login failed user=%s ip=%s reason=user_not_found", req.Username, ip)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	if user.AccountDeletedAt != nil {
		log.Printf("[Auth] login failed user=%s ip=%s reason=account_deleted", req.Username, ip)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号已删除"})
		return
	}

	if !auth.CheckPassword(req.Password, user.PassHash) {
		log.Printf("[Auth] login failed user=%s ip=%s reason=bad_password", req.Username, ip)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 检查封禁状态
	if user.IsBanned {
		// 检查封禁是否已过期
		if user.BannedUntil != nil && user.BannedUntil.Before(time.Now()) {
			// 封禁已过期，自动解除
			user.IsBanned = false
			user.BannedUntil = nil
			user.BanReason = ""
			database.DB.Save(&user)
		} else {
			// 仍在封禁中
			log.Printf("[Auth] login blocked user_id=%d username=%s ip=%s reason=banned", user.ID, user.Username, ip)
			msg := "账号已被封禁"
			if user.BanReason != "" {
				msg += "，原因：" + user.BanReason
			}
			if user.BannedUntil != nil {
				msg += "，解封时间：" + user.BannedUntil.Format("2006-01-02 15:04")
			} else {
				msg += "（永久）"
			}
			c.JSON(http.StatusForbidden, gin.H{"error": msg})
			return
		}
	}

	nameColor, nameBold := userDisplayStyle(user)
	level := resolveSponsorLevel(user)
	activity := buildUserActivityPayload(user, loadUserActivitySnapshot(user.ID, time.Now()))
	token, err := auth.GenerateToken(user.ID, user.Username, s.cfg.JWT.Expire)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	log.Printf("[Auth] login success user_id=%d username=%s ip=%s verified=%t", user.ID, user.Username, ip, user.EmailVerified)

	// 返回头像 URL 而不是 base64 数据，避免 localStorage 配额超限
	avatarURL := userAvatarURL(s.cfg.Server.ApiHost, user)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":                        user.ID,
			"username":                  user.Username,
			"email":                     user.Email,
			"avatar":                    avatarURL,
			"avatar_review_status":      user.AvatarReviewStatus,
			"role":                      user.Role,
			"is_sponsor":                level > sponsorLevelNone,
			"sponsor_level":             level,
			"sponsor_color":             user.SponsorColor,
			"sponsor_bold":              user.SponsorBold,
			"name_color":                nameColor,
			"name_bold":                 nameBold,
			"activity_points":           activity.ActivityPoints,
			"activity_experience":       activity.ActivityExperience,
			"forum_level":               activity.ForumLevel,
			"forum_level_name":          activity.ForumLevelName,
			"forum_level_color":         activity.ForumLevelColor,
			"forum_level_bold":          activity.ForumLevelBold,
			"current_level_exp":         activity.CurrentLevelExp,
			"next_level_exp":            activity.NextLevelExp,
			"level_progress_percent":    activity.LevelProgressPercent,
			"signed_in_today":           activity.SignedInToday,
			"name_style_preference":     activity.NameStylePreference,
			"avatar_change_count":       activity.AvatarChangeCount,
			"username_change_count":     activity.UsernameChangeCount,
			"next_avatar_change_cost":   activity.NextAvatarChangeCost,
			"next_username_change_cost": activity.NextUsernameChangeCost,
		},
	})
}

// forgotPassword 发送重置密码验证码
func (s *Server) forgotPassword(c *gin.Context) {
	var req SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	// 验证邮箱格式
	if !email.ValidateEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱格式不正确"})
		return
	}

	// 检查邮箱是否已注册
	var user model.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "该邮箱未注册"})
		return
	}

	ctx := context.Background()

	// 检查发送频率限制
	canSend, err := s.verificationService.CheckRateLimit(ctx, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查发送频率失败"})
		return
	}
	if !canSend {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "请等待1分钟后再试"})
		return
	}

	// 生成验证码
	code, err := s.verificationService.GenerateCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成验证码失败"})
		return
	}

	// 保存验证码到Redis
	if err := s.verificationService.SaveCode(ctx, req.Email, code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存验证码失败"})
		return
	}

	// 发送邮件
	if err := s.emailClient.SendVerificationCode(req.Email, code); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送邮件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "重置密码验证码已发送到您的邮箱",
	})
}

// resetPassword 使用验证码重置密码
func (s *Server) resetPassword(c *gin.Context) {
	var req ResetPasswordRequest
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

	// 检查邮箱是否存在
	var user model.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "该邮箱未注册"})
		return
	}

	// 哈希新密码
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// 更新密码
	if err := database.DB.Model(&user).Update("pass_hash", hash).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置密码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "密码重置成功，请使用新密码登录",
	})
}
