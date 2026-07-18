package database

import (
	"testing"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestMigrateLegacyCommentLikeNotifications(t *testing.T) {
	db := testutil.NewTestDB(t, &model.Notification{})
	legacyLike := model.Notification{
		UserID:     1,
		Type:       "post_comment",
		TargetType: "comment",
		TargetID:   1,
		Content:    "点赞了你的评论",
	}
	comment := model.Notification{
		UserID:     1,
		Type:       "post_comment",
		TargetType: "comment",
		TargetID:   2,
		Content:    "评论了你的帖子《测试》",
	}
	if err := db.Create(&legacyLike).Error; err != nil {
		t.Fatalf("create legacy like notification: %v", err)
	}
	if err := db.Create(&comment).Error; err != nil {
		t.Fatalf("create regular comment notification: %v", err)
	}

	if err := migrateLegacyCommentLikeNotifications(db); err != nil {
		t.Fatalf("migrate legacy comment like notifications: %v", err)
	}

	var migratedLike model.Notification
	if err := db.First(&migratedLike, legacyLike.ID).Error; err != nil {
		t.Fatalf("load migrated like notification: %v", err)
	}
	if migratedLike.Type != "post_comment_like" {
		t.Fatalf("expected legacy like type to migrate, got %q", migratedLike.Type)
	}

	var preservedComment model.Notification
	if err := db.First(&preservedComment, comment.ID).Error; err != nil {
		t.Fatalf("load regular comment notification: %v", err)
	}
	if preservedComment.Type != "post_comment" {
		t.Fatalf("expected regular comment type to remain unchanged, got %q", preservedComment.Type)
	}

	if err := migrateLegacyCommentLikeNotifications(db); err != nil {
		t.Fatalf("repeat legacy comment like migration: %v", err)
	}
}
