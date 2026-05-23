package api

import (
	"time"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
)

type userActivityPayload struct {
	ActivityPoints         int    `json:"activity_points"`
	ActivityExperience     int    `json:"activity_experience"`
	ForumLevel             int    `json:"forum_level"`
	ForumLevelName         string `json:"forum_level_name"`
	ForumLevelColor        string `json:"forum_level_color"`
	ForumLevelBold         bool   `json:"forum_level_bold"`
	CurrentLevelExp        int    `json:"current_level_exp"`
	NextLevelExp           int    `json:"next_level_exp"`
	LevelProgressPercent   int    `json:"level_progress_percent"`
	SignedInToday          bool   `json:"signed_in_today"`
	TotalSignInDays        int    `json:"total_sign_in_days"`
	ConsecutiveSignInDays  int    `json:"consecutive_sign_in_days"`
	NameStylePreference    string `json:"name_style_preference"`
	AvatarChangeCount      int    `json:"avatar_change_count"`
	UsernameChangeCount    int    `json:"username_change_count"`
	NextAvatarChangeCost   int    `json:"next_avatar_change_cost"`
	NextUsernameChangeCost int    `json:"next_username_change_cost"`
}

func buildUserActivityPayload(user model.User, snapshot service.DailyActivitySnapshot, signInStats service.SignInStats) userActivityPayload {
	levelInfo := resolveForumLevelInfo(user.ActivityExperience)

	nextAvatarCost := 0
	if user.AvatarChangeCount > 0 {
		nextAvatarCost = service.AvatarChangeCost
	}
	nextUsernameCost := 0
	if user.UsernameChangeCount > 0 {
		nextUsernameCost = service.UsernameChangeCost
	}

	return userActivityPayload{
		ActivityPoints:         user.ActivityPoints,
		ActivityExperience:     user.ActivityExperience,
		ForumLevel:             levelInfo.Level,
		ForumLevelName:         levelInfo.Name,
		ForumLevelColor:        levelInfo.Color,
		ForumLevelBold:         levelInfo.Bold,
		CurrentLevelExp:        levelInfo.CurrentLevelExp,
		NextLevelExp:           levelInfo.NextLevelExp,
		LevelProgressPercent:   levelInfo.ProgressPercent,
		SignedInToday:          snapshot.SignedInToday,
		TotalSignInDays:        signInStats.TotalDays,
		ConsecutiveSignInDays:  signInStats.ConsecutiveDays,
		NameStylePreference:    normalizedNameStylePreference(user),
		AvatarChangeCount:      user.AvatarChangeCount,
		UsernameChangeCount:    user.UsernameChangeCount,
		NextAvatarChangeCost:   nextAvatarCost,
		NextUsernameChangeCost: nextUsernameCost,
	}
}

func loadUserActivityPayload(user model.User, now time.Time) userActivityPayload {
	return buildUserActivityPayload(user, loadUserActivitySnapshot(user.ID, now), loadUserSignInStats(user.ID, now))
}

func loadUserActivitySnapshot(userID uint, now time.Time) service.DailyActivitySnapshot {
	snapshot, err := service.GetDailyActivitySnapshot(database.DB, userID, now)
	if err != nil {
		return service.DailyActivitySnapshot{}
	}
	return snapshot
}

func loadUserSignInStats(userID uint, now time.Time) service.SignInStats {
	stats, err := service.GetSignInStats(database.DB, userID, now)
	if err != nil {
		return service.SignInStats{}
	}
	return stats
}
