package api

import (
	"encoding/json"
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

func TestPostEditReviewUsesOnlyLatestPendingEdit(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Post{}, &model.PostEditRequest{})
	if err := db.Exec("DROP INDEX IF EXISTS idx_post_edit_requests_post_id").Error; err != nil {
		t.Fatalf("drop unique post edit index: %v", err)
	}

	author := model.User{Username: "author", Email: "author-latest@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "moderator", Email: "mod-latest@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	post := model.Post{
		AuthorID:     author.ID,
		Title:        "Original title",
		Content:      "Original content",
		ContentType:  "html",
		Category:     "other",
		Status:       "published",
		ReviewStatus: "approved",
		IsPublic:     true,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	oldEdit := model.PostEditRequest{
		PostID:      post.ID,
		AuthorID:    author.ID,
		Title:       "Old edit",
		Content:     "Old content",
		ContentType: "html",
		Category:    "other",
		Status:      "pending",
	}
	finalEdit := model.PostEditRequest{
		PostID:      post.ID,
		AuthorID:    author.ID,
		Title:       "Final edit",
		Content:     "Final content",
		ContentType: "html",
		Category:    "other",
		Status:      "pending",
	}
	if err := db.Create(&oldEdit).Error; err != nil {
		t.Fatalf("create old edit request: %v", err)
	}
	if err := db.Create(&finalEdit).Error; err != nil {
		t.Fatalf("create final edit request: %v", err)
	}

	var visiblePending int64
	if err := latestPendingPostEditQuery(db).Count(&visiblePending).Error; err != nil {
		t.Fatalf("count visible pending edits: %v", err)
	}
	if visiblePending != 1 {
		t.Fatalf("expected one visible pending edit, got %d", visiblePending)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, moderator)
	listResp := performRequest(server.router, http.MethodGet, "/api/v1/moderator/review/post-edits", nil, token)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d body=%s", listResp.Code, listResp.Body.String())
	}
	var listPayload struct {
		Edits []struct {
			ID    uint   `json:"id"`
			Title string `json:"title"`
		} `json:"edits"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode pending edits: %v", err)
	}
	if listPayload.Total != 1 || len(listPayload.Edits) != 1 {
		t.Fatalf("expected exactly one listed edit, got total=%d len=%d", listPayload.Total, len(listPayload.Edits))
	}
	if listPayload.Edits[0].ID != finalEdit.ID || listPayload.Edits[0].Title != finalEdit.Title {
		t.Fatalf("expected final edit %#v, got %#v", finalEdit.ID, listPayload.Edits[0])
	}

	staleResp := performRequest(
		server.router,
		http.MethodPost,
		fmt.Sprintf("/api/v1/moderator/review/post-edits/%d", oldEdit.ID),
		map[string]string{"action": "approve"},
		token,
	)
	if staleResp.Code != http.StatusConflict {
		t.Fatalf("expected stale edit conflict, got %d body=%s", staleResp.Code, staleResp.Body.String())
	}

	approveResp := performRequest(
		server.router,
		http.MethodPost,
		fmt.Sprintf("/api/v1/moderator/review/post-edits/%d", finalEdit.ID),
		map[string]string{"action": "approve"},
		token,
	)
	if approveResp.Code != http.StatusOK {
		t.Fatalf("expected approve 200, got %d body=%s", approveResp.Code, approveResp.Body.String())
	}

	var refreshed model.Post
	if err := db.First(&refreshed, post.ID).Error; err != nil {
		t.Fatalf("load post: %v", err)
	}
	if refreshed.Title != finalEdit.Title || refreshed.Content != finalEdit.Content {
		t.Fatalf("expected final edit applied, got title=%q content=%q", refreshed.Title, refreshed.Content)
	}

	var remaining int64
	if err := db.Model(&model.PostEditRequest{}).Where("post_id = ?", post.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("count remaining edit requests: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected all edit requests for post to be cleared, got %d", remaining)
	}
}

func TestPostEditReviewCleansOrphanEditRequest(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Post{}, &model.PostEditRequest{})
	author := model.User{Username: "author", Email: "author-orphan@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "moderator", Email: "mod-orphan@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	post := model.Post{
		AuthorID:     author.ID,
		Title:        "Deleted original",
		Content:      "Original content",
		ContentType:  "html",
		Category:     "other",
		Status:       "published",
		ReviewStatus: "approved",
		IsPublic:     true,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	edit := model.PostEditRequest{
		PostID:      post.ID,
		AuthorID:    author.ID,
		Title:       "Pending edit for deleted post",
		Content:     "Pending content",
		ContentType: "html",
		Category:    "other",
		Status:      "pending",
	}
	if err := db.Create(&edit).Error; err != nil {
		t.Fatalf("create edit request: %v", err)
	}
	if err := db.Delete(&post).Error; err != nil {
		t.Fatalf("delete original post: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, moderator)
	listResp := performRequest(server.router, http.MethodGet, "/api/v1/moderator/review/post-edits", nil, token)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d body=%s", listResp.Code, listResp.Body.String())
	}
	var listPayload struct {
		Edits []struct {
			ID uint `json:"id"`
		} `json:"edits"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listPayload); err != nil {
		t.Fatalf("decode pending edits: %v", err)
	}
	if listPayload.Total != 0 || len(listPayload.Edits) != 0 {
		t.Fatalf("expected orphan edit to be hidden, got total=%d len=%d", listPayload.Total, len(listPayload.Edits))
	}

	approveResp := performRequest(
		server.router,
		http.MethodPost,
		fmt.Sprintf("/api/v1/moderator/review/post-edits/%d", edit.ID),
		map[string]string{"action": "approve"},
		token,
	)
	if approveResp.Code != http.StatusOK {
		t.Fatalf("expected orphan cleanup 200, got %d body=%s", approveResp.Code, approveResp.Body.String())
	}

	var remaining int64
	if err := db.Model(&model.PostEditRequest{}).Where("post_id = ?", post.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("count remaining edit requests: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected orphan edit requests to be cleared, got %d", remaining)
	}
}

func TestDeletePostClearsPostEditRequests(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Post{}, &model.PostEditRequest{})
	author := model.User{Username: "author", Email: "author-delete@example.com", PassHash: "hash", Role: "user"}
	if err := db.Create(&author).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}

	post := model.Post{
		AuthorID:     author.ID,
		Title:        "Post to delete",
		Content:      "Original content",
		ContentType:  "html",
		Category:     "other",
		Status:       "published",
		ReviewStatus: "approved",
		IsPublic:     true,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	edit := model.PostEditRequest{
		PostID:      post.ID,
		AuthorID:    author.ID,
		Title:       "Pending edit",
		Content:     "Pending content",
		ContentType: "html",
		Category:    "other",
		Status:      "pending",
	}
	if err := db.Create(&edit).Error; err != nil {
		t.Fatalf("create edit request: %v", err)
	}

	server := newTestServer(t, db)
	resp := performRequest(
		server.router,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/posts/%d", post.ID),
		nil,
		newTestToken(t, author),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected delete 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var remaining int64
	if err := db.Model(&model.PostEditRequest{}).Where("post_id = ?", post.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("count remaining edit requests: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected edit request to be deleted with post, got %d", remaining)
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
