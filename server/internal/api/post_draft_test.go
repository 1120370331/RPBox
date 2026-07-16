package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestSavePostDraftReplacesIncompleteSnapshotAndTags(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Post{},
		&model.Tag{},
		&model.PostTag{},
	)
	database.DB = db

	user := model.User{Username: "draft-author", Email: "draft-author@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	tagA := model.Tag{Name: "old-tag", Type: "preset", Category: "post", IsPublic: true, UsageCount: 1}
	tagB := model.Tag{Name: "new-tag", Type: "preset", Category: "post", IsPublic: true}
	if err := db.Create(&[]*model.Tag{&tagA, &tagB}).Error; err != nil {
		t.Fatalf("create tags: %v", err)
	}
	post := model.Post{AuthorID: user.ID, Title: "old title", Content: "old content", Status: "draft", Category: "novel", IsPublic: true}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Create(&model.PostTag{PostID: post.ID, TagID: tagA.ID, AddedBy: user.ID}).Error; err != nil {
		t.Fatalf("create old post tag: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, user)
	resp := performRequest(server.router, http.MethodPut, "/api/v1/posts/"+strconv.FormatUint(uint64(post.ID), 10)+"/draft", map[string]interface{}{
		"title":        "",
		"content":      "new content",
		"content_type": "html",
		"category":     "report",
		"tag_ids":      []uint{tagB.ID},
	}, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var saved model.Post
	if err := json.Unmarshal(resp.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if saved.Title != "" || saved.Content != "new content" || saved.Category != "report" || saved.Status != "draft" {
		t.Fatalf("unexpected saved draft: %+v", saved)
	}

	var postTags []model.PostTag
	if err := db.Where("post_id = ?", post.ID).Find(&postTags).Error; err != nil {
		t.Fatalf("load post tags: %v", err)
	}
	if len(postTags) != 1 || postTags[0].TagID != tagB.ID {
		t.Fatalf("expected only tag %d, got %+v", tagB.ID, postTags)
	}
}

func TestSavePostDraftRejectsPublishedPost(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Post{})
	database.DB = db

	user := model.User{Username: "published-author", Email: "published-author@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	post := model.Post{AuthorID: user.ID, Title: "published", Content: "content", Status: "published", ReviewStatus: "approved", Category: "other", IsPublic: true}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, user)
	resp := performRequest(server.router, http.MethodPut, "/api/v1/posts/"+strconv.FormatUint(uint64(post.ID), 10)+"/draft", map[string]interface{}{
		"title":   "changed",
		"content": "changed",
	}, token)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestCreatePostAllowsEmptyDraftButRejectsEmptyPublishedPost(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Post{})
	database.DB = db

	user := model.User{Username: "blank-draft-author", Email: "blank-draft-author@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	server := newTestServer(t, db)
	token := newTestToken(t, user)

	draftResp := performRequest(server.router, http.MethodPost, "/api/v1/posts", map[string]interface{}{
		"title": "", "content": "", "status": "draft",
	}, token)
	if draftResp.Code != http.StatusCreated {
		t.Fatalf("expected blank draft to return 201, got %d body=%s", draftResp.Code, draftResp.Body.String())
	}

	publishedResp := performRequest(server.router, http.MethodPost, "/api/v1/posts", map[string]interface{}{
		"title": "", "content": "", "status": "published",
	}, token)
	if publishedResp.Code != http.StatusBadRequest {
		t.Fatalf("expected blank published post to return 400, got %d body=%s", publishedResp.Code, publishedResp.Body.String())
	}
}

func TestListMyPostDraftsReturnsOnlyCurrentUsersDrafts(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Post{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
	)
	database.DB = db

	user := model.User{Username: "draft-owner", Email: "draft-owner@example.com", PassHash: "hash"}
	other := model.User{Username: "other-owner", Email: "other-owner@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&user, &other}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	older := model.Post{
		AuthorID: user.ID, Title: "较早草稿", Content: "older", Status: "draft",
		Category: "novel", IsPublic: true,
	}
	newer := model.Post{
		AuthorID: user.ID, Title: "最新草稿", Content: "newer", Status: "draft",
		Category: "report", IsPublic: true,
	}
	published := model.Post{
		AuthorID: user.ID, Title: "已发布帖子", Content: "published", Status: "published",
		ReviewStatus: "approved", Category: "other", IsPublic: true,
	}
	foreign := model.Post{
		AuthorID: other.ID, Title: "他人草稿", Content: "foreign", Status: "draft",
		Category: "other", IsPublic: true,
	}
	if err := db.Create(&[]*model.Post{&older, &newer, &published, &foreign}).Error; err != nil {
		t.Fatalf("create posts: %v", err)
	}
	if err := db.Model(&older).Update("updated_at", older.UpdatedAt.Add(-time.Hour)).Error; err != nil {
		t.Fatalf("age older draft: %v", err)
	}

	server := newTestServer(t, db)
	resp := performRequest(server.router, http.MethodGet, "/api/v1/posts/drafts", nil, newTestToken(t, user))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected drafts 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload postListResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode drafts: %v", err)
	}
	if payload.Total != 2 || len(payload.Posts) != 2 {
		t.Fatalf("expected 2 own drafts, got total=%d posts=%#v", payload.Total, payload.Posts)
	}
	if payload.Posts[0].ID != newer.ID || payload.Posts[1].ID != older.ID {
		t.Fatalf("unexpected draft order: %#v", payload.Posts)
	}
}
