package service

import (
	"errors"
	"strconv"
	"time"

	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// DailySignInPoints is the points gained from daily sign-in.
	DailySignInPoints = 10
	// DailySignInExperience is the experience gained from daily sign-in.
	DailySignInExperience = 10
	// DailyFirstLikeExperience is the daily first-like bonus.
	DailyFirstLikeExperience = 5
	// PostLikeRewardPoints is the points earned when a post receives a like.
	PostLikeRewardPoints = 3
	// PostLikeRewardExperience is the experience earned when a post receives a like.
	PostLikeRewardExperience = 5
	// ItemLikeRewardPoints is the points earned when an item receives a like.
	ItemLikeRewardPoints = 5
	// ItemLikeRewardExperience is the experience earned when an item receives a like.
	ItemLikeRewardExperience = 10
	// CommentCreateExperience is the experience earned when publishing a comment.
	CommentCreateExperience = 3
	// CommentReceivedExperience is the experience earned when being commented on.
	CommentReceivedExperience = 3
	// PostPublishExperience is the experience earned when publishing a post.
	PostPublishExperience = 30
	// ItemPublishExperience is the experience earned when publishing an item.
	ItemPublishExperience = 50
	// ItemDownloadRewardExperience is the experience earned when an item is downloaded.
	ItemDownloadRewardExperience = 10
	// AvatarChangeCost is the point cost for avatar changes after the first upload.
	AvatarChangeCost = 50
	// UsernameChangeCost is the point cost for username changes after the first change.
	UsernameChangeCost = 30
	// StoryArchiveEntriesPerExp is the archive-entry bucket size for story EXP.
	StoryArchiveEntriesPerExp = 10
	// StoryArchiveDailyMaxExp is the daily EXP cap from story archival.
	StoryArchiveDailyMaxExp = 50
)

var (
	// ErrInsufficientPoints is returned when the user cannot afford a point spend.
	ErrInsufficientPoints = errors.New("insufficient activity points")
)

// RewardResult describes the net change applied by a reward operation.
type RewardResult struct {
	Granted         bool
	PointsDelta     int
	ExperienceDelta int
}

// DailyActivitySnapshot is the normalized daily activity state for a user.
type DailyActivitySnapshot struct {
	SignedInToday          bool
	LikeBonusAwardedToday  bool
	StoryArchiveEntries    int
	StoryArchiveExpAwarded int
}

// DayStart normalizes a timestamp to the start of the local day.
func DayStart(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// GetDailyActivitySnapshot loads today's activity state for the given user.
func GetDailyActivitySnapshot(db *gorm.DB, userID uint, now time.Time) (DailyActivitySnapshot, error) {
	state, err := getDailyActivity(db, userID, now)
	if err != nil {
		return DailyActivitySnapshot{}, err
	}
	return DailyActivitySnapshot{
		SignedInToday:          state.SignedInAt != nil,
		LikeBonusAwardedToday:  state.LikeBonusAwardedAt != nil,
		StoryArchiveEntries:    state.StoryArchiveEntries,
		StoryArchiveExpAwarded: state.StoryArchiveExpAwarded,
	}, nil
}

// AwardActivityReward applies an idempotent positive reward to the user.
func AwardActivityReward(tx *gorm.DB, userID uint, action, referenceKey string, pointsDelta, experienceDelta int) (RewardResult, error) {
	if pointsDelta == 0 && experienceDelta == 0 {
		return RewardResult{}, nil
	}

	logEntry := model.UserActivityLog{
		UserID:          userID,
		Action:          action,
		ReferenceKey:    referenceKey,
		PointsDelta:     pointsDelta,
		ExperienceDelta: experienceDelta,
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&logEntry)
	if result.Error != nil {
		return RewardResult{}, result.Error
	}
	if result.RowsAffected == 0 {
		return RewardResult{}, nil
	}

	updates := map[string]interface{}{}
	if pointsDelta != 0 {
		updates["activity_points"] = gorm.Expr("activity_points + ?", pointsDelta)
	}
	if experienceDelta != 0 {
		updates["activity_experience"] = gorm.Expr("activity_experience + ?", experienceDelta)
	}
	if err := tx.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return RewardResult{}, err
	}

	return RewardResult{
		Granted:         true,
		PointsDelta:     pointsDelta,
		ExperienceDelta: experienceDelta,
	}, nil
}

// SpendActivityPoints spends points once for a unique action reference.
func SpendActivityPoints(tx *gorm.DB, userID uint, action, referenceKey string, cost int) (RewardResult, error) {
	if cost <= 0 {
		return RewardResult{}, nil
	}

	logEntry := model.UserActivityLog{
		UserID:          userID,
		Action:          action,
		ReferenceKey:    referenceKey,
		PointsDelta:     -cost,
		ExperienceDelta: 0,
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&logEntry)
	if result.Error != nil {
		return RewardResult{}, result.Error
	}
	if result.RowsAffected == 0 {
		return RewardResult{}, nil
	}

	updateResult := tx.Model(&model.User{}).
		Where("id = ? AND activity_points >= ?", userID, cost).
		Update("activity_points", gorm.Expr("activity_points - ?", cost))
	if updateResult.Error != nil {
		return RewardResult{}, updateResult.Error
	}
	if updateResult.RowsAffected == 0 {
		return RewardResult{}, ErrInsufficientPoints
	}

	return RewardResult{
		Granted:     true,
		PointsDelta: -cost,
	}, nil
}

// ApplyDailySignIn applies the once-per-day sign-in reward.
func ApplyDailySignIn(tx *gorm.DB, userID uint, now time.Time) (RewardResult, error) {
	state, err := ensureDailyActivity(tx, userID, now)
	if err != nil {
		return RewardResult{}, err
	}
	if state.SignedInAt != nil {
		return RewardResult{}, nil
	}

	if _, err := AwardActivityReward(tx, userID, "daily_sign_in", state.ActivityDate.Format("2006-01-02"), DailySignInPoints, DailySignInExperience); err != nil {
		return RewardResult{}, err
	}
	timestamp := now
	if err := tx.Model(&model.UserDailyActivity{}).
		Where("id = ?", state.ID).
		Update("signed_in_at", &timestamp).Error; err != nil {
		return RewardResult{}, err
	}

	return RewardResult{
		Granted:         true,
		PointsDelta:     DailySignInPoints,
		ExperienceDelta: DailySignInExperience,
	}, nil
}

// ApplyDailyFirstLikeBonus grants the once-per-day like bonus.
func ApplyDailyFirstLikeBonus(tx *gorm.DB, userID uint, now time.Time) (RewardResult, error) {
	state, err := ensureDailyActivity(tx, userID, now)
	if err != nil {
		return RewardResult{}, err
	}
	if state.LikeBonusAwardedAt != nil {
		return RewardResult{}, nil
	}

	if _, err := AwardActivityReward(tx, userID, "daily_first_like", state.ActivityDate.Format("2006-01-02"), 0, DailyFirstLikeExperience); err != nil {
		return RewardResult{}, err
	}
	timestamp := now
	if err := tx.Model(&model.UserDailyActivity{}).
		Where("id = ?", state.ID).
		Update("like_bonus_awarded_at", &timestamp).Error; err != nil {
		return RewardResult{}, err
	}

	return RewardResult{
		Granted:         true,
		ExperienceDelta: DailyFirstLikeExperience,
	}, nil
}

// ApplyStoryArchiveProgress awards EXP for archived story entries in daily buckets.
func ApplyStoryArchiveProgress(tx *gorm.DB, userID uint, archivedEntries int, now time.Time) (RewardResult, error) {
	if archivedEntries <= 0 {
		return RewardResult{}, nil
	}

	state, err := ensureDailyActivity(tx, userID, now)
	if err != nil {
		return RewardResult{}, err
	}

	totalEntries := state.StoryArchiveEntries + archivedEntries
	targetAward := totalEntries / StoryArchiveEntriesPerExp
	if targetAward > StoryArchiveDailyMaxExp {
		targetAward = StoryArchiveDailyMaxExp
	}
	delta := targetAward - state.StoryArchiveExpAwarded
	if delta < 0 {
		delta = 0
	}

	if err := tx.Model(&model.UserDailyActivity{}).
		Where("id = ?", state.ID).
		Updates(map[string]interface{}{
			"story_archive_entries":     totalEntries,
			"story_archive_exp_awarded": targetAward,
		}).Error; err != nil {
		return RewardResult{}, err
	}

	if delta == 0 {
		return RewardResult{}, nil
	}

	refKey := state.ActivityDate.Format("2006-01-02") + "-archive-" + strconv.Itoa(targetAward)
	result, err := AwardActivityReward(tx, userID, "story_archive_progress", refKey, 0, delta)
	if err != nil {
		return RewardResult{}, err
	}
	result.ExperienceDelta = delta
	return result, nil
}

func getDailyActivity(db *gorm.DB, userID uint, now time.Time) (model.UserDailyActivity, error) {
	var state model.UserDailyActivity
	err := db.Where("user_id = ? AND activity_date = ?", userID, DayStart(now)).First(&state).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.UserDailyActivity{
			UserID:       userID,
			ActivityDate: DayStart(now),
		}, nil
	}
	return state, err
}

func ensureDailyActivity(tx *gorm.DB, userID uint, now time.Time) (model.UserDailyActivity, error) {
	date := DayStart(now)
	state := model.UserDailyActivity{
		UserID:       userID,
		ActivityDate: date,
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&state).Error; err != nil {
		return model.UserDailyActivity{}, err
	}
	if err := tx.Where("user_id = ? AND activity_date = ?", userID, date).First(&state).Error; err != nil {
		return model.UserDailyActivity{}, err
	}
	return state, nil
}
