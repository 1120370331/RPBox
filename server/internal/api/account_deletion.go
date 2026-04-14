package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	authpkg "github.com/rpbox/server/pkg/auth"
	"github.com/rpbox/server/pkg/validator"
	"gorm.io/gorm"
)

type deleteAccountRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

type accountDeletionCleanupPlan struct {
	user         model.User
	posts        []model.Post
	items        []accountDeletionItemCleanup
	comments     []model.Comment
	itemComments []model.ItemComment
	storyEntries []model.StoryEntry
	guilds       []model.Guild
	collections  []model.Collection
	characters   []model.Character
}

type accountDeletionItemCleanup struct {
	item   model.Item
	images []model.ItemImage
}

// deleteAccount permanently removes a user's private data and authored content,
// then anonymizes the shell account record so historical moderation snapshots stay intact.
func (s *Server) deleteAccount(c *gin.Context) {
	userID := c.GetUint("userID")

	var req deleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	var user model.User
	if err := database.DB.Select("id", "username", "pass_hash", "avatar", "account_deleted_at").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if user.AccountDeletedAt != nil {
		c.JSON(http.StatusGone, gin.H{"error": "账号已删除"})
		return
	}
	if !authpkg.CheckPassword(req.Password, user.PassHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	var cleanupPlan accountDeletionCleanupPlan
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return s.deleteAccountInTx(tx, user, &cleanupPlan)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除账号失败"})
		return
	}

	s.cleanupDeletedAccountUploads(c, cleanupPlan)
	s.invalidateUserProfileCache(c.Request.Context(), userID)
	s.bumpPostListCache(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "账号已删除"})
}

func (s *Server) deleteAccountInTx(tx *gorm.DB, user model.User, cleanupPlan *accountDeletionCleanupPlan) error {
	userID := user.ID
	cleanupPlan.user = user

	ownedPostIDs, err := pluckUintIDs(tx, &model.Post{}, "id", "author_id = ?", userID)
	if err != nil {
		return err
	}
	ownedItemIDs, err := pluckUintIDs(tx, &model.Item{}, "id", "author_id = ?", userID)
	if err != nil {
		return err
	}
	ownedStoryIDs, err := pluckUintIDs(tx, &model.Story{}, "id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	ownedGuildIDs, err := pluckUintIDs(tx, &model.Guild{}, "id", "owner_id = ?", userID)
	if err != nil {
		return err
	}
	ownedCollectionIDs, err := pluckUintIDs(tx, &model.Collection{}, "id", "author_id = ?", userID)
	if err != nil {
		return err
	}
	ownedProfileIDs, err := pluckStringIDs(tx, &model.Profile{}, "id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	backupIDs, err := pluckUintIDs(tx, &model.AccountBackup{}, "id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	createdTagIDs, err := pluckUintIDs(tx, &model.Tag{}, "id", "creator_id = ?", userID)
	if err != nil {
		return err
	}
	userCommentIDs, err := pluckUintIDs(tx, &model.Comment{}, "id", "author_id = ?", userID)
	if err != nil {
		return err
	}
	userCommentPostIDs, err := pluckUintIDs(tx, &model.Comment{}, "post_id", "author_id = ?", userID)
	if err != nil {
		return err
	}
	userItemCommentIDs, err := pluckUintIDs(tx, &model.ItemComment{}, "id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	userItemCommentItemIDs, err := pluckUintIDs(tx, &model.ItemComment{}, "item_id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	postLikeIDs, err := pluckUintIDs(tx, &model.PostLike{}, "post_id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	postFavoriteIDs, err := pluckUintIDs(tx, &model.PostFavorite{}, "post_id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	postViewIDs, err := pluckUintIDs(tx, &model.PostView{}, "post_id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	itemLikeIDs, err := pluckUintIDs(tx, &model.ItemLike{}, "item_id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	itemFavoriteIDs, err := pluckUintIDs(tx, &model.ItemFavorite{}, "item_id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	itemViewIDs, err := pluckUintIDs(tx, &model.ItemView{}, "item_id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	itemDownloadIDs, err := pluckUintIDs(tx, &model.ItemDownload{}, "item_id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	itemRatingIDs, err := pluckUintIDs(tx, &model.ItemRating{}, "item_id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	commentLikeCommentIDs, err := pluckUintIDs(tx, &model.CommentLike{}, "comment_id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	guildMembershipIDs, err := pluckUintIDs(tx, &model.GuildMember{}, "guild_id", "user_id = ?", userID)
	if err != nil {
		return err
	}

	if err := tx.Where("author_id = ?", userID).Find(&cleanupPlan.posts).Error; err != nil {
		return err
	}
	if err := tx.Where("owner_id = ?", userID).Find(&cleanupPlan.guilds).Error; err != nil {
		return err
	}
	if err := tx.Where("author_id = ?", userID).Find(&cleanupPlan.collections).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Find(&cleanupPlan.characters).Error; err != nil {
		return err
	}
	if len(userCommentIDs) > 0 {
		if err := tx.Where("id IN ?", userCommentIDs).Find(&cleanupPlan.comments).Error; err != nil {
			return err
		}
	}
	if len(userItemCommentIDs) > 0 {
		if err := tx.Where("id IN ?", userItemCommentIDs).Find(&cleanupPlan.itemComments).Error; err != nil {
			return err
		}
	}
	if len(ownedStoryIDs) > 0 {
		if err := tx.Where("story_id IN ?", ownedStoryIDs).Find(&cleanupPlan.storyEntries).Error; err != nil {
			return err
		}
	}

	if len(ownedItemIDs) > 0 {
		var items []model.Item
		if err := tx.Where("id IN ?", ownedItemIDs).Find(&items).Error; err != nil {
			return err
		}
		for _, item := range items {
			var images []model.ItemImage
			if err := tx.Where("item_id = ?", item.ID).Find(&images).Error; err != nil {
				return err
			}
			cleanupPlan.items = append(cleanupPlan.items, accountDeletionItemCleanup{
				item:   item,
				images: images,
			})
		}
	}

	affectedPostIDs := uniqueUintValues(userCommentPostIDs, postLikeIDs, postFavoriteIDs, postViewIDs)
	affectedItemIDs := uniqueUintValues(userItemCommentItemIDs, itemLikeIDs, itemFavoriteIDs, itemViewIDs, itemDownloadIDs, itemRatingIDs)
	affectedCommentIDs := uniqueUintValues(commentLikeCommentIDs)
	affectedGuildIDs := uniqueUintValues(guildMembershipIDs)

	var ownedPostCommentIDs []uint
	if len(ownedPostIDs) > 0 {
		ownedPostCommentIDs, err = pluckUintIDs(tx, &model.Comment{}, "id", "post_id IN ?", ownedPostIDs)
		if err != nil {
			return err
		}
	}
	commentTargetIDs := uniqueUintValues(userCommentIDs, ownedPostCommentIDs)

	if len(backupIDs) > 0 {
		if err := tx.Where("backup_id IN ?", backupIDs).Delete(&model.AccountBackupVersion{}).Error; err != nil {
			return err
		}
	}
	if len(ownedProfileIDs) > 0 {
		if err := tx.Where("profile_id IN ?", ownedProfileIDs).Delete(&model.ProfileVersion{}).Error; err != nil {
			return err
		}
	}

	if len(createdTagIDs) > 0 {
		if err := tx.Where("tag_id IN ?", createdTagIDs).Delete(&model.StoryTag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tag_id IN ?", createdTagIDs).Delete(&model.ItemTag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("tag_id IN ?", createdTagIDs).Delete(&model.PostTag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", createdTagIDs).Delete(&model.Tag{}).Error; err != nil {
			return err
		}
	}

	if len(ownedCollectionIDs) > 0 {
		if err := tx.Where("collection_id IN ?", ownedCollectionIDs).Delete(&model.CollectionFavorite{}).Error; err != nil {
			return err
		}
		if err := tx.Where("collection_id IN ?", ownedCollectionIDs).Delete(&model.CollectionPost{}).Error; err != nil {
			return err
		}
		if err := tx.Where("collection_id IN ?", ownedCollectionIDs).Delete(&model.CollectionItem{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ownedCollectionIDs).Delete(&model.Collection{}).Error; err != nil {
			return err
		}
	}

	if len(ownedPostIDs) > 0 {
		if err := deleteNotificationsByTarget(tx, "post", ownedPostIDs); err != nil {
			return err
		}
		if err := tx.Where("post_id IN ?", ownedPostIDs).Delete(&model.CollectionPost{}).Error; err != nil {
			return err
		}
		if len(ownedPostCommentIDs) > 0 {
			if err := deleteNotificationsByTarget(tx, "comment", ownedPostCommentIDs); err != nil {
				return err
			}
			if err := tx.Where("comment_id IN ?", ownedPostCommentIDs).Delete(&model.CommentLike{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("post_id IN ?", ownedPostIDs).Delete(&model.PostTag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id IN ?", ownedPostIDs).Delete(&model.Comment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id IN ?", ownedPostIDs).Delete(&model.PostLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id IN ?", ownedPostIDs).Delete(&model.PostFavorite{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id IN ?", ownedPostIDs).Delete(&model.PostView{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id IN ?", ownedPostIDs).Delete(&model.PostEditRequest{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ownedPostIDs).Delete(&model.Post{}).Error; err != nil {
			return err
		}
	}

	if len(ownedItemIDs) > 0 {
		if err := deleteNotificationsByTarget(tx, "item", ownedItemIDs); err != nil {
			return err
		}
		if err := tx.Where("item_id IN ?", ownedItemIDs).Delete(&model.CollectionItem{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id IN ?", ownedItemIDs).Delete(&model.ItemTag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id IN ?", ownedItemIDs).Delete(&model.ItemComment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id IN ?", ownedItemIDs).Delete(&model.ItemLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id IN ?", ownedItemIDs).Delete(&model.ItemFavorite{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id IN ?", ownedItemIDs).Delete(&model.ItemRating{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id IN ?", ownedItemIDs).Delete(&model.ItemView{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id IN ?", ownedItemIDs).Delete(&model.ItemDownload{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id IN ?", ownedItemIDs).Delete(&model.ItemPendingEdit{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id IN ?", ownedItemIDs).Delete(&model.ItemImage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ownedItemIDs).Delete(&model.Item{}).Error; err != nil {
			return err
		}
	}

	if len(ownedStoryIDs) > 0 {
		if err := tx.Where("story_id IN ?", ownedStoryIDs).Delete(&model.StoryBookmark{}).Error; err != nil {
			return err
		}
		if err := tx.Where("story_id IN ?", ownedStoryIDs).Delete(&model.StoryGuild{}).Error; err != nil {
			return err
		}
		if err := tx.Where("story_id IN ?", ownedStoryIDs).Delete(&model.StoryTag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("story_id IN ?", ownedStoryIDs).Delete(&model.StoryEntry{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ownedStoryIDs).Delete(&model.Story{}).Error; err != nil {
			return err
		}
	}

	if len(ownedGuildIDs) > 0 {
		if err := deleteNotificationsByTarget(tx, "guild", ownedGuildIDs); err != nil {
			return err
		}
		if err := tx.Model(&model.Post{}).Where("guild_id IN ?", ownedGuildIDs).Update("guild_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Where("guild_id IN ?", ownedGuildIDs).Delete(&model.GuildMember{}).Error; err != nil {
			return err
		}
		if err := tx.Where("guild_id IN ?", ownedGuildIDs).Delete(&model.GuildApplication{}).Error; err != nil {
			return err
		}
		if err := tx.Where("guild_id IN ?", ownedGuildIDs).Delete(&model.Tag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("guild_id IN ?", ownedGuildIDs).Delete(&model.StoryGuild{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ownedGuildIDs).Delete(&model.Guild{}).Error; err != nil {
			return err
		}
	}

	if len(commentTargetIDs) > 0 {
		if err := deleteNotificationsByTarget(tx, "comment", commentTargetIDs); err != nil {
			return err
		}
		if err := tx.Where("comment_id IN ?", commentTargetIDs).Delete(&model.CommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", commentTargetIDs).Delete(&model.Comment{}).Error; err != nil {
			return err
		}
	}

	if len(userItemCommentIDs) > 0 {
		if err := tx.Where("id IN ?", userItemCommentIDs).Delete(&model.ItemComment{}).Error; err != nil {
			return err
		}
	}

	if err := tx.Where("user_id = ?", userID).Delete(&model.PostLike{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.PostFavorite{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.PostView{}).Error; err != nil {
		return err
	}
	if err := tx.Where("comment_id IN ?", commentLikeCommentIDs).Where("user_id = ?", userID).Delete(&model.CommentLike{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.ItemLike{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.ItemFavorite{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.ItemView{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.ItemDownload{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.ItemRating{}).Error; err != nil {
		return err
	}

	if err := tx.Where("user_id = ?", userID).Delete(&model.GuildApplication{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.GuildMember{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.CollectionFavorite{}).Error; err != nil {
		return err
	}
	if err := tx.Where("author_id = ?", userID).Delete(&model.PostEditRequest{}).Error; err != nil {
		return err
	}
	if err := tx.Where("author_id = ?", userID).Delete(&model.ItemPendingEdit{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.StoryBookmark{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.Profile{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.AccountBackup{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.Character{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.UserDailyActivity{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.UserActivityLog{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.ContentModerationViolation{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ? OR actor_id = ?", userID, userID).Delete(&model.Notification{}).Error; err != nil {
		return err
	}

	if err := recalculateCommentLikeCounts(tx, affectedCommentIDs); err != nil {
		return err
	}
	if err := recalculatePostCounts(tx, affectedPostIDs); err != nil {
		return err
	}
	if err := recalculateItemMetrics(tx, affectedItemIDs); err != nil {
		return err
	}
	if err := recalculateGuildMemberCounts(tx, affectedGuildIDs); err != nil {
		return err
	}

	return anonymizeDeletedUser(tx, user)
}

func anonymizeDeletedUser(tx *gorm.DB, user model.User) error {
	now := time.Now()
	deletedUsername := fmt.Sprintf("deleted-user-%d", user.ID)
	deletedEmail := fmt.Sprintf("deleted+%d@rpbox.invalid", user.ID)
	randomPassword := fmt.Sprintf("deleted:%d:%d", user.ID, now.UnixNano())
	passHash, err := authpkg.HashPassword(randomPassword)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"username":                    deletedUsername,
		"email":                       deletedEmail,
		"email_verified":              false,
		"pass_hash":                   passHash,
		"avatar":                      "",
		"avatar_review_status":        "none",
		"avatar_reviewer_id":          nil,
		"avatar_reviewed_at":          nil,
		"avatar_review_comment":       "",
		"role":                        "user",
		"is_sponsor":                  false,
		"sponsor_level":               0,
		"sponsor_color":               "",
		"sponsor_bold":                false,
		"bio":                         "",
		"location":                    "",
		"website":                     "",
		"activity_points":             0,
		"activity_experience":         0,
		"avatar_change_count":         0,
		"username_change_count":       0,
		"name_style_preference":       "default",
		"post_count":                  0,
		"story_count":                 0,
		"profile_count":               0,
		"is_muted":                    false,
		"muted_until":                 nil,
		"mute_reason":                 "",
		"is_banned":                   false,
		"banned_until":                nil,
		"ban_reason":                  "",
		"banned_by":                   nil,
		"banned_at":                   nil,
		"sensitive_violation_count":   0,
		"sensitive_last_violation_at": nil,
		"account_deleted_at":          &now,
	}

	return tx.Model(&model.User{}).Where("id = ?", user.ID).Updates(updates).Error
}

func (s *Server) cleanupDeletedAccountUploads(c *gin.Context, plan accountDeletionCleanupPlan) {
	keys := make(map[string]struct{})

	collectUploadKeysFromValue(c, plan.user.Avatar, keys)

	for _, post := range plan.posts {
		collectUploadKeysFromValue(c, post.CoverImage, keys)
		collectUploadKeysFromContent(c, post.Content, keys)
	}

	for _, item := range plan.items {
		collectUploadKeysFromValue(c, item.item.PreviewImage, keys)
		collectUploadKeysFromValue(c, item.item.Icon, keys)
		collectUploadKeysFromContent(c, item.item.DetailContent, keys)
		for _, image := range item.images {
			collectUploadKeysFromValue(c, image.ImageData, keys)
		}
	}

	for _, comment := range plan.comments {
		collectUploadKeysFromValue(c, comment.ImageURL, keys)
	}

	for _, comment := range plan.itemComments {
		collectUploadKeysFromValue(c, comment.ImageURL, keys)
	}

	for _, entry := range plan.storyEntries {
		collectUploadKeysFromValue(c, entry.Content, keys)
		collectUploadKeysFromContent(c, entry.Content, keys)
	}

	for _, guild := range plan.guilds {
		collectUploadKeysFromValue(c, guild.Avatar, keys)
		collectUploadKeysFromValue(c, guild.Banner, keys)
		collectUploadKeysFromContent(c, guild.Lore, keys)
	}

	for _, collection := range plan.collections {
		collectUploadKeysFromValue(c, collection.CoverImage, keys)
	}

	for _, character := range plan.characters {
		collectUploadKeysFromValue(c, character.CustomAvatar, keys)
	}

	s.deleteUploadKeys(keys)
}

func deleteNotificationsByTarget(tx *gorm.DB, targetType string, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return tx.Where("target_type = ? AND target_id IN ?", targetType, ids).Delete(&model.Notification{}).Error
}

func recalculateCommentLikeCounts(tx *gorm.DB, commentIDs []uint) error {
	for _, commentID := range uniqueUintValues(commentIDs) {
		var count int64
		if err := tx.Model(&model.CommentLike{}).Where("comment_id = ?", commentID).Count(&count).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Comment{}).Where("id = ?", commentID).Update("like_count", count).Error; err != nil {
			return err
		}
	}
	return nil
}

func recalculatePostCounts(tx *gorm.DB, postIDs []uint) error {
	for _, postID := range uniqueUintValues(postIDs) {
		var commentCount int64
		if err := tx.Model(&model.Comment{}).Where("post_id = ?", postID).Count(&commentCount).Error; err != nil {
			return err
		}
		var likeCount int64
		if err := tx.Model(&model.PostLike{}).Where("post_id = ?", postID).Count(&likeCount).Error; err != nil {
			return err
		}
		var favoriteCount int64
		if err := tx.Model(&model.PostFavorite{}).Where("post_id = ?", postID).Count(&favoriteCount).Error; err != nil {
			return err
		}
		var viewCount int64
		if err := tx.Model(&model.PostView{}).Where("post_id = ?", postID).Count(&viewCount).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{
			"comment_count":  commentCount,
			"like_count":     likeCount,
			"favorite_count": favoriteCount,
			"view_count":     viewCount,
		}
		if err := tx.Model(&model.Post{}).Where("id = ?", postID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func recalculateItemMetrics(tx *gorm.DB, itemIDs []uint) error {
	for _, itemID := range uniqueUintValues(itemIDs) {
		var ratingCount int64
		if err := tx.Model(&model.ItemComment{}).Where("item_id = ? AND rating > 0", itemID).Count(&ratingCount).Error; err != nil {
			return err
		}
		var avgRating float64
		if ratingCount > 0 {
			if err := tx.Model(&model.ItemComment{}).Where("item_id = ? AND rating > 0", itemID).Select("AVG(rating)").Scan(&avgRating).Error; err != nil {
				return err
			}
		}
		var likeCount int64
		if err := tx.Model(&model.ItemLike{}).Where("item_id = ?", itemID).Count(&likeCount).Error; err != nil {
			return err
		}
		var favoriteCount int64
		if err := tx.Model(&model.ItemFavorite{}).Where("item_id = ?", itemID).Count(&favoriteCount).Error; err != nil {
			return err
		}
		var downloadCount int64
		if err := tx.Model(&model.ItemDownload{}).Where("item_id = ?", itemID).Count(&downloadCount).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{
			"rating_count":   ratingCount,
			"rating":         avgRating,
			"like_count":     likeCount,
			"favorite_count": favoriteCount,
			"downloads":      downloadCount,
		}
		if err := tx.Model(&model.Item{}).Where("id = ?", itemID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func recalculateGuildMemberCounts(tx *gorm.DB, guildIDs []uint) error {
	for _, guildID := range uniqueUintValues(guildIDs) {
		var memberCount int64
		if err := tx.Model(&model.GuildMember{}).Where("guild_id = ?", guildID).Count(&memberCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Guild{}).Where("id = ?", guildID).Update("member_count", memberCount).Error; err != nil {
			return err
		}
	}
	return nil
}

func pluckUintIDs(tx *gorm.DB, target interface{}, column, query string, args ...interface{}) ([]uint, error) {
	var ids []uint
	db := tx.Model(target).Distinct(column)
	if strings.TrimSpace(query) != "" {
		db = db.Where(query, args...)
	}
	if err := db.Pluck(column, &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func pluckStringIDs(tx *gorm.DB, target interface{}, column, query string, args ...interface{}) ([]string, error) {
	var ids []string
	db := tx.Model(target).Distinct(column)
	if strings.TrimSpace(query) != "" {
		db = db.Where(query, args...)
	}
	if err := db.Pluck(column, &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func uniqueUintValues(groups ...[]uint) []uint {
	seen := make(map[uint]struct{})
	result := make([]uint, 0)
	for _, group := range groups {
		for _, value := range group {
			if value == 0 {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}
