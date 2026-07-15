package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
	"gorm.io/gorm"
)

func (s *Server) likeRPDBWork(c *gin.Context) {
	s.changeRPDBWorkInteraction(c, &model.RPDBLike{}, "like_count", true)
}

func (s *Server) unlikeRPDBWork(c *gin.Context) {
	s.changeRPDBWorkInteraction(c, &model.RPDBLike{}, "like_count", false)
}

func (s *Server) favoriteRPDBWork(c *gin.Context) {
	s.changeRPDBWorkInteraction(c, &model.RPDBFavorite{}, "favorite_count", true)
}

func (s *Server) unfavoriteRPDBWork(c *gin.Context) {
	s.changeRPDBWorkInteraction(c, &model.RPDBFavorite{}, "favorite_count", false)
}

func (s *Server) listMyRPDBFavorites(c *gin.Context) {
	userID := c.GetUint("userID")
	var works []model.RPDBWork
	if err := database.DB.
		Table("rpdb_works").
		Select("rpdb_works.*").
		Joins("JOIN rpdb_favorites ON rpdb_favorites.work_id = rpdb_works.id").
		Where("rpdb_favorites.user_id = ?", userID).
		Where(
			"rpdb_works.status = ? AND rpdb_works.review_status = ? AND rpdb_works.is_public = ?",
			model.RPDBStatusPublished,
			model.RPDBReviewApproved,
			true,
		).
		Order("rpdb_favorites.created_at DESC").
		Find(&works).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载收藏作品失败"})
		return
	}
	cards, err := buildRPDBWorkCards(works, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载收藏作品信息失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"works": cards})
}

func (s *Server) changeRPDBWorkInteraction(c *gin.Context, target interface{}, counter string, add bool) {
	userID := c.GetUint("userID")
	workID, ok := parseRPDBWorkID(c)
	if !ok {
		return
	}
	if !rpdbPublishedWorkExists(workID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var result *gorm.DB
		if add {
			switch value := target.(type) {
			case *model.RPDBLike:
				*value = model.RPDBLike{WorkID: workID, UserID: userID}
			case *model.RPDBFavorite:
				*value = model.RPDBFavorite{WorkID: workID, UserID: userID}
			}
			result = tx.Where("work_id = ? AND user_id = ?", workID, userID).FirstOrCreate(target)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 1 {
				return tx.Model(&model.RPDBWork{}).Where("id = ?", workID).
					UpdateColumn(counter, gorm.Expr(counter+" + 1")).Error
			}
			return nil
		}

		result = tx.Where("work_id = ? AND user_id = ?", workID, userID).Delete(target)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 1 {
			return tx.Model(&model.RPDBWork{}).Where("id = ?", workID).
				UpdateColumn(counter, gorm.Expr("CASE WHEN "+counter+" > 0 THEN "+counter+" - 1 ELSE 0 END")).Error
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新互动状态失败"})
		return
	}
	s.bumpRPDBListCache(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"active": add})
}

type rpdbCommentResponse struct {
	model.RPDBComment
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

func (s *Server) listRPDBComments(c *gin.Context) {
	workID, ok := parseRPDBWorkID(c)
	if !ok {
		return
	}
	var comments []model.RPDBComment
	if err := database.DB.Where("work_id = ? AND status = ?", workID, "published").
		Order("created_at ASC").Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载评论失败"})
		return
	}

	viewerID := optionalRPDBUserID(c)
	if viewerID != 0 {
		blockedIDs, err := getBlockedUserIDs(viewerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "加载评论失败"})
			return
		}
		hiddenIDs, err := hiddenContentIDs(viewerID, reportTargetRPDBComment)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "加载评论失败"})
			return
		}
		blockedMap := make(map[uint]struct{}, len(blockedIDs))
		for _, blockedID := range blockedIDs {
			blockedMap[blockedID] = struct{}{}
		}
		hiddenMap := make(map[uint]struct{}, len(hiddenIDs))
		for _, hiddenID := range hiddenIDs {
			hiddenMap[hiddenID] = struct{}{}
		}
		filtered := make([]model.RPDBComment, 0, len(comments))
		for _, comment := range comments {
			if _, blocked := blockedMap[comment.AuthorID]; blocked {
				continue
			}
			if _, hidden := hiddenMap[comment.ID]; hidden {
				continue
			}
			filtered = append(filtered, comment)
		}
		comments = filtered
	}

	authorIDs := make([]uint, 0, len(comments))
	for _, comment := range comments {
		authorIDs = append(authorIDs, comment.AuthorID)
	}
	var authors []model.User
	database.DB.Select("id", "username", "avatar", "sponsor_color", "sponsor_bold", "activity_experience").Where("id IN ?", authorIDs).Find(&authors)
	authorMap := map[uint]model.User{}
	for _, author := range authors {
		authorMap[author.ID] = author
	}

	liked := map[uint]bool{}
	if viewerID != 0 {
		var likes []model.RPDBCommentLike
		commentIDs := make([]uint, 0, len(comments))
		for _, comment := range comments {
			commentIDs = append(commentIDs, comment.ID)
		}
		if len(commentIDs) > 0 {
			database.DB.Where("user_id = ? AND comment_id IN ?", viewerID, commentIDs).Find(&likes)
		}
		for _, like := range likes {
			liked[like.CommentID] = true
		}
	}

	response := make([]rpdbCommentResponse, 0, len(comments))
	for _, comment := range comments {
		author := authorMap[comment.AuthorID]
		levelInfo := resolveForumLevelInfo(author.ActivityExperience)
		response = append(response, rpdbCommentResponse{
			RPDBComment:           comment,
			AuthorName:            author.Username,
			AuthorAvatar:          author.Avatar,
			AuthorNameColor:       author.SponsorColor,
			AuthorNameBold:        author.SponsorBold,
			AuthorForumLevel:      levelInfo.Level,
			AuthorForumLevelName:  levelInfo.Name,
			AuthorForumLevelColor: levelInfo.Color,
			AuthorForumLevelBold:  levelInfo.Bold,
			Liked:                 liked[comment.ID],
		})
	}
	c.JSON(http.StatusOK, gin.H{"comments": response})
}

func (s *Server) createRPDBComment(c *gin.Context) {
	userID := c.GetUint("userID")
	workID, ok := parseRPDBWorkID(c)
	if !ok {
		return
	}
	if !rpdbPublishedWorkExists(workID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	var request struct {
		Content  string `json:"content"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}
	request.Content = strings.TrimSpace(request.Content)
	if request.Content == "" || len([]rune(request.Content)) > 2000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论长度必须为 1 到 2000 个字符"})
		return
	}
	if request.ParentID != nil {
		var parent model.RPDBComment
		if err := database.DB.Where("id = ? AND work_id = ?", *request.ParentID, workID).First(&parent).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "回复目标不存在"})
			return
		}
	}

	comment := model.RPDBComment{
		WorkID:   workID,
		AuthorID: userID,
		ParentID: request.ParentID,
		Content:  request.Content,
		Status:   "published",
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&comment).Error; err != nil {
			return err
		}
		return tx.Model(&model.RPDBWork{}).Where("id = ?", workID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发布评论失败"})
		return
	}
	s.bumpRPDBListCache(c.Request.Context())

	var work model.RPDBWork
	if err := database.DB.Select("id", "author_id", "title").First(&work, workID).Error; err == nil && work.AuthorID != userID {
		actorID := userID
		_ = service.CreateNotification(&model.Notification{
			UserID:     work.AuthorID,
			Type:       "rpdb_comment",
			ActorID:    &actorID,
			TargetType: "rpdb_work",
			TargetID:   workID,
			Content:    "有人评论了你的 RP 数据库作品《" + work.Title + "》",
		})
	}

	c.JSON(http.StatusCreated, gin.H{"comment": comment})
}

func (s *Server) deleteRPDBComment(c *gin.Context) {
	userID := c.GetUint("userID")
	commentID, err := strconv.ParseUint(c.Param("commentId"), 10, 64)
	if err != nil || commentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的评论 ID"})
		return
	}
	var comment model.RPDBComment
	if err := database.DB.First(&comment, uint(commentID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "评论不存在"})
		return
	}
	var work model.RPDBWork
	if err := database.DB.Select("id", "author_id").First(&work, comment.WorkID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	role, _ := rpdbUserRole(userID)
	if comment.AuthorID != userID && work.AuthorID != userID && role != "moderator" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除该评论"})
		return
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return deleteRPDBCommentRecord(tx, comment)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除评论失败"})
		return
	}
	s.bumpRPDBListCache(c.Request.Context())
	c.Status(http.StatusNoContent)
}

func deleteRPDBCommentRecord(tx *gorm.DB, comment model.RPDBComment) error {
	if err := tx.Where("comment_id = ?", comment.ID).Delete(&model.RPDBCommentLike{}).Error; err != nil {
		return err
	}
	if comment.ParentID == nil {
		if err := tx.Model(&model.RPDBComment{}).Where("parent_id = ?", comment.ID).
			Update("parent_id", nil).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Model(&model.RPDBComment{}).Where("parent_id = ?", comment.ID).
			Update("parent_id", *comment.ParentID).Error; err != nil {
			return err
		}
	}
	if err := tx.Delete(&comment).Error; err != nil {
		return err
	}
	return tx.Model(&model.RPDBWork{}).Where("id = ?", comment.WorkID).
		UpdateColumn("comment_count", gorm.Expr("CASE WHEN comment_count > 0 THEN comment_count - 1 ELSE 0 END")).Error
}

func (s *Server) verifyRPDBWork(c *gin.Context) {
	userID := c.GetUint("userID")
	workID, ok := parseRPDBWorkID(c)
	if !ok {
		return
	}
	var work model.RPDBWork
	if err := database.DB.First(&work, workID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	var request struct {
		Result  string `json:"result"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}
	if request.Result != "valid" && request.Result != "outdated" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证结果无效"})
		return
	}
	verification := model.RPDBVerification{
		WorkID:      workID,
		UserID:      userID,
		WorkVersion: work.Version,
		Result:      request.Result,
		Comment:     strings.TrimSpace(request.Comment),
	}
	if err := database.DB.Where("work_id = ? AND user_id = ?", workID, userID).
		Assign(verification).
		FirstOrCreate(&verification).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存验证结果失败"})
		return
	}
	recalculateRPDBVerification(workID)
	s.bumpRPDBListCache(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"verification": verification})
}

func recalculateRPDBVerification(workID uint) {
	var validCount int64
	var outdatedCount int64
	database.DB.Model(&model.RPDBVerification{}).Where("work_id = ? AND result = ?", workID, "valid").Count(&validCount)
	database.DB.Model(&model.RPDBVerification{}).Where("work_id = ? AND result = ?", workID, "outdated").Count(&outdatedCount)
	status := model.RPDBVerificationUnverified
	if validCount >= 2 && validCount > outdatedCount {
		status = model.RPDBVerificationVerified
	} else if outdatedCount > validCount && outdatedCount >= 1 {
		status = model.RPDBVerificationStale
	} else if validCount > 0 && outdatedCount > 0 {
		status = model.RPDBVerificationDisputed
	}
	updates := map[string]interface{}{
		"verified_count":      validCount,
		"outdated_count":      outdatedCount,
		"verification_status": status,
	}
	if validCount > 0 {
		updates["last_verified_at"] = gorm.Expr("CURRENT_TIMESTAMP")
	}
	database.DB.Model(&model.RPDBWork{}).Where("id = ?", workID).Updates(updates)
}

func parseRPDBWorkID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品 ID"})
		return 0, false
	}
	return uint(value), true
}

func rpdbPublishedWorkExists(workID uint) bool {
	var count int64
	database.DB.Model(&model.RPDBWork{}).
		Where("id = ? AND status = ? AND review_status = ? AND is_public = ?",
			workID,
			model.RPDBStatusPublished,
			model.RPDBReviewApproved,
			true,
		).
		Count(&count)
	return count == 1
}
