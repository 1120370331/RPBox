package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestContentFollowListsAndUpdateNotifications(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{}, &model.Post{}, &model.Item{},
		&model.PostFollow{}, &model.ItemFollow{}, &model.Notification{},
		&model.PostLike{}, &model.PostFavorite{}, &model.PostView{}, &model.PostTag{},
		&model.ItemLike{}, &model.ItemFavorite{}, &model.ItemView{}, &model.ItemTag{},
		&model.UserBlock{}, &model.UserHiddenContent{}, &model.Tag{},
	)
	author := model.User{Username: "follow-author", Email: "follow-author@example.com", PassHash: "hash", AvatarReviewStatus: "none"}
	follower := model.User{Username: "follower", Email: "follower@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&author, &follower}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	post := model.Post{AuthorID: author.ID, Title: "Watched post", Content: "body", Status: "published", ReviewStatus: "approved", IsPublic: true}
	item := model.Item{AuthorID: author.ID, Name: "Watched item", Type: "item", Status: "published", ReviewStatus: "approved", IsPublic: true}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, follower)
	for _, path := range []string{
		fmt.Sprintf("/api/v1/posts/%d/follow", post.ID),
		fmt.Sprintf("/api/v1/items/%d/follow", item.ID),
	} {
		resp := performRequest(server.router, http.MethodPost, path, nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("follow %s: expected 200, got %d body=%s", path, resp.Code, resp.Body.String())
		}
	}
	selfFollow := performRequest(server.router, http.MethodPost, fmt.Sprintf("/api/v1/items/%d/follow", item.ID), nil, newTestToken(t, author))
	if selfFollow.Code != http.StatusOK {
		t.Fatalf("authors should be able to add their own item to following, got %d body=%s", selfFollow.Code, selfFollow.Body.String())
	}

	postList := performRequest(server.router, http.MethodGet, "/api/v1/posts/follows", nil, token)
	if postList.Code != http.StatusOK {
		t.Fatalf("list followed posts: expected 200, got %d body=%s", postList.Code, postList.Body.String())
	}
	var postsPayload struct {
		Posts []model.Post `json:"posts"`
	}
	if err := json.Unmarshal(postList.Body.Bytes(), &postsPayload); err != nil || len(postsPayload.Posts) != 1 || postsPayload.Posts[0].ID != post.ID {
		t.Fatalf("unexpected followed posts payload: err=%v body=%s", err, postList.Body.String())
	}

	itemList := performRequest(server.router, http.MethodGet, "/api/v1/items/follows", nil, token)
	if itemList.Code != http.StatusOK {
		t.Fatalf("list followed items: expected 200, got %d body=%s", itemList.Code, itemList.Body.String())
	}
	var itemsPayload struct {
		Data struct {
			Items []model.Item `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(itemList.Body.Bytes(), &itemsPayload); err != nil || len(itemsPayload.Data.Items) != 1 || itemsPayload.Data.Items[0].ID != item.ID {
		t.Fatalf("unexpected followed items payload: err=%v body=%s", err, itemList.Body.String())
	}

	notifyPostFollowers(post)
	notifyItemFollowers(item)
	var notifications []model.Notification
	if err := db.Where("user_id = ? AND type = ?", follower.ID, "follow_update").Order("target_type").Find(&notifications).Error; err != nil {
		t.Fatalf("load update notifications: %v", err)
	}
	if len(notifications) != 2 {
		t.Fatalf("expected two update notifications, got %d", len(notifications))
	}
	if notifications[0].TargetType != "item" || notifications[1].TargetType != "post" {
		t.Fatalf("unexpected notification targets: %#v", notifications)
	}

	followNotifications := performRequest(server.router, http.MethodGet, "/api/v1/notifications?type=follow", nil, token)
	if followNotifications.Code != http.StatusOK {
		t.Fatalf("list follow notifications: expected 200, got %d body=%s", followNotifications.Code, followNotifications.Body.String())
	}
	var notificationPayload struct {
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(followNotifications.Body.Bytes(), &notificationPayload); err != nil || notificationPayload.Total != 2 {
		t.Fatalf("unexpected follow notification payload: err=%v body=%s", err, followNotifications.Body.String())
	}
}

func TestApprovedContentEditsNotifyFollowers(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{}, &model.Post{}, &model.PostEditRequest{}, &model.PostFollow{},
		&model.Item{}, &model.ItemPendingEdit{}, &model.ItemFollow{}, &model.Notification{},
		&model.UserBlock{},
	)
	author := model.User{Username: "edit-author", Email: "edit-author@example.com", PassHash: "hash"}
	follower := model.User{Username: "edit-follower", Email: "edit-follower@example.com", PassHash: "hash"}
	moderator := model.User{Username: "edit-moderator", Email: "edit-moderator@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &follower, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	post := model.Post{AuthorID: author.ID, Title: "Old post", Content: "old", ContentType: "html", Category: "other", Status: "published", ReviewStatus: "approved", IsPublic: true}
	item := model.Item{AuthorID: author.ID, Name: "Old item", Type: "item", Status: "published", ReviewStatus: "approved", IsPublic: true}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.Create(&model.PostFollow{PostID: post.ID, UserID: follower.ID}).Error; err != nil {
		t.Fatalf("follow post: %v", err)
	}
	if err := db.Create(&model.ItemFollow{ItemID: item.ID, UserID: follower.ID}).Error; err != nil {
		t.Fatalf("follow item: %v", err)
	}
	postEdit := model.PostEditRequest{PostID: post.ID, AuthorID: author.ID, Title: "New post", Content: "new", ContentType: "html", Category: "other", Status: "pending"}
	itemEdit := model.ItemPendingEdit{ItemID: item.ID, AuthorID: author.ID, Name: "New item", ReviewStatus: "pending", IsPublic: true}
	if err := db.Create(&postEdit).Error; err != nil {
		t.Fatalf("create post edit: %v", err)
	}
	if err := db.Create(&itemEdit).Error; err != nil {
		t.Fatalf("create item edit: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, moderator)
	for _, path := range []string{
		fmt.Sprintf("/api/v1/moderator/review/post-edits/%d", postEdit.ID),
		fmt.Sprintf("/api/v1/moderator/review/item-edits/%d", itemEdit.ID),
	} {
		resp := performRequest(server.router, http.MethodPost, path, map[string]string{"action": "approve"}, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("approve %s: expected 200, got %d body=%s", path, resp.Code, resp.Body.String())
		}
	}

	var notifications []model.Notification
	if err := db.Where("user_id = ? AND type = ?", follower.ID, "follow_update").Find(&notifications).Error; err != nil {
		t.Fatalf("load notifications: %v", err)
	}
	if len(notifications) != 2 {
		t.Fatalf("expected one update notification per approved edit, got %d", len(notifications))
	}
}
