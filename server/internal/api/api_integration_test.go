package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"github.com/rpbox/server/pkg/auth"
	"gorm.io/gorm"
)

func TestHealthEndpoint(t *testing.T) {
	server := newTestServer(t, testutil.NewTestDB(t))

	resp := performRequest(server.router, http.MethodGet, "/health", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestPresetTagsEndpoint(t *testing.T) {
	db := testutil.NewTestDB(t, &model.Tag{})
	database.DB = db
	db.Create(&model.Tag{Name: "tag-a", Type: "preset", Category: "story", IsPublic: true})
	db.Create(&model.Tag{Name: "tag-b", Type: "preset", Category: "item", IsPublic: true})
	db.Create(&model.Tag{Name: "tag-c", Type: "custom", Category: "story", IsPublic: false})

	server := newTestServer(t, db)

	resp := performRequest(server.router, http.MethodGet, "/api/v1/tags/preset", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var payload struct {
		Tags []model.Tag `json:"tags"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Tags) != 2 {
		t.Fatalf("expected 2 preset tags, got %d", len(payload.Tags))
	}
}

func TestProfileVersionFlow(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Profile{}, &model.ProfileVersion{})
	database.DB = db

	user := model.User{Username: "tester", Email: "tester@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, user)

	createReq := map[string]string{
		"id":           "profile-1",
		"profile_name": "First",
		"checksum":     "abc",
	}
	resp := performRequest(server.router, http.MethodPost, "/api/v1/profiles", createReq, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}

	updateReq := map[string]string{
		"profile_name": "Second",
		"checksum":     "def",
	}
	resp = performRequest(server.router, http.MethodPut, "/api/v1/profiles/profile-1", updateReq, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	resp = performRequest(server.router, http.MethodGet, "/api/v1/profiles/profile-1/versions", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var versions struct {
		Versions []model.ProfileVersion `json:"versions"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &versions); err != nil {
		t.Fatalf("decode versions: %v", err)
	}
	if len(versions.Versions) != 1 {
		t.Fatalf("expected 1 version entry, got %d", len(versions.Versions))
	}
}

func TestListPostsSupportsAuthorNameSearch(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Post{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
	)
	database.DB = db

	viewer := model.User{Username: "viewer", Email: "viewer@example.com", PassHash: "hash"}
	target := model.User{Username: "targetAuthor", Email: "target@example.com", PassHash: "hash"}
	other := model.User{Username: "otherAuthor", Email: "other@example.com", PassHash: "hash"}
	if err := db.Create(&viewer).Error; err != nil {
		t.Fatalf("create viewer: %v", err)
	}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("create target: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other: %v", err)
	}

	posts := []model.Post{
		{
			AuthorID:     target.ID,
			Title:        "Target post",
			Content:      "for author filtering",
			Status:       "published",
			ReviewStatus: "approved",
			IsPublic:     true,
			Category:     "other",
		},
		{
			AuthorID:     other.ID,
			Title:        "Other post",
			Content:      "should be filtered out",
			Status:       "published",
			ReviewStatus: "approved",
			IsPublic:     true,
			Category:     "other",
		},
	}
	if err := db.Create(&posts).Error; err != nil {
		t.Fatalf("create posts: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, viewer)

	resp := performRequest(server.router, http.MethodGet, "/api/v1/posts?author_name=target", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Posts []struct {
			ID         uint   `json:"id"`
			AuthorName string `json:"author_name"`
			Title      string `json:"title"`
		} `json:"posts"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 1 {
		t.Fatalf("expected total=1, got %d", payload.Total)
	}
	if len(payload.Posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(payload.Posts))
	}
	if payload.Posts[0].AuthorName != target.Username {
		t.Fatalf("expected author %q, got %q", target.Username, payload.Posts[0].AuthorName)
	}
}

func TestCreateReplyCommentNotifiesParentAndPostAuthor(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Post{},
		&model.Comment{},
		&model.Notification{},
		&model.UserActivityLog{},
	)

	postAuthor := model.User{Username: "post-author", Email: "post-author@example.com", PassHash: "hash"}
	parentAuthor := model.User{Username: "parent-author", Email: "parent-author@example.com", PassHash: "hash"}
	replier := model.User{Username: "replier", Email: "replier@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&postAuthor, &parentAuthor, &replier}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	post := model.Post{
		AuthorID:     postAuthor.ID,
		Title:        "Reply target",
		Content:      "post content",
		Status:       "published",
		ReviewStatus: "approved",
		IsPublic:     true,
		Category:     "other",
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	parentComment := model.Comment{
		PostID:            post.ID,
		AuthorID:          parentAuthor.ID,
		Content:           "parent comment",
		ImageReviewStatus: "none",
	}
	if err := db.Create(&parentComment).Error; err != nil {
		t.Fatalf("create parent comment: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, replier)
	resp := performRequest(server.router, http.MethodPost, "/api/v1/posts/"+strconv.FormatUint(uint64(post.ID), 10)+"/comments", map[string]interface{}{
		"content":   "reply content",
		"parent_id": parentComment.ID,
	}, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", resp.Code, resp.Body.String())
	}

	var reply model.Comment
	if err := json.Unmarshal(resp.Body.Bytes(), &reply); err != nil {
		t.Fatalf("decode reply: %v", err)
	}
	if reply.ParentID == nil || *reply.ParentID != parentComment.ID {
		t.Fatalf("expected parent_id %d, got %#v", parentComment.ID, reply.ParentID)
	}

	var notifications []model.Notification
	if err := db.Where("type = ?", "post_comment").Order("user_id ASC").Find(&notifications).Error; err != nil {
		t.Fatalf("load notifications: %v", err)
	}
	if len(notifications) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifications))
	}
	recipients := map[uint]model.Notification{}
	for _, notification := range notifications {
		recipients[notification.UserID] = notification
	}
	for _, userID := range []uint{postAuthor.ID, parentAuthor.ID} {
		notification, ok := recipients[userID]
		if !ok {
			t.Fatalf("missing notification for user %d", userID)
		}
		if notification.TargetType != "comment" || notification.TargetID != reply.ID {
			t.Fatalf("expected comment target %d, got type=%q id=%d", reply.ID, notification.TargetType, notification.TargetID)
		}
	}

	parentToken := newTestToken(t, parentAuthor)
	notifResp := performRequest(server.router, http.MethodGet, "/api/v1/notifications?type=comment", nil, parentToken)
	if notifResp.Code != http.StatusOK {
		t.Fatalf("expected notifications 200, got %d body=%s", notifResp.Code, notifResp.Body.String())
	}
	var payload struct {
		Notifications []struct {
			TargetPostID uint `json:"target_post_id"`
		} `json:"notifications"`
	}
	if err := json.Unmarshal(notifResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode notifications: %v", err)
	}
	if len(payload.Notifications) != 1 || payload.Notifications[0].TargetPostID != post.ID {
		t.Fatalf("expected target_post_id %d, got %#v", post.ID, payload.Notifications)
	}
}

func newTestServer(t *testing.T, db *gorm.DB) *Server {
	t.Helper()

	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	cfg := testutil.NewTestConfig(t)
	auth.Init(cfg.JWT.Secret)
	database.DB = db

	return NewServer(cfg)
}

func TestListPostsSearchIncludesAuthorAndLocation(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Post{}, &model.UserBlock{}, &model.UserHiddenContent{})

	viewer := model.User{Username: "viewer", Email: "viewer@example.com", PassHash: "hash"}
	target := model.User{Username: "target", Email: "target@example.com", PassHash: "hash"}
	other := model.User{Username: "other", Email: "other@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&viewer, &target, &other}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	posts := []model.Post{
		{
			AuthorID:     target.ID,
			Title:        "A tale",
			Content:      "plain content",
			Region:       "Goldshire",
			Address:      "Lion's Pride Inn",
			Status:       "published",
			ReviewStatus: "approved",
			IsPublic:     true,
			Category:     "other",
		},
		{
			AuthorID:     other.ID,
			Title:        "Another tale",
			Content:      "plain content",
			Region:       "Stormwind",
			Address:      "Trade District",
			Status:       "published",
			ReviewStatus: "approved",
			IsPublic:     true,
			Category:     "other",
		},
	}
	if err := db.Create(&posts).Error; err != nil {
		t.Fatalf("create posts: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, viewer)

	assertSinglePostSearch := func(path string) {
		t.Helper()

		resp := performRequest(server.router, http.MethodGet, path, nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
		}

		var payload struct {
			Posts []struct {
				ID         uint   `json:"id"`
				AuthorName string `json:"author_name"`
				Title      string `json:"title"`
			} `json:"posts"`
			Total int64 `json:"total"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if payload.Total != 1 {
			t.Fatalf("expected total=1, got %d", payload.Total)
		}
		if len(payload.Posts) != 1 {
			t.Fatalf("expected 1 post, got %d", len(payload.Posts))
		}
		if payload.Posts[0].AuthorName != target.Username {
			t.Fatalf("expected author %q, got %q", target.Username, payload.Posts[0].AuthorName)
		}
	}

	assertSinglePostSearch("/api/v1/posts?search=target")
	assertSinglePostSearch("/api/v1/posts?search=Goldshire")
	assertSinglePostSearch("/api/v1/posts?search=Lion")
}

func newTestToken(t *testing.T, user model.User) string {
	t.Helper()

	token, err := auth.GenerateToken(user.ID, user.Username, 1)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}

func performRequest(router http.Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		payload, _ := json.Marshal(body)
		reader = bytes.NewBuffer(payload)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}
