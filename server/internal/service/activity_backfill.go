package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
)

const storyArchiveProgressBackfillAction = "story_archive_progress_backfill"

// ActivityBackfillOptions controls how historical activity rewards are reconciled.
type ActivityBackfillOptions struct {
	UserID uint
	Before *time.Time
	DryRun bool
}

// ActivityBackfillSummary is the final result of one backfill run.
type ActivityBackfillSummary struct {
	SelectedUsers   int
	AffectedUsers   int
	AppliedRewards  int
	TotalPoints     int
	TotalExperience int
	Users           []UserActivityBackfillSummary
}

// UserActivityBackfillSummary describes the delta applied to one user.
type UserActivityBackfillSummary struct {
	UserID          uint
	Username        string
	RewardCount     int
	PointsDelta     int
	ExperienceDelta int
}

type activityBackfillRunner struct {
	opts              ActivityBackfillOptions
	selectedUsers     int64
	exactLogs         map[string]struct{}
	storyArchiveAward map[string]int
	userSummaries     map[uint]*UserActivityBackfillSummary
}

// BackfillLegacyCommunityActivity compensates historical community rewards for legacy users.
func BackfillLegacyCommunityActivity(db *gorm.DB, opts ActivityBackfillOptions) (ActivityBackfillSummary, error) {
	runner, err := newActivityBackfillRunner(db, opts)
	if err != nil {
		return ActivityBackfillSummary{}, err
	}

	if opts.DryRun {
		if err := runner.run(db); err != nil {
			return ActivityBackfillSummary{}, err
		}
		return runner.summary(db)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return runner.run(tx)
	}); err != nil {
		return ActivityBackfillSummary{}, err
	}

	return runner.summary(db)
}

func newActivityBackfillRunner(db *gorm.DB, opts ActivityBackfillOptions) (*activityBackfillRunner, error) {
	runner := &activityBackfillRunner{
		opts:              opts,
		exactLogs:         make(map[string]struct{}),
		storyArchiveAward: make(map[string]int),
		userSummaries:     make(map[uint]*UserActivityBackfillSummary),
	}

	userQuery := db.Model(&model.User{})
	if opts.UserID != 0 {
		userQuery = userQuery.Where("id = ?", opts.UserID)
	}
	if err := userQuery.Count(&runner.selectedUsers).Error; err != nil {
		return nil, err
	}

	type existingLogRow struct {
		UserID          uint
		Action          string
		ReferenceKey    string
		ExperienceDelta int
	}

	actions := []string{
		"post_publish",
		"item_publish",
		"post_like_received",
		"item_like_received",
		"post_comment_create",
		"post_comment_received",
		"item_comment_create",
		"item_comment_received",
		"item_download_received",
		"daily_first_like",
		"story_archive_progress",
		storyArchiveProgressBackfillAction,
	}

	var logs []existingLogRow
	logQuery := db.Model(&model.UserActivityLog{}).
		Select("user_id, action, reference_key, experience_delta").
		Where("action IN ?", actions)
	if opts.UserID != 0 {
		logQuery = logQuery.Where("user_id = ?", opts.UserID)
	}
	if err := logQuery.Find(&logs).Error; err != nil {
		return nil, err
	}

	for _, row := range logs {
		switch row.Action {
		case "story_archive_progress", storyArchiveProgressBackfillAction:
			day := storyArchiveLogDay(row.Action, row.ReferenceKey)
			if day != "" {
				runner.storyArchiveAward[storyArchiveDayKey(row.UserID, day)] += row.ExperienceDelta
			}
			if row.Action == storyArchiveProgressBackfillAction {
				runner.exactLogs[exactRewardKey(row.UserID, row.Action, row.ReferenceKey)] = struct{}{}
			}
		default:
			runner.exactLogs[exactRewardKey(row.UserID, row.Action, row.ReferenceKey)] = struct{}{}
		}
	}

	return runner, nil
}

func (r *activityBackfillRunner) run(db *gorm.DB) error {
	steps := []func(*gorm.DB) error{
		r.backfillPublishedPosts,
		r.backfillPublishedItems,
		r.backfillPostLikes,
		r.backfillItemLikes,
		r.backfillPostComments,
		r.backfillItemComments,
		r.backfillItemDownloads,
		r.backfillDailyFirstLikeBonus,
		r.backfillStoryArchiveProgress,
	}

	for _, step := range steps {
		if err := step(db); err != nil {
			return err
		}
	}

	return nil
}

func (r *activityBackfillRunner) summary(db *gorm.DB) (ActivityBackfillSummary, error) {
	userIDs := make([]uint, 0, len(r.userSummaries))
	for userID := range r.userSummaries {
		userIDs = append(userIDs, userID)
	}

	if len(userIDs) > 0 {
		var users []struct {
			ID       uint
			Username string
		}
		if err := db.Model(&model.User{}).
			Select("id, username").
			Where("id IN ?", userIDs).
			Find(&users).Error; err != nil {
			return ActivityBackfillSummary{}, err
		}
		for _, user := range users {
			if summary := r.userSummaries[user.ID]; summary != nil {
				summary.Username = user.Username
			}
		}
	}

	result := ActivityBackfillSummary{
		SelectedUsers: int(r.selectedUsers),
		Users:         make([]UserActivityBackfillSummary, 0, len(r.userSummaries)),
	}

	for _, summary := range r.userSummaries {
		result.AffectedUsers++
		result.AppliedRewards += summary.RewardCount
		result.TotalPoints += summary.PointsDelta
		result.TotalExperience += summary.ExperienceDelta
		result.Users = append(result.Users, *summary)
	}

	sort.Slice(result.Users, func(i, j int) bool {
		if result.Users[i].ExperienceDelta != result.Users[j].ExperienceDelta {
			return result.Users[i].ExperienceDelta > result.Users[j].ExperienceDelta
		}
		if result.Users[i].PointsDelta != result.Users[j].PointsDelta {
			return result.Users[i].PointsDelta > result.Users[j].PointsDelta
		}
		return result.Users[i].UserID < result.Users[j].UserID
	})

	return result, nil
}

func (r *activityBackfillRunner) backfillPublishedPosts(db *gorm.DB) error {
	var rows []struct {
		ID       uint
		AuthorID uint
	}

	query := db.Model(&model.Post{}).
		Select("id, author_id").
		Where("status = ? AND review_status = ?", "published", "approved").
		Order("id ASC")
	query = r.applyBeforeFilter(query, "created_at")
	if r.opts.UserID != 0 {
		query = query.Where("author_id = ?", r.opts.UserID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		if err := r.applyExactReward(db, row.AuthorID, "post_publish", fmt.Sprintf("post:%d", row.ID), 0, PostPublishExperience); err != nil {
			return err
		}
	}
	return nil
}

func (r *activityBackfillRunner) backfillPublishedItems(db *gorm.DB) error {
	var rows []struct {
		ID       uint
		AuthorID uint
	}

	query := db.Model(&model.Item{}).
		Select("id, author_id").
		Where("review_status = ?", "approved").
		Where("status IN ?", []string{"published", "removed"}).
		Order("id ASC")
	query = r.applyBeforeFilter(query, "created_at")
	if r.opts.UserID != 0 {
		query = query.Where("author_id = ?", r.opts.UserID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		if err := r.applyExactReward(db, row.AuthorID, "item_publish", fmt.Sprintf("item:%d", row.ID), 0, ItemPublishExperience); err != nil {
			return err
		}
	}
	return nil
}

func (r *activityBackfillRunner) backfillPostLikes(db *gorm.DB) error {
	var rows []struct {
		PostID   uint
		LikerID  uint
		AuthorID uint
	}

	query := db.Table("post_likes AS pl").
		Select("pl.post_id, pl.user_id AS liker_id, p.author_id").
		Joins("JOIN posts AS p ON p.id = pl.post_id").
		Order("pl.id ASC")
	query = r.applyBeforeFilter(query, "pl.created_at")
	if r.opts.UserID != 0 {
		query = query.Where("p.author_id = ?", r.opts.UserID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		if row.AuthorID == row.LikerID {
			continue
		}
		if err := r.applyExactReward(
			db,
			row.AuthorID,
			"post_like_received",
			fmt.Sprintf("post:%d:liker:%d", row.PostID, row.LikerID),
			PostLikeRewardPoints,
			PostLikeRewardExperience,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *activityBackfillRunner) backfillItemLikes(db *gorm.DB) error {
	var rows []struct {
		ItemID   uint
		LikerID  uint
		AuthorID uint
	}

	query := db.Table("item_likes AS il").
		Select("il.item_id, il.user_id AS liker_id, i.author_id").
		Joins("JOIN items AS i ON i.id = il.item_id").
		Order("il.id ASC")
	query = r.applyBeforeFilter(query, "il.created_at")
	if r.opts.UserID != 0 {
		query = query.Where("i.author_id = ?", r.opts.UserID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		if row.AuthorID == row.LikerID {
			continue
		}
		if err := r.applyExactReward(
			db,
			row.AuthorID,
			"item_like_received",
			fmt.Sprintf("item:%d:liker:%d", row.ItemID, row.LikerID),
			ItemLikeRewardPoints,
			ItemLikeRewardExperience,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *activityBackfillRunner) backfillPostComments(db *gorm.DB) error {
	var rows []struct {
		ID             uint
		AuthorID       uint
		ParentID       *uint
		PostAuthorID   uint
		ParentAuthorID *uint
	}

	query := db.Table("comments AS c").
		Select("c.id, c.author_id, c.parent_id, p.author_id AS post_author_id, parent.author_id AS parent_author_id").
		Joins("JOIN posts AS p ON p.id = c.post_id").
		Joins("LEFT JOIN comments AS parent ON parent.id = c.parent_id").
		Order("c.id ASC")
	query = r.applyBeforeFilter(query, "c.created_at")
	if err := query.Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		if r.opts.UserID == 0 || row.AuthorID == r.opts.UserID {
			if err := r.applyExactReward(db, row.AuthorID, "post_comment_create", fmt.Sprintf("post-comment:%d", row.ID), 0, CommentCreateExperience); err != nil {
				return err
			}
		}

		ownerID := row.PostAuthorID
		if row.ParentID != nil && row.ParentAuthorID != nil {
			ownerID = *row.ParentAuthorID
		}
		if ownerID == row.AuthorID {
			continue
		}
		if r.opts.UserID != 0 && ownerID != r.opts.UserID {
			continue
		}
		if err := r.applyExactReward(db, ownerID, "post_comment_received", fmt.Sprintf("comment:%d:owner:%d", row.ID, ownerID), 0, CommentReceivedExperience); err != nil {
			return err
		}
	}
	return nil
}

func (r *activityBackfillRunner) backfillItemComments(db *gorm.DB) error {
	var rows []struct {
		ID       uint
		UserID   uint
		ItemID   uint
		AuthorID uint
	}

	query := db.Table("item_comments AS ic").
		Select("ic.id, ic.user_id, ic.item_id, i.author_id").
		Joins("JOIN items AS i ON i.id = ic.item_id").
		Order("ic.id ASC")
	query = r.applyBeforeFilter(query, "ic.created_at")
	if err := query.Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		if r.opts.UserID == 0 || row.UserID == r.opts.UserID {
			if err := r.applyExactReward(db, row.UserID, "item_comment_create", fmt.Sprintf("item-comment:%d", row.ID), 0, CommentCreateExperience); err != nil {
				return err
			}
		}

		if row.AuthorID == row.UserID {
			continue
		}
		if r.opts.UserID != 0 && row.AuthorID != r.opts.UserID {
			continue
		}
		if err := r.applyExactReward(db, row.AuthorID, "item_comment_received", fmt.Sprintf("item:%d:comment:%d", row.ItemID, row.ID), 0, CommentReceivedExperience); err != nil {
			return err
		}
	}
	return nil
}

func (r *activityBackfillRunner) backfillItemDownloads(db *gorm.DB) error {
	var rows []struct {
		ItemID       uint
		DownloaderID uint
		AuthorID     uint
	}

	query := db.Table("item_downloads AS idl").
		Select("idl.item_id, idl.user_id AS downloader_id, i.author_id").
		Joins("JOIN items AS i ON i.id = idl.item_id").
		Order("idl.id ASC")
	query = r.applyBeforeFilter(query, "idl.created_at")
	if r.opts.UserID != 0 {
		query = query.Where("i.author_id = ?", r.opts.UserID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		if row.AuthorID == row.DownloaderID {
			continue
		}
		if err := r.applyExactReward(
			db,
			row.AuthorID,
			"item_download_received",
			fmt.Sprintf("item:%d:downloader:%d", row.ItemID, row.DownloaderID),
			0,
			ItemDownloadRewardExperience,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *activityBackfillRunner) backfillDailyFirstLikeBonus(db *gorm.DB) error {
	daySet := make(map[string]struct{})
	if err := r.collectLikeDays(db.Model(&model.PostLike{}).Select("user_id, created_at"), daySet); err != nil {
		return err
	}
	if err := r.collectLikeDays(db.Model(&model.ItemLike{}).Select("user_id, created_at"), daySet); err != nil {
		return err
	}
	if err := r.collectLikeDays(db.Model(&model.CommentLike{}).Select("user_id, created_at"), daySet); err != nil {
		return err
	}

	keys := make([]string, 0, len(daySet))
	for key := range daySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		userID, day, ok := parseDailyLikeKey(key)
		if !ok {
			continue
		}
		if err := r.applyExactReward(db, userID, "daily_first_like", day, 0, DailyFirstLikeExperience); err != nil {
			return err
		}
	}
	return nil
}

func (r *activityBackfillRunner) backfillStoryArchiveProgress(db *gorm.DB) error {
	var rows []struct {
		UserID    uint
		CreatedAt time.Time
	}

	query := db.Table("story_entries AS se").
		Select("s.user_id, se.created_at").
		Joins("JOIN stories AS s ON s.id = se.story_id")
	query = r.applyBeforeFilter(query, "se.created_at")
	if r.opts.UserID != 0 {
		query = query.Where("s.user_id = ?", r.opts.UserID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return err
	}

	dailyEntries := make(map[string]int)
	for _, row := range rows {
		dailyEntries[storyArchiveDayKey(row.UserID, activityDay(row.CreatedAt))]++
	}

	keys := make([]string, 0, len(dailyEntries))
	for key := range dailyEntries {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		userID, day, ok := parseStoryArchiveDayKey(key)
		if !ok {
			continue
		}

		targetAward := storyArchiveTarget(dailyEntries[key])
		if targetAward == 0 {
			continue
		}
		existingAward := r.storyArchiveAward[key]
		delta := targetAward - existingAward
		if delta <= 0 {
			continue
		}

		refKey := fmt.Sprintf("%s:%d", day, targetAward)
		if err := r.applyExactReward(db, userID, storyArchiveProgressBackfillAction, refKey, 0, delta); err != nil {
			return err
		}
		r.storyArchiveAward[key] += delta
	}

	return nil
}

func (r *activityBackfillRunner) collectLikeDays(query *gorm.DB, daySet map[string]struct{}) error {
	type row struct {
		UserID    uint
		CreatedAt time.Time
	}

	query = r.applyBeforeFilter(query, "created_at")
	if r.opts.UserID != 0 {
		query = query.Where("user_id = ?", r.opts.UserID)
	}

	var rows []row
	if err := query.Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		daySet[dailyLikeKey(row.UserID, activityDay(row.CreatedAt))] = struct{}{}
	}
	return nil
}

func (r *activityBackfillRunner) applyExactReward(db *gorm.DB, userID uint, action, referenceKey string, pointsDelta, experienceDelta int) error {
	if userID == 0 || (pointsDelta == 0 && experienceDelta == 0) {
		return nil
	}
	if r.opts.UserID != 0 && userID != r.opts.UserID {
		return nil
	}

	key := exactRewardKey(userID, action, referenceKey)
	if _, exists := r.exactLogs[key]; exists {
		return nil
	}

	appliedPoints := pointsDelta
	appliedExperience := experienceDelta

	if !r.opts.DryRun {
		result, err := AwardActivityReward(db, userID, action, referenceKey, pointsDelta, experienceDelta)
		if err != nil {
			return err
		}
		if !result.Granted {
			r.exactLogs[key] = struct{}{}
			return nil
		}
		appliedPoints = result.PointsDelta
		appliedExperience = result.ExperienceDelta
	}

	r.exactLogs[key] = struct{}{}
	summary := r.userSummaries[userID]
	if summary == nil {
		summary = &UserActivityBackfillSummary{UserID: userID}
		r.userSummaries[userID] = summary
	}
	summary.RewardCount++
	summary.PointsDelta += appliedPoints
	summary.ExperienceDelta += appliedExperience
	return nil
}

func (r *activityBackfillRunner) applyBeforeFilter(query *gorm.DB, column string) *gorm.DB {
	if r.opts.Before == nil {
		return query
	}
	return query.Where(column+" <= ?", *r.opts.Before)
}

func exactRewardKey(userID uint, action, referenceKey string) string {
	return fmt.Sprintf("%d|%s|%s", userID, action, referenceKey)
}

func dailyLikeKey(userID uint, day string) string {
	return fmt.Sprintf("%d|%s", userID, day)
}

func parseDailyLikeKey(key string) (uint, string, bool) {
	parts := strings.SplitN(key, "|", 2)
	if len(parts) != 2 {
		return 0, "", false
	}
	var userID uint
	if _, err := fmt.Sscanf(parts[0], "%d", &userID); err != nil {
		return 0, "", false
	}
	return userID, parts[1], true
}

func storyArchiveDayKey(userID uint, day string) string {
	return fmt.Sprintf("%d|%s", userID, day)
}

func parseStoryArchiveDayKey(key string) (uint, string, bool) {
	return parseDailyLikeKey(key)
}

func storyArchiveLogDay(action, referenceKey string) string {
	switch action {
	case "story_archive_progress":
		if idx := strings.Index(referenceKey, "-archive-"); idx > 0 {
			return referenceKey[:idx]
		}
	case storyArchiveProgressBackfillAction:
		if idx := strings.Index(referenceKey, ":"); idx > 0 {
			return referenceKey[:idx]
		}
		return referenceKey
	}
	return ""
}

func activityDay(ts time.Time) string {
	return ts.In(time.Local).Format("2006-01-02")
}

func storyArchiveTarget(totalEntries int) int {
	if totalEntries <= 0 {
		return 0
	}
	target := totalEntries / StoryArchiveEntriesPerExp
	if target > StoryArchiveDailyMaxExp {
		target = StoryArchiveDailyMaxExp
	}
	return target
}
