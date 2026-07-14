package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func TestRPDBWorkReportCanBeReviewedAndDeleted(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.RPDBWork{}, &model.RPDBReference{}, &model.RPDBMedia{},
		&model.RPDBTransmogSlot{}, &model.RPDBGuideStep{}, &model.RPDBTag{},
		&model.RPDBLike{}, &model.RPDBFavorite{}, &model.RPDBView{},
		&model.RPDBComment{}, &model.RPDBCommentLike{}, &model.RPDBListEntry{},
		&model.RPDBRevision{}, &model.RPDBVerification{}, &model.RPDBSetWork{},
		&model.ContentReport{}, &model.AdminActionLog{},
	)

	author := model.User{Username: "rpdb-author", Email: "rpdb-author@example.com", PassHash: "hash", Role: "user"}
	reporter := model.User{Username: "rpdb-reporter", Email: "rpdb-reporter@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "rpdb-moderator", Email: "rpdb-moderator@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &reporter, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	work := model.RPDBWork{
		AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "来源可疑的道具档案",
		Summary: "摘要说明", Content: "<p>被举报的正文内容</p>", CoverImage: "/uploads/rpdb/report-cover.jpg",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true,
	}
	if err := db.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}

	server := newTestServer(t, db)
	reportResp := performRequest(server.router, http.MethodPost, "/api/v1/reports", map[string]interface{}{
		"target_type": "rpdb_work", "target_id": work.ID, "reason": "fraud",
		"detail": "来源链接与展示内容不符", "submit_report": true,
	}, newTestToken(t, reporter))
	if reportResp.Code != http.StatusOK {
		t.Fatalf("expected create report 200, got %d body=%s", reportResp.Code, reportResp.Body.String())
	}

	listResp := performRequest(server.router, http.MethodGet, "/api/v1/moderator/reports?target_scope=content", nil, newTestToken(t, moderator))
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected report list 200, got %d body=%s", listResp.Code, listResp.Body.String())
	}
	var payload struct {
		Reports []struct {
			ID                 uint   `json:"id"`
			TargetType         string `json:"target_type"`
			TargetTitle        string `json:"target_title"`
			TargetPreviewText  string `json:"target_preview_text"`
			TargetPreviewImage string `json:"target_preview_image"`
			TargetURL          string `json:"target_url"`
		} `json:"reports"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(payload.Reports) != 1 {
		t.Fatalf("expected one report, got %d", len(payload.Reports))
	}
	report := payload.Reports[0]
	if report.TargetType != "rpdb_work" || report.TargetTitle != work.Title || report.TargetURL != fmt.Sprintf("/rpdb/%d", work.ID) {
		t.Fatalf("unexpected report target: %#v", report)
	}
	if !strings.Contains(report.TargetPreviewText, "摘要说明") || report.TargetPreviewImage != work.CoverImage {
		t.Fatalf("unexpected report preview: %#v", report)
	}

	reviewResp := performRequest(server.router, http.MethodPost, fmt.Sprintf("/api/v1/moderator/reports/%d/review", report.ID), map[string]interface{}{
		"action": "delete_content", "comment": "确认违规",
	}, newTestToken(t, moderator))
	if reviewResp.Code != http.StatusOK {
		t.Fatalf("expected delete review 200, got %d body=%s", reviewResp.Code, reviewResp.Body.String())
	}
	if err := db.First(&model.RPDBWork{}, work.ID).Error; err == nil {
		t.Fatalf("expected RPDB work deleted")
	} else if err != gorm.ErrRecordNotFound {
		t.Fatalf("load deleted work: %v", err)
	}
}
