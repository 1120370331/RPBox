package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func TestCommentImagesRequireModeratorApprovalAcrossAllCommentTypes(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Post{},
		&model.Comment{},
		&model.CommentLike{},
		&model.Item{},
		&model.ItemComment{},
		&model.RPDBWork{},
		&model.RPDBComment{},
		&model.RPDBCommentLike{},
		&model.Notification{},
		&model.UserActivityLog{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
		&model.AdminActionLog{},
	)
	database.DB = db

	author := model.User{Username: "image-target-author", Email: "image-target@example.com", PassHash: "hash", Role: "user"}
	commenter := model.User{Username: "image-commenter", Email: "image-commenter@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "image-moderator", Email: "image-moderator@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &commenter, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	post := model.Post{
		AuthorID: author.ID, Title: "帖子评论配图审核", Content: "正文", Category: "other",
		Status: "published", ReviewStatus: "approved", IsPublic: true,
	}
	item := model.Item{
		AuthorID: author.ID, Name: "道具评论配图审核", Type: "item",
		Status: "published", ReviewStatus: "approved", IsPublic: true,
	}
	work := model.RPDBWork{
		AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "数据库评论配图审核",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true, Version: 1,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.Create(&work).Error; err != nil {
		t.Fatalf("create RPDB work: %v", err)
	}

	server := newTestServer(t, db)
	enableCommentImageTestOSS(server)
	commenterToken := newTestToken(t, commenter)
	moderatorToken := newTestToken(t, moderator)

	postCommentPath := "/api/v1/posts/" + strconv.FormatUint(uint64(post.ID), 10) + "/comments"
	postCreate := performRequest(server.router, http.MethodPost, postCommentPath, map[string]interface{}{
		"content":   "帖子评论中的动画配图",
		"image_url": commentImageTestURL(commenter.ID, 1, "gif"),
	}, commenterToken)
	requireCommentImageResponseCode(t, postCreate.Code, http.StatusCreated, postCreate.Body.String())
	var postComment model.Comment
	if err := json.Unmarshal(postCreate.Body.Bytes(), &postComment); err != nil {
		t.Fatalf("decode post comment: %v", err)
	}
	if postComment.ImageReviewStatus != commentImageReviewPending || postComment.ImageURL != commentImageTestURL(commenter.ID, 1, "gif") {
		t.Fatalf("unexpected pending post comment: %#v", postComment)
	}
	requirePostCommentCount(t, db, post.ID, 0)
	requireActivityLogCount(t, db, "post_comment_create", fmt.Sprintf("post-comment:%d", postComment.ID), 0)
	requireNotificationCount(t, db, "post_comment", 0)
	requirePostCommentList(t, server, postCommentPath, commenterToken, nil)

	postApprove := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/post-comment-images/"+strconv.FormatUint(uint64(postComment.ID), 10),
		map[string]string{"action": "approve"},
		moderatorToken,
	)
	requireCommentImageResponseCode(t, postApprove.Code, http.StatusOK, postApprove.Body.String())
	requirePostCommentCount(t, db, post.ID, 1)
	requireActivityLogCount(t, db, "post_comment_create", fmt.Sprintf("post-comment:%d", postComment.ID), 1)
	requireActivityLogCount(t, db, "post_comment_received", fmt.Sprintf("comment:%d:owner:%d", postComment.ID, author.ID), 1)
	requireNotificationCount(t, db, "post_comment", 1)
	requirePostCommentList(t, server, postCommentPath, commenterToken, []uint{postComment.ID})

	postApproveAgain := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/post-comment-images/"+strconv.FormatUint(uint64(postComment.ID), 10),
		map[string]string{"action": "approve"},
		moderatorToken,
	)
	requireCommentImageResponseCode(t, postApproveAgain.Code, http.StatusConflict, postApproveAgain.Body.String())
	requirePostCommentCount(t, db, post.ID, 1)
	requireActivityLogCount(t, db, "post_comment_create", fmt.Sprintf("post-comment:%d", postComment.ID), 1)
	requireNotificationCount(t, db, "post_comment", 1)

	itemCommentPath := "/api/v1/items/" + strconv.FormatUint(uint64(item.ID), 10) + "/comments"
	itemCreate := performRequest(server.router, http.MethodPost, itemCommentPath, map[string]interface{}{
		"content":   "这是一条至少十个字符的配图评分评价",
		"image_url": commentImageTestURL(commenter.ID, 2, "webp"),
		"rating":    5,
	}, commenterToken)
	requireCommentImageResponseCode(t, itemCreate.Code, http.StatusCreated, itemCreate.Body.String())
	var itemPayload struct {
		Data model.ItemComment `json:"data"`
	}
	if err := json.Unmarshal(itemCreate.Body.Bytes(), &itemPayload); err != nil {
		t.Fatalf("decode item comment: %v", err)
	}
	itemComment := itemPayload.Data
	if itemComment.ImageReviewStatus != commentImageReviewPending {
		t.Fatalf("expected pending item comment, got %#v", itemComment)
	}
	requireItemRating(t, db, item.ID, 0, 0)
	requireItemCommentList(t, server, itemCommentPath, commenterToken, nil)

	itemApprove := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/item-comment-images/"+strconv.FormatUint(uint64(itemComment.ID), 10),
		map[string]string{"action": "approve"},
		moderatorToken,
	)
	requireCommentImageResponseCode(t, itemApprove.Code, http.StatusOK, itemApprove.Body.String())
	requireItemRating(t, db, item.ID, 5, 1)
	requireItemCommentList(t, server, itemCommentPath, commenterToken, []uint{itemComment.ID})
	requireActivityLogCount(t, db, "item_comment_create", fmt.Sprintf("item-comment:%d", itemComment.ID), 1)
	requireActivityLogCount(t, db, "item_comment_received", fmt.Sprintf("item:%d:comment:%d", item.ID, itemComment.ID), 1)
	requireNotificationCount(t, db, "item_comment", 1)

	itemApproveAgain := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/item-comment-images/"+strconv.FormatUint(uint64(itemComment.ID), 10),
		map[string]string{"action": "approve"},
		moderatorToken,
	)
	requireCommentImageResponseCode(t, itemApproveAgain.Code, http.StatusConflict, itemApproveAgain.Body.String())
	requireItemRating(t, db, item.ID, 5, 1)
	requireActivityLogCount(t, db, "item_comment_create", fmt.Sprintf("item-comment:%d", itemComment.ID), 1)
	requireNotificationCount(t, db, "item_comment", 1)

	rpdbCommentPath := "/api/v1/rpdb/works/" + strconv.FormatUint(uint64(work.ID), 10) + "/comments"
	rpdbApprovedCreate := performRequest(server.router, http.MethodPost, rpdbCommentPath, map[string]interface{}{
		"content":   "数据库 GIF 评论",
		"image_url": commentImageTestURL(commenter.ID, 3, "gif"),
	}, commenterToken)
	requireCommentImageResponseCode(t, rpdbApprovedCreate.Code, http.StatusCreated, rpdbApprovedCreate.Body.String())
	var rpdbApprovedPayload struct {
		Comment model.RPDBComment `json:"comment"`
	}
	if err := json.Unmarshal(rpdbApprovedCreate.Body.Bytes(), &rpdbApprovedPayload); err != nil {
		t.Fatalf("decode RPDB comment: %v", err)
	}
	rpdbApproved := rpdbApprovedPayload.Comment

	rpdbRejectedCreate := performRequest(server.router, http.MethodPost, rpdbCommentPath, map[string]interface{}{
		"content":   "将被拒绝的数据库配图评论",
		"image_url": commentImageTestURL(commenter.ID, 4, "png"),
	}, commenterToken)
	requireCommentImageResponseCode(t, rpdbRejectedCreate.Code, http.StatusCreated, rpdbRejectedCreate.Body.String())
	var rpdbRejectedPayload struct {
		Comment model.RPDBComment `json:"comment"`
	}
	if err := json.Unmarshal(rpdbRejectedCreate.Body.Bytes(), &rpdbRejectedPayload); err != nil {
		t.Fatalf("decode rejected RPDB comment: %v", err)
	}
	rpdbRejected := rpdbRejectedPayload.Comment

	requireRPDBCommentCount(t, db, work.ID, 0)
	requireRPDBCommentList(t, server, rpdbCommentPath, commenterToken, nil)
	pendingQueue := performRequest(server.router, http.MethodGet, "/api/v1/moderator/review/rpdb-comment-images?status=pending", nil, moderatorToken)
	requireCommentImageResponseCode(t, pendingQueue.Code, http.StatusOK, pendingQueue.Body.String())
	var queuePayload struct {
		Comments []model.RPDBComment `json:"comments"`
		Total    int64               `json:"total"`
	}
	if err := json.Unmarshal(pendingQueue.Body.Bytes(), &queuePayload); err != nil {
		t.Fatalf("decode RPDB moderation queue: %v", err)
	}
	if queuePayload.Total != 2 || len(queuePayload.Comments) != 2 {
		t.Fatalf("expected two pending RPDB images, got total=%d comments=%d", queuePayload.Total, len(queuePayload.Comments))
	}

	rpdbReject := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/rpdb-comment-images/"+strconv.FormatUint(uint64(rpdbRejected.ID), 10),
		map[string]string{"action": "reject", "comment": "图片不符合规范"},
		moderatorToken,
	)
	requireCommentImageResponseCode(t, rpdbReject.Code, http.StatusOK, rpdbReject.Body.String())
	requireRPDBCommentCount(t, db, work.ID, 0)
	requireRPDBCommentList(t, server, rpdbCommentPath, commenterToken, nil)
	requireActivityLogCount(t, db, "rpdb_comment_create", fmt.Sprintf("rpdb-comment:%d", rpdbRejected.ID), 0)

	rpdbApprove := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/rpdb-comment-images/"+strconv.FormatUint(uint64(rpdbApproved.ID), 10),
		map[string]string{"action": "approve"},
		moderatorToken,
	)
	requireCommentImageResponseCode(t, rpdbApprove.Code, http.StatusOK, rpdbApprove.Body.String())
	requireRPDBCommentCount(t, db, work.ID, 1)
	requireRPDBCommentList(t, server, rpdbCommentPath, commenterToken, []uint{rpdbApproved.ID})
	requireActivityLogCount(t, db, "rpdb_comment_create", fmt.Sprintf("rpdb-comment:%d", rpdbApproved.ID), 1)
	requireActivityLogCount(t, db, "rpdb_comment_received", fmt.Sprintf("rpdb-comment:%d:owner:%d", rpdbApproved.ID, author.ID), 1)
	requireNotificationCount(t, db, "rpdb_comment", 1)

	rpdbApproveAgain := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/rpdb-comment-images/"+strconv.FormatUint(uint64(rpdbApproved.ID), 10),
		map[string]string{"action": "approve"},
		moderatorToken,
	)
	requireCommentImageResponseCode(t, rpdbApproveAgain.Code, http.StatusConflict, rpdbApproveAgain.Body.String())
	requireRPDBCommentCount(t, db, work.ID, 1)
	requireActivityLogCount(t, db, "rpdb_comment_create", fmt.Sprintf("rpdb-comment:%d", rpdbApproved.ID), 1)
	requireNotificationCount(t, db, "rpdb_comment", 1)
}

func TestPendingPostReplyImagePublishesOnlyAfterApproval(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Post{},
		&model.Comment{},
		&model.CommentLike{},
		&model.ItemComment{},
		&model.RPDBComment{},
		&model.Notification{},
		&model.UserActivityLog{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
		&model.AdminActionLog{},
	)
	database.DB = db

	postAuthor := model.User{Username: "reply-post-author", Email: "reply-post-author@example.com", PassHash: "hash", Role: "user"}
	parentAuthor := model.User{Username: "reply-parent-author", Email: "reply-parent-author@example.com", PassHash: "hash", Role: "user"}
	replier := model.User{Username: "reply-image-author", Email: "reply-image-author@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "reply-image-moderator", Email: "reply-image-moderator@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&postAuthor, &parentAuthor, &replier, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	post := model.Post{
		AuthorID: postAuthor.ID, Title: "配图回复审核", Content: "正文", Category: "other",
		Status: "published", ReviewStatus: "approved", IsPublic: true, CommentCount: 1,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	parent := model.Comment{
		PostID: post.ID, AuthorID: parentAuthor.ID, Content: "已公开的父评论",
		ImageReviewStatus: commentImageReviewNone,
	}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("create parent comment: %v", err)
	}

	server := newTestServer(t, db)
	enableCommentImageTestOSS(server)
	commentPath := "/api/v1/posts/" + strconv.FormatUint(uint64(post.ID), 10) + "/comments"
	createResponse := performRequest(server.router, http.MethodPost, commentPath, map[string]interface{}{
		"content":   "带动画配图的回复",
		"image_url": commentImageTestURL(replier.ID, 1, "gif"),
		"parent_id": parent.ID,
	}, newTestToken(t, replier))
	requireCommentImageResponseCode(t, createResponse.Code, http.StatusCreated, createResponse.Body.String())

	var reply model.Comment
	if err := json.Unmarshal(createResponse.Body.Bytes(), &reply); err != nil {
		t.Fatalf("decode reply: %v", err)
	}
	if reply.ParentID == nil || *reply.ParentID != parent.ID || reply.ImageReviewStatus != commentImageReviewPending {
		t.Fatalf("unexpected pending reply: %#v", reply)
	}
	requirePostCommentCount(t, db, post.ID, 1)
	requirePostCommentList(t, server, commentPath, newTestToken(t, replier), []uint{parent.ID})
	requireActivityLogCount(t, db, "post_comment_create", fmt.Sprintf("post-comment:%d", reply.ID), 0)
	requireNotificationCount(t, db, "post_comment", 0)

	approveResponse := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/post-comment-images/"+strconv.FormatUint(uint64(reply.ID), 10),
		map[string]string{"action": "approve"},
		newTestToken(t, moderator),
	)
	requireCommentImageResponseCode(t, approveResponse.Code, http.StatusOK, approveResponse.Body.String())
	requirePostCommentCount(t, db, post.ID, 2)
	requirePostCommentList(t, server, commentPath, newTestToken(t, replier), []uint{parent.ID, reply.ID})
	requireActivityLogCount(t, db, "post_comment_create", fmt.Sprintf("post-comment:%d", reply.ID), 1)
	requireActivityLogCount(t, db, "post_comment_received", fmt.Sprintf("comment:%d:owner:%d", reply.ID, parentAuthor.ID), 1)
	requireNotificationCount(t, db, "post_comment", 2)

	approveAgainResponse := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/post-comment-images/"+strconv.FormatUint(uint64(reply.ID), 10),
		map[string]string{"action": "approve"},
		newTestToken(t, moderator),
	)
	requireCommentImageResponseCode(t, approveAgainResponse.Code, http.StatusConflict, approveAgainResponse.Body.String())
	requirePostCommentCount(t, db, post.ID, 2)
	requireNotificationCount(t, db, "post_comment", 2)
}

func TestLegacyPendingCommentApprovalDoesNotDuplicatePublicationEffects(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Post{},
		&model.Comment{},
		&model.Notification{},
		&model.UserActivityLog{},
		&model.AdminActionLog{},
	)
	database.DB = db

	author := model.User{Username: "legacy-image-author", Email: "legacy-image-author@example.com", PassHash: "hash", Role: "user"}
	commenter := model.User{Username: "legacy-image-commenter", Email: "legacy-image-commenter@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "legacy-image-moderator", Email: "legacy-image-moderator@example.com", PassHash: "hash", Role: "moderator"}
	if err := db.Create(&[]*model.User{&author, &commenter, &moderator}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	post := model.Post{
		AuthorID: author.ID, Title: "历史待审配图评论", Content: "正文", Category: "other",
		Status: "published", ReviewStatus: "approved", IsPublic: true, CommentCount: 0,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	comment := model.Comment{
		PostID: post.ID, AuthorID: commenter.ID, Content: "旧版本已经发放过奖励的评论",
		ImageURL: "/uploads/images/legacy-pending.gif", ImageReviewStatus: commentImageReviewPending,
	}
	if err := db.Create(&comment).Error; err != nil {
		t.Fatalf("create legacy pending comment: %v", err)
	}
	logs := []model.UserActivityLog{
		{
			UserID: commenter.ID, Action: "post_comment_create",
			ReferenceKey: fmt.Sprintf("post-comment:%d", comment.ID), ExperienceDelta: 3,
		},
		{
			UserID: author.ID, Action: "post_comment_received",
			ReferenceKey: fmt.Sprintf("comment:%d:owner:%d", comment.ID, author.ID), ExperienceDelta: 3,
		},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("create legacy reward logs: %v", err)
	}
	actorID := commenter.ID
	if err := db.Create(&model.Notification{
		UserID: author.ID, Type: "post_comment", ActorID: &actorID,
		TargetType: "comment", TargetID: comment.ID, Content: "历史评论通知",
	}).Error; err != nil {
		t.Fatalf("create legacy notification: %v", err)
	}

	server := newTestServer(t, db)
	enableCommentImageTestOSS(server)
	resp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/review/post-comment-images/"+strconv.FormatUint(uint64(comment.ID), 10),
		map[string]string{"action": "approve"},
		newTestToken(t, moderator),
	)
	requireCommentImageResponseCode(t, resp.Code, http.StatusOK, resp.Body.String())
	requirePostCommentCount(t, db, post.ID, 1)
	requireActivityLogCount(t, db, "post_comment_create", fmt.Sprintf("post-comment:%d", comment.ID), 1)
	requireActivityLogCount(t, db, "post_comment_received", fmt.Sprintf("comment:%d:owner:%d", comment.ID, author.ID), 1)
	requireNotificationCount(t, db, "post_comment", 1)
}

func TestPendingCommentImageLimitIncludesRPDBComments(t *testing.T) {
	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Post{},
		&model.Comment{},
		&model.Item{},
		&model.ItemComment{},
		&model.RPDBWork{},
		&model.RPDBComment{},
	)
	database.DB = db

	user := model.User{Username: "pending-image-user", Email: "pending-image@example.com", PassHash: "hash", Role: "user"}
	author := model.User{Username: "pending-image-author", Email: "pending-image-author@example.com", PassHash: "hash", Role: "user"}
	if err := db.Create(&[]*model.User{&user, &author}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	post := model.Post{AuthorID: author.ID, Title: "配图上限帖子", Content: "正文", Category: "other", Status: "published", ReviewStatus: "approved", IsPublic: true}
	item := model.Item{AuthorID: author.ID, Name: "配图上限道具", Type: "item", Status: "published", ReviewStatus: "approved", IsPublic: true}
	work := model.RPDBWork{
		AuthorID: author.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "配图上限数据库作品",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true, Version: 1,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.Create(&work).Error; err != nil {
		t.Fatalf("create RPDB work: %v", err)
	}

	postComments := []model.Comment{
		{PostID: post.ID, AuthorID: user.ID, Content: "待审核一", ImageURL: "/uploads/images/pending-1.png", ImageReviewStatus: commentImageReviewPending},
		{PostID: post.ID, AuthorID: user.ID, Content: "待审核二", ImageURL: "/uploads/images/pending-2.png", ImageReviewStatus: commentImageReviewPending},
	}
	itemComments := []model.ItemComment{
		{ItemID: item.ID, UserID: user.ID, Content: "待审核三", ImageURL: "/uploads/images/pending-3.png", ImageReviewStatus: commentImageReviewPending},
		{ItemID: item.ID, UserID: user.ID, Content: "待审核四", ImageURL: "/uploads/images/pending-4.png", ImageReviewStatus: commentImageReviewPending},
	}
	rpdbComment := model.RPDBComment{
		WorkID: work.ID, AuthorID: user.ID, Content: "待审核五", ImageURL: "/uploads/images/pending-5.gif",
		ImageReviewStatus: commentImageReviewPending, Status: model.RPDBStatusPublished,
	}
	if err := db.Create(&postComments).Error; err != nil {
		t.Fatalf("create pending post comments: %v", err)
	}
	if err := db.Create(&itemComments).Error; err != nil {
		t.Fatalf("create pending item comments: %v", err)
	}
	if err := db.Create(&rpdbComment).Error; err != nil {
		t.Fatalf("create pending RPDB comment: %v", err)
	}

	pending, err := pendingCommentReviewRequestCount(user.ID)
	if err != nil {
		t.Fatalf("count pending image comments: %v", err)
	}
	if pending != maxPendingCommentReviewRequests {
		t.Fatalf("expected %d pending image comments, got %d", maxPendingCommentReviewRequests, pending)
	}

	server := newTestServer(t, db)
	enableCommentImageTestOSS(server)
	resp := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/rpdb/works/"+strconv.FormatUint(uint64(work.ID), 10)+"/comments",
		map[string]interface{}{"content": "第六条", "image_url": commentImageTestURL(user.ID, 6, "gif")},
		newTestToken(t, user),
	)
	requireCommentImageResponseCode(t, resp.Code, http.StatusBadRequest, resp.Body.String())
}

func enableCommentImageTestOSS(server *Server) {
	server.cfg.OSS = config.OSSConfig{
		Enabled:         true,
		Endpoint:        "https://oss.example.invalid",
		Bucket:          "rpbox-test",
		AccessKeyID:     "test-access-key",
		AccessKeySecret: "test-access-secret",
		Prefix:          "images",
	}
}

func commentImageTestURL(userID uint, sequence int, extension string) string {
	return fmt.Sprintf("/uploads/%s/%d/%032x.%s", commentImageUploadSubdir, userID, sequence, extension)
}

func requireCommentImageResponseCode(t *testing.T, actual, expected int, body string) {
	t.Helper()
	if actual != expected {
		t.Fatalf("expected status %d, got %d body=%s", expected, actual, body)
	}
}

func requirePostCommentCount(t *testing.T, db *gorm.DB, postID uint, expected int) {
	t.Helper()
	var post model.Post
	if err := db.First(&post, postID).Error; err != nil {
		t.Fatalf("load post: %v", err)
	}
	if post.CommentCount != expected {
		t.Fatalf("expected post comment count %d, got %d", expected, post.CommentCount)
	}
}

func requireItemRating(t *testing.T, db *gorm.DB, itemID uint, expectedRating float64, expectedCount int) {
	t.Helper()
	var item model.Item
	if err := db.First(&item, itemID).Error; err != nil {
		t.Fatalf("load item: %v", err)
	}
	if item.Rating != expectedRating || item.RatingCount != expectedCount {
		t.Fatalf("expected item rating %.1f/%d, got %.1f/%d", expectedRating, expectedCount, item.Rating, item.RatingCount)
	}
}

func requireRPDBCommentCount(t *testing.T, db *gorm.DB, workID uint, expected int) {
	t.Helper()
	var work model.RPDBWork
	if err := db.First(&work, workID).Error; err != nil {
		t.Fatalf("load RPDB work: %v", err)
	}
	if work.CommentCount != expected {
		t.Fatalf("expected RPDB comment count %d, got %d", expected, work.CommentCount)
	}
}

func requireActivityLogCount(t *testing.T, db *gorm.DB, action, reference string, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.UserActivityLog{}).
		Where("action = ? AND reference_key = ?", action, reference).
		Count(&count).Error; err != nil {
		t.Fatalf("count activity logs: %v", err)
	}
	if count != expected {
		t.Fatalf("expected %d activity logs for %s/%s, got %d", expected, action, reference, count)
	}
}

func requireNotificationCount(t *testing.T, db *gorm.DB, notificationType string, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.Notification{}).Where("type = ?", notificationType).Count(&count).Error; err != nil {
		t.Fatalf("count %s notifications: %v", notificationType, err)
	}
	if count != expected {
		t.Fatalf("expected %d %s notifications, got %d", expected, notificationType, count)
	}
}

func requirePostCommentList(t *testing.T, server *Server, path, token string, expectedIDs []uint) {
	t.Helper()
	resp := performRequest(server.router, http.MethodGet, path, nil, token)
	requireCommentImageResponseCode(t, resp.Code, http.StatusOK, resp.Body.String())
	var payload struct {
		Comments []model.Comment `json:"comments"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode post comment list: %v", err)
	}
	requireCommentIDs(t, "post", commentIDs(payload.Comments), expectedIDs)
}

func requireItemCommentList(t *testing.T, server *Server, path, token string, expectedIDs []uint) {
	t.Helper()
	resp := performRequest(server.router, http.MethodGet, path, nil, token)
	requireCommentImageResponseCode(t, resp.Code, http.StatusOK, resp.Body.String())
	var payload struct {
		Data []model.ItemComment `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode item comment list: %v", err)
	}
	ids := make([]uint, len(payload.Data))
	for i, comment := range payload.Data {
		ids[i] = comment.ID
	}
	requireCommentIDs(t, "item", ids, expectedIDs)
}

func requireRPDBCommentList(t *testing.T, server *Server, path, token string, expectedIDs []uint) {
	t.Helper()
	resp := performRequest(server.router, http.MethodGet, path, nil, token)
	requireCommentImageResponseCode(t, resp.Code, http.StatusOK, resp.Body.String())
	var payload struct {
		Comments []model.RPDBComment `json:"comments"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode RPDB comment list: %v", err)
	}
	ids := make([]uint, len(payload.Comments))
	for i, comment := range payload.Comments {
		ids[i] = comment.ID
		if comment.ID != 0 && comment.ImageURL == "" {
			t.Fatalf("expected approved RPDB comment %d to retain its image URL", comment.ID)
		}
	}
	requireCommentIDs(t, "RPDB", ids, expectedIDs)
}

func commentIDs(comments []model.Comment) []uint {
	ids := make([]uint, len(comments))
	for i, comment := range comments {
		ids[i] = comment.ID
	}
	return ids
}

func requireCommentIDs(t *testing.T, commentType string, actual, expected []uint) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("expected %d visible %s comments, got %d (%v)", len(expected), commentType, len(actual), actual)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("expected visible %s comment IDs %v, got %v", commentType, expected, actual)
		}
	}
}
