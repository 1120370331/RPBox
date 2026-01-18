package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
)

// listNotifications 获取通知列表
func (s *Server) listNotifications(c *gin.Context) {
	userID := c.GetUint("userID")

	// 获取查询参数
	notifType := c.DefaultQuery("type", "all")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取通知列表
	notifications, total, err := service.GetNotifications(userID, notifType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取通知失败"})
		return
	}

	// 获取通知相关的用户信息（actor）
	actorIDs := make([]uint, 0)
	for _, notif := range notifications {
		if notif.ActorID != nil {
			actorIDs = append(actorIDs, *notif.ActorID)
		}
	}

	var users []model.User
	if len(actorIDs) > 0 {
		database.DB.Where("id IN ?", actorIDs).Find(&users)
	}
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 组装响应
	type NotificationWithActor struct {
		model.Notification
		ActorName   string `json:"actor_name,omitempty"`
		ActorAvatar string `json:"actor_avatar,omitempty"`
	}
	result := make([]NotificationWithActor, len(notifications))
	for i, notif := range notifications {
		item := NotificationWithActor{
			Notification: notif,
		}
		if notif.ActorID != nil {
			if actor, ok := userMap[*notif.ActorID]; ok {
				item.ActorName = actor.Username
				item.ActorAvatar = actor.Avatar
			}
		}
		result[i] = item
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": result,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	})
}

// markNotificationAsRead 标记通知为已读
func (s *Server) markNotificationAsRead(c *gin.Context) {
	userID := c.GetUint("userID")
	notificationID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查通知是否存在且属于当前用户
	var notification model.Notification
	if err := database.DB.First(&notification, notificationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "通知不存在"})
		return
	}

	if notification.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	// 标记为已读
	if err := service.MarkAsRead(uint(notificationID), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已标记为已读"})
}

// markAllNotificationsAsRead 标记所有通知为已读
func (s *Server) markAllNotificationsAsRead(c *gin.Context) {
	userID := c.GetUint("userID")

	if err := service.MarkAllAsRead(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已全部标记为已读"})
}

// getUnreadCount 获取未读通知数量
func (s *Server) getUnreadCount(c *gin.Context) {
	userID := c.GetUint("userID")

	count, err := service.GetUnreadCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// deleteNotification 删除单个通知
func (s *Server) deleteNotification(c *gin.Context) {
	userID := c.GetUint("userID")
	notificationID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查通知是否存在且属于当前用户
	var notification model.Notification
	if err := database.DB.First(&notification, notificationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "通知不存在"})
		return
	}

	if notification.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	// 删除通知
	if err := service.DeleteNotification(uint(notificationID), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// deleteAllNotifications 删除所有通知
func (s *Server) deleteAllNotifications(c *gin.Context) {
	userID := c.GetUint("userID")

	if err := service.DeleteAllNotifications(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已清空所有通知"})
}
