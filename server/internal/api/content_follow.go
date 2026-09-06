package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
	"gorm.io/gorm/clause"
)

func (s *Server) followPost(c *gin.Context) {
	userID := c.GetUint("userID")
	postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || postID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子 ID"})
		return
	}

	var post model.Post
	if err := database.DB.First(&post, uint(postID)).Error; err != nil || !canAccessPost(userID, post) {
		c.JSON(http.StatusNotFound, gin.H{"error": "帖子不存在"})
		return
	}
	follow := model.PostFollow{PostID: post.ID, UserID: userID}
	result := database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&follow)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "关注失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"followed": true})
}

func (s *Server) unfollowPost(c *gin.Context) {
	userID := c.GetUint("userID")
	postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || postID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子 ID"})
		return
	}
	if err := database.DB.Where("post_id = ? AND user_id = ?", uint(postID), userID).Delete(&model.PostFollow{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消关注失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"followed": false})
}

func (s *Server) listMyPostFollows(c *gin.Context) {
	s.listUserPostsByRelation(c, "post_follows", "created_at")
}

func (s *Server) followItem(c *gin.Context) {
	userID := c.GetUint("userID")
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || itemID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品 ID"})
		return
	}

	var item model.Item
	if err := database.DB.First(&item, uint(itemID)).Error; err != nil ||
		item.Status != "published" || item.ReviewStatus != "approved" || !item.IsPublic {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	follow := model.ItemFollow{ItemID: item.ID, UserID: userID}
	result := database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&follow)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "关注失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"followed": true})
}

func (s *Server) unfollowItem(c *gin.Context) {
	userID := c.GetUint("userID")
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || itemID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品 ID"})
		return
	}
	if err := database.DB.Where("item_id = ? AND user_id = ?", uint(itemID), userID).Delete(&model.ItemFollow{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消关注失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"followed": false})
}

func (s *Server) listMyItemFollows(c *gin.Context) {
	s.listUserItemsByRelation(c, "item_follows", "created_at")
}

func notifyPostFollowers(post model.Post) {
	if post.Status != "published" || post.ReviewStatus != "approved" {
		return
	}
	var follows []model.PostFollow
	if err := database.DB.Where("post_id = ?", post.ID).Find(&follows).Error; err != nil {
		return
	}
	for _, follow := range follows {
		if !canAccessPost(follow.UserID, post) || isUserBlocked(follow.UserID, post.AuthorID) {
			continue
		}
		authorID := post.AuthorID
		_ = service.CreateNotification(&model.Notification{
			UserID: follow.UserID, Type: "follow_update", ActorID: &authorID,
			TargetType: "post", TargetID: post.ID,
			Content: fmt.Sprintf("更新了你关注的帖子《%s》", post.Title),
		})
	}
}

func notifyItemFollowers(item model.Item) {
	if item.Status != "published" || item.ReviewStatus != "approved" || !item.IsPublic {
		return
	}
	var follows []model.ItemFollow
	if err := database.DB.Where("item_id = ?", item.ID).Find(&follows).Error; err != nil {
		return
	}
	for _, follow := range follows {
		if isUserBlocked(follow.UserID, item.AuthorID) {
			continue
		}
		authorID := item.AuthorID
		_ = service.CreateNotification(&model.Notification{
			UserID: follow.UserID, Type: "follow_update", ActorID: &authorID,
			TargetType: "item", TargetID: item.ID,
			Content: fmt.Sprintf("更新了你关注的作品《%s》", item.Name),
		})
	}
}
