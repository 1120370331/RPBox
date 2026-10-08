package api

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

type commentCompatibilitySnapshot struct {
	ID                uint      `json:"id"`
	Content           string    `json:"content"`
	ImageURL          string    `json:"image_url"`
	ImageReviewStatus string    `json:"image_review_status"`
	ParentID          *uint     `json:"parent_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func TestPostCommentCompatibilityPreservesVisibilityAndStoredContent(t *testing.T) {
	testCommentCompatibilityEndpoint(t, false)
}

// testCommentCompatibilityEndpoint exercises both existing response envelopes
// against stored legacy image-only records, rather than testing a helper mock.
func testCommentCompatibilityEndpoint(t *testing.T, itemComments bool) {
	t.Helper()
	db := testutil.NewTestDB(t, &model.User{}, &model.Post{}, &model.Comment{}, &model.CommentLike{},
		&model.Item{}, &model.ItemComment{}, &model.UserBlock{}, &model.UserHiddenContent{})
	author := model.User{Username: "compat-author", Email: "compat-author@example.invalid", PassHash: "fixture", Role: "user"}
	viewer := model.User{Username: "compat-viewer", Email: "compat-viewer@example.invalid", PassHash: "fixture", Role: "user"}
	blocked := model.User{Username: "compat-blocked", Email: "compat-blocked@example.invalid", PassHash: "fixture", Role: "user"}
	if err := db.Create(&[]*model.User{&author, &viewer, &blocked}).Error; err != nil {
		t.Fatal(err)
	}
	post := model.Post{ID: 1, AuthorID: viewer.ID, Title: "兼容性验证", Content: "正文", Status: "published", ReviewStatus: "approved", IsPublic: true}
	item := model.Item{ID: 1, AuthorID: viewer.ID, Name: "兼容性验证", Type: "item", Status: "published", ReviewStatus: "approved", IsPublic: true}
	for _, record := range []interface{}{&post, &item} {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	parentID := uint(1)
	rows := []commentCompatibilitySnapshot{
		{ID: 1, Content: "\n原有文字[[mention:7:玩家]]\n", ImageReviewStatus: "none"},
		{ID: 2, Content: "原有配图文字", ImageURL: "/uploads/text.png", ImageReviewStatus: "approved"},
		{ID: 3, ImageURL: "/uploads/only-image.png", ImageReviewStatus: "approved"},
		{ID: 4, Content: " \n\t ", ImageURL: "/uploads/whitespace.png", ImageReviewStatus: "approved"},
		{ID: 5, ParentID: &parentID, ImageURL: "/uploads/reply.png", ImageReviewStatus: "approved"},
		{ID: 6, ImageURL: "/uploads/pending-secret.png", ImageReviewStatus: "pending"},
		{ID: 7, ImageURL: "/uploads/rejected-secret.png", ImageReviewStatus: "rejected"},
		{ID: 8, ImageURL: "/uploads/hidden-secret.png", ImageReviewStatus: "approved"},
		{ID: 9, ImageURL: "/uploads/blocked-secret.png", ImageReviewStatus: "approved"},
	}
	var table interface{} = &model.Comment{}
	path := "/api/v1/posts/1/comments"
	targetType := reportTargetComment
	if itemComments {
		table = &model.ItemComment{}
		path = "/api/v1/items/1/comments"
		targetType = reportTargetItemComment
	}
	for _, row := range rows {
		authorID := author.ID
		if row.ID == 9 {
			authorID = blocked.ID
		}
		var record interface{} = &model.Comment{ID: row.ID, PostID: 1, AuthorID: authorID,
			Content: row.Content, ImageURL: row.ImageURL, ImageReviewStatus: row.ImageReviewStatus, ParentID: row.ParentID}
		if itemComments {
			record = &model.ItemComment{ID: row.ID, ItemID: 1, UserID: authorID,
				Content: row.Content, ImageURL: row.ImageURL, ImageReviewStatus: row.ImageReviewStatus, ParentID: row.ParentID}
		}
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.UserHiddenContent{UserID: viewer.ID, TargetType: targetType, TargetID: 8}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.UserBlock{BlockerID: viewer.ID, BlockedUserID: blocked.ID}).Error; err != nil {
		t.Fatal(err)
	}
	var before []commentCompatibilitySnapshot
	if err := db.Model(table).Order("id").Find(&before).Error; err != nil {
		t.Fatal(err)
	}
	server := newTestServer(t, db)
	token := newTestToken(t, viewer)
	read := func(token string) []commentCompatibilitySnapshot {
		t.Helper()
		response := performRequest(server.router, http.MethodGet, path, nil, token)
		if response.Code != http.StatusOK {
			t.Fatalf("read comments: status=%d body=%s", response.Code, response.Body.String())
		}
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		key := "comments"
		if itemComments {
			key = "data"
			var code int
			if err := json.Unmarshal(payload["code"], &code); err != nil || code != 0 {
				t.Fatalf("market envelope changed: %s", response.Body.String())
			}
		}
		var got []commentCompatibilitySnapshot
		if err := json.Unmarshal(payload[key], &got); err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"pending-secret", "rejected-secret"} {
			if strings.Contains(response.Body.String(), secret) {
				t.Fatalf("unapproved image leaked: %s", secret)
			}
		}
		return got
	}
	for repeat := 0; repeat < 2; repeat++ {
		got := read(token)
		if len(got) != 5 {
			t.Fatalf("viewer received hidden or blocked comments: %+v", got)
		}
		for index, comment := range got {
			if comment.ID != uint(index+1) {
				t.Fatalf("unexpected public comment ID %d", comment.ID)
			}
			wantContent := rows[index].Content
			if index >= 2 {
				wantContent = "【图片评论】请使用新版 RPBox 查看配图。"
			}
			if comment.Content != wantContent || comment.ImageURL != rows[index].ImageURL || comment.ImageReviewStatus != rows[index].ImageReviewStatus || !reflect.DeepEqual(comment.ParentID, rows[index].ParentID) {
				t.Fatalf("comment response compatibility mismatch: %+v", comment)
			}
		}
	}
	// Preserve the existing authenticated endpoint contract as well.
	unauthorized := performRequest(server.router, http.MethodGet, path, nil, "")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous request was not rejected: status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
	if strings.Contains(unauthorized.Body.String(), "/uploads/") || strings.Contains(unauthorized.Body.String(), "图片评论") {
		t.Fatalf("anonymous response leaked comment content: %s", unauthorized.Body.String())
	}
	var after []commentCompatibilitySnapshot
	if err := db.Model(table).Order("id").Find(&after).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("GET rewrote stored content, image metadata or timestamps: before=%+v after=%+v", before, after)
	}
}
