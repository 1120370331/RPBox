package service

import (
	"testing"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestGetVersionsAndCleanOldVersions(t *testing.T) {
	db := testutil.NewTestDB(t, &model.ProfileVersion{})
	database.DB = db

	profileID := "profile-1"
	for i := 1; i <= MaxVersions+2; i++ {
		db.Create(&model.ProfileVersion{
			ProfileID: profileID,
			Version:   i,
			Checksum:  "v",
		})
	}

	versions, err := GetVersions(profileID)
	if err != nil {
		t.Fatalf("get versions: %v", err)
	}
	if len(versions) != MaxVersions {
		t.Fatalf("expected %d versions, got %d", MaxVersions, len(versions))
	}
	if versions[0].Version != MaxVersions+2 {
		t.Fatalf("expected latest version %d, got %d", MaxVersions+2, versions[0].Version)
	}

	if err := CleanOldVersions(profileID); err != nil {
		t.Fatalf("clean versions: %v", err)
	}

	var count int64
	db.Model(&model.ProfileVersion{}).Where("profile_id = ?", profileID).Count(&count)
	if count != int64(MaxVersions) {
		t.Fatalf("expected %d versions after cleanup, got %d", MaxVersions, count)
	}
}
