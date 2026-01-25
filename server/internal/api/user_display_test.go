package api

import (
	"testing"

	"github.com/rpbox/server/internal/model"
)

func TestNormalizeHexColor(t *testing.T) {
	if color := normalizeHexColor("#a1b2c3"); color != "#A1B2C3" {
		t.Fatalf("expected #A1B2C3, got %s", color)
	}
	if color := normalizeHexColor("bad"); color != "" {
		t.Fatalf("expected empty color, got %s", color)
	}
}

func TestResolveSponsorLevel(t *testing.T) {
	if level := resolveSponsorLevel(userFixture(0, true)); level != sponsorLevelStyle {
		t.Fatalf("expected sponsor style level, got %d", level)
	}
	if level := resolveSponsorLevel(userFixture(3, false)); level != 3 {
		t.Fatalf("expected sponsor level 3, got %d", level)
	}
}

func TestUserDisplayStyle(t *testing.T) {
	color, bold := userDisplayStyle(userFixture(2, true))
	if color != defaultNameColor || !bold {
		t.Fatalf("expected default color with bold for sponsor without color, got %s %v", color, bold)
	}

	user := userFixture(2, true)
	user.SponsorColor = "00ff00"
	color, _ = userDisplayStyle(user)
	if color != "#00FF00" {
		t.Fatalf("expected sponsor color, got %s", color)
	}

	admin := userFixture(0, false)
	admin.Role = "admin"
	color, _ = userDisplayStyle(admin)
	if color != adminNameColor {
		t.Fatalf("expected admin color, got %s", color)
	}
}

func TestHslToHex(t *testing.T) {
	if color := hslToHex(0, 1, 0.5); color != "#FF0000" {
		t.Fatalf("expected red, got %s", color)
	}
}

func userFixture(level int, sponsor bool) model.User {
	return model.User{
		SponsorLevel: level,
		IsSponsor:    sponsor,
		SponsorBold:  true,
	}
}
