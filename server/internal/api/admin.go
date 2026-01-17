package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

// listUsers 获取用户列表（支持分页和筛选）
func (s *Server) listUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	role := c.Query("role")
	keyword := c.Query("keyword")

	query := database.DB.Model(&model.User{})

	if role != "" {
		query = query.Where("role = ?", role)
	}
	if keyword != "" {
		query = query.Where("username LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * pageSize
	var users []model.User
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users)

	// 隐藏敏感信息
	type SafeUser struct {
		ID          uint       `json:"id"`
		Username    string     `json:"username"`
		Email       string     `json:"email"`
		Avatar      string     `json:"avatar"`
		Role        string     `json:"role"`
		IsMuted     bool       `json:"is_muted"`
		MutedUntil  *time.Time `json:"muted_until"`
		MuteReason  string     `json:"mute_reason"`
		IsBanned    bool       `json:"is_banned"`
		BannedUntil *time.Time `json:"banned_until"`
		BanReason   string     `json:"ban_reason"`
		PostCount   int        `json:"post_count"`
		CreatedAt   time.Time  `json:"created_at"`
	}
	result := make([]SafeUser, len(users))
	for i, u := range users {
		result[i] = SafeUser{
			ID:          u.ID,
			Username:    u.Username,
			Email:       u.Email,
			Avatar:      u.Avatar,
			Role:        u.Role,
			IsMuted:     u.IsMuted,
			MutedUntil:  u.MutedUntil,
			MuteReason:  u.MuteReason,
			IsBanned:    u.IsBanned,
			BannedUntil: u.BannedUntil,
			BanReason:   u.BanReason,
			PostCount:   u.PostCount,
			CreatedAt:   u.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"users": result, "total": total})
}

// setUserRole 设置用户角色
func (s *Server) setUserRole(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var user model.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 不能修改管理员角色
	if user.Role == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能修改管理员角色"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 只允许设置为 user 或 moderator
	if req.Role != "user" && req.Role != "moderator" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色"})
		return
	}

	user.Role = req.Role
	database.DB.Save(&user)

	// 记录日志
	logAdminAction(c, "set_role", "user", uint(id), user.Username, map[string]interface{}{
		"new_role": req.Role,
	})

	c.JSON(http.StatusOK, gin.H{"message": "角色已更新", "user": gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	}})
}

// ========== 用户封禁管理 ==========

// MuteUserRequest 禁言请求
type MuteUserRequest struct {
	Duration int    `json:"duration"` // 禁言时长（小时），0=永久
	Reason   string `json:"reason"`   // 禁言原因
}

// muteUser 禁言用户
func (s *Server) muteUser(c *gin.Context) {
	modID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var user model.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 不能禁言管理员或版主
	if user.Role == "admin" || user.Role == "moderator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能禁言管理员或版主"})
		return
	}

	var req MuteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	user.IsMuted = true
	user.MuteReason = req.Reason
	user.BannedBy = &modID
	user.BannedAt = &now

	if req.Duration > 0 {
		until := now.Add(time.Duration(req.Duration) * time.Hour)
		user.MutedUntil = &until
	} else {
		user.MutedUntil = nil // 永久禁言
	}

	database.DB.Save(&user)

	// 记录日志
	durationStr := "永久"
	if req.Duration > 0 {
		durationStr = strconv.Itoa(req.Duration) + "小时"
	}
	logAdminAction(c, "mute_user", "user", uint(id), user.Username, map[string]interface{}{
		"duration": durationStr,
		"reason":   req.Reason,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "用户已禁言",
		"user": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"is_muted":    user.IsMuted,
			"muted_until": user.MutedUntil,
			"mute_reason": user.MuteReason,
		},
	})
}

// unmuteUser 解除禁言
func (s *Server) unmuteUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var user model.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	user.IsMuted = false
	user.MutedUntil = nil
	user.MuteReason = ""

	database.DB.Save(&user)

	// 记录日志
	logAdminAction(c, "unmute_user", "user", uint(id), user.Username, nil)

	c.JSON(http.StatusOK, gin.H{
		"message": "已解除禁言",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"is_muted": user.IsMuted,
		},
	})
}

// BanUserRequest 禁止登录请求
type BanUserRequest struct {
	Duration int    `json:"duration"` // 封禁时长（小时），0=永久
	Reason   string `json:"reason"`   // 封禁原因
}

// banUser 禁止用户登录
func (s *Server) banUser(c *gin.Context) {
	modID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var user model.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 不能封禁管理员或版主
	if user.Role == "admin" || user.Role == "moderator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能封禁管理员或版主"})
		return
	}

	var req BanUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	user.IsBanned = true
	user.BanReason = req.Reason
	user.BannedBy = &modID
	user.BannedAt = &now

	if req.Duration > 0 {
		until := now.Add(time.Duration(req.Duration) * time.Hour)
		user.BannedUntil = &until
	} else {
		user.BannedUntil = nil // 永久封禁
	}

	database.DB.Save(&user)

	// 记录日志
	durationStr := "永久"
	if req.Duration > 0 {
		durationStr = strconv.Itoa(req.Duration) + "小时"
	}
	logAdminAction(c, "ban_user", "user", uint(id), user.Username, map[string]interface{}{
		"duration": durationStr,
		"reason":   req.Reason,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "用户已被禁止登录",
		"user": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"is_banned":    user.IsBanned,
			"banned_until": user.BannedUntil,
			"ban_reason":   user.BanReason,
		},
	})
}

// unbanUser 解除登录禁止
func (s *Server) unbanUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var user model.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	user.IsBanned = false
	user.BannedUntil = nil
	user.BanReason = ""

	database.DB.Save(&user)

	// 记录日志
	logAdminAction(c, "unban_user", "user", uint(id), user.Username, nil)

	c.JSON(http.StatusOK, gin.H{
		"message": "已解除登录禁止",
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"is_banned": user.IsBanned,
		},
	})
}

// ========== 用户内容管理 ==========

// disableUserPosts 禁用用户所有帖子（设为不可见）
func (s *Server) disableUserPosts(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var user model.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 不能操作管理员或版主
	if user.Role == "admin" || user.Role == "moderator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能操作管理员或版主的内容"})
		return
	}

	// 将用户所有帖子设为removed状态
	result := database.DB.Model(&model.Post{}).
		Where("author_id = ?", id).
		Updates(map[string]interface{}{
			"status":        "removed",
			"review_status": "rejected",
		})

	// 记录日志
	logAdminAction(c, "disable_posts", "user", uint(id), user.Username, map[string]interface{}{
		"affected_count": result.RowsAffected,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":        "已禁用该用户所有帖子",
		"affected_count": result.RowsAffected,
	})
}

// deleteUserPosts 删除用户所有帖子
func (s *Server) deleteUserPosts(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var user model.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 不能操作管理员或版主
	if user.Role == "admin" || user.Role == "moderator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能操作管理员或版主的内容"})
		return
	}

	// 获取用户所有帖子ID
	var postIDs []uint
	database.DB.Model(&model.Post{}).Where("author_id = ?", id).Pluck("id", &postIDs)

	if len(postIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":        "该用户没有帖子",
			"affected_count": 0,
		})
		return
	}

	// 删除关联数据
	database.DB.Where("post_id IN ?", postIDs).Delete(&model.PostTag{})
	database.DB.Where("post_id IN ?", postIDs).Delete(&model.Comment{})
	database.DB.Where("post_id IN ?", postIDs).Delete(&model.PostLike{})
	database.DB.Where("post_id IN ?", postIDs).Delete(&model.PostFavorite{})

	// 删除帖子
	result := database.DB.Where("author_id = ?", id).Delete(&model.Post{})

	// 更新用户帖子计数
	database.DB.Model(&user).Update("post_count", 0)

	// 记录日志
	logAdminAction(c, "delete_posts", "user", uint(id), user.Username, map[string]interface{}{
		"affected_count": result.RowsAffected,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":        "已删除该用户所有帖子",
		"affected_count": result.RowsAffected,
	})
}
