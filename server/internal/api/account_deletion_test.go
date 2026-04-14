package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	authpkg "github.com/rpbox/server/pkg/auth"
)

func TestDeleteAccount(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Profile{},
		&model.ProfileVersion{},
		&model.AccountBackup{},
		&model.AccountBackupVersion{},
		&model.Story{},
		&model.StoryEntry{},
		&model.StoryBookmark{},
		&model.Character{},
		&model.Tag{},
		&model.StoryTag{},
		&model.Guild{},
		&model.GuildMember{},
		&model.GuildApplication{},
		&model.StoryGuild{},
		&model.Item{},
		&model.ItemTag{},
		&model.ItemRating{},
		&model.ItemComment{},
		&model.ItemLike{},
		&model.ItemFavorite{},
		&model.ItemView{},
		&model.ItemDownload{},
		&model.ItemPendingEdit{},
		&model.ItemImage{},
		&model.Post{},
		&model.PostEditRequest{},
		&model.PostTag{},
		&model.Comment{},
		&model.PostLike{},
		&model.PostFavorite{},
		&model.PostView{},
		&model.CommentLike{},
		&model.ContentModerationViolation{},
		&model.Notification{},
		&model.UserDailyActivity{},
		&model.UserActivityLog{},
		&model.Collection{},
		&model.CollectionPost{},
		&model.CollectionItem{},
		&model.CollectionFavorite{},
	)
	database.DB = db

	passHash, err := authpkg.HashPassword("secret123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := model.User{Username: "deleter", Email: "deleter@example.com", PassHash: passHash}
	otherUser := model.User{Username: "other", Email: "other@example.com", PassHash: passHash}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}

	otherPost := model.Post{AuthorID: otherUser.ID, Title: "Other Post", Content: "body", CommentCount: 1, LikeCount: 1, FavoriteCount: 1, ViewCount: 1}
	ownedPost := model.Post{AuthorID: user.ID, Title: "Owned Post", Content: "owned"}
	if err := db.Create(&otherPost).Error; err != nil {
		t.Fatalf("create other post: %v", err)
	}
	if err := db.Create(&ownedPost).Error; err != nil {
		t.Fatalf("create owned post: %v", err)
	}

	otherComment := model.Comment{PostID: otherPost.ID, AuthorID: otherUser.ID, Content: "other comment", LikeCount: 1}
	userComment := model.Comment{PostID: otherPost.ID, AuthorID: user.ID, Content: "user comment"}
	if err := db.Create(&otherComment).Error; err != nil {
		t.Fatalf("create other comment: %v", err)
	}
	if err := db.Create(&userComment).Error; err != nil {
		t.Fatalf("create user comment: %v", err)
	}
	if err := db.Create(&model.CommentLike{CommentID: otherComment.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create comment like: %v", err)
	}
	if err := db.Create(&model.PostLike{PostID: otherPost.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create post like: %v", err)
	}
	if err := db.Create(&model.PostFavorite{PostID: otherPost.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create post favorite: %v", err)
	}
	if err := db.Create(&model.PostView{PostID: otherPost.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create post view: %v", err)
	}

	otherItem := model.Item{
		AuthorID:      otherUser.ID,
		Name:          "Other Item",
		Type:          "item",
		Downloads:     1,
		Rating:        4,
		RatingCount:   1,
		LikeCount:     1,
		FavoriteCount: 1,
	}
	ownedItem := model.Item{AuthorID: user.ID, Name: "Owned Item", Type: "item"}
	if err := db.Create(&otherItem).Error; err != nil {
		t.Fatalf("create other item: %v", err)
	}
	if err := db.Create(&ownedItem).Error; err != nil {
		t.Fatalf("create owned item: %v", err)
	}

	userItemComment := model.ItemComment{ItemID: otherItem.ID, UserID: user.ID, Rating: 4, Content: "great"}
	if err := db.Create(&userItemComment).Error; err != nil {
		t.Fatalf("create item comment: %v", err)
	}
	if err := db.Create(&model.ItemLike{ItemID: otherItem.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create item like: %v", err)
	}
	if err := db.Create(&model.ItemFavorite{ItemID: otherItem.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create item favorite: %v", err)
	}
	if err := db.Create(&model.ItemView{ItemID: otherItem.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create item view: %v", err)
	}
	if err := db.Create(&model.ItemDownload{ItemID: otherItem.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create item download: %v", err)
	}

	profile := model.Profile{ID: "profile-1", UserID: user.ID, ProfileName: "Profile", Checksum: "abc"}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := db.Create(&model.ProfileVersion{ProfileID: profile.ID, Version: 1, Checksum: "abc"}).Error; err != nil {
		t.Fatalf("create profile version: %v", err)
	}

	backup := model.AccountBackup{UserID: user.ID, AccountID: "acc-1", Checksum: "sum"}
	if err := db.Create(&backup).Error; err != nil {
		t.Fatalf("create backup: %v", err)
	}
	if err := db.Create(&model.AccountBackupVersion{BackupID: backup.ID, Version: 1, Checksum: "sum"}).Error; err != nil {
		t.Fatalf("create backup version: %v", err)
	}

	story := model.Story{UserID: user.ID, Title: "Story"}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}
	if err := db.Create(&model.StoryEntry{StoryID: story.ID, Content: "entry"}).Error; err != nil {
		t.Fatalf("create story entry: %v", err)
	}

	collection := model.Collection{AuthorID: user.ID, Name: "Collection"}
	if err := db.Create(&collection).Error; err != nil {
		t.Fatalf("create collection: %v", err)
	}

	guild := model.Guild{Name: "Guild", OwnerID: otherUser.ID, MemberCount: 2}
	if err := db.Create(&guild).Error; err != nil {
		t.Fatalf("create guild: %v", err)
	}
	if err := db.Create(&model.GuildMember{GuildID: guild.ID, UserID: otherUser.ID, Role: "owner"}).Error; err != nil {
		t.Fatalf("create owner guild member: %v", err)
	}
	if err := db.Create(&model.GuildMember{GuildID: guild.ID, UserID: user.ID, Role: "member"}).Error; err != nil {
		t.Fatalf("create user guild member: %v", err)
	}

	if err := db.Create(&model.Notification{UserID: user.ID, Type: "system", Content: "hello"}).Error; err != nil {
		t.Fatalf("create notification: %v", err)
	}
	if err := db.Create(&model.UserDailyActivity{UserID: user.ID}).Error; err != nil {
		t.Fatalf("create daily activity: %v", err)
	}
	if err := db.Create(&model.UserActivityLog{UserID: user.ID, Action: "a", ReferenceKey: "b"}).Error; err != nil {
		t.Fatalf("create activity log: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, user)

	resp := performRequest(server.router, http.MethodDelete, "/api/v1/user/account", map[string]string{
		"password": "secret123",
	}, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var deletedUser model.User
	if err := db.First(&deletedUser, user.ID).Error; err != nil {
		t.Fatalf("reload deleted user: %v", err)
	}
	if deletedUser.AccountDeletedAt == nil {
		t.Fatalf("expected account_deleted_at to be set")
	}
	expectedUsername := fmt.Sprintf("deleted-user-%d", user.ID)
	if deletedUser.Username != expectedUsername {
		t.Fatalf("expected anonymized username, got %q", deletedUser.Username)
	}
	expectedEmail := fmt.Sprintf("deleted+%d@rpbox.invalid", user.ID)
	if deletedUser.Email != expectedEmail {
		t.Fatalf("expected anonymized email, got %q", deletedUser.Email)
	}

	assertCount := func(name string, target interface{}, expected int64, query string, args ...interface{}) {
		t.Helper()
		var count int64
		if err := db.Model(target).Where(query, args...).Count(&count).Error; err != nil {
			t.Fatalf("%s count failed: %v", name, err)
		}
		if count != expected {
			t.Fatalf("%s expected %d, got %d", name, expected, count)
		}
	}

	assertCount("profiles", &model.Profile{}, 0, "user_id = ?", user.ID)
	assertCount("profile_versions", &model.ProfileVersion{}, 0, "profile_id = ?", profile.ID)
	assertCount("account_backups", &model.AccountBackup{}, 0, "user_id = ?", user.ID)
	assertCount("account_backup_versions", &model.AccountBackupVersion{}, 0, "backup_id = ?", backup.ID)
	assertCount("owned_posts", &model.Post{}, 0, "author_id = ?", user.ID)
	assertCount("owned_items", &model.Item{}, 0, "author_id = ?", user.ID)
	assertCount("stories", &model.Story{}, 0, "user_id = ?", user.ID)
	assertCount("collections", &model.Collection{}, 0, "author_id = ?", user.ID)
	assertCount("notifications", &model.Notification{}, 0, "user_id = ? OR actor_id = ?", user.ID, user.ID)

	var refreshedPost model.Post
	if err := db.First(&refreshedPost, otherPost.ID).Error; err != nil {
		t.Fatalf("reload other post: %v", err)
	}
	if refreshedPost.CommentCount != 1 || refreshedPost.LikeCount != 0 || refreshedPost.FavoriteCount != 0 || refreshedPost.ViewCount != 0 {
		t.Fatalf("unexpected post counters after deletion: %+v", refreshedPost)
	}

	var refreshedComment model.Comment
	if err := db.First(&refreshedComment, otherComment.ID).Error; err != nil {
		t.Fatalf("reload other comment: %v", err)
	}
	if refreshedComment.LikeCount != 0 {
		t.Fatalf("expected comment like count 0, got %d", refreshedComment.LikeCount)
	}

	var refreshedItem model.Item
	if err := db.First(&refreshedItem, otherItem.ID).Error; err != nil {
		t.Fatalf("reload other item: %v", err)
	}
	if refreshedItem.RatingCount != 0 || refreshedItem.Rating != 0 || refreshedItem.LikeCount != 0 || refreshedItem.FavoriteCount != 0 || refreshedItem.Downloads != 0 {
		t.Fatalf("unexpected item counters after deletion: %+v", refreshedItem)
	}

	var refreshedGuild model.Guild
	if err := db.First(&refreshedGuild, guild.ID).Error; err != nil {
		t.Fatalf("reload guild: %v", err)
	}
	if refreshedGuild.MemberCount != 1 {
		t.Fatalf("expected guild member_count 1, got %d", refreshedGuild.MemberCount)
	}

	unauthorizedResp := performRequest(server.router, http.MethodGet, "/api/v1/user/info", nil, token)
	if unauthorizedResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after deletion, got %d: %s", unauthorizedResp.Code, unauthorizedResp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["message"] != "账号已删除" {
		t.Fatalf("unexpected response message: %+v", payload)
	}
}
