package service

import (
	"testing"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestNotificationLifecycle(t *testing.T) {
	db := testutil.NewTestDB(t, &model.Notification{})
	database.DB = db
	notificationHub = nil

	like := model.Notification{UserID: 1, Type: "post_like", TargetType: "post", TargetID: 10}
	comment := model.Notification{UserID: 1, Type: "item_comment", TargetType: "item", TargetID: 11}
	mention := model.Notification{UserID: 1, Type: "mention", TargetType: "post", TargetID: 12}

	if err := CreateNotification(&like); err != nil {
		t.Fatalf("create notification: %v", err)
	}
	if err := CreateNotification(&comment); err != nil {
		t.Fatalf("create notification: %v", err)
	}
	if err := CreateNotification(&mention); err != nil {
		t.Fatalf("create notification: %v", err)
	}

	count, err := GetUnreadCount(1)
	if err != nil {
		t.Fatalf("unread count: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 unread, got %d", count)
	}

	if err := MarkAsRead(like.ID, 1); err != nil {
		t.Fatalf("mark as read: %v", err)
	}

	list, total, err := GetNotifications(1, "comment", 1, 10)
	if err != nil {
		t.Fatalf("get notifications: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("expected 2 comment notifications, got %d", total)
	}

	if err := MarkAllAsRead(1); err != nil {
		t.Fatalf("mark all read: %v", err)
	}

	count, err = GetUnreadCount(1)
	if err != nil {
		t.Fatalf("unread count after read: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 unread, got %d", count)
	}
}
