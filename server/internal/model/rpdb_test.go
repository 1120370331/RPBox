package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRPDBTableNames(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"work", (RPDBWork{}).TableName(), "rpdb_works"},
		{"draft", (RPDBDraft{}).TableName(), "rpdb_drafts"},
		{"reference", (RPDBReference{}).TableName(), "rpdb_references"},
		{"media", (RPDBMedia{}).TableName(), "rpdb_media"},
		{"transmog slot", (RPDBTransmogSlot{}).TableName(), "rpdb_transmog_slots"},
		{"guide step", (RPDBGuideStep{}).TableName(), "rpdb_guide_steps"},
		{"list", (RPDBList{}).TableName(), "rpdb_lists"},
		{"list entry", (RPDBListEntry{}).TableName(), "rpdb_list_entries"},
		{"revision", (RPDBRevision{}).TableName(), "rpdb_revisions"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("expected %q, got %q", test.want, test.got)
			}
		})
	}
}

func TestRPDBUniqueUserInteractions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:rpdb_model_unique?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&RPDBWork{},
		&RPDBFavorite{},
		&RPDBList{},
		&RPDBListEntry{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	work := RPDBWork{
		AuthorID:     1,
		Type:         RPDBWorkTypeItemShowcase,
		Title:        "测试作品",
		Status:       RPDBStatusPublished,
		ReviewStatus: RPDBReviewApproved,
		IsPublic:     true,
	}
	if err := db.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}

	favorite := RPDBFavorite{WorkID: work.ID, UserID: 2}
	if err := db.Create(&favorite).Error; err != nil {
		t.Fatalf("create favorite: %v", err)
	}
	if err := db.Create(&RPDBFavorite{WorkID: work.ID, UserID: 2}).Error; err == nil {
		t.Fatal("expected duplicate favorite to fail")
	}

	list := RPDBList{UserID: 2, Name: "我的收藏", IsDefault: true}
	if err := db.Create(&list).Error; err != nil {
		t.Fatalf("create list: %v", err)
	}
	entry := RPDBListEntry{ListID: list.ID, WorkID: work.ID, Status: RPDBListStatusWanted}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("create list entry: %v", err)
	}
	if err := db.Create(&RPDBListEntry{ListID: list.ID, WorkID: work.ID, Status: RPDBListStatusOwned}).Error; err == nil {
		t.Fatal("expected duplicate list entry to fail")
	}
}

func TestRPDBReferenceCanonicalUniqueness(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:rpdb_reference_unique?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&RPDBWork{}, &RPDBReference{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	work := RPDBWork{
		AuthorID:     1,
		Type:         RPDBWorkTypeItemShowcase,
		Title:        "测试作品",
		Status:       RPDBStatusDraft,
		ReviewStatus: RPDBReviewNone,
		IsPublic:     false,
	}
	if err := db.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}

	ref := RPDBReference{
		WorkID:       work.ID,
		ExternalType: "item",
		ExternalID:   "19019",
		Name:         "雷霆之怒",
	}
	if err := db.Create(&ref).Error; err != nil {
		t.Fatalf("create reference: %v", err)
	}
	if err := db.Create(&RPDBReference{
		WorkID:       work.ID,
		ExternalType: "item",
		ExternalID:   "19019",
		Name:         "重复引用",
	}).Error; err == nil {
		t.Fatal("expected duplicate work reference to fail")
	}
}
