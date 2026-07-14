package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestRedeemLimitedSponsorCodeDoesNotInheritPermanentThanksExpiry(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.SponsorRedeemCode{})
	user := model.User{
		Username:     "thanks-only",
		Email:        "thanks-only@example.com",
		PassHash:     "hash",
		IsSponsor:    true,
		SponsorLevel: 1,
	}
	code := model.SponsorRedeemCode{
		Code:           "RPB-ABCD-EFGH-JKLM",
		SponsorLevel:   3,
		DurationMonths: 1,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&code).Error; err != nil {
		t.Fatalf("create code: %v", err)
	}

	server := newTestServer(t, db)
	resp := performRequest(server.router, http.MethodPost, "/api/v1/sponsor-codes/redeem", map[string]string{
		"code": code.Code,
	}, newTestToken(t, user))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var refreshed model.User
	if err := db.First(&refreshed, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if !refreshed.IsSponsor {
		t.Fatal("expected user to keep permanent acknowledgement flag")
	}
	if refreshed.SponsorLevel != 3 {
		t.Fatalf("expected current sponsor level 3, got %d", refreshed.SponsorLevel)
	}
	if refreshed.SponsorAcknowledgementLevel != 3 {
		t.Fatalf("expected acknowledgement level 3, got %d", refreshed.SponsorAcknowledgementLevel)
	}
	if refreshed.SponsorExpiresAt == nil {
		t.Fatal("expected limited level 3 perks to receive an expiry")
	}
}

func TestListSponsorsIncludesExpiredLimitedSponsor(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{})
	expired := userFixture(3, true)
	expired.Username = "expired-sponsor"
	expired.Email = "expired-sponsor@example.com"
	expired.PassHash = "hash"
	expiresAt := testPastTime()
	expired.SponsorExpiresAt = &expiresAt
	expired.SponsorAcknowledgementLevel = 3
	if err := db.Create(&expired).Error; err != nil {
		t.Fatalf("create expired sponsor: %v", err)
	}

	server := newTestServer(t, db)
	resp := performRequest(server.router, http.MethodGet, "/api/v1/sponsors", nil, newTestToken(t, expired))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Users []struct {
			Username     string `json:"username"`
			IsSponsor    bool   `json:"is_sponsor"`
			SponsorLevel int    `json:"sponsor_level"`
		} `json:"users"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Users) != 1 {
		t.Fatalf("expected 1 sponsor, got %d", len(payload.Users))
	}
	if payload.Users[0].Username != expired.Username {
		t.Fatalf("expected sponsor %q, got %q", expired.Username, payload.Users[0].Username)
	}
	if !payload.Users[0].IsSponsor || payload.Users[0].SponsorLevel != 3 {
		t.Fatalf("expected expired sponsor acknowledgement level 3, got is_sponsor=%v level=%d", payload.Users[0].IsSponsor, payload.Users[0].SponsorLevel)
	}
}

func testPastTime() time.Time {
	return time.Now().Add(-time.Hour)
}
