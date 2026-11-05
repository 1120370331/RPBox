package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestReviewPostApprovalNotifiesAuthor(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{}, &model.Post{}, &model.Notification{},
		&model.UserActivityLog{}, &model.AdminActionLog{},
	)
	author := model.User{Username: "author", Email: "post-notify-author@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "moderator", Email: "post-notify-moderator@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &moderator}).Error; err != nil {
		t.Fatal(err)
	}
	post := model.Post{AuthorID: author.ID, Title: "待审核帖子", Content: "正文", Category: "other", Status: "pending", ReviewStatus: "pending", IsPublic: true}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	server := newTestServer(t, db)
	response := performRequest(server.router, http.MethodPost,
		fmt.Sprintf("/api/v1/moderator/review/posts/%d", post.ID),
		map[string]string{"action": "approve"}, newTestToken(t, moderator))
	if response.Code != http.StatusOK {
		t.Fatalf("approve post: code=%d body=%s", response.Code, response.Body.String())
	}
	list := performRequest(server.router, http.MethodGet, "/api/v1/notifications?type=system", nil, newTestToken(t, author))
	if list.Code != http.StatusOK {
		t.Fatalf("list author notifications: code=%d body=%s", list.Code, list.Body.String())
	}
	var payload struct {
		Notifications []model.Notification `json:"notifications"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Notifications) != 1 {
		t.Fatalf("expected one author notification, got %d", len(payload.Notifications))
	}
	notification := payload.Notifications[0]
	if notification.Type != "system" || notification.TargetType != "post" || notification.TargetID != post.ID || notification.IsRead {
		t.Fatalf("unexpected approval notification: %+v", notification)
	}
	if notification.Content != "你的帖子《待审核帖子》审核已通过。" {
		t.Fatalf("unexpected approval message: %q", notification.Content)
	}
}
