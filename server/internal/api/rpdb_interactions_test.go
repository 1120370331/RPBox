package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func newRPDBInteractionTestServer(t *testing.T) (*Server, model.User, model.RPDBWork, string) {
	t.Helper()

	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Notification{},
		&model.RPDBWork{},
		&model.RPDBLike{},
		&model.RPDBFavorite{},
		&model.RPDBComment{},
		&model.RPDBCommentLike{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
		&model.UserDailyActivity{},
		&model.UserActivityLog{},
		&model.RPDBVerification{},
		&model.RPDBList{},
		&model.RPDBListEntry{},
		&model.RPDBReference{},
		&model.RPDBGuideStep{},
		&model.RPDBViewEvent{},
	)
	user := model.User{Username: "collector", Email: "collector@example.com", PassHash: "hash"}
	author := model.User{Username: "author", Email: "author@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&user, &author}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	work := model.RPDBWork{
		AuthorID:     author.ID,
		Type:         model.RPDBWorkTypeItemShowcase,
		Title:        "路线作品",
		Status:       model.RPDBStatusPublished,
		ReviewStatus: model.RPDBReviewApproved,
		IsPublic:     true,
		Version:      1,
	}
	if err := db.Create(&work).Error; err != nil {
		t.Fatalf("create work: %v", err)
	}
	if err := db.Create(&model.RPDBGuideStep{
		WorkID:    work.ID,
		SortOrder: 1,
		Title:     "第一站",
		MapID:     "84",
		X:         42.1,
		Y:         65.3,
		Label:     "旧城区",
	}).Error; err != nil {
		t.Fatalf("create guide step: %v", err)
	}
	return newTestServer(t, db), user, work, newTestToken(t, user)
}

func TestRPDBWorkAuthorCanDeleteAnotherUsersComment(t *testing.T) {
	server, commenter, work, _ := newRPDBInteractionTestServer(t)
	var author model.User
	if err := database.DB.First(&author, work.AuthorID).Error; err != nil {
		t.Fatalf("load work author: %v", err)
	}
	root := model.RPDBComment{
		WorkID: work.ID, AuthorID: commenter.ID, Content: "需要作者清理的评论",
		Status: model.RPDBStatusPublished,
	}
	if err := database.DB.Create(&root).Error; err != nil {
		t.Fatalf("create root comment: %v", err)
	}
	reply := model.RPDBComment{
		WorkID: work.ID, AuthorID: commenter.ID, ParentID: &root.ID, Content: "保留的回复",
		Status: model.RPDBStatusPublished,
	}
	if err := database.DB.Create(&reply).Error; err != nil {
		t.Fatalf("create reply: %v", err)
	}
	if err := database.DB.Create(&model.RPDBCommentLike{CommentID: root.ID, UserID: author.ID}).Error; err != nil {
		t.Fatalf("create comment like: %v", err)
	}
	if err := database.DB.Model(&work).Update("comment_count", 2).Error; err != nil {
		t.Fatalf("update comment count: %v", err)
	}

	resp := performRequest(
		server.router,
		http.MethodDelete,
		"/api/v1/rpdb/comments/"+strconv.FormatUint(uint64(root.ID), 10),
		nil,
		newTestToken(t, author),
	)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected work author delete 204, got %d body=%s", resp.Code, resp.Body.String())
	}
	if err := database.DB.First(&model.RPDBComment{}, root.ID).Error; err == nil {
		t.Fatalf("expected root comment deleted")
	}
	var storedReply model.RPDBComment
	if err := database.DB.First(&storedReply, reply.ID).Error; err != nil {
		t.Fatalf("load preserved reply: %v", err)
	}
	if storedReply.ParentID != nil {
		t.Fatalf("expected preserved reply reparented to root, got parent %v", *storedReply.ParentID)
	}
	var likeCount int64
	if err := database.DB.Model(&model.RPDBCommentLike{}).Where("comment_id = ?", root.ID).Count(&likeCount).Error; err != nil {
		t.Fatalf("count deleted comment likes: %v", err)
	}
	if likeCount != 0 {
		t.Fatalf("expected deleted comment likes removed, got %d", likeCount)
	}
	var storedWork model.RPDBWork
	if err := database.DB.First(&storedWork, work.ID).Error; err != nil {
		t.Fatalf("load work: %v", err)
	}
	if storedWork.CommentCount != 1 {
		t.Fatalf("expected comment count 1, got %d", storedWork.CommentCount)
	}
}

func TestRPDBCommentsExcludeHiddenCommentsAndBlockedAuthors(t *testing.T) {
	server, viewer, work, token := newRPDBInteractionTestServer(t)
	visibleAuthor := model.User{Username: "visible-commenter", Email: "visible@example.com", PassHash: "hash"}
	blockedAuthor := model.User{Username: "blocked-commenter", Email: "blocked@example.com", PassHash: "hash"}
	if err := database.DB.Create(&[]*model.User{&visibleAuthor, &blockedAuthor}).Error; err != nil {
		t.Fatalf("create comment authors: %v", err)
	}
	comments := []model.RPDBComment{
		{WorkID: work.ID, AuthorID: visibleAuthor.ID, Content: "可见评论", Status: model.RPDBStatusPublished},
		{WorkID: work.ID, AuthorID: visibleAuthor.ID, Content: "已隐藏评论", Status: model.RPDBStatusPublished},
		{WorkID: work.ID, AuthorID: blockedAuthor.ID, Content: "已屏蔽作者评论", Status: model.RPDBStatusPublished},
	}
	if err := database.DB.Create(&comments).Error; err != nil {
		t.Fatalf("create comments: %v", err)
	}
	if err := database.DB.Create(&model.UserHiddenContent{
		UserID: viewer.ID, TargetType: reportTargetRPDBComment, TargetID: comments[1].ID,
	}).Error; err != nil {
		t.Fatalf("hide comment: %v", err)
	}
	if err := database.DB.Create(&model.UserBlock{
		BlockerID: viewer.ID, BlockedUserID: blockedAuthor.ID,
	}).Error; err != nil {
		t.Fatalf("block comment author: %v", err)
	}

	resp := performRequest(
		server.router,
		http.MethodGet,
		"/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10)+"/comments",
		nil,
		token,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected comments 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	var payload struct {
		Comments []rpdbCommentResponse `json:"comments"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode comments: %v", err)
	}
	if len(payload.Comments) != 1 || payload.Comments[0].ID != comments[0].ID {
		t.Fatalf("unexpected visible comments: %#v", payload.Comments)
	}
}

func TestRPDBInteractionLikeAndFavoriteAreIdempotent(t *testing.T) {
	server, user, work, token := newRPDBInteractionTestServer(t)
	base := "/api/v1/rpdb/works/" + strconv.FormatUint(uint64(work.ID), 10)

	for i := 0; i < 2; i++ {
		if resp := performRequest(server.router, http.MethodPost, base+"/like", nil, token); resp.Code != http.StatusOK {
			t.Fatalf("like attempt %d returned %d body=%s", i+1, resp.Code, resp.Body.String())
		}
		if resp := performRequest(server.router, http.MethodPost, base+"/favorite", nil, token); resp.Code != http.StatusOK {
			t.Fatalf("favorite attempt %d returned %d body=%s", i+1, resp.Code, resp.Body.String())
		}
	}

	var stored model.RPDBWork
	if err := database.DB.First(&stored, work.ID).Error; err != nil {
		t.Fatalf("load work: %v", err)
	}
	if stored.LikeCount != 1 || stored.FavoriteCount != 1 {
		t.Fatalf("expected counters 1/1, got likes=%d favorites=%d", stored.LikeCount, stored.FavoriteCount)
	}

	var rewardedUser model.User
	if err := database.DB.First(&rewardedUser, user.ID).Error; err != nil {
		t.Fatalf("load liker rewards: %v", err)
	}
	if rewardedUser.ActivityExperience != 5 {
		t.Fatalf("expected liker daily first-like experience 5, got %d", rewardedUser.ActivityExperience)
	}
	var rewardedAuthor model.User
	if err := database.DB.First(&rewardedAuthor, work.AuthorID).Error; err != nil {
		t.Fatalf("load author rewards: %v", err)
	}
	if rewardedAuthor.ActivityPoints != 3 || rewardedAuthor.ActivityExperience != 5 {
		t.Fatalf("expected author rewards 3 points/5 experience, got %d/%d", rewardedAuthor.ActivityPoints, rewardedAuthor.ActivityExperience)
	}

	var notifications []model.Notification
	if err := database.DB.Where("user_id = ?", work.AuthorID).Find(&notifications).Error; err != nil {
		t.Fatalf("load work like notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected one work like notification after duplicate likes, got %d", len(notifications))
	}
	notification := notifications[0]
	if notification.Type != "rpdb_like" || notification.TargetType != "rpdb_work" || notification.TargetID != work.ID {
		t.Fatalf("unexpected work like notification: %#v", notification)
	}
	if notification.ActorID == nil || *notification.ActorID != user.ID {
		t.Fatalf("expected liker %d as notification actor, got %#v", user.ID, notification.ActorID)
	}

	if resp := performRequest(server.router, http.MethodPost, base+"/like", nil, newTestToken(t, rewardedAuthor)); resp.Code != http.StatusOK {
		t.Fatalf("self-like returned %d body=%s", resp.Code, resp.Body.String())
	}
	var selfLikeNotificationCount int64
	if err := database.DB.Model(&model.Notification{}).Where("user_id = ?", work.AuthorID).Count(&selfLikeNotificationCount).Error; err != nil {
		t.Fatalf("count notifications after self-like: %v", err)
	}
	if selfLikeNotificationCount != 1 {
		t.Fatalf("expected self-like to create no notification, got %d total", selfLikeNotificationCount)
	}
}

func TestRPDBCommentAndReplyLikesAreIdempotent(t *testing.T) {
	server, user, work, token := newRPDBInteractionTestServer(t)
	var author model.User
	if err := database.DB.First(&author, work.AuthorID).Error; err != nil {
		t.Fatalf("load work author: %v", err)
	}
	root := model.RPDBComment{
		WorkID: work.ID, AuthorID: author.ID, Content: "根评论",
		Status: model.RPDBStatusPublished,
	}
	if err := database.DB.Create(&root).Error; err != nil {
		t.Fatalf("create root comment: %v", err)
	}
	reply := model.RPDBComment{
		WorkID: work.ID, AuthorID: author.ID, ParentID: &root.ID, Content: "评论回复",
		Status: model.RPDBStatusPublished,
	}
	if err := database.DB.Create(&reply).Error; err != nil {
		t.Fatalf("create reply: %v", err)
	}

	for _, commentID := range []uint{root.ID, reply.ID} {
		endpoint := "/api/v1/rpdb/comments/" + strconv.FormatUint(uint64(commentID), 10) + "/like"
		for attempt := 0; attempt < 2; attempt++ {
			resp := performRequest(server.router, http.MethodPost, endpoint, nil, token)
			if resp.Code != http.StatusOK {
				t.Fatalf("like comment %d attempt %d returned %d body=%s", commentID, attempt+1, resp.Code, resp.Body.String())
			}
		}
	}

	var storedRoot model.RPDBComment
	var storedReply model.RPDBComment
	if err := database.DB.First(&storedRoot, root.ID).Error; err != nil {
		t.Fatalf("load root comment: %v", err)
	}
	if err := database.DB.First(&storedReply, reply.ID).Error; err != nil {
		t.Fatalf("load reply: %v", err)
	}
	if storedRoot.LikeCount != 1 || storedReply.LikeCount != 1 {
		t.Fatalf("expected comment like counts 1/1, got %d/%d", storedRoot.LikeCount, storedReply.LikeCount)
	}
	var rewardedUser model.User
	if err := database.DB.First(&rewardedUser, user.ID).Error; err != nil {
		t.Fatalf("load comment liker rewards: %v", err)
	}
	if rewardedUser.ActivityExperience != 5 {
		t.Fatalf("expected one daily first-like reward across comment likes, got %d", rewardedUser.ActivityExperience)
	}

	notificationResp := performRequest(server.router, http.MethodGet, "/api/v1/notifications?type=all", nil, newTestToken(t, author))
	if notificationResp.Code != http.StatusOK {
		t.Fatalf("list comment like notifications returned %d body=%s", notificationResp.Code, notificationResp.Body.String())
	}
	var notificationPayload struct {
		Notifications []struct {
			Type             string `json:"type"`
			TargetType       string `json:"target_type"`
			TargetID         uint   `json:"target_id"`
			TargetRPDBWorkID uint   `json:"target_rpdb_work_id"`
		} `json:"notifications"`
	}
	if err := json.Unmarshal(notificationResp.Body.Bytes(), &notificationPayload); err != nil {
		t.Fatalf("decode comment like notifications: %v", err)
	}
	if len(notificationPayload.Notifications) != 2 {
		t.Fatalf("expected two comment like notifications, got %#v", notificationPayload.Notifications)
	}
	seenCommentNotifications := make(map[uint]bool, len(notificationPayload.Notifications))
	for _, notification := range notificationPayload.Notifications {
		if notification.Type != "rpdb_comment_like" || notification.TargetType != "rpdb_comment" ||
			notification.TargetRPDBWorkID != work.ID {
			t.Fatalf("unexpected RPDB comment like notification: %#v", notification)
		}
		seenCommentNotifications[notification.TargetID] = true
	}
	if !seenCommentNotifications[root.ID] || !seenCommentNotifications[reply.ID] {
		t.Fatalf("expected notifications for root/reply comments, got %#v", notificationPayload.Notifications)
	}

	listResp := performRequest(
		server.router,
		http.MethodGet,
		"/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10)+"/comments",
		nil,
		token,
	)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list comments returned %d body=%s", listResp.Code, listResp.Body.String())
	}
	var listed struct {
		Comments []rpdbCommentResponse `json:"comments"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode comments: %v", err)
	}
	if len(listed.Comments) != 2 || !listed.Comments[0].Liked || !listed.Comments[1].Liked {
		t.Fatalf("expected root and reply liked for viewer, got %#v", listed.Comments)
	}

	for _, commentID := range []uint{root.ID, reply.ID} {
		endpoint := "/api/v1/rpdb/comments/" + strconv.FormatUint(uint64(commentID), 10) + "/like"
		for attempt := 0; attempt < 2; attempt++ {
			resp := performRequest(server.router, http.MethodDelete, endpoint, nil, token)
			if resp.Code != http.StatusOK {
				t.Fatalf("unlike comment %d attempt %d returned %d body=%s", commentID, attempt+1, resp.Code, resp.Body.String())
			}
		}
	}

	if err := database.DB.First(&storedRoot, root.ID).Error; err != nil {
		t.Fatalf("reload root comment: %v", err)
	}
	if err := database.DB.First(&storedReply, reply.ID).Error; err != nil {
		t.Fatalf("reload reply: %v", err)
	}
	if storedRoot.LikeCount != 0 || storedReply.LikeCount != 0 {
		t.Fatalf("expected comment like counts 0/0, got %d/%d", storedRoot.LikeCount, storedReply.LikeCount)
	}
	var likes int64
	if err := database.DB.Model(&model.RPDBCommentLike{}).Where("user_id = ?", user.ID).Count(&likes).Error; err != nil {
		t.Fatalf("count comment likes: %v", err)
	}
	if likes != 0 {
		t.Fatalf("expected comment likes removed, got %d", likes)
	}

	listResp = performRequest(
		server.router,
		http.MethodGet,
		"/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10)+"/comments",
		nil,
		token,
	)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list comments after unlike returned %d body=%s", listResp.Code, listResp.Body.String())
	}
	listed.Comments = nil
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode comments after unlike: %v", err)
	}
	if len(listed.Comments) != 2 || listed.Comments[0].Liked || listed.Comments[1].Liked {
		t.Fatalf("expected root and reply unliked for viewer, got %#v", listed.Comments)
	}
}

func TestRPDBInteractionCreatesCommentAndDefaultList(t *testing.T) {
	server, user, work, token := newRPDBInteractionTestServer(t)
	base := "/api/v1/rpdb/works/" + strconv.FormatUint(uint64(work.ID), 10)

	commentResp := performRequest(server.router, http.MethodPost, base+"/comments", map[string]interface{}{
		"content": "这条路线很适合调查员角色。",
	}, token)
	if commentResp.Code != http.StatusCreated {
		t.Fatalf("expected comment 201, got %d body=%s", commentResp.Code, commentResp.Body.String())
	}
	var rewardedCommenter model.User
	if err := database.DB.First(&rewardedCommenter, user.ID).Error; err != nil {
		t.Fatalf("load commenter rewards: %v", err)
	}
	if rewardedCommenter.ActivityExperience != 3 {
		t.Fatalf("expected commenter experience 3, got %d", rewardedCommenter.ActivityExperience)
	}
	var rewardedAuthor model.User
	if err := database.DB.First(&rewardedAuthor, work.AuthorID).Error; err != nil {
		t.Fatalf("load work author rewards: %v", err)
	}
	if rewardedAuthor.ActivityExperience != 3 {
		t.Fatalf("expected work author received-comment experience 3, got %d", rewardedAuthor.ActivityExperience)
	}

	listResp := performRequest(server.router, http.MethodPost, base+"/list", map[string]interface{}{
		"status": model.RPDBListStatusFarming,
	}, token)
	if listResp.Code != http.StatusCreated {
		t.Fatalf("expected list 201, got %d body=%s", listResp.Code, listResp.Body.String())
	}

	var list model.RPDBList
	if err := database.DB.Where("user_id = ? AND is_default = ?", user.ID, true).First(&list).Error; err != nil {
		t.Fatalf("load default list: %v", err)
	}
	if list.Name != "默认收集清单" {
		t.Fatalf("expected default collection checklist name, got %q", list.Name)
	}
	var entry model.RPDBListEntry
	if err := database.DB.Where("list_id = ? AND work_id = ?", list.ID, work.ID).First(&entry).Error; err != nil {
		t.Fatalf("load list entry: %v", err)
	}
	if entry.Status != model.RPDBListStatusFarming {
		t.Fatalf("expected farming status, got %q", entry.Status)
	}
}

func TestRPDBListEndpointCreatesDefaultCollectionChecklist(t *testing.T) {
	server, user, _, token := newRPDBInteractionTestServer(t)

	resp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/lists", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected list endpoint 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Lists []rpdbListResponse `json:"lists"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode lists: %v", err)
	}
	if len(payload.Lists) != 1 {
		t.Fatalf("expected one default list, got %d", len(payload.Lists))
	}
	if payload.Lists[0].Name != "默认收集清单" || !payload.Lists[0].IsDefault {
		t.Fatalf("unexpected default list: %#v", payload.Lists[0].RPDBList)
	}

	var favoriteCount int64
	if err := database.DB.Model(&model.RPDBFavorite{}).Where("user_id = ?", user.ID).Count(&favoriteCount).Error; err != nil {
		t.Fatalf("count favorites: %v", err)
	}
	if favoriteCount != 0 {
		t.Fatalf("default collection checklist must not create favorites, got %d", favoriteCount)
	}
}

func TestRPDBInteractionAddsWorkToSelectedList(t *testing.T) {
	server, user, work, token := newRPDBInteractionTestServer(t)
	customList := model.RPDBList{
		UserID:      user.ID,
		Name:        "剧情道具清单",
		Description: "手动选择的清单",
	}
	if err := database.DB.Create(&customList).Error; err != nil {
		t.Fatalf("create custom list: %v", err)
	}

	resp := performRequest(server.router, http.MethodPost, "/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10)+"/list", map[string]interface{}{
		"status":  model.RPDBListStatusWanted,
		"list_id": customList.ID,
	}, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected selected list add 201, got %d body=%s", resp.Code, resp.Body.String())
	}

	var entry model.RPDBListEntry
	if err := database.DB.Where("list_id = ? AND work_id = ?", customList.ID, work.ID).First(&entry).Error; err != nil {
		t.Fatalf("load selected list entry: %v", err)
	}
	var defaultCount int64
	if err := database.DB.
		Table("rpdb_list_entries").
		Joins("JOIN rpdb_lists ON rpdb_lists.id = rpdb_list_entries.list_id").
		Where("rpdb_lists.user_id = ? AND rpdb_lists.is_default = ? AND rpdb_list_entries.work_id = ?", user.ID, true, work.ID).
		Count(&defaultCount).Error; err != nil {
		t.Fatalf("count default entries: %v", err)
	}
	if defaultCount != 0 {
		t.Fatalf("expected no default-list entry, got %d", defaultCount)
	}
}

func TestRPDBListTomTomExportUsesGuideCoordinates(t *testing.T) {
	server, _, work, token := newRPDBInteractionTestServer(t)
	if err := database.DB.Create(&model.RPDBGuideStep{
		WorkID:    work.ID,
		SortOrder: 2,
		Title:     "第二站",
		Zone:      "暮色森林",
		X:         48.6,
		Y:         72.4,
		Label:     "大教堂广场",
	}).Error; err != nil {
		t.Fatalf("create second guide step: %v", err)
	}
	base := "/api/v1/rpdb/works/" + strconv.FormatUint(uint64(work.ID), 10)
	if resp := performRequest(server.router, http.MethodPost, base+"/list", map[string]interface{}{}, token); resp.Code != http.StatusCreated {
		t.Fatalf("add to list: %d body=%s", resp.Code, resp.Body.String())
	}

	var list model.RPDBList
	if err := database.DB.Where("is_default = ?", true).First(&list).Error; err != nil {
		t.Fatalf("load default list: %v", err)
	}
	resp := performRequest(
		server.router,
		http.MethodGet,
		"/api/v1/rpdb/lists/"+strconv.FormatUint(uint64(list.ID), 10)+"/export?format=tomtom",
		nil,
		token,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected export 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	var payload struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	if !strings.Contains(payload.Content, "/way #84 42.10 65.30 [1/2] 路线作品 · 旧城区") ||
		!strings.Contains(payload.Content, "/way 暮色森林 48.60 72.40 [2/2] 路线作品 · 大教堂广场") {
		t.Fatalf("unexpected TomTom content %q", payload.Content)
	}
}

func TestRPDBInteractionListsFavoritesNewestFirst(t *testing.T) {
	server, user, firstWork, token := newRPDBInteractionTestServer(t)
	secondWork := model.RPDBWork{
		AuthorID:     firstWork.AuthorID,
		Type:         model.RPDBWorkTypeTransmog,
		Title:        "暮色巡林幻化",
		Status:       model.RPDBStatusPublished,
		ReviewStatus: model.RPDBReviewApproved,
		IsPublic:     true,
		Version:      1,
	}
	if err := database.DB.Create(&secondWork).Error; err != nil {
		t.Fatalf("create second work: %v", err)
	}
	if err := database.DB.Create(&[]model.RPDBFavorite{
		{WorkID: firstWork.ID, UserID: user.ID, CreatedAt: time.Now().Add(-time.Hour)},
		{WorkID: secondWork.ID, UserID: user.ID, CreatedAt: time.Now()},
	}).Error; err != nil {
		t.Fatalf("create favorites: %v", err)
	}

	resp := performRequest(server.router, http.MethodGet, "/api/v1/rpdb/my/favorites", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected favorites 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload struct {
		Works []rpdbWorkCard `json:"works"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode favorites: %v", err)
	}
	if len(payload.Works) != 2 {
		t.Fatalf("expected 2 favorites, got %d", len(payload.Works))
	}
	if payload.Works[0].ID != secondWork.ID || payload.Works[1].ID != firstWork.ID {
		t.Fatalf("unexpected favorite order: %#v", payload.Works)
	}
	if !payload.Works[0].IsFavorited || !payload.Works[1].IsFavorited {
		t.Fatalf("favorites missing viewer state: %#v", payload.Works)
	}
}
