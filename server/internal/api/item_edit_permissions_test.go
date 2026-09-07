package api

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestModeratorCanEditAnotherUsersItem(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Item{},
		&model.ItemFollow{},
		&model.UserActivityLog{},
	)
	author := model.User{Username: "item-author", Email: "item-author@example.com", PassHash: "hash"}
	moderator := model.User{Username: "item-moderator", Email: "item-moderator@example.com", PassHash: "hash", Role: "moderator"}
	viewer := model.User{Username: "item-viewer", Email: "item-viewer@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&author, &moderator, &viewer}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	item := model.Item{
		AuthorID: author.ID, Name: "Original item", Type: "item", ImportCode: "code",
		Status: "published", ReviewStatus: "approved", IsPublic: true,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}

	server := newTestServer(t, db)
	path := fmt.Sprintf("/api/v1/items/%d", item.ID)
	denied := performRequest(server.router, http.MethodPut, path, map[string]string{"name": "Viewer revision"}, newTestToken(t, viewer))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("expected ordinary viewer edit to be forbidden, got %d body=%s", denied.Code, denied.Body.String())
	}

	updated := performRequest(server.router, http.MethodPut, path, map[string]string{"name": "Moderator revision"}, newTestToken(t, moderator))
	if updated.Code != http.StatusOK {
		t.Fatalf("expected moderator edit to succeed, got %d body=%s", updated.Code, updated.Body.String())
	}

	var refreshed model.Item
	if err := db.First(&refreshed, item.ID).Error; err != nil {
		t.Fatalf("load item: %v", err)
	}
	if refreshed.Name != "Moderator revision" {
		t.Fatalf("expected moderator revision to persist, got %q", refreshed.Name)
	}
}
