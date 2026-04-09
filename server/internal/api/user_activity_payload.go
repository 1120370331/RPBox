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
	NameStylePreference    string `json:"name_style_preference"`
	AvatarChangeCount      int    `json:"avatar_change_count"`
	UsernameChangeCount    int    `json:"username_change_count"`
	NextAvatarChangeCost   int    `json:"next_avatar_change_cost"`
	NextUsernameChangeCost int    `json:"next_username_change_cost"`
}

func buildUserActivityPayload(user model.User, snapshot service.DailyActivitySnapshot) userActivityPayload {
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
		NameStylePreference:    normalizedNameStylePreference(user),
		AvatarChangeCount:      user.AvatarChangeCount,
		UsernameChangeCount:    user.UsernameChangeCount,
		NextAvatarChangeCost:   nextAvatarCost,
		NextUsernameChangeCost: nextUsernameCost,
	}
}

func loadUserActivitySnapshot(userID uint, now time.Time) service.DailyActivitySnapshot {
	snapshot, err := service.GetDailyActivitySnapshot(database.DB, userID, now)
	if err != nil {
		return service.DailyActivitySnapshot{}
	}
	return snapshot
}
