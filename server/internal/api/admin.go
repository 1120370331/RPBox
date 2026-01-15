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
		ID        uint      `json:"id"`
		Username  string    `json:"username"`
		Email     string    `json:"email"`
		Avatar    string    `json:"avatar"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
	}
	result := make([]SafeUser, len(users))
	for i, u := range users {
		result[i] = SafeUser{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			Avatar:    u.Avatar,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
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

	c.JSON(http.StatusOK, gin.H{"message": "角色已更新", "user": gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	}})
}
