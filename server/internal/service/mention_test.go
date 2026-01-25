package service

import (
	"reflect"
	"testing"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestExtractMentionIDs(t *testing.T) {
	ids := ExtractMentionIDs(
		"hi [[mention:12:Bob]] [[mention:12:Bob]] [[mention:0:Skip]]",
		`<span data-mention-id="34">@Eve</span>`,
	)

	expected := []uint{12, 34}
	if !reflect.DeepEqual(ids, expected) {
		t.Fatalf("expected %v, got %v", expected, ids)
	}
}

func TestNormalizeMentionPreview(t *testing.T) {
	input := "hello [[mention:12:Foo%20Bar]]!"
	output := NormalizeMentionPreview(input)
	if output != "hello @Foo Bar!" {
		t.Fatalf("expected normalized preview, got %s", output)
	}
}

func TestCreateMentionNotifications(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Notification{})
	database.DB = db

	if err := db.Create(&model.User{ID: 1, Username: "actor", Email: "actor@example.com"}).Error; err != nil {
		t.Fatalf("create actor: %v", err)
	}
	if err := db.Create(&model.User{ID: 2, Username: "bob", Email: "bob@example.com"}).Error; err != nil {
		t.Fatalf("create bob: %v", err)
	}
	if err := db.Create(&model.User{ID: 3, Username: "eve", Email: "eve@example.com"}).Error; err != nil {
		t.Fatalf("create eve: %v", err)
	}

	content := "Hi [[mention:2:Bob]] and [[mention:3:Eve]] and [[mention:1:Actor]]"
	CreateMentionNotifications(1, "post", 99, "hello", content)
	CreateMentionNotifications(1, "post", 99, "hello", content)

	var notifications []model.Notification
	if err := db.Where("type = ?", "mention").Find(&notifications).Error; err != nil {
		t.Fatalf("load notifications: %v", err)
	}
	if len(notifications) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifications))
	}
}
