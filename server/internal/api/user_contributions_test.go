package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestUserContributionsMergedPaginationAndVisibility(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Post{}, &model.Item{}, &model.RPDBWork{}, &model.UserBlock{}, &model.UserHiddenContent{})
	previousDB := database.DB
	t.Cleanup(func() { database.DB = previousDB })
	s := newTestServer(t, db)
	user := model.User{Username: "contributor", Email: "contributor@example.com", PassHash: "hash"}
	viewer := model.User{Username: "reader", Email: "reader@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&user, &viewer}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	post := model.Post{AuthorID: user.ID, Title: "New post", Content: "private body", CoverImage: "data:image/png;base64,secret", Status: "published", ReviewStatus: "approved", IsPublic: true, CreatedAt: now, LikeCount: 1}
	item := model.Item{AuthorID: user.ID, Name: "Popular item", Status: "published", ReviewStatus: "approved", IsPublic: true, CreatedAt: now.Add(-time.Hour), LikeCount: 40}
	work := model.RPDBWork{AuthorID: user.ID, Title: "Database work", Status: "published", ReviewStatus: "approved", IsPublic: true, CoverImage: "data:image/png;base64,secret", CreatedAt: now.Add(-2 * time.Hour), LikeCount: 20}
	for _, value := range []interface{}{&post, &item, &work} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	// The owner and visitors must never see unapproved/private material in this public feed.
	for _, table := range []string{"posts", "items", "rpdb_works"} {
		for _, values := range []map[string]interface{}{
			{"status": "draft", "review_status": "approved", "is_public": true},
			{"status": "published", "review_status": "pending", "is_public": true},
			{"status": "published", "review_status": "approved", "is_public": false},
		} {
			values["author_id"] = user.ID
			if table == "items" {
				values["name"] = "Hidden"
			} else {
				values["title"] = "Hidden"
			}
			if table == "posts" {
				values["content"] = "hidden body"
			}
			if table == "rpdb_works" {
				values["type"] = "item_showcase"
			}
			if err := db.Table(table).Create(values).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	load := func(query string, identity model.User) userContributionResponse {
		t.Helper()
		resp := performRequest(s.router, http.MethodGet, fmt.Sprintf("/api/v1/users/%d/contributions?%s", user.ID, query), nil, newTestToken(t, identity))
		if resp.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
		}
		if strings.Contains(resp.Body.String(), "base64") || strings.Contains(resp.Body.String(), "private body") {
			t.Fatalf("list leaked inline images/body: %s", resp.Body.String())
		}
		var payload userContributionResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}
	first := load("page_size=2", viewer)
	second := load("page_size=2&page=2", viewer)
	if first.Total != 3 || len(first.Entries) != 2 || first.Entries[0].Kind != "post" || first.Entries[1].Kind != "item" || len(second.Entries) != 1 || second.Entries[0].Kind != "rpdb" {
		t.Fatalf("incorrect global pagination: first=%+v second=%+v", first, second)
	}
	if !first.Entries[0].CreatedAt.Equal(now) || !first.Entries[0].HasCover {
		t.Fatalf("lost date/cover metadata: %+v", first.Entries[0])
	}
	popular := load("sort=popular", user)
	if popular.Total != 3 || popular.Entries[0].Kind != "item" || popular.Entries[1].Kind != "rpdb" {
		t.Fatalf("incorrect global popularity: %+v", popular)
	}
	filtered := load("kind=rpdb", viewer)
	if filtered.Total != 1 || len(filtered.Entries) != 1 || filtered.Entries[0].Kind != "rpdb" {
		t.Fatalf("incorrect type filter: %+v", filtered)
	}
	if err := db.Create(&model.UserHiddenContent{UserID: viewer.ID, TargetType: reportTargetItem, TargetID: item.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if hidden := load("", viewer); hidden.Total != 2 {
		t.Fatalf("hidden content appeared: %+v", hidden)
	}
	if err := db.Create(&model.UserBlock{BlockerID: viewer.ID, BlockedUserID: user.ID}).Error; err != nil {
		t.Fatal(err)
	}
	if blocked := load("", viewer); blocked.Total != 0 || len(blocked.Entries) != 0 {
		t.Fatalf("blocked author appeared: %+v", blocked)
	}
}
