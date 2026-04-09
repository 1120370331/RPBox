package service

import (
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func TestAwardActivityRewardIsIdempotent(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.UserActivityLog{})
	user := model.User{Username: "reward-user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	var first RewardResult
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		first, err = AwardActivityReward(tx, user.ID, "post_approved", "post:42", 3, 30)
		return err
	}); err != nil {
		t.Fatalf("award first reward: %v", err)
	}

	var duplicate RewardResult
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		duplicate, err = AwardActivityReward(tx, user.ID, "post_approved", "post:42", 3, 30)
		return err
	}); err != nil {
		t.Fatalf("award duplicate reward: %v", err)
	}

	if !first.Granted || first.PointsDelta != 3 || first.ExperienceDelta != 30 {
		t.Fatalf("unexpected first reward result: %+v", first)
	}
	if duplicate.Granted || duplicate.PointsDelta != 0 || duplicate.ExperienceDelta != 0 {
		t.Fatalf("duplicate reward should be ignored, got %+v", duplicate)
	}

	var updated model.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if updated.ActivityPoints != 3 || updated.ActivityExperience != 30 {
		t.Fatalf("expected points/exp to update once, got points=%d exp=%d", updated.ActivityPoints, updated.ActivityExperience)
	}

	var logs int64
	if err := db.Model(&model.UserActivityLog{}).Where("user_id = ?", user.ID).Count(&logs).Error; err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if logs != 1 {
		t.Fatalf("expected one activity log, got %d", logs)
	}
}

func TestApplyDailySignInOnlyAwardsOncePerDay(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.UserDailyActivity{}, &model.UserActivityLog{})
	user := model.User{Username: "sign-in-user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	now := time.Date(2026, 4, 7, 9, 30, 0, 0, time.FixedZone("CST", 8*3600))

	var first RewardResult
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		first, err = ApplyDailySignIn(tx, user.ID, now)
		return err
	}); err != nil {
		t.Fatalf("apply first sign-in: %v", err)
	}

	var second RewardResult
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		second, err = ApplyDailySignIn(tx, user.ID, now.Add(2*time.Hour))
		return err
	}); err != nil {
		t.Fatalf("apply second sign-in: %v", err)
	}

	if !first.Granted || first.PointsDelta != DailySignInPoints || first.ExperienceDelta != DailySignInExperience {
		t.Fatalf("unexpected first sign-in result: %+v", first)
	}
	if second.Granted || second.PointsDelta != 0 || second.ExperienceDelta != 0 {
		t.Fatalf("duplicate sign-in should not reward again, got %+v", second)
	}

	snapshot, err := GetDailyActivitySnapshot(db, user.ID, now)
	if err != nil {
		t.Fatalf("get daily snapshot: %v", err)
	}
	if !snapshot.SignedInToday {
		t.Fatalf("expected signed-in snapshot after reward")
	}
	if snapshot.LikeBonusAwardedToday {
		t.Fatalf("did not expect like bonus to be awarded")
	}

	var updated model.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if updated.ActivityPoints != DailySignInPoints || updated.ActivityExperience != DailySignInExperience {
		t.Fatalf("unexpected sign-in totals: points=%d exp=%d", updated.ActivityPoints, updated.ActivityExperience)
	}
}

func TestApplyStoryArchiveProgressBucketsAndCaps(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.UserDailyActivity{}, &model.UserActivityLog{})
	user := model.User{Username: "story-user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	now := time.Date(2026, 4, 7, 14, 0, 0, 0, time.FixedZone("CST", 8*3600))

	var result RewardResult
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = ApplyStoryArchiveProgress(tx, user.ID, 9, now)
		return err
	}); err != nil {
		t.Fatalf("apply initial archive progress: %v", err)
	}
	if result.Granted || result.ExperienceDelta != 0 {
		t.Fatalf("expected no reward before first 10 entries, got %+v", result)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = ApplyStoryArchiveProgress(tx, user.ID, 1, now)
		return err
	}); err != nil {
		t.Fatalf("apply bucket-completing archive progress: %v", err)
	}
	if !result.Granted || result.ExperienceDelta != 1 {
		t.Fatalf("expected +1 exp on 10th entry, got %+v", result)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = ApplyStoryArchiveProgress(tx, user.ID, 500, now)
		return err
	}); err != nil {
		t.Fatalf("apply capped archive progress: %v", err)
	}
	if !result.Granted || result.ExperienceDelta != 49 {
		t.Fatalf("expected archive exp to cap at daily max, got %+v", result)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = ApplyStoryArchiveProgress(tx, user.ID, 100, now)
		return err
	}); err != nil {
		t.Fatalf("apply archive progress after cap: %v", err)
	}
	if result.Granted || result.ExperienceDelta != 0 {
		t.Fatalf("expected no reward after hitting daily cap, got %+v", result)
	}

	snapshot, err := GetDailyActivitySnapshot(db, user.ID, now)
	if err != nil {
		t.Fatalf("get daily snapshot: %v", err)
	}
	if snapshot.StoryArchiveEntries != 610 {
		t.Fatalf("expected 610 tracked archive entries, got %d", snapshot.StoryArchiveEntries)
	}
	if snapshot.StoryArchiveExpAwarded != StoryArchiveDailyMaxExp {
		t.Fatalf("expected archive exp cap %d, got %d", StoryArchiveDailyMaxExp, snapshot.StoryArchiveExpAwarded)
	}

	var updated model.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if updated.ActivityExperience != StoryArchiveDailyMaxExp {
		t.Fatalf("expected total archive exp %d, got %d", StoryArchiveDailyMaxExp, updated.ActivityExperience)
	}
}
