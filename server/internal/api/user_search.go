package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

// searchUsers 搜索用户（用于@提及）
func (s *Server) searchUsers(c *gin.Context) {
	userID := c.GetUint("userID")
	keyword := strings.TrimSpace(c.Query("q"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 20 {
		limit = 10
	}

	query := database.DB.Model(&model.User{}).Select("id", "username", "avatar", "role", "is_sponsor", "sponsor_level", "sponsor_color", "sponsor_bold")
	if keyword != "" {
		query = query.Where("username LIKE ?", "%"+keyword+"%")
	}
	query = query.Where("id != ?", userID).Order("username ASC").Limit(limit)

	var users []model.User
	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	type UserSummary struct {
		ID        uint   `json:"id"`
		Username  string `json:"username"`
		Avatar    string `json:"avatar"`
		NameColor string `json:"name_color"`
		NameBold  bool   `json:"name_bold"`
	}

	result := make([]UserSummary, len(users))
	for i, user := range users {
		nameColor, nameBold := userDisplayStyle(user)
		result[i] = UserSummary{
			ID:        user.ID,
			Username:  user.Username,
			Avatar:    user.Avatar,
			NameColor: nameColor,
			NameBold:  nameBold,
		}
	}

	c.JSON(http.StatusOK, gin.H{"users": result})
}
