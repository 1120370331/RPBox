package api

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestApplySensitiveViolationDoesNotBanOnThirdStrike(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.ContentModerationViolation{})
	database.DB = db

	user := model.User{
		Username: "tester",
		Email:    "tester@example.com",
		PassHash: "hash",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	server := &Server{}
	var decision sensitiveDecision
	for i := 0; i < 3; i++ {
		var err error
		decision, _, err = server.applySensitiveViolation(user.ID, "post", nil, []string{"开盒"})
		if err != nil {
			t.Fatalf("apply violation %d: %v", i+1, err)
		}
	}

	var updated model.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}

	if updated.IsBanned {
		t.Fatal("expected user to remain unbanned after third violation")
	}
	if updated.BanReason != "" {
		t.Fatalf("expected no ban reason, got %q", updated.BanReason)
	}
	if updated.SensitiveViolationCount != 3 {
		t.Fatalf("expected violation count 3, got %d", updated.SensitiveViolationCount)
	}
	if decision.Action != moderationActionSevereWarning {
		t.Fatalf("expected severe warning action, got %q", decision.Action)
	}
}

func TestEnsureUserCanPublishClearsLegacySensitiveBan(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{})
	database.DB = db

	now := time.Now()
	user := model.User{
		Username:    "tester",
		Email:       "tester@example.com",
		PassHash:    "hash",
		IsBanned:    true,
		BanReason:   legacySensitiveBanReason,
		BannedAt:    &now,
		BannedBy:    nil,
		BannedUntil: nil,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	server := &Server{}
	updated, ok := server.ensureUserCanPublish(c, user.ID)
	if !ok {
		t.Fatalf("expected user to be allowed after clearing legacy ban, response: %s", recorder.Body.String())
	}
	if updated == nil || updated.IsBanned {
		t.Fatal("expected returned user to be unbanned")
	}

	var reloaded model.User
	if err := db.First(&reloaded, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.IsBanned {
		t.Fatal("expected persisted user ban to be cleared")
	}
	if reloaded.BanReason != "" {
		t.Fatalf("expected cleared ban reason, got %q", reloaded.BanReason)
	}
}
