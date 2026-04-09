package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
	"github.com/rpbox/server/pkg/validator"
	"gorm.io/gorm"
)

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	Content  string `json:"content"`
	ImageURL string `json:"image_url"`
	ParentID *uint  `json:"parent_id"`
}

const maxPendingCommentReviewRequests = 5

func pendingCommentReviewRequestCount(userID uint) (int64, error) {
	var postCommentPending int64
	if err := database.DB.Model(&model.Comment{}).
		Where("author_id = ? AND image_review_status = ?", userID, "pending").
		Count(&postCommentPending).Error; err != nil {
		return 0, err
	}

	var itemCommentPending int64
	if err := database.DB.Model(&model.ItemComment{}).
		Where("user_id = ? AND image_review_status = ?", userID, "pending").
		Count(&itemCommentPending).Error; err != nil {
		return 0, err
	}

	return postCommentPending + itemCommentPending, nil
}

// listComments 获取帖子的评论列表
func (s *Server) listComments(c *gin.Context) {
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID := c.GetUint("userID")

	// 检查帖子是否存在
	var post model.Post
	if err := database.DB.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	var comments []model.Comment
	database.DB.Where("post_id = ?", postID).Order("created_at ASC").Find(&comments)

	likedMap := make(map[uint]bool)
	if userID != 0 && len(comments) > 0 {
		commentIDs := make([]uint, 0, len(comments))
		for _, comment := range comments {
			commentIDs = append(commentIDs, comment.ID)
		}
		var likes []model.CommentLike
		database.DB.Where("user_id = ? AND comment_id IN ?", userID, commentIDs).Find(&likes)
		for _, like := range likes {
			likedMap[like.CommentID] = true
		}
	}

	// 获取评论作者信息
	authorIDs := make([]uint, len(comments))
	for i, comment := range comments {
		authorIDs[i] = comment.AuthorID
	}

	var users []model.User
	if len(authorIDs) > 0 {
		database.DB.Where("id IN ?", authorIDs).Find(&users)
	}
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 组装响应
	type CommentWithAuthor struct {
		model.Comment
		AuthorName            string `json:"author_name"`
		AuthorAvatar          string `json:"author_avatar"`
		AuthorNameColor       string `json:"author_name_color"`
		AuthorNameBold        bool   `json:"author_name_bold"`
		AuthorForumLevel      int    `json:"author_forum_level"`
		AuthorForumLevelName  string `json:"author_forum_level_name"`
		AuthorForumLevelColor string `json:"author_forum_level_color"`
		AuthorForumLevelBold  bool   `json:"author_forum_level_bold"`
		Liked                 bool   `json:"liked"`
	}
	result := make([]CommentWithAuthor, len(comments))
	for i, comment := range comments {
		if comment.ImageURL != "" && comment.ImageReviewStatus != "approved" {
			comment.ImageURL = ""
		}
		author := userMap[comment.AuthorID]
		nameColor, nameBold := userDisplayStyle(author)
		levelInfo := resolveForumLevelInfo(author.ActivityExperience)
		result[i] = CommentWithAuthor{
			Comment:               comment,
			AuthorName:            author.Username,
			AuthorAvatar:          userAvatarURL(s.cfg.Server.ApiHost, author),
			AuthorNameColor:       nameColor,
			AuthorNameBold:        nameBold,
			AuthorForumLevel:      levelInfo.Level,
			AuthorForumLevelName:  levelInfo.Name,
			AuthorForumLevelColor: levelInfo.Color,
			AuthorForumLevelBold:  levelInfo.Bold,
			Liked:                 likedMap[comment.ID],
		}
	}

	c.JSON(http.StatusOK, gin.H{"comments": result})
}

// createComment 创建评论
func (s *Server) createComment(c *gin.Context) {
	userID := c.GetUint("userID")
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 检查帖子是否存在
	var post model.Post
	if err := database.DB.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	req.ImageURL = strings.TrimSpace(req.ImageURL)
	if req.Content == "" && req.ImageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论内容和图片不能同时为空"})
		return
	}
	if s.enforcePostCommentHardRules(c, userID, "comment", nil, req.Content) {
		return
	}
	if req.ImageURL != "" {
		pendingCount, err := pendingCommentReviewRequestCount(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "检查待审核评论数量失败"})
			return
		}
		if pendingCount >= maxPendingCommentReviewRequests {
			c.JSON(http.StatusBadRequest, gin.H{"error": "你最多只能同时有5条待审核评论申请"})
			return
		}
	}

	// 如果是回复评论，检查父评论是否存在
	var parent model.Comment
	if req.ParentID != nil {
		if err := database.DB.First(&parent, *req.ParentID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "父评论不存在"})
			return
		}
		if parent.PostID != uint(postID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "父评论不属于该帖子"})
			return
		}
	}

	comment := model.Comment{
		PostID:            uint(postID),
		AuthorID:          userID,
		Content:           req.Content,
		ImageURL:          req.ImageURL,
		ImageReviewStatus: "none",
		ParentID:          req.ParentID,
	}
	if req.ImageURL != "" {
		comment.ImageReviewStatus = "pending"
	}

	commentOwnerID := post.AuthorID
	if req.ParentID != nil {
		commentOwnerID = parent.AuthorID
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&comment).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Post{}).
			Where("id = ?", post.ID).
			Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error; err != nil {
			return err
		}
		if _, err := service.AwardActivityReward(tx, userID, "post_comment_create", fmt.Sprintf("post-comment:%d", comment.ID), 0, service.CommentCreateExperience); err != nil {
			return err
		}
		if commentOwnerID != userID {
			if _, err := service.AwardActivityReward(tx, commentOwnerID, "post_comment_received", fmt.Sprintf("comment:%d:owner:%d", comment.ID, commentOwnerID), 0, service.CommentReceivedExperience); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	// 创建通知
	if req.ParentID != nil {
		// 回复评论：通知被回复的评论作者
		if parent.AuthorID != userID {
			// 构建通知内容：包含帖子标题和回复片段
			replyPreview := req.Content
			if replyPreview == "" {
				replyPreview = "[图片评论]"
			}
			if len([]rune(replyPreview)) > 50 {
				replyPreview = string([]rune(replyPreview)[:50]) + "..."
			}
			content := "在《" + post.Title + "》中回复了你的评论：" + replyPreview

			notification := model.Notification{
				UserID:     parent.AuthorID,
				Type:       "post_comment",
				ActorID:    &userID,
				TargetType: "comment",
				TargetID:   comment.ID,
				Content:    content,
			}
			service.CreateNotification(&notification)
		}
	} else {
		// 直接评论帖子：通知帖子作者
		if post.AuthorID != userID {
			// 构建通知内容：包含帖子标题和评论片段
			commentPreview := req.Content
			if commentPreview == "" {
				commentPreview = "[图片评论]"
			}
			if len([]rune(commentPreview)) > 50 {
				commentPreview = string([]rune(commentPreview)[:50]) + "..."
			}
			content := "评论了你的帖子《" + post.Title + "》：" + commentPreview

			notification := model.Notification{
				UserID:     post.AuthorID,
				Type:       "post_comment",
				ActorID:    &userID,
				TargetType: "post",
				TargetID:   uint(postID),
				Content:    content,
			}
			service.CreateNotification(&notification)
		}
	}

	// @提及通知
	if req.Content != "" {
		mentionPreview := service.NormalizeMentionPreview(req.Content)
		if len([]rune(mentionPreview)) > 50 {
			mentionPreview = string([]rune(mentionPreview)[:50]) + "..."
		}
		mentionMessage := "在《" + post.Title + "》的评论中提到了你：" + mentionPreview
		service.CreateMentionNotifications(userID, "comment", comment.ID, mentionMessage, req.Content)
	}

	c.JSON(http.StatusCreated, comment)
}

// deleteComment 删除评论
func (s *Server) deleteComment(c *gin.Context) {
	userID := c.GetUint("userID")
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	commentID, _ := strconv.ParseUint(c.Param("commentId"), 10, 32)

	var comment model.Comment
	if err := database.DB.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "评论不存在"})
		return
	}

	// 检查评论是否属于该帖子
	if comment.PostID != uint(postID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论不属于该帖子"})
		return
	}

	// 权限检查：评论作者、帖子作者、版主/管理员可以删除
	var post model.Post
	if err := database.DB.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}

	isModerator := checkModerator(userID)
	isCommentAuthor := comment.AuthorID == userID
	isPostAuthor := post.AuthorID == userID

	if !isCommentAuthor && !isPostAuthor && !isModerator {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return
	}

	// 删除评论及其点赞记录
	database.DB.Where("comment_id = ?", commentID).Delete(&model.CommentLike{})
	database.DB.Delete(&comment)

	// 更新帖子评论数
	database.DB.Model(&model.Post{}).Where("id = ?", postID).Update("comment_count", database.DB.Raw("comment_count - 1"))

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// likeComment 点赞评论
func (s *Server) likeComment(c *gin.Context) {
	userID := c.GetUint("userID")
	commentID, _ := strconv.ParseUint(c.Param("commentId"), 10, 32)

	// 检查评论是否存在
	var comment model.Comment
	if err := database.DB.First(&comment, commentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "评论不存在"})
		return
	}

	// 检查是否已点赞
	var existing model.CommentLike
	if err := database.DB.Where("comment_id = ? AND user_id = ?", commentID, userID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已点赞"})
		return
	}

	// 创建点赞记录
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		commentLike := model.CommentLike{
			CommentID: uint(commentID),
			UserID:    userID,
		}
		if err := tx.Create(&commentLike).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Comment{}).
			Where("id = ?", comment.ID).
			Update("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
			return err
		}
		if _, err := service.ApplyDailyFirstLikeBonus(tx, userID, time.Now()); err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "点赞失败"})
		return
	}

	// 创建通知（不给自己发通知）
	if comment.AuthorID != userID {
		notification := model.Notification{
			UserID:     comment.AuthorID,
			Type:       "post_comment",
			ActorID:    &userID,
			TargetType: "comment",
			TargetID:   uint(commentID),
			Content:    "点赞了你的评论",
		}
		service.CreateNotification(&notification)
	}

	c.JSON(http.StatusOK, gin.H{"message": "点赞成功"})
}

// unlikeComment 取消点赞评论
func (s *Server) unlikeComment(c *gin.Context) {
	userID := c.GetUint("userID")
	commentID, _ := strconv.ParseUint(c.Param("commentId"), 10, 32)

	result := database.DB.Where("comment_id = ? AND user_id = ?", commentID, userID).Delete(&model.CommentLike{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未点赞"})
		return
	}

	// 更新点赞数
	database.DB.Model(&model.Comment{}).Where("id = ?", commentID).Update("like_count", database.DB.Raw("like_count - 1"))

	c.JSON(http.StatusOK, gin.H{"message": "取消点赞成功"})
}
