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

func TestRPDBCommentCanBeReportedHiddenAndModerated(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{}, &model.RPDBWork{}, &model.RPDBComment{}, &model.RPDBCommentLike{},
		&model.UserBlock{}, &model.UserHiddenContent{}, &model.ContentReport{}, &model.AdminActionLog{},
	)
	workAuthor := model.User{Username: "work-author", Email: "work-author@example.com", PassHash: "hash"}
	commentAuthor := model.User{Username: "comment-author", Email: "comment-author@example.com", PassHash: "hash"}
	reporter := model.User{Username: "comment-reporter", Email: "comment-reporter@example.com", PassHash: "hash"}
	moderator := model.User{Username: "comment-moderator", Email: "comment-moderator@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&workAuthor, &commentAuthor, &reporter, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	work := model.RPDBWork{
		AuthorID: workAuthor.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "评论所属作品",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true, CommentCount: 1,
	}
	if err := db.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	comment := model.RPDBComment{
		WorkID: work.ID, AuthorID: commentAuthor.ID, Content: "需要处理的 RPDB 评论",
		Status: model.RPDBStatusPublished,
	}
	if err := db.Create(&comment).Error; err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if err := db.Create(&model.RPDBCommentLike{CommentID: comment.ID, UserID: reporter.ID}).Error; err != nil {
		t.Fatalf("create comment like: %v", err)
	}

	server := newTestServer(t, db)
	reporterToken := newTestToken(t, reporter)
	reportResp := performRequest(server.router, http.MethodPost, "/api/v1/reports", map[string]interface{}{
		"target_type": "rpdb_comment", "target_id": comment.ID, "reason": "abuse",
		"detail": "评论包含人身攻击", "hide_target": true, "submit_report": true,
	}, reporterToken)
	if reportResp.Code != http.StatusOK {
		t.Fatalf("expected create report 200, got %d body=%s", reportResp.Code, reportResp.Body.String())
	}

	commentsResp := performRequest(
		server.router,
		http.MethodGet,
		fmt.Sprintf("/api/v1/rpdb/works/%d/comments", work.ID),
		nil,
		reporterToken,
	)
	if commentsResp.Code != http.StatusOK {
		t.Fatalf("expected comments 200, got %d body=%s", commentsResp.Code, commentsResp.Body.String())
	}
	var commentsPayload struct {
		Comments []rpdbCommentResponse `json:"comments"`
	}
	if err := json.Unmarshal(commentsResp.Body.Bytes(), &commentsPayload); err != nil {
		t.Fatalf("decode comments: %v", err)
	}
	if len(commentsPayload.Comments) != 0 {
		t.Fatalf("expected hidden comment excluded, got %#v", commentsPayload.Comments)
	}

	listResp := performRequest(server.router, http.MethodGet, "/api/v1/moderator/reports?target_scope=comment", nil, newTestToken(t, moderator))
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected report list 200, got %d body=%s", listResp.Code, listResp.Body.String())
	}
	var reportPayload struct {
		Reports []struct {
			ID                uint   `json:"id"`
			TargetType        string `json:"target_type"`
			ParentTargetID    uint   `json:"parent_target_id"`
			ParentTargetTitle string `json:"parent_target_title"`
			TargetURL         string `json:"target_url"`
		} `json:"reports"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &reportPayload); err != nil {
		t.Fatalf("decode report list: %v", err)
	}
	if len(reportPayload.Reports) != 1 {
		t.Fatalf("expected one RPDB comment report, got %d", len(reportPayload.Reports))
	}
	report := reportPayload.Reports[0]
	if report.TargetType != reportTargetRPDBComment || report.ParentTargetID != work.ID ||
		report.ParentTargetTitle != work.Title || report.TargetURL != fmt.Sprintf("/rpdb/%d?comment=%d", work.ID, comment.ID) {
		t.Fatalf("unexpected RPDB comment report: %#v", report)
	}

	reviewResp := performRequest(server.router, http.MethodPost, fmt.Sprintf("/api/v1/moderator/reports/%d/review", report.ID), map[string]interface{}{
		"action": "delete_content", "comment": "确认违规",
	}, newTestToken(t, moderator))
	if reviewResp.Code != http.StatusOK {
		t.Fatalf("expected review 200, got %d body=%s", reviewResp.Code, reviewResp.Body.String())
	}
	if err := db.First(&model.RPDBComment{}, comment.ID).Error; err == nil {
		t.Fatalf("expected RPDB comment deleted")
	}
	var storedWork model.RPDBWork
	if err := db.First(&storedWork, work.ID).Error; err != nil {
		t.Fatalf("load work: %v", err)
	}
	if storedWork.CommentCount != 0 {
		t.Fatalf("expected comment count 0, got %d", storedWork.CommentCount)
	}
	var likes int64
	if err := db.Model(&model.RPDBCommentLike{}).Where("comment_id = ?", comment.ID).Count(&likes).Error; err != nil {
		t.Fatalf("count likes: %v", err)
	}
	if likes != 0 {
		t.Fatalf("expected comment likes deleted, got %d", likes)
	}
}
