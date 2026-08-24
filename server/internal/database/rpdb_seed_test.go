package database

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	authpkg "github.com/rpbox/server/pkg/auth"
	"gorm.io/gorm"
)

func TestSeedRPDBDemoCreatesCompleteIdempotentDataset(t *testing.T) {
	db := newRPDBSeedTestDB(t)
	compromisedPassword := "test-only-known:" + rpdbDemoUsername
	compromisedHash, err := authpkg.HashPassword(compromisedPassword)
	if err != nil {
		t.Fatalf("hash test-only compromised credential: %v", err)
	}

	existingAuthor := model.User{
		Username:      "rpdb_demo",
		Email:         "rpdb-demo@local.invalid",
		EmailVerified: true,
		PassHash:      compromisedHash,
		Role:          "moderator",
		Bio:           "保留现有演示账号资料",
		Location:      "用户自定义位置",
	}
	if err := db.Create(&existingAuthor).Error; err != nil {
		t.Fatalf("create existing demo author: %v", err)
	}
	unrelated := model.RPDBWork{
		AuthorID:     existingAuthor.ID,
		Type:         model.RPDBWorkTypeItemShowcase,
		Title:        "用户原有作品",
		Slug:         "user-owned-work",
		Restrictions: `{}`,
		Extra:        `{}`,
		Status:       model.RPDBStatusDraft,
		ReviewStatus: model.RPDBReviewNone,
	}
	if err := db.Create(&unrelated).Error; err != nil {
		t.Fatalf("create unrelated work: %v", err)
	}
	existingDefaultList := model.RPDBList{
		UserID:      existingAuthor.ID,
		Name:        "用户已有默认清单",
		Description: "保留已有默认清单资料",
		IsDefault:   true,
		IsPublic:    false,
	}
	if err := db.Create(&existingDefaultList).Error; err != nil {
		t.Fatalf("create existing default list: %v", err)
	}

	if err := SeedRPDBDemo(db); err != nil {
		t.Fatalf("seed RPDB demo: %v", err)
	}

	var author model.User
	if err := db.Where("username = ?", "rpdb_demo").First(&author).Error; err != nil {
		t.Fatalf("find demo author: %v", err)
	}
	if author.ID != existingAuthor.ID {
		t.Fatalf("expected existing demo author ID %d, got %d", existingAuthor.ID, author.ID)
	}
	if author.Email != "rpdb-demo@local.invalid" || author.EmailVerified {
		t.Fatalf("unexpected demo login identity: email=%q verified=%v", author.Email, author.EmailVerified)
	}
	if authpkg.CheckPassword(compromisedPassword, author.PassHash) {
		t.Fatal("demo account retained a deterministic seed credential")
	}
	assertRPDBDemoAccountDisabled(t, db, rpdbDemoAccountSpec{Username: rpdbDemoUsername, Email: rpdbDemoEmail})
	if author.Bio != existingAuthor.Bio || author.Location != existingAuthor.Location {
		t.Fatal("seed overwrote existing demo account profile fields")
	}

	for _, username := range []string{"rpdb_demo_curator", "rpdb_demo_explorer"} {
		var count int64
		if err := db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
			t.Fatalf("count demo account %q: %v", username, err)
		}
		if count != 1 {
			t.Fatalf("expected one stable demo account %q, got %d", username, count)
		}
	}
	var curator model.User
	if err := db.Where("username = ?", "rpdb_demo_curator").First(&curator).Error; err != nil {
		t.Fatalf("find demo curator: %v", err)
	}
	if curator.Role != "user" {
		t.Fatalf("expected demo curator role user, got %q", curator.Role)
	}
	assertRPDBDemoAccountDisabled(t, db, rpdbDemoAccountSpec{Username: curator.Username, Email: "rpdb-demo-curator@local.invalid"})
	assertRPDBDemoAccountDisabled(t, db, rpdbDemoAccountSpec{Username: "rpdb_demo_explorer", Email: "rpdb-demo-explorer@local.invalid"})

	var works []model.RPDBWork
	if err := db.Where("slug LIKE ?", "rpdb-demo-%").Order("slug ASC").Find(&works).Error; err != nil {
		t.Fatalf("load demo works: %v", err)
	}
	if len(works) != 12 {
		t.Fatalf("expected 12 demo works, got %d", len(works))
	}

	typeCounts := map[string]int{}
	for _, work := range works {
		typeCounts[work.Type]++
		if work.AuthorID != author.ID {
			t.Fatalf("work %q has unexpected author %d", work.Slug, work.AuthorID)
		}
		if work.Status != model.RPDBStatusPublished || work.ReviewStatus != model.RPDBReviewApproved || !work.IsPublic {
			t.Fatalf("work %q is not published, approved, and public", work.Slug)
		}
		if work.ReviewerID == nil || *work.ReviewerID != curator.ID || work.ReviewedAt == nil {
			t.Fatalf("work %q is missing review metadata", work.Slug)
		}
		expectedImageURL := rpdbDemoImageURL(work.Slug)
		if work.CoverImage != expectedImageURL {
			t.Fatalf("work %q cover image: expected %q, got %q", work.Slug, expectedImageURL, work.CoverImage)
		}
		if work.Type == model.RPDBWorkTypeHomeShowcase {
			assertRPDBDemoHomeExtra(t, work)
		}
		assertRPDBDemoWorkChildren(t, db, work)
		assertRPDBDemoWorkCounters(t, db, work)
	}

	for workType, want := range map[string]int{
		model.RPDBWorkTypeItemShowcase: 4,
		model.RPDBWorkTypeTransmog:     4,
		model.RPDBWorkTypeHomeShowcase: 4,
	} {
		if typeCounts[workType] != want {
			t.Fatalf("expected %d %s works, got %d", want, workType, typeCounts[workType])
		}
	}

	var list model.RPDBList
	if err := db.Where("user_id = ? AND is_default = ?", author.ID, true).First(&list).Error; err != nil {
		t.Fatalf("find demo list: %v", err)
	}
	if list.ID != existingDefaultList.ID {
		t.Fatalf("expected seed to reuse default list %d, got %d", existingDefaultList.ID, list.ID)
	}
	if list.Name != existingDefaultList.Name || list.Description != existingDefaultList.Description || list.IsPublic != existingDefaultList.IsPublic {
		t.Fatal("seed overwrote existing default list profile fields")
	}
	if !list.IsDefault || list.ItemCount != 12 {
		t.Fatalf("unexpected demo list state: default=%v item_count=%d", list.IsDefault, list.ItemCount)
	}
	var defaultLists int64
	if err := db.Model(&model.RPDBList{}).Where("user_id = ? AND is_default = ?", author.ID, true).Count(&defaultLists).Error; err != nil {
		t.Fatalf("count default lists: %v", err)
	}
	if defaultLists != 1 {
		t.Fatalf("expected exactly one default list, got %d", defaultLists)
	}
	var listEntries int64
	if err := db.Model(&model.RPDBListEntry{}).Where("list_id = ?", list.ID).Count(&listEntries).Error; err != nil {
		t.Fatalf("count demo list entries: %v", err)
	}
	if listEntries != 12 {
		t.Fatalf("expected 12 demo list entries, got %d", listEntries)
	}

	var set model.RPDBSet
	if err := db.Where("name = ?", "艾泽拉斯 RP 灵感精选").First(&set).Error; err != nil {
		t.Fatalf("find demo set: %v", err)
	}
	if !set.IsPublic || set.Status != model.RPDBStatusPublished || set.ReviewStatus != model.RPDBReviewApproved || set.ItemCount != 12 {
		t.Fatalf("unexpected demo set state: %+v", set)
	}
	var setWorks int64
	if err := db.Model(&model.RPDBSetWork{}).Where("set_id = ?", set.ID).Count(&setWorks).Error; err != nil {
		t.Fatalf("count demo set works: %v", err)
	}
	if setWorks != 12 {
		t.Fatalf("expected 12 demo set works, got %d", setWorks)
	}

	var storedUnrelated model.RPDBWork
	if err := db.First(&storedUnrelated, unrelated.ID).Error; err != nil {
		t.Fatalf("unrelated work was removed: %v", err)
	}
	if storedUnrelated.Title != unrelated.Title || storedUnrelated.Status != unrelated.Status {
		t.Fatal("unrelated work was modified")
	}

	var mutableWork model.RPDBWork
	if err := db.Where("slug = ?", "rpdb-demo-item-01").First(&mutableWork).Error; err != nil {
		t.Fatalf("find work for guide-step idempotency test: %v", err)
	}
	var mutableStep model.RPDBGuideStep
	if err := db.Where("work_id = ? AND sort_order = ?", mutableWork.ID, 1).First(&mutableStep).Error; err != nil {
		t.Fatalf("find guide step for metadata mutation: %v", err)
	}
	if err := db.Model(&mutableStep).Update("meta", `{"changed":"outside-seed"}`).Error; err != nil {
		t.Fatalf("mutate guide step metadata: %v", err)
	}
	duplicateStep := model.RPDBGuideStep{
		WorkID:    mutableWork.ID,
		SortOrder: 1,
		Title:     "重复演示步骤",
		Body:      "用于验证种子清理重复 sort_order。",
		Meta:      `{"duplicate":true}`,
	}
	if err := db.Create(&duplicateStep).Error; err != nil {
		t.Fatalf("create duplicate guide step: %v", err)
	}

	var homeWork model.RPDBWork
	if err := db.Where("slug = ?", "rpdb-demo-home-01").First(&homeWork).Error; err != nil {
		t.Fatalf("find home work for legacy cleanup test: %v", err)
	}
	legacyHomeStep := model.RPDBGuideStep{
		WorkID:    homeWork.ID,
		SortOrder: 1,
		Title:     "旧家宅结构化步骤",
		Meta:      `{"legacy":true}`,
	}
	if err := db.Create(&legacyHomeStep).Error; err != nil {
		t.Fatalf("create legacy home guide step: %v", err)
	}

	if err := SeedRPDBDemo(db); err != nil {
		t.Fatalf("seed RPDB demo a second time: %v", err)
	}
	for sortOrder := 1; sortOrder <= 3; sortOrder++ {
		var count int64
		if err := db.Model(&model.RPDBGuideStep{}).
			Where("work_id = ? AND sort_order = ?", mutableWork.ID, sortOrder).
			Count(&count).Error; err != nil {
			t.Fatalf("count guide step sort order %d: %v", sortOrder, err)
		}
		if count != 1 {
			t.Fatalf("expected one guide step at sort order %d, got %d", sortOrder, count)
		}
	}
	var homeSteps int64
	if err := db.Model(&model.RPDBGuideStep{}).Where("work_id = ?", homeWork.ID).Count(&homeSteps).Error; err != nil {
		t.Fatalf("count home guide steps after reseed: %v", err)
	}
	if homeSteps != 0 {
		t.Fatalf("expected home guide steps to be cleared, got %d", homeSteps)
	}

	stableCounts := rpdbSeedTableCounts(t, db)
	stableHashes := rpdbDemoAccountHashes(t, db)
	if err := SeedRPDBDemo(db); err != nil {
		t.Fatalf("seed RPDB demo a third time: %v", err)
	}
	thirdCounts := rpdbSeedTableCounts(t, db)
	for table, stable := range stableCounts {
		if thirdCounts[table] != stable {
			t.Fatalf("table %s changed after stable reseed: second=%d third=%d", table, stable, thirdCounts[table])
		}
	}
	thirdHashes := rpdbDemoAccountHashes(t, db)
	for username, stableHash := range stableHashes {
		if thirdHashes[username] != stableHash {
			t.Fatalf("disabled credential for %s changed after an idempotent reseed", username)
		}
	}

	if err := db.Where("slug LIKE ?", "rpdb-demo-%").Order("slug ASC").Find(&works).Error; err != nil {
		t.Fatalf("reload demo works: %v", err)
	}
	for _, work := range works {
		assertRPDBDemoWorkCounters(t, db, work)
	}
}

func TestHardenRPDBDemoAccountsRehardensExistingIdentitiesIdempotently(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{})
	now := time.Now().UTC().Truncate(time.Second)
	legacyModeratorID := uint(999999)
	compromisedPasswords := make(map[string]string)
	originalIDs := make(map[string]uint)
	originalHashes := make(map[string]string)

	for _, spec := range rpdbDemoAccountSpecs() {
		compromisedPassword := "test-only-known:" + spec.Username
		compromisedHash, err := authpkg.HashPassword(compromisedPassword)
		if err != nil {
			t.Fatalf("hash test-only compromised credential for %s: %v", spec.Username, err)
		}
		user := model.User{
			Username:      spec.Username,
			Email:         "legacy-" + spec.Email,
			EmailVerified: true,
			PassHash:      compromisedHash,
			Role:          "admin",
			MutedUntil:    &now,
			MuteReason:    "legacy moderation state",
			BannedUntil:   &now,
			BanReason:     "legacy moderation state",
			BannedBy:      &legacyModeratorID,
			BannedAt:      &now,
		}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create legacy demo identity %s: %v", spec.Username, err)
		}
		compromisedPasswords[spec.Username] = compromisedPassword
		originalIDs[spec.Username] = user.ID
		originalHashes[spec.Username] = compromisedHash
	}

	unrelatedPassword := "test-only-known:unrelated"
	unrelatedHash, err := authpkg.HashPassword(unrelatedPassword)
	if err != nil {
		t.Fatalf("hash unrelated test credential: %v", err)
	}
	unrelated := model.User{
		Username:      "unrelated_user",
		Email:         "unrelated@example.invalid",
		EmailVerified: true,
		PassHash:      unrelatedHash,
		Role:          "moderator",
	}
	if err := db.Create(&unrelated).Error; err != nil {
		t.Fatalf("create unrelated identity: %v", err)
	}

	if err := hardenRPDBDemoAccounts(db); err != nil {
		t.Fatalf("harden existing demo identities: %v", err)
	}
	firstHardenedHashes := make(map[string]string)
	for _, spec := range rpdbDemoAccountSpecs() {
		var user model.User
		if err := db.Where("username = ?", spec.Username).First(&user).Error; err != nil {
			t.Fatalf("reload hardened identity %s: %v", spec.Username, err)
		}
		if user.ID != originalIDs[spec.Username] {
			t.Fatalf("hardening changed stable ID for %s", spec.Username)
		}
		if user.Email != "legacy-"+spec.Email {
			t.Fatalf("hardening overwrote the existing email for %s", spec.Username)
		}
		if user.PassHash == originalHashes[spec.Username] || authpkg.CheckPassword(compromisedPasswords[spec.Username], user.PassHash) {
			t.Fatalf("hardening retained a known credential for %s", spec.Username)
		}
		assertRPDBDemoAccountStateDisabled(t, user, spec)
		firstHardenedHashes[spec.Username] = user.PassHash
	}

	if err := hardenRPDBDemoAccounts(db); err != nil {
		t.Fatalf("repeat demo identity hardening: %v", err)
	}
	secondHardenedHashes := rpdbDemoAccountHashes(t, db)
	for username, firstHash := range firstHardenedHashes {
		if secondHardenedHashes[username] != firstHash {
			t.Fatalf("disabled credential for %s changed during idempotent startup hardening", username)
		}
	}

	var storedUnrelated model.User
	if err := db.First(&storedUnrelated, unrelated.ID).Error; err != nil {
		t.Fatalf("reload unrelated identity: %v", err)
	}
	if storedUnrelated.PassHash != unrelatedHash || storedUnrelated.Role != unrelated.Role || storedUnrelated.IsBanned {
		t.Fatal("demo identity hardening modified an unrelated account")
	}
}

func TestHardenRPDBDemoAccountsFindsRenamedSeedIdentityByStableEmail(t *testing.T) {
	db := newRPDBSeedTestDB(t)
	spec := rpdbDemoAccountSpecs()[0]
	compromisedPassword := "test-only-known:renamed"
	compromisedHash, err := authpkg.HashPassword(compromisedPassword)
	if err != nil {
		t.Fatalf("hash test-only compromised credential: %v", err)
	}
	user := model.User{
		Username:      "renamed_demo_identity",
		Email:         spec.Email,
		EmailVerified: true,
		PassHash:      compromisedHash,
		Role:          "moderator",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create renamed demo identity: %v", err)
	}

	if err := hardenRPDBDemoAccounts(db); err != nil {
		t.Fatalf("harden renamed demo identity: %v", err)
	}
	var reloaded model.User
	if err := db.First(&reloaded, user.ID).Error; err != nil {
		t.Fatalf("reload renamed demo identity: %v", err)
	}
	if reloaded.Username != user.Username || reloaded.Email != user.Email {
		t.Fatal("hardening changed stable identity fields")
	}
	if authpkg.CheckPassword(compromisedPassword, reloaded.PassHash) {
		t.Fatal("renamed demo identity retained a known credential")
	}
	assertRPDBDemoAccountStateDisabled(t, reloaded, spec)

	if err := SeedRPDBDemo(db); err != nil {
		t.Fatalf("refresh demo data after identity rename: %v", err)
	}
	var afterSeed model.User
	if err := db.First(&afterSeed, user.ID).Error; err != nil {
		t.Fatalf("reload renamed demo identity after seed: %v", err)
	}
	if afterSeed.Username != user.Username || afterSeed.Email != user.Email {
		t.Fatal("demo refresh replaced or rewrote the renamed stable identity")
	}
	assertRPDBDemoAccountStateDisabled(t, afterSeed, spec)
	var authoredWorks int64
	if err := db.Model(&model.RPDBWork{}).Where("author_id = ? AND slug LIKE ?", user.ID, "rpdb-demo-%").Count(&authoredWorks).Error; err != nil {
		t.Fatalf("count renamed identity demo works: %v", err)
	}
	if authoredWorks != 12 {
		t.Fatalf("expected renamed stable identity to retain 12 demo works, got %d", authoredWorks)
	}
}

func assertRPDBDemoAccountDisabled(t *testing.T, db *gorm.DB, spec rpdbDemoAccountSpec) {
	t.Helper()
	var user model.User
	if err := db.Where("username = ?", spec.Username).First(&user).Error; err != nil {
		t.Fatalf("find disabled demo identity %s: %v", spec.Username, err)
	}
	assertRPDBDemoAccountStateDisabled(t, user, spec)
}

func assertRPDBDemoAccountStateDisabled(t *testing.T, user model.User, spec rpdbDemoAccountSpec) {
	t.Helper()
	if !isRPDBDemoAccountHardened(&user, spec) {
		t.Fatalf("demo identity %s is not fully disabled and unprivileged", spec.Username)
	}
}

func rpdbDemoAccountHashes(t *testing.T, db *gorm.DB) map[string]string {
	t.Helper()
	hashes := make(map[string]string)
	for _, spec := range rpdbDemoAccountSpecs() {
		var user model.User
		if err := db.Where("username = ?", spec.Username).First(&user).Error; err != nil {
			t.Fatalf("load disabled credential for %s: %v", spec.Username, err)
		}
		hashes[spec.Username] = user.PassHash
	}
	return hashes
}

func newRPDBSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Tag{},
		&model.RPDBWork{},
		&model.RPDBReference{},
		&model.RPDBMedia{},
		&model.RPDBTransmogSlot{},
		&model.RPDBGuideStep{},
		&model.RPDBTag{},
		&model.RPDBLike{},
		&model.RPDBFavorite{},
		&model.RPDBView{},
		&model.RPDBComment{},
		&model.RPDBCommentLike{},
		&model.RPDBList{},
		&model.RPDBListEntry{},
		&model.RPDBRevision{},
		&model.RPDBVerification{},
		&model.RPDBSet{},
		&model.RPDBSetWork{},
	)
	installRPDBJSONValidationTriggers(t, db)
	return db
}

func installRPDBJSONValidationTriggers(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		`CREATE TRIGGER validate_rpdb_works_json_before_insert
BEFORE INSERT ON rpdb_works
FOR EACH ROW
WHEN json_valid(NEW.restrictions) = 0 OR json_valid(NEW.extra) = 0
BEGIN
  SELECT RAISE(ABORT, 'invalid rpdb_works json');
END`,
		`CREATE TRIGGER validate_rpdb_works_json_before_update
BEFORE UPDATE ON rpdb_works
FOR EACH ROW
WHEN json_valid(NEW.restrictions) = 0 OR json_valid(NEW.extra) = 0
BEGIN
  SELECT RAISE(ABORT, 'invalid rpdb_works json');
END`,
		`CREATE TRIGGER validate_rpdb_media_json_before_insert
BEFORE INSERT ON rpdb_media
FOR EACH ROW
WHEN json_valid(NEW.meta) = 0
BEGIN
  SELECT RAISE(ABORT, 'invalid rpdb_media json');
END`,
		`CREATE TRIGGER validate_rpdb_media_json_before_update
BEFORE UPDATE ON rpdb_media
FOR EACH ROW
WHEN json_valid(NEW.meta) = 0
BEGIN
  SELECT RAISE(ABORT, 'invalid rpdb_media json');
END`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("install JSON validation trigger: %v", err)
		}
	}
}

func assertRPDBDemoWorkChildren(t *testing.T, db *gorm.DB, work model.RPDBWork) {
	t.Helper()

	var references int64
	db.Model(&model.RPDBReference{}).Where("work_id = ?", work.ID).Count(&references)
	if references < 1 {
		t.Fatalf("work %q has no references", work.Slug)
	}

	var media []model.RPDBMedia
	if err := db.Where("work_id = ?", work.ID).Find(&media).Error; err != nil {
		t.Fatalf("load media for %q: %v", work.Slug, err)
	}
	if len(media) != 1 {
		t.Fatalf("work %q should have exactly one demo media record, got %d", work.Slug, len(media))
	}
	for _, item := range media {
		if item.ReviewStatus != model.RPDBReviewApproved || item.ReviewerID == nil || item.ReviewedAt == nil {
			t.Fatalf("work %q has unapproved media", work.Slug)
		}
		if item.URL != rpdbDemoImageURL(work.Slug) {
			t.Fatalf("work %q media URL: expected %q, got %q", work.Slug, rpdbDemoImageURL(work.Slug), item.URL)
		}
	}

	var steps int64
	if err := db.Model(&model.RPDBGuideStep{}).Where("work_id = ?", work.ID).Count(&steps).Error; err != nil {
		t.Fatalf("count guide steps for %q: %v", work.Slug, err)
	}
	if work.Type == model.RPDBWorkTypeHomeShowcase {
		if steps != 0 {
			t.Fatalf("home showcase %q should not have guide steps, got %d", work.Slug, steps)
		}
	} else if steps < 3 {
		t.Fatalf("work %q has fewer than three guide steps", work.Slug)
	}

	var tags int64
	db.Model(&model.RPDBTag{}).Where("work_id = ?", work.ID).Count(&tags)
	if tags < 2 {
		t.Fatalf("work %q has fewer than two tags", work.Slug)
	}
	var nonStyleTags int64
	if err := db.Model(&model.Tag{}).
		Joins("JOIN rpdb_tags ON rpdb_tags.tag_id = tags.id").
		Where("rpdb_tags.work_id = ? AND tags.name NOT LIKE ?", work.ID, "%风格").
		Count(&nonStyleTags).Error; err != nil {
		t.Fatalf("count non-style tags for %q: %v", work.Slug, err)
	}
	if nonStyleTags != 0 {
		t.Fatalf("work %q should only use RP style tags, got %d non-style tags", work.Slug, nonStyleTags)
	}

	if work.Type == model.RPDBWorkTypeTransmog {
		var slots int64
		db.Model(&model.RPDBTransmogSlot{}).Where("work_id = ?", work.ID).Count(&slots)
		if slots < 4 {
			t.Fatalf("transmog work %q has fewer than four slots", work.Slug)
		}
	}
}

func assertRPDBDemoHomeExtra(t *testing.T, work model.RPDBWork) {
	t.Helper()
	var extra struct {
		Server      string `json:"server"`
		Region      string `json:"region"`
		HomeStyle   string `json:"home_style"`
		ShareCode   string `json:"share_code"`
		VisitNotes  string `json:"visit_notes"`
		CopyStatus  string `json:"copy_status"`
		VisitStatus string `json:"visit_status"`
		SpaceType   string `json:"space_type"`
	}
	if err := json.Unmarshal([]byte(work.Extra), &extra); err != nil {
		t.Fatalf("parse home extra for %q: %v", work.Slug, err)
	}
	fields := map[string]string{
		"server":       extra.Server,
		"region":       extra.Region,
		"home_style":   extra.HomeStyle,
		"share_code":   extra.ShareCode,
		"visit_notes":  extra.VisitNotes,
		"copy_status":  extra.CopyStatus,
		"visit_status": extra.VisitStatus,
		"space_type":   extra.SpaceType,
	}
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			t.Fatalf("home extra field %s is empty for %q", name, work.Slug)
		}
	}
}

func rpdbDemoImageURL(slug string) string {
	return "/uploads/rpdb/demo/" + strings.TrimPrefix(slug, "rpdb-demo-") + ".jpg"
}

func assertRPDBDemoWorkCounters(t *testing.T, db *gorm.DB, work model.RPDBWork) {
	t.Helper()

	counts := map[string]int64{}
	queries := []struct {
		name  string
		model interface{}
		where string
		args  []interface{}
	}{
		{"views", &model.RPDBView{}, "work_id = ?", []interface{}{work.ID}},
		{"likes", &model.RPDBLike{}, "work_id = ?", []interface{}{work.ID}},
		{"favorites", &model.RPDBFavorite{}, "work_id = ?", []interface{}{work.ID}},
		{"comments", &model.RPDBComment{}, "work_id = ? AND status = ?", []interface{}{work.ID, "published"}},
		{"lists", &model.RPDBListEntry{}, "work_id = ?", []interface{}{work.ID}},
		{"media", &model.RPDBMedia{}, "work_id = ?", []interface{}{work.ID}},
		{"valid", &model.RPDBVerification{}, "work_id = ? AND result = ?", []interface{}{work.ID, "valid"}},
		{"outdated", &model.RPDBVerification{}, "work_id = ? AND result = ?", []interface{}{work.ID, "outdated"}},
	}
	for _, query := range queries {
		var count int64
		if err := db.Model(query.model).Where(query.where, query.args...).Count(&count).Error; err != nil {
			t.Fatalf("count %s for %q: %v", query.name, work.Slug, err)
		}
		counts[query.name] = count
	}

	if counts["views"] < 1 || counts["likes"] < 1 || counts["favorites"] < 1 || counts["comments"] < 1 || counts["lists"] < 1 {
		t.Fatalf("work %q is missing seeded interactions: %+v", work.Slug, counts)
	}
	if counts["valid"] < 2 || work.VerificationStatus != model.RPDBVerificationVerified || work.LastVerifiedAt == nil {
		t.Fatalf("work %q is not verified: valid=%d status=%q", work.Slug, counts["valid"], work.VerificationStatus)
	}

	if int64(work.ViewCount) != counts["views"] ||
		int64(work.LikeCount) != counts["likes"] ||
		int64(work.FavoriteCount) != counts["favorites"] ||
		int64(work.CommentCount) != counts["comments"] ||
		int64(work.ListCount) != counts["lists"] ||
		int64(work.MediaCount) != counts["media"] ||
		int64(work.VerifiedCount) != counts["valid"] ||
		int64(work.OutdatedCount) != counts["outdated"] {
		t.Fatalf("work %q counters do not match rows: work=%+v rows=%+v", work.Slug, work, counts)
	}
}

func rpdbSeedTableCounts(t *testing.T, db *gorm.DB) map[string]int64 {
	t.Helper()

	models := map[string]interface{}{
		"users":          &model.User{},
		"tags":           &model.Tag{},
		"works":          &model.RPDBWork{},
		"references":     &model.RPDBReference{},
		"media":          &model.RPDBMedia{},
		"transmog_slots": &model.RPDBTransmogSlot{},
		"guide_steps":    &model.RPDBGuideStep{},
		"work_tags":      &model.RPDBTag{},
		"likes":          &model.RPDBLike{},
		"favorites":      &model.RPDBFavorite{},
		"views":          &model.RPDBView{},
		"comments":       &model.RPDBComment{},
		"comment_likes":  &model.RPDBCommentLike{},
		"lists":          &model.RPDBList{},
		"list_entries":   &model.RPDBListEntry{},
		"verifications":  &model.RPDBVerification{},
		"sets":           &model.RPDBSet{},
		"set_works":      &model.RPDBSetWork{},
	}
	counts := make(map[string]int64, len(models))
	for name, value := range models {
		var count int64
		if err := db.Model(value).Count(&count).Error; err != nil {
			t.Fatalf("count %s: %v", name, err)
		}
		counts[name] = count
	}
	return counts
}
