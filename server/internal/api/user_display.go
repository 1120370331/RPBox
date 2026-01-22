package api

import (
	"fmt"
	"hash/fnv"
	"math"
	"strings"

	"github.com/rpbox/server/internal/model"
)

const adminNameColor = "#D4A373"
const defaultNameColor = "#6B4F3A"

const (
	sponsorLevelNone    = 0
	sponsorLevelStyle   = 2
	sponsorLevelPremium = 3
)

func resolveSponsorLevel(user model.User) int {
	if user.SponsorLevel > 0 {
		return user.SponsorLevel
	}
	if user.IsSponsor {
		return sponsorLevelStyle
	}
	return sponsorLevelNone
}

func normalizeHexValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "#") {
		value = value[1:]
	}
	if len(value) != 6 {
		return ""
	}
	for _, r := range value {
		if !isHexDigit(r) {
			return ""
		}
	}
	return strings.ToUpper(value)
}

func normalizeHexColor(value string) string {
	hex := normalizeHexValue(value)
	if hex == "" {
		return ""
	}
	return "#" + hex
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func defaultUserNameColor(user model.User) string {
	_ = user
	return defaultNameColor
}

func userDisplayStyle(user model.User) (string, bool) {
	level := resolveSponsorLevel(user)
	bold := level >= sponsorLevelStyle && user.SponsorBold
	if level >= sponsorLevelStyle {
		if color := normalizeHexColor(user.SponsorColor); color != "" {
			return color, bold
		}
	}
	if user.Role == "admin" || user.Role == "moderator" {
		return adminNameColor, bold
	}
	return defaultUserNameColor(user), bold
}

func hslToHex(h, s, l float64) string {
	h = math.Mod(h, 360)
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := l - c/2

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	r = (r + m) * 255
	g = (g + m) * 255
	b = (b + m) * 255

	return fmt.Sprintf("#%02X%02X%02X", uint8(math.Round(r)), uint8(math.Round(g)), uint8(math.Round(b)))
}

func hashString(value string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(value))
	return h.Sum32()
}
