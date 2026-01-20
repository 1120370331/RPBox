package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
	"github.com/rpbox/server/pkg/validator"
)

// testSendNotification 测试发送通知（仅用于开发测试）
func (s *Server) testSendNotification(c *gin.Context) {
	var req struct {
		UserID  uint   `json:"user_id" binding:"required"`
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	// 创建测试通知
	notification := &model.Notification{
		UserID:     req.UserID,
		Type:       "system",
		Content:    req.Content,
		TargetType: "system",
		TargetID:   0,
		IsRead:     false,
	}

	if err := service.CreateNotification(notification); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建通知失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "通知已发送",
		"id":      notification.ID,
	})
}
