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

func TestResolveForumLevelUsesIndependentThresholds(t *testing.T) {
	tests := []struct {
		name  string
		total int
		want  int
	}{
		{name: "below level 2", total: 179, want: 1},
		{name: "at level 2", total: 180, want: 2},
		{name: "below level 3", total: 323, want: 2},
		{name: "at level 3", total: 324, want: 3},
		{name: "at level 4", total: 583, want: 4},
		{name: "at level 10", total: 19836, want: 10},
		{name: "beyond max level threshold", total: 25000, want: 10},
	}

	for _, tt := range tests {
		if level := resolveForumLevel(tt.total); level != tt.want {
			t.Fatalf("%s: expected level %d, got %d", tt.name, tt.want, level)
		}
	}
}

func TestResolveForumLevelInfoResetsProgressAtLevelUp(t *testing.T) {
	info := resolveForumLevelInfo(180)
	if info.Level != 2 {
		t.Fatalf("expected level 2, got %d", info.Level)
	}
	if info.CurrentLevelExp != 0 {
		t.Fatalf("expected current level exp to reset at threshold, got %d", info.CurrentLevelExp)
	}
	if info.NextLevelExp != 144 {
		t.Fatalf("expected 144 exp to next level, got %d", info.NextLevelExp)
	}
	if info.ProgressPercent != 0 {
		t.Fatalf("expected zero progress right after level up, got %d", info.ProgressPercent)
	}

	info = resolveForumLevelInfo(252)
	if info.Level != 2 {
		t.Fatalf("expected level 2 for mid-progress sample, got %d", info.Level)
	}
	if info.CurrentLevelExp != 72 {
		t.Fatalf("expected 72 exp into current level, got %d", info.CurrentLevelExp)
	}
	if info.NextLevelExp != 144 {
		t.Fatalf("expected 144 total exp span for level 2, got %d", info.NextLevelExp)
	}
	if info.ProgressPercent != 50 {
		t.Fatalf("expected 50 percent progress, got %d", info.ProgressPercent)
	}
}

func TestResolveForumLevelInfoAtMaxLevel(t *testing.T) {
	info := resolveForumLevelInfo(20000)
	if info.Level != 10 {
		t.Fatalf("expected level 10, got %d", info.Level)
	}
	if info.CurrentLevelExp != 164 {
		t.Fatalf("expected overflow within max level to be retained, got %d", info.CurrentLevelExp)
	}
	if info.NextLevelExp != 0 {
		t.Fatalf("expected no next level exp at max level, got %d", info.NextLevelExp)
	}
	if info.ProgressPercent != 100 {
		t.Fatalf("expected max level progress to stay full, got %d", info.ProgressPercent)
	}
}

func userFixture(level int, sponsor bool) model.User {
	return model.User{
		SponsorLevel: level,
		IsSponsor:    sponsor,
		SponsorBold:  true,
	}
}
