package service

import (
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	ws "github.com/rpbox/server/internal/websocket"
)

// 全局 WebSocket Hub 引用
var notificationHub *ws.Hub

// SetNotificationHub 设置通知推送的 WebSocket Hub
func SetNotificationHub(hub *ws.Hub) {
	notificationHub = hub
}

// CreateNotification 创建新通知
func CreateNotification(notification *model.Notification) error {
	// 保存到数据库
	if err := database.DB.Create(notification).Error; err != nil {
		return err
	}

	// 如果 Hub 已设置，推送 WebSocket 消息
	if notificationHub != nil {
		// 推送新通知事件
		notificationHub.SendToUser(notification.UserID, ws.MessageTypeNewNotification, map[string]interface{}{
			"id":      notification.ID,
			"type":    notification.Type,
			"content": notification.Content,
		})

		// 推送未读数量更新
		count, _ := GetUnreadCount(notification.UserID)
		notificationHub.SendToUser(notification.UserID, ws.MessageTypeUnreadCountUpdate, map[string]interface{}{
			"count": count,
		})
	}

	return nil
}

// GetNotifications 获取用户通知列表（支持分页和类型过滤）
func GetNotifications(userID uint, notifType string, page, pageSize int) ([]model.Notification, int64, error) {
	var notifications []model.Notification
	var total int64

	query := database.DB.Model(&model.Notification{}).Where("user_id = ?", userID)

	// 类型过滤
	if notifType != "" && notifType != "all" {
		switch notifType {
		case "like":
			query = query.Where("type IN ?", []string{"post_like", "item_like"})
		case "comment":
			query = query.Where("type IN ?", []string{"post_comment", "item_comment"})
		case "guild":
			query = query.Where("type IN ?", []string{"guild_application", "guild_invite"})
		case "system":
			query = query.Where("type = ?", "system")
		}
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&notifications).Error

	return notifications, total, err
}

// MarkAsRead 标记通知为已读
func MarkAsRead(notificationID, userID uint) error {
	return database.DB.
		Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("is_read", true).Error
}

// MarkAllAsRead 标记所有通知为已读
func MarkAllAsRead(userID uint) error {
	return database.DB.
		Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

// GetUnreadCount 获取未读通知数量
func GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := database.DB.
		Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// DeleteNotification 删除单个通知
func DeleteNotification(notificationID, userID uint) error {
	return database.DB.
		Where("id = ? AND user_id = ?", notificationID, userID).
		Delete(&model.Notification{}).Error
}

// DeleteAllNotifications 删除所有通知
func DeleteAllNotifications(userID uint) error {
	return database.DB.
		Where("user_id = ?", userID).
		Delete(&model.Notification{}).Error
}
