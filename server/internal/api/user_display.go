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
	maxForumLevel       = 10
)

type forumLevelDefinition struct {
	Level int
	Name  string
	Color string
	Bold  bool
}

type forumLevelInfo struct {
	Level           int
	Name            string
	Color           string
	Bold            bool
	CurrentLevelExp int
	NextLevelExp    int
	ProgressPercent int
}

var forumLevelDefinitions = map[int]forumLevelDefinition{
	1:  {Level: 1, Name: "新人", Color: "#403B33"},
	2:  {Level: 2, Name: "启源", Color: "#808080"},
	3:  {Level: 3, Name: "常态", Color: "#FFFFFF"},
	4:  {Level: 4, Name: "优秀", Color: "#00C100"},
	5:  {Level: 5, Name: "精良", Color: "#0080FF"},
	6:  {Level: 6, Name: "史诗", Color: "#800080"},
	7:  {Level: 7, Name: "传奇", Color: "#F59B00", Bold: true},
	8:  {Level: 8, Name: "传承", Color: "#0080C0", Bold: true},
	9:  {Level: 9, Name: "神话", Color: "#EBD7A7", Bold: true},
	10: {Level: 10, Name: "顶级", Color: "#8E1027", Bold: true},
}

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
	if user.Role == "admin" || user.Role == "moderator" {
		return adminNameColor, false
	}

	preference := strings.ToLower(strings.TrimSpace(user.NameStylePreference))
	if preference == "" {
		if resolveSponsorLevel(user) >= sponsorLevelStyle && (strings.TrimSpace(user.SponsorColor) != "" || user.SponsorBold) {
			preference = "sponsor"
		} else {
			preference = "level"
		}
	}

	switch preference {
	case "sponsor":
		level := resolveSponsorLevel(user)
		if level >= sponsorLevelStyle {
			bold := user.SponsorBold
			if color := normalizeHexColor(user.SponsorColor); color != "" {
				return color, bold
			}
			return defaultUserNameColor(user), bold
		}
	case "level":
		info := resolveForumLevelInfo(user.ActivityExperience)
		return info.Color, info.Bold
	}

	return defaultUserNameColor(user), false
}

func levelDefinition(level int) forumLevelDefinition {
	if level < 1 {
		level = 1
	}
	if level > maxForumLevel {
		level = maxForumLevel
	}
	return forumLevelDefinitions[level]
}

func levelStepExperience(level int) int {
	if level < 1 {
		return 0
	}
	return int(math.Round(100 * math.Pow(1.8, float64(level-1))))
}

func levelThresholdExperience(level int) int {
	if level <= 1 {
		return 0
	}
	return levelStepExperience(level)
}

func resolveForumLevel(totalExperience int) int {
	if totalExperience <= 0 {
		return 1
	}

	level := 1
	for level < maxForumLevel {
		nextThreshold := levelThresholdExperience(level + 1)
		if totalExperience < nextThreshold {
			break
		}
		level++
	}
	return level
}

func resolveForumLevelInfo(totalExperience int) forumLevelInfo {
	level := resolveForumLevel(totalExperience)
	definition := levelDefinition(level)
	currentThreshold := levelThresholdExperience(level)

	if level >= maxForumLevel {
		return forumLevelInfo{
			Level:           definition.Level,
			Name:            definition.Name,
			Color:           definition.Color,
			Bold:            definition.Bold,
			CurrentLevelExp: totalExperience - currentThreshold,
			NextLevelExp:    0,
			ProgressPercent: 100,
		}
	}

	nextThreshold := levelThresholdExperience(level + 1)
	currentExp := totalExperience - currentThreshold
	nextExp := nextThreshold - currentThreshold
	progress := 0
	if nextExp > 0 {
		progress = int(math.Round(float64(currentExp) / float64(nextExp) * 100))
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	return forumLevelInfo{
		Level:           definition.Level,
		Name:            definition.Name,
		Color:           definition.Color,
		Bold:            definition.Bold,
		CurrentLevelExp: currentExp,
		NextLevelExp:    nextExp,
		ProgressPercent: progress,
	}
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
