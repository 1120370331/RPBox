package api

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func newRPDBModerationTestServer(t *testing.T) (*Server, model.User, model.User, string) {
	t.Helper()
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Tag{},
		&model.Notification{},
		&model.AdminActionLog{},
		&model.RPDBWork{},
		&model.RPDBReference{},
		&model.RPDBMedia{},
		&model.RPDBTransmogSlot{},
		&model.RPDBGuideStep{},
		&model.RPDBTag{},
		&model.RPDBRevision{},
	)
	author := model.User{Username: "rpdb-author", Email: "rpdb-author@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "rpdb-moderator", Email: "rpdb-moderator@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	return newTestServer(t, db), author, moderator, newTestToken(t, moderator)
}

func TestRPDBModerationApprovesPendingWork(t *testing.T) {
	server, author, moderator, token := newRPDBModerationTestServer(t)
	work := model.RPDBWork{
		AuthorID:     author.ID,
		Type:         model.RPDBWorkTypeItemShowcase,
		Title:        "待审核作品",
		Status:       model.RPDBStatusPending,
		ReviewStatus: model.RPDBReviewPending,
		Version:      1,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}

	resp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10),
		map[string]interface{}{"action": "approve", "comment": "信息完整"},
		token,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var stored model.RPDBWork
	if err := database.DB.First(&stored, work.ID).Error; err != nil {
		t.Fatalf("load work: %v", err)
	}
	if stored.Status != model.RPDBStatusPublished || stored.ReviewStatus != model.RPDBReviewApproved || !stored.IsPublic {
		t.Fatalf("unexpected approved state: %#v", stored)
	}
	if stored.ReviewerID == nil || *stored.ReviewerID != moderator.ID || stored.ReviewedAt == nil {
		t.Fatalf("missing reviewer metadata: %#v", stored)
	}
}

func TestRPDBModerationApprovesPendingWorkPublishesCustomRPDBTags(t *testing.T) {
	server, author, _, token := newRPDBModerationTestServer(t)
	work := model.RPDBWork{
		AuthorID:     author.ID,
		Type:         model.RPDBWorkTypeItemShowcase,
		Title:        "暮色森林巡林灯",
		Status:       model.RPDBStatusPending,
		ReviewStatus: model.RPDBReviewPending,
		IsPublic:     true,
		Version:      1,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	tag := model.Tag{Name: "暮色森林风格", Color: "B87333", Category: "rpdb", Type: "custom", CreatorID: author.ID, IsPublic: false}
	if err := database.DB.Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if err := database.DB.Create(&model.RPDBTag{WorkID: work.ID, TagID: tag.ID, AddedBy: author.ID}).Error; err != nil {
		t.Fatalf("create work tag: %v", err)
	}

	resp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10),
		map[string]interface{}{"action": "approve"},
		token,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var storedTag model.Tag
	if err := database.DB.First(&storedTag, tag.ID).Error; err != nil {
		t.Fatalf("load tag: %v", err)
	}
	if !storedTag.IsPublic {
		t.Fatalf("expected custom tag to become searchable after approval: %#v", storedTag)
	}
}

func TestRPDBModerationRejectsStaleRevisionConflict(t *testing.T) {
	server, author, _, token := newRPDBModerationTestServer(t)
	work := model.RPDBWork{
		AuthorID:     author.ID,
		Type:         model.RPDBWorkTypeItemShowcase,
		Title:        "正式标题",
		Status:       model.RPDBStatusPublished,
		ReviewStatus: model.RPDBReviewApproved,
		IsPublic:     true,
		Version:      2,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	revision := model.RPDBRevision{
		WorkID:      work.ID,
		ProposerID:  author.ID,
		BaseVersion: 1,
		Payload:     `{"title":"过期修订"}`,
		Status:      model.RPDBReviewPending,
	}
	if err := database.DB.Create(&revision).Error; err != nil {
		t.Fatalf("create revision: %v", err)
	}

	resp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/rpdb/revisions/"+strconv.FormatUint(uint64(revision.ID), 10),
		map[string]interface{}{"action": "approve"},
		token,
	)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", resp.Code, resp.Body.String())
	}

	var storedRevision model.RPDBRevision
	if err := database.DB.First(&storedRevision, revision.ID).Error; err != nil {
		t.Fatalf("load revision: %v", err)
	}
	if storedRevision.Status != model.RPDBReviewPending {
		t.Fatalf("stale revision status changed: %#v", storedRevision)
	}
}

func TestRPDBModerationApprovesTitleOnlyRevision(t *testing.T) {
	server, author, _, moderatorToken := newRPDBModerationTestServer(t)
	work := model.RPDBWork{
		AuthorID:          author.ID,
		Type:              model.RPDBWorkTypeItemShowcase,
		Title:             "原始标题",
		Summary:           "原始摘要",
		Content:           "<p>原始正文</p>",
		EffectDescription: "原始效果",
		Status:            model.RPDBStatusPublished,
		ReviewStatus:      model.RPDBReviewApproved,
		IsPublic:          true,
		Version:           1,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	reference := model.RPDBReference{
		WorkID:       work.ID,
		ExternalType: "item",
		ExternalID:   "190001",
		Name:         "仪式蜡烛",
	}
	if err := database.DB.Create(&reference).Error; err != nil {
		t.Fatalf("create reference: %v", err)
	}

	authorToken := newTestToken(t, author)
	updateResp := performRequest(
		server.router,
		http.MethodPut,
		"/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10),
		map[string]interface{}{"title": "修订后的标题", "change_summary": "修正标题"},
		authorToken,
	)
	if updateResp.Code != http.StatusAccepted {
		t.Fatalf("expected revision creation 202, got %d body=%s", updateResp.Code, updateResp.Body.String())
	}

	var revision model.RPDBRevision
	if err := database.DB.Where("work_id = ?", work.ID).First(&revision).Error; err != nil {
		t.Fatalf("load revision: %v", err)
	}
	reviewResp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/rpdb/revisions/"+strconv.FormatUint(uint64(revision.ID), 10),
		map[string]interface{}{"action": "approve"},
		moderatorToken,
	)
	if reviewResp.Code != http.StatusOK {
		t.Fatalf("expected revision approval 200, got %d body=%s payload=%s", reviewResp.Code, reviewResp.Body.String(), revision.Payload)
	}

	var stored model.RPDBWork
	if err := database.DB.First(&stored, work.ID).Error; err != nil {
		t.Fatalf("load work: %v", err)
	}
	if stored.Title != "修订后的标题" {
		t.Fatalf("title not updated: %q", stored.Title)
	}
	if stored.Type != work.Type || stored.Summary != work.Summary || stored.Content != work.Content || stored.EffectDescription != work.EffectDescription {
		t.Fatalf("omitted fields changed: %#v", stored)
	}
	if stored.Version != 2 {
		t.Fatalf("expected version 2, got %d", stored.Version)
	}

	var referenceCount int64
	if err := database.DB.Model(&model.RPDBReference{}).Where("work_id = ?", work.ID).Count(&referenceCount).Error; err != nil {
		t.Fatalf("count references: %v", err)
	}
	if referenceCount != 1 {
		t.Fatalf("expected reference to be preserved, got %d", referenceCount)
	}
}

func TestRPDBModerationApprovesLegacyFullShapeTitleRevision(t *testing.T) {
	server, author, _, moderatorToken := newRPDBModerationTestServer(t)
	work := model.RPDBWork{
		AuthorID:          author.ID,
		Type:              model.RPDBWorkTypeItemShowcase,
		Title:             "旧标题",
		Summary:           "必须保留的摘要",
		Content:           "<p>必须保留的正文</p>",
		EffectDescription: "必须保留的效果说明",
		Status:            model.RPDBStatusPublished,
		ReviewStatus:      model.RPDBReviewApproved,
		IsPublic:          true,
		Version:           2,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	if err := database.DB.Create(&model.RPDBReference{
		WorkID:       work.ID,
		ExternalType: "item",
		ExternalID:   "190002",
		Name:         "旧版修订关联物品",
	}).Error; err != nil {
		t.Fatalf("create reference: %v", err)
	}
	revision := model.RPDBRevision{
		WorkID:        work.ID,
		ProposerID:    author.ID,
		BaseVersion:   2,
		ChangeSummary: "旧版标题修订",
		Status:        model.RPDBReviewPending,
		Payload:       `{"type":"","title":"兼容后的新标题","summary":"","content":"","content_type":"","cover_image":"","rp_use_cases":"","effect_description":"","restrictions":null,"extra":null,"game_version":"","expansion":"","availability_status":"","bind_type":"","faction":"","armor_type":"","status":"","is_public":false,"references":null,"media":null,"transmog_slots":null,"guide_steps":null,"tag_ids":null,"change_summary":"旧版标题修订"}`,
	}
	if err := database.DB.Create(&revision).Error; err != nil {
		t.Fatalf("create revision: %v", err)
	}

	resp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/rpdb/revisions/"+strconv.FormatUint(uint64(revision.ID), 10),
		map[string]interface{}{"action": "approve"},
		moderatorToken,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected legacy revision approval 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var stored model.RPDBWork
	if err := database.DB.First(&stored, work.ID).Error; err != nil {
		t.Fatalf("load work: %v", err)
	}
	if stored.Title != "兼容后的新标题" || stored.Version != 3 {
		t.Fatalf("legacy revision not applied: %#v", stored)
	}
	if stored.Type != work.Type || stored.Summary != work.Summary || stored.Content != work.Content || stored.EffectDescription != work.EffectDescription {
		t.Fatalf("legacy revision cleared omitted fields: %#v", stored)
	}
	var referenceCount int64
	if err := database.DB.Model(&model.RPDBReference{}).Where("work_id = ?", work.ID).Count(&referenceCount).Error; err != nil {
		t.Fatalf("count references: %v", err)
	}
	if referenceCount != 1 {
		t.Fatalf("legacy revision removed references: %d", referenceCount)
	}
}
