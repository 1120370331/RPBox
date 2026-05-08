package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestReviewPostEditAppliesCoverImage(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Post{}, &model.PostEditRequest{})
	author := model.User{Username: "author", Email: "author@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "moderator", Email: "mod@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	oldCoverTime := time.Now().Add(-time.Hour).Truncate(time.Second)
	post := model.Post{
		AuthorID:            author.ID,
		Title:               "Old title",
		Content:             "Old content",
		ContentType:         "html",
		CoverImage:          "/uploads/old-cover.jpg",
		CoverImageUpdatedAt: &oldCoverTime,
		Category:            "other",
		Status:              "published",
		ReviewStatus:        "approved",
		IsPublic:            true,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	edit := model.PostEditRequest{
		PostID:      post.ID,
		AuthorID:    author.ID,
		Title:       "New title",
		Content:     "New content",
		ContentType: "html",
		CoverImage:  "/uploads/new-cover.jpg",
		Category:    "other",
		Status:      "pending",
	}
	if err := db.Create(&edit).Error; err != nil {
		t.Fatalf("create edit request: %v", err)
	}

	server := newTestServer(t, db)
	resp := performRequest(
		server.router,
		http.MethodPost,
		fmt.Sprintf("/api/v1/moderator/review/post-edits/%d", edit.ID),
		map[string]string{"action": "approve"},
		newTestToken(t, moderator),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var refreshed model.Post
	if err := db.First(&refreshed, post.ID).Error; err != nil {
		t.Fatalf("load post: %v", err)
	}
	if refreshed.CoverImage != edit.CoverImage {
		t.Fatalf("expected cover %q, got %q", edit.CoverImage, refreshed.CoverImage)
	}
	if refreshed.CoverImageUpdatedAt == nil || !refreshed.CoverImageUpdatedAt.After(oldCoverTime) {
		t.Fatalf("expected cover timestamp to be refreshed, got %v", refreshed.CoverImageUpdatedAt)
	}

	var remaining int64
	db.Model(&model.PostEditRequest{}).Where("id = ?", edit.ID).Count(&remaining)
	if remaining != 0 {
		t.Fatalf("expected edit request to be deleted, got %d", remaining)
	}
}

func TestReviewItemEditAppliesPreviewImage(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Item{}, &model.ItemPendingEdit{})
	author := model.User{Username: "author", Email: "author@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "moderator", Email: "mod@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	oldPreviewTime := time.Now().Add(-time.Hour).Truncate(time.Second)
	item := model.Item{
		AuthorID:              author.ID,
		Name:                  "Old item",
		Type:                  "item",
		Icon:                  "old-icon",
		PreviewImage:          "/uploads/old-preview.jpg",
		PreviewImageUpdatedAt: &oldPreviewTime,
		Description:           "Old description",
		DetailContent:         "Old detail",
		ImportCode:            "old-import",
		RequiresPermission:    false,
		EnableWatermark:       true,
		Status:                "published",
		ReviewStatus:          "approved",
		IsPublic:              true,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	edit := model.ItemPendingEdit{
		ItemID:             item.ID,
		AuthorID:           author.ID,
		Name:               "New item",
		Icon:               "new-icon",
		PreviewImage:       "/uploads/new-preview.jpg",
		Description:        "New description",
		DetailContent:      "New detail",
		ImportCode:         "new-import",
		RawData:            "new-raw",
		RequiresPermission: true,
		EnableWatermark:    false,
		IsPublic:           false,
		ReviewStatus:       "pending",
	}
	if err := db.Create(&edit).Error; err != nil {
		t.Fatalf("create edit request: %v", err)
	}

	server := newTestServer(t, db)
	resp := performRequest(
		server.router,
		http.MethodPost,
		fmt.Sprintf("/api/v1/moderator/review/item-edits/%d", edit.ID),
		map[string]string{"action": "approve"},
		newTestToken(t, moderator),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var refreshed model.Item
	if err := db.First(&refreshed, item.ID).Error; err != nil {
		t.Fatalf("load item: %v", err)
	}
	if refreshed.PreviewImage != edit.PreviewImage {
		t.Fatalf("expected preview %q, got %q", edit.PreviewImage, refreshed.PreviewImage)
	}
	if refreshed.PreviewImageUpdatedAt == nil || !refreshed.PreviewImageUpdatedAt.After(oldPreviewTime) {
		t.Fatalf("expected preview timestamp to be refreshed, got %v", refreshed.PreviewImageUpdatedAt)
	}
	if refreshed.DetailContent != edit.DetailContent || refreshed.RawData != edit.RawData {
		t.Fatalf("expected extended edit fields to be applied")
	}
	if refreshed.RequiresPermission != edit.RequiresPermission || refreshed.EnableWatermark != edit.EnableWatermark || refreshed.IsPublic != edit.IsPublic {
		t.Fatalf("expected boolean edit fields to be applied")
	}

	var remaining int64
	db.Model(&model.ItemPendingEdit{}).Where("id = ?", edit.ID).Count(&remaining)
	if remaining != 0 {
		t.Fatalf("expected edit request to be deleted, got %d", remaining)
	}
}
