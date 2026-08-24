package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func newStructuredReportTargetTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.NewTestDB(
		t,
		&model.User{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
		&model.ContentReport{},
		&model.AdminActionLog{},
		&model.CharacterCard{},
		&model.CharacterCardPublication{},
		&model.CharacterCardSubmission{},
		&model.CharacterCardPortrait{},
		&model.CharacterCardImpression{},
		&model.StoryEntry{},
		&model.Guild{},
		&model.GuildMember{},
	)
}

func createStructuredReportTargetUsers(t *testing.T, db *gorm.DB) (model.User, model.User, model.User) {
	t.Helper()
	author := model.User{Username: "structured-author", Email: "structured-author@example.com", PassHash: "hash", Role: "user"}
	reporter := model.User{Username: "structured-reporter", Email: "structured-reporter@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "structured-moderator", Email: "structured-moderator@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &reporter, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	return author, reporter, moderator
}

func createStructuredCharacterCardsAndGuilds(t *testing.T, db *gorm.DB, authorID uint) ([]model.CharacterCard, []model.Guild) {
	t.Helper()
	cards := []model.CharacterCard{
		{
			UserID: authorID, DisplayName: "艾琳·星语", Summary: "艾琳的人物卡摘要", PortraitImage: "/uploads/character-cards/author/one.png",
			Status: model.CharacterCardStatusPublished, Visibility: model.CharacterCardVisibilityPublic, ReviewStatus: model.CharacterCardReviewApproved,
		},
		{
			UserID: authorID, DisplayName: "罗兰·灰羽", Summary: "罗兰的人物卡摘要", PortraitImage: "/uploads/character-cards/author/two.png",
			Status: model.CharacterCardStatusPublished, Visibility: model.CharacterCardVisibilityPublic, ReviewStatus: model.CharacterCardReviewApproved,
		},
	}
	if err := db.Create(&cards).Error; err != nil {
		t.Fatalf("create character cards: %v", err)
	}
	guilds := []model.Guild{
		{Name: "星语议会", OwnerID: authorID, Description: "守护星语的公会", Banner: "/uploads/guilds/one/banner.png", InviteCode: "report-guild-1", Status: "approved", IsPublic: true},
		{Name: "灰羽旅团", OwnerID: authorID, Description: "游历艾泽拉斯的公会", Banner: "/uploads/guilds/two/banner.png", InviteCode: "report-guild-2", Status: "approved", IsPublic: true},
	}
	if err := db.Create(&guilds).Error; err != nil {
		t.Fatalf("create guilds: %v", err)
	}
	return cards, guilds
}

func submitStructuredTargetReport(t *testing.T, server *Server, token, targetType string, targetID uint) {
	t.Helper()
	resp := performRequest(server.router, http.MethodPost, "/api/v1/reports", map[string]interface{}{
		"target_type": targetType,
		"target_id":   targetID,
		"reason":      "abuse",
		"detail":      "需要版主核查的具体内容",
	}, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("submit %s report: status=%d body=%s", targetType, resp.Code, resp.Body.String())
	}
}

func TestCharacterCardAndGuildReportsStayObjectScopedAndListExactTargets(t *testing.T) {
	db := newStructuredReportTargetTestDB(t)
	author, reporter, moderator := createStructuredReportTargetUsers(t, db)
	cards, guilds := createStructuredCharacterCardsAndGuilds(t, db, author.ID)
	publicSnapshot := characterCardSnapshot{
		Card: characterCardSnapshotCard{
			ID: cards[0].ID, UserID: author.ID, DisplayName: cards[0].DisplayName, Summary: cards[0].Summary,
			PortraitImage: cards[0].PortraitImage, CreatedAt: cards[0].CreatedAt, UpdatedAt: cards[0].UpdatedAt,
		},
	}
	publicPayload, err := json.Marshal(publicSnapshot)
	if err != nil {
		t.Fatalf("encode public card snapshot: %v", err)
	}
	if err := db.Create(&model.CharacterCardPublication{
		CharacterCardID: cards[0].ID, UserID: author.ID, Payload: string(publicPayload), ApprovedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("create public card snapshot: %v", err)
	}
	if err := db.Model(&model.CharacterCard{}).Where("id = ?", cards[0].ID).Updates(map[string]interface{}{
		"display_name": "未审核的私密名称",
		"summary":      "未审核的私密摘要",
	}).Error; err != nil {
		t.Fatalf("update live card draft: %v", err)
	}
	server := newTestServer(t, db)
	reporterToken := newTestToken(t, reporter)

	for _, card := range cards {
		submitStructuredTargetReport(t, server, reporterToken, reportTargetCharacterCard, card.ID)
	}
	for _, guild := range guilds {
		submitStructuredTargetReport(t, server, reporterToken, reportTargetGuild, guild.ID)
	}

	for _, target := range []struct {
		targetType string
		targetID   uint
	}{
		{reportTargetCharacterCard, cards[0].ID},
		{reportTargetGuild, guilds[0].ID},
	} {
		resp := performRequest(server.router, http.MethodPost, "/api/v1/reports", map[string]interface{}{
			"target_type": target.targetType,
			"target_id":   target.targetID,
			"reason":      "other",
			"detail":      "self report",
		}, newTestToken(t, author))
		if resp.Code != http.StatusBadRequest || !strings.Contains(resp.Body.String(), "不能举报自己") {
			t.Fatalf("expected self-report rejection for %s, got status=%d body=%s", target.targetType, resp.Code, resp.Body.String())
		}
	}

	resp := performRequest(server.router, http.MethodGet, "/api/v1/moderator/reports?target_scope=content", nil, newTestToken(t, moderator))
	if resp.Code != http.StatusOK {
		t.Fatalf("list structured reports: status=%d body=%s", resp.Code, resp.Body.String())
	}
	var payload struct {
		Reports []struct {
			TargetType         string `json:"target_type"`
			TargetID           uint   `json:"target_id"`
			TargetUserID       uint   `json:"target_user_id"`
			TargetTitle        string `json:"target_title"`
			TargetAuthorName   string `json:"target_author_name"`
			TargetPreviewText  string `json:"target_preview_text"`
			TargetPreviewImage string `json:"target_preview_image"`
			TargetURL          string `json:"target_url"`
			ReportCount        int64  `json:"report_count"`
		} `json:"reports"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode report list: %v", err)
	}
	if payload.Total != 4 || len(payload.Reports) != 4 {
		t.Fatalf("expected four independent object groups, total=%d reports=%#v", payload.Total, payload.Reports)
	}

	groups := make(map[string]struct {
		Title, Author, Preview, Image, URL string
		TargetUserID                       uint
		Count                              int64
	})
	for _, report := range payload.Reports {
		key := fmt.Sprintf("%s:%d", report.TargetType, report.TargetID)
		groups[key] = struct {
			Title, Author, Preview, Image, URL string
			TargetUserID                       uint
			Count                              int64
		}{report.TargetTitle, report.TargetAuthorName, report.TargetPreviewText, report.TargetPreviewImage, report.TargetURL, report.TargetUserID, report.ReportCount}
	}
	for _, card := range cards {
		group, ok := groups[fmt.Sprintf("%s:%d", reportTargetCharacterCard, card.ID)]
		if !ok || group.Title != card.DisplayName || group.Author != author.Username || group.TargetUserID != author.ID || group.Count != 1 {
			t.Fatalf("unexpected character-card group for %d: %#v", card.ID, group)
		}
		if !strings.Contains(group.Preview, card.Summary) || !strings.Contains(group.Image, fmt.Sprintf("/api/v1/images/character-card-portrait/%d", card.ID)) || group.URL != fmt.Sprintf("/character-cards/%d", card.ID) {
			t.Fatalf("unexpected character-card preview for %d: %#v", card.ID, group)
		}
	}
	for _, guild := range guilds {
		group, ok := groups[fmt.Sprintf("%s:%d", reportTargetGuild, guild.ID)]
		if !ok || group.Title != guild.Name || group.Author != author.Username || group.TargetUserID != author.ID || group.Count != 1 {
			t.Fatalf("unexpected guild group for %d: %#v", guild.ID, group)
		}
		if !strings.Contains(group.Preview, guild.Description) || group.Image != guild.Banner || group.URL != fmt.Sprintf("/guild/%d", guild.ID) {
			t.Fatalf("unexpected guild preview for %d: %#v", guild.ID, group)
		}
	}
}

func TestCharacterCardAndGuildHideAndBlockPersistAcrossDetailsAndLists(t *testing.T) {
	db := newStructuredReportTargetTestDB(t)
	author, viewer, _ := createStructuredReportTargetUsers(t, db)
	blocker := model.User{Username: "structured-blocker", Email: "structured-blocker@example.com", PassHash: "hash", Role: "user"}
	if err := db.Create(&blocker).Error; err != nil {
		t.Fatalf("create blocker: %v", err)
	}
	cards, guilds := createStructuredCharacterCardsAndGuilds(t, db, author.ID)
	server := newTestServer(t, db)
	viewerToken := newTestToken(t, viewer)

	hide := func(targetType string, targetID uint, token string) {
		t.Helper()
		resp := performRequest(server.router, http.MethodPost, "/api/v1/reports", map[string]interface{}{
			"target_type":   targetType,
			"target_id":     targetID,
			"hide_target":   true,
			"submit_report": false,
		}, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("hide %s: status=%d body=%s", targetType, resp.Code, resp.Body.String())
		}
	}
	hide(reportTargetCharacterCard, cards[0].ID, viewerToken)
	hide(reportTargetGuild, guilds[0].ID, viewerToken)

	for _, target := range []struct {
		path       string
		wantStatus int
	}{
		{fmt.Sprintf("/api/v1/character-cards/%d", cards[0].ID), http.StatusNotFound},
		{fmt.Sprintf("/api/v1/character-cards/%d", cards[1].ID), http.StatusOK},
		{fmt.Sprintf("/api/v1/guilds/%d", guilds[0].ID), http.StatusNotFound},
		{fmt.Sprintf("/api/v1/guilds/%d", guilds[1].ID), http.StatusOK},
	} {
		resp := performRequest(server.router, http.MethodGet, target.path, nil, viewerToken)
		if resp.Code != target.wantStatus {
			t.Fatalf("GET %s: got=%d want=%d body=%s", target.path, resp.Code, target.wantStatus, resp.Body.String())
		}
	}

	cardListResp := performRequest(server.router, http.MethodGet, fmt.Sprintf("/api/v1/users/%d/character-cards", author.ID), nil, viewerToken)
	if cardListResp.Code != http.StatusOK {
		t.Fatalf("list visible cards: status=%d body=%s", cardListResp.Code, cardListResp.Body.String())
	}
	var cardList struct {
		Cards []struct {
			ID uint `json:"id"`
		} `json:"character_cards"`
	}
	if err := json.Unmarshal(cardListResp.Body.Bytes(), &cardList); err != nil {
		t.Fatalf("decode card list: %v", err)
	}
	if len(cardList.Cards) != 1 || cardList.Cards[0].ID != cards[1].ID {
		t.Fatalf("expected only unhidden card %d, got %#v", cards[1].ID, cardList.Cards)
	}

	guildListResp := performRequest(server.router, http.MethodGet, "/api/v1/public/guilds", nil, viewerToken)
	if guildListResp.Code != http.StatusOK {
		t.Fatalf("list visible guilds: status=%d body=%s", guildListResp.Code, guildListResp.Body.String())
	}
	var guildList struct {
		Guilds []struct {
			ID uint `json:"id"`
		} `json:"guilds"`
	}
	if err := json.Unmarshal(guildListResp.Body.Bytes(), &guildList); err != nil {
		t.Fatalf("decode guild list: %v", err)
	}
	if len(guildList.Guilds) != 1 || guildList.Guilds[0].ID != guilds[1].ID {
		t.Fatalf("expected only unhidden guild %d, got %#v", guilds[1].ID, guildList.Guilds)
	}

	blockResp := performRequest(server.router, http.MethodPost, "/api/v1/reports", map[string]interface{}{
		"target_type":   reportTargetGuild,
		"target_id":     guilds[0].ID,
		"block_author":  true,
		"submit_report": false,
	}, newTestToken(t, blocker))
	if blockResp.Code != http.StatusOK {
		t.Fatalf("block guild owner: status=%d body=%s", blockResp.Code, blockResp.Body.String())
	}
	for _, targetPath := range []string{
		fmt.Sprintf("/api/v1/character-cards/%d", cards[1].ID),
		fmt.Sprintf("/api/v1/guilds/%d", guilds[1].ID),
	} {
		resp := performRequest(server.router, http.MethodGet, targetPath, nil, newTestToken(t, blocker))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("blocked owner target remained visible at %s: status=%d body=%s", targetPath, resp.Code, resp.Body.String())
		}
	}
}

func TestModeratorDeleteCharacterCardReportCleansExactAggregateAndMutesOwner(t *testing.T) {
	db := newStructuredReportTargetTestDB(t)
	author, reporter, moderator := createStructuredReportTargetUsers(t, db)
	secondReporter := model.User{Username: "second-card-reporter", Email: "second-card-reporter@example.com", PassHash: "hash", Role: "user"}
	if err := db.Create(&secondReporter).Error; err != nil {
		t.Fatalf("create second reporter: %v", err)
	}
	cards, _ := createStructuredCharacterCardsAndGuilds(t, db, author.ID)
	targetCard := cards[0]
	server := newTestServer(t, db)

	livePortrait := writeCharacterCardTestPNG(t, server, fmt.Sprintf("character-cards/%d/portrait/live.png", author.ID))
	liveGallery := writeCharacterCardTestPNG(t, server, fmt.Sprintf("character-cards/%d/portrait/gallery.png", author.ID))
	liveIcon := writeCharacterCardTestPNG(t, server, fmt.Sprintf("character-cards/%d/impression-icon/live.png", author.ID))
	snapshotAsset := writeCharacterCardTestPNG(t, server, fmt.Sprintf("character-cards/%d/archive/snapshot.png", author.ID))
	if err := db.Model(&model.CharacterCard{}).Where("id = ?", targetCard.ID).Update("portrait_image", livePortrait).Error; err != nil {
		t.Fatalf("set live portrait: %v", err)
	}
	targetCard.PortraitImage = livePortrait
	portrait := model.CharacterCardPortrait{CharacterCardID: targetCard.ID, SortOrder: 0, Image: liveGallery}
	impression := model.CharacterCardImpression{CharacterCardID: targetCard.ID, Slot: 1, Active: true, Title: "印象", IconImage: liveIcon}
	if err := db.Create(&portrait).Error; err != nil {
		t.Fatalf("create card portrait: %v", err)
	}
	if err := db.Create(&impression).Error; err != nil {
		t.Fatalf("create card impression: %v", err)
	}
	cardID := targetCard.ID
	entry := model.StoryEntry{StoryID: 99, SourceID: "structured-card-entry", CharacterCardID: &cardID, Content: "linked entry", Timestamp: time.Now()}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("create linked story entry: %v", err)
	}

	snapshot := characterCardSnapshot{
		Card:      characterCardSnapshotCard{ID: targetCard.ID, UserID: author.ID, DisplayName: targetCard.DisplayName, PortraitImage: snapshotAsset, UpdatedAt: time.Now()},
		Portraits: []characterCardSnapshotPortrait{{ID: 5001, SortOrder: 0, Image: snapshotAsset, UpdatedAt: time.Now()}},
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("encode character-card snapshot: %v", err)
	}
	if err := db.Create(&model.CharacterCardPublication{CharacterCardID: targetCard.ID, UserID: author.ID, Payload: string(payload), ApprovedAt: time.Now()}).Error; err != nil {
		t.Fatalf("create publication: %v", err)
	}
	if err := db.Create(&model.CharacterCardSubmission{CharacterCardID: targetCard.ID, UserID: author.ID, Payload: string(payload), SubmittedAt: time.Now()}).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	submitStructuredTargetReport(t, server, newTestToken(t, reporter), reportTargetCharacterCard, targetCard.ID)
	submitStructuredTargetReport(t, server, newTestToken(t, secondReporter), reportTargetCharacterCard, targetCard.ID)
	var report model.ContentReport
	if err := db.Where("target_type = ? AND target_id = ?", reportTargetCharacterCard, targetCard.ID).First(&report).Error; err != nil {
		t.Fatalf("load pending report: %v", err)
	}
	reviewResp := performRequest(server.router, http.MethodPost, fmt.Sprintf("/api/v1/moderator/reports/%d/review", report.ID), map[string]interface{}{
		"action": "delete_and_mute_user", "duration": 24, "comment": "confirmed",
	}, newTestToken(t, moderator))
	if reviewResp.Code != http.StatusOK {
		t.Fatalf("delete character card report: status=%d body=%s", reviewResp.Code, reviewResp.Body.String())
	}

	if err := db.First(&model.CharacterCard{}, targetCard.ID).Error; err == nil {
		t.Fatalf("expected exact reported card deleted")
	} else if err != gorm.ErrRecordNotFound {
		t.Fatalf("load deleted card: %v", err)
	}
	if err := db.First(&model.CharacterCard{}, cards[1].ID).Error; err != nil {
		t.Fatalf("other card by same author must remain: %v", err)
	}
	for _, check := range []struct {
		model interface{}
		where string
	}{
		{&model.CharacterCardPortrait{}, "character_card_id = ?"},
		{&model.CharacterCardImpression{}, "character_card_id = ?"},
		{&model.CharacterCardPublication{}, "character_card_id = ?"},
		{&model.CharacterCardSubmission{}, "character_card_id = ?"},
	} {
		var count int64
		if err := db.Model(check.model).Where(check.where, targetCard.ID).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("expected %T cascade removed, count=%d err=%v", check.model, count, err)
		}
	}
	var storedEntry model.StoryEntry
	if err := db.First(&storedEntry, entry.ID).Error; err != nil || storedEntry.CharacterCardID != nil {
		t.Fatalf("expected story entry detached, entry=%#v err=%v", storedEntry, err)
	}
	for _, asset := range []string{livePortrait, liveGallery, liveIcon, snapshotAsset} {
		if _, err := os.Stat(characterCardTestUploadFile(server, asset)); !os.IsNotExist(err) {
			t.Fatalf("expected unreferenced card asset removed %q: %v", asset, err)
		}
	}
	var unresolved int64
	if err := db.Model(&model.ContentReport{}).Where("target_type = ? AND target_id = ? AND status = ?", reportTargetCharacterCard, targetCard.ID, "pending").Count(&unresolved).Error; err != nil || unresolved != 0 {
		t.Fatalf("expected all pending card reports resolved, count=%d err=%v", unresolved, err)
	}
	var resolved int64
	if err := db.Model(&model.ContentReport{}).Where("target_type = ? AND target_id = ? AND status = ?", reportTargetCharacterCard, targetCard.ID, "resolved").Count(&resolved).Error; err != nil || resolved != 2 {
		t.Fatalf("expected two resolved card reports, count=%d err=%v", resolved, err)
	}
	var storedAuthor model.User
	if err := db.First(&storedAuthor, author.ID).Error; err != nil || !storedAuthor.IsMuted {
		t.Fatalf("expected card owner muted, user=%#v err=%v", storedAuthor, err)
	}
}

func TestModeratorDeleteGuildReportUsesGuildCascadeAndBansOwner(t *testing.T) {
	db := newGuildDeletionTestDB(t)
	owner, reporter, guild, _, post := createGuildDeletionFixture(t, db)
	moderator := model.User{Username: "guild-report-moderator", Email: "guild-report-moderator@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&moderator).Error; err != nil {
		t.Fatalf("create moderator: %v", err)
	}
	if err := db.AutoMigrate(&model.ContentReport{}, &model.AdminActionLog{}); err != nil {
		t.Fatalf("migrate report models: %v", err)
	}
	otherGuild := model.Guild{Name: "Keep Same Owner Guild", OwnerID: owner.ID, InviteCode: "keep-owner-guild", Status: "approved", IsPublic: true}
	if err := db.Create(&otherGuild).Error; err != nil {
		t.Fatalf("create other guild: %v", err)
	}

	server := newTestServer(t, db)
	avatar := writeCharacterCardTestPNG(t, server, fmt.Sprintf("guilds/%d/avatar/report.png", guild.ID))
	if err := db.Model(&model.Guild{}).Where("id = ?", guild.ID).Update("avatar", avatar).Error; err != nil {
		t.Fatalf("set guild avatar: %v", err)
	}
	submitStructuredTargetReport(t, server, newTestToken(t, reporter), reportTargetGuild, guild.ID)
	currentOwner := model.User{Username: "current-guild-owner", Email: "current-guild-owner@example.com", PassHash: "hash", Role: "user"}
	if err := db.Create(&currentOwner).Error; err != nil {
		t.Fatalf("create current owner: %v", err)
	}
	if err := db.Model(&model.Guild{}).Where("id = ?", guild.ID).Update("owner_id", currentOwner.ID).Error; err != nil {
		t.Fatalf("transfer reported guild before review: %v", err)
	}
	listResp := performRequest(server.router, http.MethodGet, "/api/v1/moderator/reports?target_scope=content&target_type=guild", nil, newTestToken(t, moderator))
	if listResp.Code != http.StatusOK {
		t.Fatalf("list transferred guild report: status=%d body=%s", listResp.Code, listResp.Body.String())
	}
	var listPayload struct {
		Reports []struct {
			TargetUserID     uint   `json:"target_user_id"`
			TargetAuthorName string `json:"target_author_name"`
		} `json:"reports"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listPayload); err != nil || len(listPayload.Reports) != 1 {
		t.Fatalf("decode transferred guild report: payload=%#v err=%v", listPayload, err)
	}
	if listPayload.Reports[0].TargetUserID != currentOwner.ID || listPayload.Reports[0].TargetAuthorName != currentOwner.Username {
		t.Fatalf("expected current guild owner in report payload, got %#v", listPayload.Reports[0])
	}
	var report model.ContentReport
	if err := db.Where("target_type = ? AND target_id = ?", reportTargetGuild, guild.ID).First(&report).Error; err != nil {
		t.Fatalf("load guild report: %v", err)
	}
	reviewResp := performRequest(server.router, http.MethodPost, fmt.Sprintf("/api/v1/moderator/reports/%d/review", report.ID), map[string]interface{}{
		"action": "delete_and_ban_user", "duration": 48, "comment": "confirmed",
	}, newTestToken(t, moderator))
	if reviewResp.Code != http.StatusOK {
		t.Fatalf("delete guild report: status=%d body=%s", reviewResp.Code, reviewResp.Body.String())
	}

	assertRecordCount(t, db, &model.Guild{}, "id = ?", []interface{}{guild.ID}, 0)
	assertRecordCount(t, db, &model.GuildMember{}, "guild_id = ?", []interface{}{guild.ID}, 0)
	assertRecordCount(t, db, &model.GuildApplication{}, "guild_id = ?", []interface{}{guild.ID}, 0)
	if err := db.First(&model.Guild{}, otherGuild.ID).Error; err != nil {
		t.Fatalf("other guild by same owner must remain: %v", err)
	}
	var preservedPost model.Post
	if err := db.First(&preservedPost, post.ID).Error; err != nil || preservedPost.GuildID != nil {
		t.Fatalf("expected guild post preserved and detached, post=%#v err=%v", preservedPost, err)
	}
	if _, err := os.Stat(characterCardTestUploadFile(server, avatar)); !os.IsNotExist(err) {
		t.Fatalf("expected guild avatar removed: %v", err)
	}
	var storedOwner model.User
	if err := db.First(&storedOwner, owner.ID).Error; err != nil || storedOwner.IsBanned {
		t.Fatalf("expected former guild owner to remain unbanned, user=%#v err=%v", storedOwner, err)
	}
	var storedCurrentOwner model.User
	if err := db.First(&storedCurrentOwner, currentOwner.ID).Error; err != nil || !storedCurrentOwner.IsBanned {
		t.Fatalf("expected current guild owner banned, user=%#v err=%v", storedCurrentOwner, err)
	}
	var storedReport model.ContentReport
	if err := db.First(&storedReport, report.ID).Error; err != nil || storedReport.Status != "resolved" {
		t.Fatalf("expected guild report resolved, report=%#v err=%v", storedReport, err)
	}
}
