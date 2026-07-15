package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func newRPDBAuthoringTestServer(t *testing.T) (*Server, model.User, string) {
	t.Helper()

	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Tag{},
		&model.RPDBWork{},
		&model.RPDBDraft{},
		&model.RPDBReference{},
		&model.RPDBMedia{},
		&model.RPDBGuideStep{},
		&model.RPDBTransmogSlot{},
		&model.RPDBTag{},
		&model.RPDBRevision{},
		&model.Guild{},
		&model.GuildMember{},
	)
	for _, statement := range []string{
		`CREATE TRIGGER validate_rpdb_media_json_before_insert
			BEFORE INSERT ON rpdb_media
			WHEN json_valid(NEW.meta) = 0
			BEGIN
				SELECT RAISE(ABORT, 'invalid rpdb_media json');
			END`,
		`CREATE TRIGGER validate_rpdb_media_json_before_update
			BEFORE UPDATE ON rpdb_media
			WHEN json_valid(NEW.meta) = 0
			BEGIN
				SELECT RAISE(ABORT, 'invalid rpdb_media json');
			END`,
		`CREATE TRIGGER validate_rpdb_guide_steps_json_before_insert
			BEFORE INSERT ON rpdb_guide_steps
			WHEN json_valid(NEW.meta) = 0
			BEGIN
				SELECT RAISE(ABORT, 'invalid rpdb_guide_steps json');
			END`,
		`CREATE TRIGGER validate_rpdb_guide_steps_json_before_update
			BEFORE UPDATE ON rpdb_guide_steps
			WHEN json_valid(NEW.meta) = 0
			BEGIN
				SELECT RAISE(ABORT, 'invalid rpdb_guide_steps json');
			END`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("install RPDB JSON validation trigger: %v", err)
		}
	}
	user := model.User{Username: "creator", Email: "creator@example.com", PassHash: "hash", Role: "user"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return newTestServer(t, db), user, newTestToken(t, user)
}

func TestRPDBAuthorCanManageVisibility(t *testing.T) {
	server, user, token := newRPDBAuthoringTestServer(t)
	work := model.RPDBWork{
		AuthorID:     user.ID,
		Type:         model.RPDBWorkTypeItemShowcase,
		Title:        "守夜人的提灯",
		Status:       model.RPDBStatusPublished,
		ReviewStatus: model.RPDBReviewApproved,
		Visibility:   model.RPDBVisibilityPublic,
		IsPublic:     true,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}

	privateResp := performRequest(
		server.router,
		http.MethodPut,
		"/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10)+"/visibility",
		map[string]interface{}{"visibility": model.RPDBVisibilityPrivate},
		token,
	)
	if privateResp.Code != http.StatusOK {
		t.Fatalf("expected private visibility 200, got %d body=%s", privateResp.Code, privateResp.Body.String())
	}

	guild := model.Guild{Name: "暮色守望", OwnerID: user.ID, Status: "approved", InviteCode: "DUSK-WATCH"}
	if err := database.DB.Create(&guild).Error; err != nil {
		t.Fatalf("create guild: %v", err)
	}
	if err := database.DB.Create(&model.GuildMember{GuildID: guild.ID, UserID: user.ID, Role: "owner"}).Error; err != nil {
		t.Fatalf("create membership: %v", err)
	}
	secondGuild := model.Guild{Name: "夜色议会", OwnerID: user.ID, Status: "approved", InviteCode: "NIGHT-COUNCIL"}
	if err := database.DB.Create(&secondGuild).Error; err != nil {
		t.Fatalf("create second guild: %v", err)
	}
	if err := database.DB.Create(&model.GuildMember{GuildID: secondGuild.ID, UserID: user.ID, Role: "owner"}).Error; err != nil {
		t.Fatalf("create second membership: %v", err)
	}
	guildResp := performRequest(
		server.router,
		http.MethodPut,
		"/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10)+"/visibility",
		map[string]interface{}{"visibility": model.RPDBVisibilityGuild, "guild_ids": []uint{guild.ID, secondGuild.ID}},
		token,
	)
	if guildResp.Code != http.StatusOK {
		t.Fatalf("expected guild visibility 200, got %d body=%s", guildResp.Code, guildResp.Body.String())
	}

	var stored model.RPDBWork
	if err := database.DB.First(&stored, work.ID).Error; err != nil {
		t.Fatalf("load work: %v", err)
	}
	if stored.Visibility != model.RPDBVisibilityGuild || stored.GuildID == nil || *stored.GuildID != guild.ID || len(stored.GuildIDs) != 2 || stored.GuildIDs[1] != secondGuild.ID || stored.IsPublic {
		t.Fatalf("unexpected stored visibility: %#v", stored)
	}
}

func TestRPDBAuthorCannotSelectUnjoinedGuild(t *testing.T) {
	server, user, token := newRPDBAuthoringTestServer(t)
	work := model.RPDBWork{AuthorID: user.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "私有作品", Status: model.RPDBStatusDraft}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	otherGuild := model.Guild{Name: "未加入的公会", OwnerID: user.ID + 100, Status: "approved"}
	if err := database.DB.Create(&otherGuild).Error; err != nil {
		t.Fatalf("create guild: %v", err)
	}

	resp := performRequest(
		server.router,
		http.MethodPut,
		"/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10)+"/visibility",
		map[string]interface{}{"visibility": model.RPDBVisibilityGuild, "guild_ids": []uint{otherGuild.ID}},
		token,
	)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unjoined guild, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRPDBAuthoringCreatesSeparateItemDraftWithAcquisitionGuide(t *testing.T) {
	server, _, token := newRPDBAuthoringTestServer(t)

	payload := map[string]interface{}{
		"type":               model.RPDBWorkTypeItemShowcase,
		"title":              "仪式蜡烛",
		"summary":            "适合神秘学 RP 的仪式道具与获取路线",
		"content":            "<p>从夜色镇出发。</p>",
		"status":             model.RPDBStatusDraft,
		"is_public":          false,
		"effect_description": "路线覆盖墓园与林地祭坛",
		"references": []map[string]interface{}{
			{"external_type": "item", "external_id": "1001", "name": "仪式蜡烛", "source": "wowhead", "url": "https://www.wowhead.com/item=1001"},
		},
		"media": []map[string]interface{}{
			{"type": "image", "url": "/uploads/rpdb/test/candle.jpg", "caption": "仪式蜡烛效果图", "sort_order": 1},
		},
		"guide_steps": []map[string]interface{}{
			{"sort_order": 1, "title": "夜色镇集合", "zone": "暮色森林", "map_id": "47", "x": 73.8, "y": 44.5},
		},
	}

	resp := performRequest(server.router, http.MethodPost, "/api/v1/rpdb/drafts", map[string]interface{}{"payload": payload}, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", resp.Code, resp.Body.String())
	}

	var response struct {
		Draft struct {
			ID      uint                   `json:"id"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"draft"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Draft.ID == 0 || response.Draft.Payload["title"] != "仪式蜡烛" {
		t.Fatalf("unexpected draft response: %#v", response.Draft)
	}

	var workCount int64
	database.DB.Model(&model.RPDBWork{}).Count(&workCount)
	if workCount != 0 {
		t.Fatalf("saving a draft must not create or update formal works, got %d works", workCount)
	}
}

func TestRPDBAuthoringStoresCustomTagsAsPrivateUntilReview(t *testing.T) {
	server, _, token := newRPDBAuthoringTestServer(t)

	resp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/rpdb/works",
		map[string]interface{}{
			"type":      model.RPDBWorkTypeItemShowcase,
			"title":     "暮色森林巡林灯",
			"status":    model.RPDBStatusPublished,
			"is_public": true,
			"tag_names": []string{"暮色森林风格", "#暮色森林风格", "  "},
		},
		token,
	)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", resp.Code, resp.Body.String())
	}

	var tag model.Tag
	if err := database.DB.Where("name = ? AND category = ?", "暮色森林风格", "rpdb").First(&tag).Error; err != nil {
		t.Fatalf("load custom tag: %v", err)
	}
	if tag.Type != "custom" || tag.IsPublic {
		t.Fatalf("custom tag should stay private before approval: %#v", tag)
	}

	var workTagCount int64
	if err := database.DB.Model(&model.RPDBTag{}).Where("tag_id = ?", tag.ID).Count(&workTagCount).Error; err != nil {
		t.Fatalf("count work tags: %v", err)
	}
	if workTagCount != 1 {
		t.Fatalf("expected one work tag link, got %d", workTagCount)
	}
}

func TestRPDBAuthoringRejectsStandaloneGuideType(t *testing.T) {
	server, _, token := newRPDBAuthoringTestServer(t)

	resp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/rpdb/works",
		map[string]interface{}{
			"type":  "guide",
			"title": "不应独立存在的攻略",
		},
		token,
	)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRPDBAuthoringCreatesHomeShowcase(t *testing.T) {
	server, _, token := newRPDBAuthoringTestServer(t)

	resp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/rpdb/works",
		map[string]interface{}{
			"type":    "home_showcase",
			"title":   "暮色森林炼金小屋",
			"summary": "分享一套适合隐居炼金术士的家宅布置",
			"extra": map[string]interface{}{
				"region":     "暮色森林",
				"home_style": "炼金工坊",
				"share_code": "RPBOX-HOME-001",
			},
		},
		token,
	)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRPDBAuthoringPublishedEditRequiresSeparateDraft(t *testing.T) {
	server, user, token := newRPDBAuthoringTestServer(t)

	work := model.RPDBWork{
		AuthorID:     user.ID,
		Type:         model.RPDBWorkTypeItemShowcase,
		Title:        "原始标题",
		Status:       model.RPDBStatusPublished,
		ReviewStatus: model.RPDBReviewApproved,
		IsPublic:     true,
		Version:      3,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}

	resp := performRequest(
		server.router,
		http.MethodPut,
		"/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10),
		map[string]interface{}{"title": "修订后的标题", "change_summary": "更新了标题"},
		token,
	)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", resp.Code, resp.Body.String())
	}

	var stored model.RPDBWork
	if err := database.DB.First(&stored, work.ID).Error; err != nil {
		t.Fatalf("load work: %v", err)
	}
	if stored.Title != "原始标题" {
		t.Fatalf("published work was overwritten: %q", stored.Title)
	}

	var revisionCount int64
	database.DB.Model(&model.RPDBRevision{}).Where("work_id = ?", work.ID).Count(&revisionCount)
	if revisionCount != 0 {
		t.Fatalf("direct edit must not create a revision, got %d", revisionCount)
	}
}

func TestRPDBPublishingRelatedDraftCreatesRevisionWithoutOverwritingWork(t *testing.T) {
	server, user, token := newRPDBAuthoringTestServer(t)
	work := model.RPDBWork{
		AuthorID:     user.ID,
		Type:         model.RPDBWorkTypeItemShowcase,
		Title:        "线上标题",
		Status:       model.RPDBStatusPublished,
		ReviewStatus: model.RPDBReviewApproved,
		IsPublic:     true,
		Visibility:   model.RPDBVisibilityPublic,
		Version:      4,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}

	createResp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/rpdb/drafts",
		map[string]interface{}{"work_id": work.ID},
		token,
	)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create draft: expected 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}
	var created struct {
		Draft model.RPDBDraft `json:"draft"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode draft: %v", err)
	}
	updateResp := performRequest(
		server.router,
		http.MethodPut,
		"/api/v1/rpdb/drafts/"+strconv.FormatUint(uint64(created.Draft.ID), 10),
		map[string]interface{}{
			"payload": map[string]interface{}{
				"type":       model.RPDBWorkTypeItemShowcase,
				"title":      "草稿中的新标题",
				"visibility": model.RPDBVisibilityPublic,
				"is_public":  true,
			},
		},
		token,
	)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("update draft: expected 200, got %d body=%s", updateResp.Code, updateResp.Body.String())
	}
	var unchanged model.RPDBWork
	if err := database.DB.First(&unchanged, work.ID).Error; err != nil {
		t.Fatalf("load unchanged work: %v", err)
	}
	if unchanged.Title != "线上标题" {
		t.Fatalf("saving the related draft changed the formal work: %q", unchanged.Title)
	}

	publishResp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/rpdb/drafts/"+strconv.FormatUint(uint64(created.Draft.ID), 10)+"/publish",
		nil,
		token,
	)
	if publishResp.Code != http.StatusAccepted {
		t.Fatalf("publish draft: expected 202, got %d body=%s", publishResp.Code, publishResp.Body.String())
	}

	var stored model.RPDBWork
	if err := database.DB.First(&stored, work.ID).Error; err != nil {
		t.Fatalf("load work: %v", err)
	}
	if stored.Title != "线上标题" {
		t.Fatalf("draft publication overwrote the formal work before review: %q", stored.Title)
	}
	var revision model.RPDBRevision
	if err := database.DB.Where("work_id = ?", work.ID).First(&revision).Error; err != nil {
		t.Fatalf("load revision: %v", err)
	}
	if revision.BaseVersion != 4 || !strings.Contains(revision.Payload, "草稿中的新标题") {
		t.Fatalf("unexpected revision: %#v", revision)
	}
	var draftCount int64
	database.DB.Model(&model.RPDBDraft{}).Where("id = ?", created.Draft.ID).Count(&draftCount)
	if draftCount != 0 {
		t.Fatalf("published draft should leave the active draft box")
	}
}

func TestRPDBRelatedDraftRejectsTypeChangesAndRepairsContaminatedPayload(t *testing.T) {
	server, user, token := newRPDBAuthoringTestServer(t)
	work := model.RPDBWork{
		AuthorID:     user.ID,
		Type:         model.RPDBWorkTypeTransmog,
		Title:        "银月巡礼幻化",
		Content:      "<p>正式幻化资料</p>",
		Status:       model.RPDBStatusPublished,
		ReviewStatus: model.RPDBReviewApproved,
		Visibility:   model.RPDBVisibilityPublic,
		Version:      5,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	draft := model.RPDBDraft{
		AuthorID:    user.ID,
		WorkID:      &work.ID,
		Type:        model.RPDBWorkTypeHomeShowcase,
		Title:       "错误的住宅草稿",
		Payload:     `{"type":"home_showcase","title":"错误的住宅草稿","content":"住宅资料"}`,
		BaseVersion: work.Version,
		Status:      model.RPDBDraftStatusActive,
	}
	if err := database.DB.Create(&draft).Error; err != nil {
		t.Fatalf("create contaminated draft: %v", err)
	}

	getResp := performRequest(
		server.router,
		http.MethodGet,
		"/api/v1/rpdb/drafts/"+strconv.FormatUint(uint64(draft.ID), 10),
		nil,
		token,
	)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get repaired draft: expected 200, got %d body=%s", getResp.Code, getResp.Body.String())
	}
	if !strings.Contains(getResp.Body.String(), `"type":"transmog"`) ||
		!strings.Contains(getResp.Body.String(), "正式幻化资料") ||
		strings.Contains(getResp.Body.String(), "住宅资料") {
		t.Fatalf("contaminated draft was not rebuilt from the formal transmog work: %s", getResp.Body.String())
	}

	updateResp := performRequest(
		server.router,
		http.MethodPut,
		"/api/v1/rpdb/drafts/"+strconv.FormatUint(uint64(draft.ID), 10),
		map[string]interface{}{
			"payload": map[string]interface{}{
				"type":  model.RPDBWorkTypeHomeShowcase,
				"title": "再次写成住宅",
			},
		},
		token,
	)
	if updateResp.Code != http.StatusConflict {
		t.Fatalf("expected 409 for associated draft type change, got %d body=%s", updateResp.Code, updateResp.Body.String())
	}
}

func TestRPDBDraftBoxMigratesLegacyDraftWorksWithoutLosingContent(t *testing.T) {
	server, user, token := newRPDBAuthoringTestServer(t)
	legacy := model.RPDBWork{
		AuthorID:     user.ID,
		Type:         model.RPDBWorkTypeHomeShowcase,
		Title:        "旧版家宅草稿",
		Content:      "<p>不能丢失的旧正文</p>",
		Extra:        `{"share_code":"HOME-LEGACY"}`,
		Status:       model.RPDBStatusDraft,
		ReviewStatus: model.RPDBReviewNone,
		Version:      1,
	}
	if err := database.DB.Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy draft work: %v", err)
	}
	if err := database.DB.Create(&model.RPDBMedia{
		WorkID: legacy.ID, Type: "image", URL: "/uploads/images/legacy.jpg", Meta: "{}",
		ReviewStatus: model.RPDBReviewPending,
	}).Error; err != nil {
		t.Fatalf("create legacy media: %v", err)
	}

	resp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/drafts", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("list drafts: expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "旧版家宅草稿") ||
		!strings.Contains(resp.Body.String(), "不能丢失的旧正文") ||
		!strings.Contains(resp.Body.String(), "legacy.jpg") {
		t.Fatalf("legacy draft content was not preserved: %s", resp.Body.String())
	}
	var workCount int64
	database.DB.Model(&model.RPDBWork{}).Where("id = ?", legacy.ID).Count(&workCount)
	if workCount != 0 {
		t.Fatalf("legacy draft work should be moved out of the formal works table")
	}
}

func TestRPDBAuthoringPartialDraftEditPreservesOmittedFieldsAndChildren(t *testing.T) {
	server, user, token := newRPDBAuthoringTestServer(t)

	work := model.RPDBWork{
		AuthorID:          user.ID,
		Type:              model.RPDBWorkTypeItemShowcase,
		Title:             "原始标题",
		Summary:           "原始摘要",
		Content:           "<p>原始正文</p>",
		EffectDescription: "原始效果",
		Status:            model.RPDBStatusDraft,
		ReviewStatus:      model.RPDBReviewNone,
		Version:           1,
	}
	if err := database.DB.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	reference := model.RPDBReference{
		WorkID:       work.ID,
		ExternalType: "item",
		ExternalID:   "1001",
		Name:         "仪式蜡烛",
	}
	if err := database.DB.Create(&reference).Error; err != nil {
		t.Fatalf("create reference: %v", err)
	}
	step := model.RPDBGuideStep{
		WorkID:    work.ID,
		SortOrder: 1,
		Title:     "第一步",
		MapID:     "47",
		X:         73.8,
		Y:         44.5,
		Meta:      "{}",
	}
	if err := database.DB.Create(&step).Error; err != nil {
		t.Fatalf("create guide step: %v", err)
	}

	resp := performRequest(
		server.router,
		http.MethodPut,
		"/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10),
		map[string]interface{}{"title": "仅更新标题"},
		token,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var stored model.RPDBWork
	if err := database.DB.First(&stored, work.ID).Error; err != nil {
		t.Fatalf("load work: %v", err)
	}
	if stored.Title != "仅更新标题" {
		t.Fatalf("expected updated title, got %q", stored.Title)
	}
	if stored.Summary != work.Summary || stored.Content != work.Content || stored.EffectDescription != work.EffectDescription {
		t.Fatalf("omitted fields were overwritten: %#v", stored)
	}

	var referenceCount int64
	var stepCount int64
	database.DB.Model(&model.RPDBReference{}).Where("work_id = ?", work.ID).Count(&referenceCount)
	database.DB.Model(&model.RPDBGuideStep{}).Where("work_id = ?", work.ID).Count(&stepCount)
	if referenceCount != 1 || stepCount != 1 {
		t.Fatalf("omitted children were replaced: references=%d steps=%d", referenceCount, stepCount)
	}
}
