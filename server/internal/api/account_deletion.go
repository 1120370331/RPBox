package api

import (
	"encoding/json"
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
	user                model.User
	posts               []model.Post
	items               []accountDeletionItemCleanup
	comments            []model.Comment
	itemComments        []model.ItemComment
	rpdbComments        []model.RPDBComment
	rpdbWorks           []model.RPDBWork
	rpdbDrafts          []model.RPDBDraft
	rpdbReferences      []model.RPDBReference
	rpdbMedia           []model.RPDBMedia
	rpdbTransmogSlots   []model.RPDBTransmogSlot
	rpdbGuideSteps      []model.RPDBGuideStep
	rpdbRevisions       []model.RPDBRevision
	rpdbSets            []model.RPDBSet
	storyEntries        []model.StoryEntry
	storyMusicTracks    []model.StoryMusicTrack
	storyMusicPlaylists []model.StoryMusicPlaylist
	guilds              []model.Guild
	collections         []model.Collection
	characters          []model.Character
	characterCards      []model.CharacterCard
	cardImpressions     []model.CharacterCardImpression
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
	s.bumpRPDBListCache(c.Request.Context())

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
	storyMusicTrackIDs, err := pluckUintIDs(tx, &model.StoryMusicTrack{}, "id", "user_id = ?", userID)
	if err != nil {
		return err
	}
	storyMusicPlaylistIDs, err := pluckUintIDs(tx, &model.StoryMusicPlaylist{}, "id", "user_id = ?", userID)
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
	if err := tx.Where("user_id = ?", userID).Find(&cleanupPlan.characterCards).Error; err != nil {
		return err
	}
	cardIDs := characterCardIDs(cleanupPlan.characterCards)
	if len(cardIDs) > 0 {
		if err := tx.Where("character_card_id IN ?", cardIDs).Find(&cleanupPlan.cardImpressions).Error; err != nil {
			return err
		}
	}
	postCommentCleanupQuery := tx.Where("author_id = ?", userID)
	if len(ownedPostIDs) > 0 {
		postCommentCleanupQuery = tx.Where("author_id = ? OR post_id IN ?", userID, ownedPostIDs)
	}
	if err := postCommentCleanupQuery.Find(&cleanupPlan.comments).Error; err != nil {
		return err
	}
	itemCommentCleanupQuery := tx.Where("user_id = ?", userID)
	if len(ownedItemIDs) > 0 {
		itemCommentCleanupQuery = tx.Where("user_id = ? OR item_id IN ?", userID, ownedItemIDs)
	}
	if err := itemCommentCleanupQuery.Find(&cleanupPlan.itemComments).Error; err != nil {
		return err
	}
	if len(ownedStoryIDs) > 0 {
		if err := tx.Where("story_id IN ?", ownedStoryIDs).Find(&cleanupPlan.storyEntries).Error; err != nil {
			return err
		}
	}
	if len(storyMusicTrackIDs) > 0 {
		if err := tx.Where("id IN ?", storyMusicTrackIDs).Find(&cleanupPlan.storyMusicTracks).Error; err != nil {
			return err
		}
	}
	if len(storyMusicPlaylistIDs) > 0 {
		if err := tx.Where("id IN ?", storyMusicPlaylistIDs).Find(&cleanupPlan.storyMusicPlaylists).Error; err != nil {
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
		if err := tx.Where("tag_id IN ?", createdTagIDs).Delete(&model.RPDBTag{}).Error; err != nil {
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
		if err := tx.Where("post_id IN ?", ownedPostIDs).Delete(&model.PostFollow{}).Error; err != nil {
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
		if err := tx.Where("item_id IN ?", ownedItemIDs).Delete(&model.ItemFollow{}).Error; err != nil {
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
		if err := tx.Where("story_id IN ?", ownedStoryIDs).Delete(&model.StoryMusicSegment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("story_id IN ?", ownedStoryIDs).Delete(&model.StoryMusicTrackStory{}).Error; err != nil {
			return err
		}
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

	if len(storyMusicTrackIDs) > 0 {
		if err := tx.Where("track_id IN ?", storyMusicTrackIDs).Delete(&model.StoryMusicSegment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("track_id IN ?", storyMusicTrackIDs).Delete(&model.StoryMusicPlaylistTrack{}).Error; err != nil {
			return err
		}
		if err := tx.Where("track_id IN ?", storyMusicTrackIDs).Delete(&model.StoryMusicTrackStory{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", storyMusicTrackIDs).Delete(&model.StoryMusicTrack{}).Error; err != nil {
			return err
		}
	}

	if len(storyMusicPlaylistIDs) > 0 {
		if err := tx.Where("playlist_id IN ?", storyMusicPlaylistIDs).Delete(&model.StoryMusicPlaylistTrack{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", storyMusicPlaylistIDs).Delete(&model.StoryMusicPlaylist{}).Error; err != nil {
			return err
		}
	}

	if len(ownedGuildIDs) > 0 {
		if err := tx.Where("target_type = ? AND target_id IN ?", "guild", ownedGuildIDs).
			Delete(&model.ContentReport{}).Error; err != nil {
			return fmt.Errorf("delete reports for removed guilds: %w", err)
		}
		if err := deleteGuildRecords(tx, ownedGuildIDs); err != nil {
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

	if err := deleteAccountRPDBData(tx, userID, cleanupPlan); err != nil {
		return err
	}

	if err := tx.Where("user_id = ?", userID).Delete(&model.PostLike{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.PostFavorite{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.PostFollow{}).Error; err != nil {
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
	if err := tx.Where("user_id = ?", userID).Delete(&model.ItemFollow{}).Error; err != nil {
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
	if len(cleanupPlan.characterCards) > 0 {
		if err := tx.Where("target_type = ? AND target_id IN ?", "character_card", cardIDs).
			Delete(&model.ContentReport{}).Error; err != nil {
			return fmt.Errorf("delete reports for removed character cards: %w", err)
		}
		if err := tx.Where("character_card_id IN ?", characterCardIDs(cleanupPlan.characterCards)).Delete(&model.CharacterCardPublication{}).Error; err != nil {
			return err
		}
		if err := tx.Where("character_card_id IN ?", characterCardIDs(cleanupPlan.characterCards)).Delete(&model.CharacterCardSubmission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("character_card_id IN ?", characterCardIDs(cleanupPlan.characterCards)).Delete(&model.CharacterCardPortrait{}).Error; err != nil {
			return err
		}
		if err := tx.Where("character_card_id IN ?", characterCardIDs(cleanupPlan.characterCards)).Delete(&model.CharacterCardImpression{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.CharacterCard{}).Error; err != nil {
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
	if err := tx.Where("blocker_id = ? OR blocked_user_id = ?", userID, userID).Delete(&model.UserBlock{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.UserHiddenContent{}).Error; err != nil {
		return err
	}
	if err := tx.Where("reporter_id = ?", userID).Delete(&model.ContentReport{}).Error; err != nil {
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

// deleteAccountRPDBData removes private/account-specific RPDB data while
// retaining only authored works that are already approved public knowledge.
// The retained works continue to point at the anonymized user shell.
func deleteAccountRPDBData(tx *gorm.DB, userID uint, cleanupPlan *accountDeletionCleanupPlan) error {
	var ownedWorks []model.RPDBWork
	if err := tx.Where("author_id = ?", userID).Find(&ownedWorks).Error; err != nil {
		return fmt.Errorf("collect authored RPDB works: %w", err)
	}

	deletedWorkIDs := make([]uint, 0, len(ownedWorks))
	retainedWorkIDs := make([]uint, 0, len(ownedWorks))
	for _, work := range ownedWorks {
		if isRetainableDeletedAccountRPDBWork(work) {
			retainedWorkIDs = append(retainedWorkIDs, work.ID)
			continue
		}
		deletedWorkIDs = append(deletedWorkIDs, work.ID)
		cleanupPlan.rpdbWorks = append(cleanupPlan.rpdbWorks, work)
	}

	ownedListIDs, err := pluckUintIDs(tx, &model.RPDBList{}, "id", "user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("collect RPDB lists: %w", err)
	}
	ownedSetIDs, err := pluckUintIDs(tx, &model.RPDBSet{}, "id", "author_id = ?", userID)
	if err != nil {
		return fmt.Errorf("collect RPDB sets: %w", err)
	}
	if len(ownedSetIDs) > 0 {
		if err := tx.Where("id IN ?", ownedSetIDs).Find(&cleanupPlan.rpdbSets).Error; err != nil {
			return fmt.Errorf("collect authored RPDB sets for cleanup: %w", err)
		}
	}

	if err := tx.Where("author_id = ?", userID).Find(&cleanupPlan.rpdbDrafts).Error; err != nil {
		return fmt.Errorf("collect RPDB drafts for cleanup: %w", err)
	}

	if len(deletedWorkIDs) > 0 {
		if err := tx.Where("work_id IN ?", deletedWorkIDs).Find(&cleanupPlan.rpdbReferences).Error; err != nil {
			return fmt.Errorf("collect RPDB references for cleanup: %w", err)
		}
		if err := tx.Where("work_id IN ?", deletedWorkIDs).Find(&cleanupPlan.rpdbMedia).Error; err != nil {
			return fmt.Errorf("collect RPDB media for cleanup: %w", err)
		}
		if err := tx.Where("work_id IN ?", deletedWorkIDs).Find(&cleanupPlan.rpdbTransmogSlots).Error; err != nil {
			return fmt.Errorf("collect RPDB transmog slots for cleanup: %w", err)
		}
		if err := tx.Where("work_id IN ?", deletedWorkIDs).Find(&cleanupPlan.rpdbGuideSteps).Error; err != nil {
			return fmt.Errorf("collect RPDB guide steps for cleanup: %w", err)
		}
	}

	revisionQuery := tx.Where("proposer_id = ?", userID)
	if len(deletedWorkIDs) > 0 {
		revisionQuery = tx.Where("proposer_id = ? OR work_id IN ?", userID, deletedWorkIDs)
	}
	if err := revisionQuery.Find(&cleanupPlan.rpdbRevisions).Error; err != nil {
		return fmt.Errorf("collect RPDB revisions for cleanup: %w", err)
	}

	var contributedMedia []model.RPDBMedia
	contributedMediaQuery := tx.Where("author_id = ?", userID)
	if len(deletedWorkIDs) > 0 {
		contributedMediaQuery = contributedMediaQuery.Where("work_id NOT IN ?", deletedWorkIDs)
	}
	if err := contributedMediaQuery.Find(&contributedMedia).Error; err != nil {
		return fmt.Errorf("collect contributed RPDB media: %w", err)
	}
	contributedWorkIDs := make([]uint, 0, len(contributedMedia))
	for _, media := range contributedMedia {
		contributedWorkIDs = append(contributedWorkIDs, media.WorkID)
	}
	var contributedWorks []model.RPDBWork
	if len(contributedWorkIDs) > 0 {
		if err := tx.Where("id IN ?", uniqueUintValues(contributedWorkIDs)).Find(&contributedWorks).Error; err != nil {
			return fmt.Errorf("collect contributed-media RPDB works: %w", err)
		}
	}
	contributedWorkByID := make(map[uint]model.RPDBWork, len(contributedWorks))
	for _, work := range contributedWorks {
		contributedWorkByID[work.ID] = work
	}
	retainedContributedMediaIDs := make([]uint, 0, len(contributedMedia))
	deletedContributedMediaIDs := make([]uint, 0, len(contributedMedia))
	deletedContributedMediaWorkIDs := make([]uint, 0, len(contributedMedia))
	for _, media := range contributedMedia {
		work, exists := contributedWorkByID[media.WorkID]
		if exists && isRetainableDeletedAccountRPDBWork(work) {
			retainedContributedMediaIDs = append(retainedContributedMediaIDs, media.ID)
			continue
		}
		deletedContributedMediaIDs = append(deletedContributedMediaIDs, media.ID)
		deletedContributedMediaWorkIDs = append(deletedContributedMediaWorkIDs, media.WorkID)
		cleanupPlan.rpdbMedia = append(cleanupPlan.rpdbMedia, media)
	}

	commentQuery := tx.Where("author_id = ?", userID)
	if len(deletedWorkIDs) > 0 {
		commentQuery = tx.Where("author_id = ? OR work_id IN ?", userID, deletedWorkIDs)
	}
	if err := commentQuery.Find(&cleanupPlan.rpdbComments).Error; err != nil {
		return fmt.Errorf("collect RPDB comments for cleanup: %w", err)
	}
	removedCommentIDs := rpdbCommentIDs(cleanupPlan.rpdbComments)
	removedCommentWorkIDs := rpdbCommentWorkIDs(cleanupPlan.rpdbComments)

	rpdbLikeWorkIDs, err := pluckUintIDs(tx, &model.RPDBLike{}, "work_id", "user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("collect RPDB likes: %w", err)
	}
	rpdbFavoriteWorkIDs, err := pluckUintIDs(tx, &model.RPDBFavorite{}, "work_id", "user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("collect RPDB favorites: %w", err)
	}
	rpdbViewWorkIDs, err := pluckUintIDs(tx, &model.RPDBView{}, "work_id", "user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("collect RPDB views: %w", err)
	}
	rpdbViewEventWorkIDs, err := pluckUintIDs(tx, &model.RPDBViewEvent{}, "work_id", "user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("collect RPDB view events: %w", err)
	}
	rpdbVerificationWorkIDs, err := pluckUintIDs(tx, &model.RPDBVerification{}, "work_id", "user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("collect RPDB verifications: %w", err)
	}
	rpdbCommentLikeIDs, err := pluckUintIDs(tx, &model.RPDBCommentLike{}, "comment_id", "user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("collect RPDB comment likes: %w", err)
	}

	var ownedListWorkIDs []uint
	if len(ownedListIDs) > 0 {
		ownedListWorkIDs, err = pluckUintIDs(tx, &model.RPDBListEntry{}, "work_id", "list_id IN ?", ownedListIDs)
		if err != nil {
			return fmt.Errorf("collect deleted-list RPDB works: %w", err)
		}
	}
	var affectedListIDs []uint
	if len(deletedWorkIDs) > 0 {
		affectedListIDs, err = pluckUintIDs(tx, &model.RPDBListEntry{}, "list_id", "work_id IN ?", deletedWorkIDs)
		if err != nil {
			return fmt.Errorf("collect RPDB lists affected by deleted works: %w", err)
		}
	}
	var affectedSetIDs []uint
	if len(deletedWorkIDs) > 0 {
		affectedSetIDs, err = pluckUintIDs(tx, &model.RPDBSetWork{}, "set_id", "work_id IN ?", deletedWorkIDs)
		if err != nil {
			return fmt.Errorf("collect RPDB sets affected by deleted works: %w", err)
		}
	}

	affectedWorkIDs := uniqueUintValues(
		retainedWorkIDs,
		removedCommentWorkIDs,
		rpdbLikeWorkIDs,
		rpdbFavoriteWorkIDs,
		rpdbViewWorkIDs,
		rpdbViewEventWorkIDs,
		rpdbVerificationWorkIDs,
		ownedListWorkIDs,
		deletedContributedMediaWorkIDs,
	)

	if len(removedCommentIDs) > 0 {
		if err := deleteNotificationsByTarget(tx, "rpdb_comment", removedCommentIDs); err != nil {
			return fmt.Errorf("delete RPDB comment notifications: %w", err)
		}
		if err := tx.Where("target_type = ? AND target_id IN ?", "rpdb_comment", removedCommentIDs).
			Delete(&model.UserHiddenContent{}).Error; err != nil {
			return fmt.Errorf("delete hidden RPDB comment references: %w", err)
		}
		if err := tx.Where("target_type = ? AND target_id IN ?", "rpdb_comment", removedCommentIDs).
			Delete(&model.ContentReport{}).Error; err != nil {
			return fmt.Errorf("delete reports for removed RPDB comments: %w", err)
		}
	}
	if err := deleteAccountRPDBComments(tx, cleanupPlan.rpdbComments); err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.RPDBCommentLike{}).Error; err != nil {
		return fmt.Errorf("delete RPDB comment likes: %w", err)
	}

	if len(cleanupPlan.rpdbDrafts) > 0 {
		if err := tx.Where("id IN ?", rpdbDraftIDs(cleanupPlan.rpdbDrafts)).Delete(&model.RPDBDraft{}).Error; err != nil {
			return fmt.Errorf("delete RPDB drafts: %w", err)
		}
	}
	if len(deletedWorkIDs) > 0 {
		if err := tx.Model(&model.RPDBDraft{}).
			Where("work_id IN ?", deletedWorkIDs).
			Updates(map[string]interface{}{"work_id": nil, "base_version": 0}).Error; err != nil {
			return fmt.Errorf("detach other users' RPDB drafts from deleted works: %w", err)
		}
	}
	if len(cleanupPlan.rpdbRevisions) > 0 {
		if err := tx.Where("id IN ?", rpdbRevisionIDs(cleanupPlan.rpdbRevisions)).Delete(&model.RPDBRevision{}).Error; err != nil {
			return fmt.Errorf("delete RPDB revisions: %w", err)
		}
	}
	if len(deletedContributedMediaIDs) > 0 {
		if err := tx.Where("id IN ?", deletedContributedMediaIDs).Delete(&model.RPDBMedia{}).Error; err != nil {
			return fmt.Errorf("delete non-public contributed RPDB media: %w", err)
		}
	}

	if len(deletedWorkIDs) > 0 {
		if err := deleteNotificationsByTarget(tx, "rpdb_work", deletedWorkIDs); err != nil {
			return fmt.Errorf("delete RPDB work notifications: %w", err)
		}
		if err := tx.Where("target_type = ? AND target_id IN ?", "rpdb_work", deletedWorkIDs).
			Delete(&model.UserHiddenContent{}).Error; err != nil {
			return fmt.Errorf("delete hidden RPDB work references: %w", err)
		}
		if err := tx.Where("target_type = ? AND target_id IN ?", "rpdb_work", deletedWorkIDs).
			Delete(&model.ContentReport{}).Error; err != nil {
			return fmt.Errorf("delete reports for removed RPDB works: %w", err)
		}
		if err := tx.Where("work_id IN ?", deletedWorkIDs).Delete(&model.RPDBListEntry{}).Error; err != nil {
			return fmt.Errorf("delete RPDB list entries for deleted works: %w", err)
		}
		if err := tx.Where("work_id IN ?", deletedWorkIDs).Delete(&model.RPDBSetWork{}).Error; err != nil {
			return fmt.Errorf("delete RPDB set entries for deleted works: %w", err)
		}
		for _, target := range []interface{}{
			&model.RPDBReference{},
			&model.RPDBMedia{},
			&model.RPDBTransmogSlot{},
			&model.RPDBGuideStep{},
			&model.RPDBTag{},
			&model.RPDBLike{},
			&model.RPDBFavorite{},
			&model.RPDBView{},
			&model.RPDBViewEvent{},
			&model.RPDBVerification{},
		} {
			if err := tx.Where("work_id IN ?", deletedWorkIDs).Delete(target).Error; err != nil {
				return fmt.Errorf("delete RPDB work dependency: %w", err)
			}
		}
		if err := tx.Where("id IN ?", deletedWorkIDs).Delete(&model.RPDBWork{}).Error; err != nil {
			return fmt.Errorf("delete private RPDB works: %w", err)
		}
	}

	if len(ownedListIDs) > 0 {
		if err := tx.Where("list_id IN ?", ownedListIDs).Delete(&model.RPDBListEntry{}).Error; err != nil {
			return fmt.Errorf("delete account RPDB list entries: %w", err)
		}
		if err := tx.Where("id IN ?", ownedListIDs).Delete(&model.RPDBList{}).Error; err != nil {
			return fmt.Errorf("delete account RPDB lists: %w", err)
		}
	}
	if len(ownedSetIDs) > 0 {
		if err := tx.Where("set_id IN ?", ownedSetIDs).Delete(&model.RPDBSetWork{}).Error; err != nil {
			return fmt.Errorf("delete account RPDB set entries: %w", err)
		}
		if err := tx.Where("id IN ?", ownedSetIDs).Delete(&model.RPDBSet{}).Error; err != nil {
			return fmt.Errorf("delete account RPDB sets: %w", err)
		}
	}

	for _, target := range []interface{}{
		&model.RPDBLike{},
		&model.RPDBFavorite{},
		&model.RPDBView{},
		&model.RPDBViewEvent{},
		&model.RPDBVerification{},
	} {
		if err := tx.Where("user_id = ?", userID).Delete(target).Error; err != nil {
			return fmt.Errorf("delete account RPDB interaction: %w", err)
		}
	}
	if len(retainedContributedMediaIDs) > 0 {
		if err := tx.Model(&model.RPDBMedia{}).Where("id IN ?", retainedContributedMediaIDs).Update("author_id", nil).Error; err != nil {
			return fmt.Errorf("anonymize retained RPDB media: %w", err)
		}
	}

	if err := recalculateRPDBCommentLikeCounts(tx, rpdbCommentLikeIDs); err != nil {
		return err
	}
	if err := recalculateRPDBListItemCounts(tx, affectedListIDs); err != nil {
		return err
	}
	if err := recalculateRPDBSetItemCounts(tx, affectedSetIDs); err != nil {
		return err
	}
	if err := recalculateRPDBWorkMetrics(tx, affectedWorkIDs); err != nil {
		return err
	}

	return nil
}

func isRetainableDeletedAccountRPDBWork(work model.RPDBWork) bool {
	return work.Status == model.RPDBStatusPublished &&
		work.ReviewStatus == model.RPDBReviewApproved &&
		work.IsPublic &&
		work.Visibility == model.RPDBVisibilityPublic
}

func deleteAccountRPDBComments(tx *gorm.DB, comments []model.RPDBComment) error {
	commentIDs := rpdbCommentIDs(comments)
	if len(commentIDs) == 0 {
		return nil
	}

	deletedParents := make(map[uint]*uint, len(comments))
	for _, comment := range comments {
		deletedParents[comment.ID] = comment.ParentID
	}
	resolveSurvivingParent := func(parentID *uint) *uint {
		seen := make(map[uint]struct{}, len(deletedParents))
		for parentID != nil {
			if _, repeated := seen[*parentID]; repeated {
				return nil
			}
			seen[*parentID] = struct{}{}
			next, deleted := deletedParents[*parentID]
			if !deleted {
				resolved := *parentID
				return &resolved
			}
			parentID = next
		}
		return nil
	}

	var survivingChildren []model.RPDBComment
	if err := tx.Where("parent_id IN ? AND id NOT IN ?", commentIDs, commentIDs).Find(&survivingChildren).Error; err != nil {
		return fmt.Errorf("collect surviving RPDB comment replies: %w", err)
	}
	for _, child := range survivingChildren {
		if err := tx.Model(&model.RPDBComment{}).Where("id = ?", child.ID).
			Update("parent_id", resolveSurvivingParent(child.ParentID)).Error; err != nil {
			return fmt.Errorf("reparent surviving RPDB comment %d: %w", child.ID, err)
		}
	}
	if err := tx.Where("comment_id IN ?", commentIDs).Delete(&model.RPDBCommentLike{}).Error; err != nil {
		return fmt.Errorf("delete likes on removed RPDB comments: %w", err)
	}
	if err := tx.Where("id IN ?", commentIDs).Delete(&model.RPDBComment{}).Error; err != nil {
		return fmt.Errorf("delete RPDB comments: %w", err)
	}
	return nil
}

func recalculateRPDBCommentLikeCounts(tx *gorm.DB, commentIDs []uint) error {
	for _, commentID := range uniqueUintValues(commentIDs) {
		var count int64
		if err := tx.Model(&model.RPDBCommentLike{}).Where("comment_id = ?", commentID).Count(&count).Error; err != nil {
			return fmt.Errorf("count RPDB comment likes for %d: %w", commentID, err)
		}
		if err := tx.Model(&model.RPDBComment{}).Where("id = ?", commentID).Update("like_count", count).Error; err != nil {
			return fmt.Errorf("update RPDB comment likes for %d: %w", commentID, err)
		}
	}
	return nil
}

func recalculateRPDBListItemCounts(tx *gorm.DB, listIDs []uint) error {
	for _, listID := range uniqueUintValues(listIDs) {
		var count int64
		if err := tx.Model(&model.RPDBListEntry{}).Where("list_id = ?", listID).Count(&count).Error; err != nil {
			return fmt.Errorf("count RPDB list entries for %d: %w", listID, err)
		}
		if err := tx.Model(&model.RPDBList{}).Where("id = ?", listID).Update("item_count", count).Error; err != nil {
			return fmt.Errorf("update RPDB list item count for %d: %w", listID, err)
		}
	}
	return nil
}

func recalculateRPDBSetItemCounts(tx *gorm.DB, setIDs []uint) error {
	for _, setID := range uniqueUintValues(setIDs) {
		var count int64
		if err := tx.Model(&model.RPDBSetWork{}).Where("set_id = ?", setID).Count(&count).Error; err != nil {
			return fmt.Errorf("count RPDB set entries for %d: %w", setID, err)
		}
		if err := tx.Model(&model.RPDBSet{}).Where("id = ?", setID).Update("item_count", count).Error; err != nil {
			return fmt.Errorf("update RPDB set item count for %d: %w", setID, err)
		}
	}
	return nil
}

func recalculateRPDBWorkMetrics(tx *gorm.DB, workIDs []uint) error {
	for _, workID := range uniqueUintValues(workIDs) {
		count := func(target interface{}, query string, args ...interface{}) (int64, error) {
			var value int64
			err := tx.Model(target).Where(query, args...).Count(&value).Error
			return value, err
		}

		likeCount, err := count(&model.RPDBLike{}, "work_id = ?", workID)
		if err != nil {
			return fmt.Errorf("count RPDB likes for work %d: %w", workID, err)
		}
		favoriteCount, err := count(&model.RPDBFavorite{}, "work_id = ?", workID)
		if err != nil {
			return fmt.Errorf("count RPDB favorites for work %d: %w", workID, err)
		}
		viewCount, err := count(&model.RPDBViewEvent{}, "work_id = ?", workID)
		if err != nil {
			return fmt.Errorf("count RPDB view events for work %d: %w", workID, err)
		}
		var commentCount int64
		if err := visibleCommentImages(tx.Model(&model.RPDBComment{})).
			Where("work_id = ? AND status = ?", workID, model.RPDBStatusPublished).
			Count(&commentCount).Error; err != nil {
			return fmt.Errorf("count RPDB comments for work %d: %w", workID, err)
		}
		listCount, err := count(&model.RPDBListEntry{}, "work_id = ?", workID)
		if err != nil {
			return fmt.Errorf("count RPDB list entries for work %d: %w", workID, err)
		}
		mediaCount, err := count(&model.RPDBMedia{}, "work_id = ?", workID)
		if err != nil {
			return fmt.Errorf("count RPDB media for work %d: %w", workID, err)
		}
		verifiedCount, err := count(&model.RPDBVerification{}, "work_id = ? AND result = ?", workID, "valid")
		if err != nil {
			return fmt.Errorf("count valid RPDB verifications for work %d: %w", workID, err)
		}
		outdatedCount, err := count(&model.RPDBVerification{}, "work_id = ? AND result = ?", workID, "outdated")
		if err != nil {
			return fmt.Errorf("count outdated RPDB verifications for work %d: %w", workID, err)
		}

		verificationStatus := model.RPDBVerificationUnverified
		if verifiedCount >= 2 && verifiedCount > outdatedCount {
			verificationStatus = model.RPDBVerificationVerified
		} else if outdatedCount > verifiedCount && outdatedCount >= 1 {
			verificationStatus = model.RPDBVerificationStale
		} else if verifiedCount > 0 && outdatedCount > 0 {
			verificationStatus = model.RPDBVerificationDisputed
		}
		updates := map[string]interface{}{
			"like_count":          likeCount,
			"favorite_count":      favoriteCount,
			"view_count":          viewCount,
			"comment_count":       commentCount,
			"list_count":          listCount,
			"media_count":         mediaCount,
			"verified_count":      verifiedCount,
			"outdated_count":      outdatedCount,
			"verification_status": verificationStatus,
			"last_verified_at":    nil,
		}
		if verifiedCount > 0 {
			var latest model.RPDBVerification
			if err := tx.Select("updated_at").
				Where("work_id = ? AND result = ?", workID, "valid").
				Order("updated_at DESC").First(&latest).Error; err != nil {
				return fmt.Errorf("load latest RPDB verification for work %d: %w", workID, err)
			}
			updates["last_verified_at"] = latest.UpdatedAt
		}
		if err := tx.Model(&model.RPDBWork{}).Where("id = ?", workID).Updates(updates).Error; err != nil {
			return fmt.Errorf("update RPDB metrics for work %d: %w", workID, err)
		}
	}
	return nil
}

func rpdbCommentIDs(comments []model.RPDBComment) []uint {
	ids := make([]uint, 0, len(comments))
	for _, comment := range comments {
		ids = append(ids, comment.ID)
	}
	return ids
}

func rpdbCommentWorkIDs(comments []model.RPDBComment) []uint {
	ids := make([]uint, 0, len(comments))
	for _, comment := range comments {
		ids = append(ids, comment.WorkID)
	}
	return ids
}

func rpdbDraftIDs(drafts []model.RPDBDraft) []uint {
	ids := make([]uint, 0, len(drafts))
	for _, draft := range drafts {
		ids = append(ids, draft.ID)
	}
	return ids
}

func rpdbRevisionIDs(revisions []model.RPDBRevision) []uint {
	ids := make([]uint, 0, len(revisions))
	for _, revision := range revisions {
		ids = append(ids, revision.ID)
	}
	return ids
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
		"username":                      deletedUsername,
		"email":                         deletedEmail,
		"email_verified":                false,
		"pass_hash":                     passHash,
		"avatar":                        "",
		"avatar_review_status":          "none",
		"avatar_reviewer_id":            nil,
		"avatar_reviewed_at":            nil,
		"avatar_review_comment":         "",
		"role":                          "user",
		"is_sponsor":                    false,
		"sponsor_level":                 0,
		"sponsor_acknowledgement_level": 0,
		"sponsor_color":                 "",
		"sponsor_bold":                  false,
		"bio":                           "",
		"location":                      "",
		"website":                       "",
		"activity_points":               0,
		"activity_experience":           0,
		"avatar_change_count":           0,
		"username_change_count":         0,
		"name_style_preference":         "default",
		"post_count":                    0,
		"story_count":                   0,
		"profile_count":                 0,
		"is_muted":                      false,
		"muted_until":                   nil,
		"mute_reason":                   "",
		"is_banned":                     false,
		"banned_until":                  nil,
		"ban_reason":                    "",
		"banned_by":                     nil,
		"banned_at":                     nil,
		"sensitive_violation_count":     0,
		"sensitive_last_violation_at":   nil,
		"account_deleted_at":            &now,
	}

	return tx.Model(&model.User{}).Where("id = ?", user.ID).Updates(updates).Error
}

func (s *Server) cleanupDeletedAccountUploads(c *gin.Context, plan accountDeletionCleanupPlan) {
	keys := make(map[string]struct{})
	rpdbKeys := make(map[string]struct{})
	commentImageURLs := make([]string, 0, len(plan.comments)+len(plan.itemComments)+len(plan.rpdbComments))

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
		if extractCommentImageStorageKey(nil, comment.ImageURL) != "" {
			commentImageURLs = append(commentImageURLs, comment.ImageURL)
		} else {
			collectUploadKeysFromValue(c, comment.ImageURL, keys)
		}
	}

	for _, comment := range plan.itemComments {
		if extractCommentImageStorageKey(nil, comment.ImageURL) != "" {
			commentImageURLs = append(commentImageURLs, comment.ImageURL)
		} else {
			collectUploadKeysFromValue(c, comment.ImageURL, keys)
		}
	}

	for _, comment := range plan.rpdbComments {
		if extractCommentImageStorageKey(nil, comment.ImageURL) != "" {
			commentImageURLs = append(commentImageURLs, comment.ImageURL)
		} else {
			collectUploadKeysFromValue(c, comment.ImageURL, keys)
		}
	}

	for _, work := range plan.rpdbWorks {
		collectUploadKeysFromValue(c, work.CoverImage, rpdbKeys)
		collectUploadKeysFromContent(c, work.Content, rpdbKeys)
		collectUploadKeysFromContent(c, work.RPUseCases, rpdbKeys)
		collectUploadKeysFromContent(c, work.EffectDescription, rpdbKeys)
		collectRPDBStructuredUploadKeys(c, work.Restrictions, rpdbKeys)
		collectRPDBStructuredUploadKeys(c, work.Extra, rpdbKeys)
	}
	for _, draft := range plan.rpdbDrafts {
		collectUploadKeysFromValue(c, draft.CoverImage, rpdbKeys)
		collectRPDBStructuredUploadKeys(c, draft.Payload, rpdbKeys)
	}
	for _, reference := range plan.rpdbReferences {
		collectUploadKeysFromValue(c, reference.Icon, rpdbKeys)
		collectUploadKeysFromValue(c, reference.URL, rpdbKeys)
		collectUploadKeysFromContent(c, reference.Description, rpdbKeys)
		collectUploadKeysFromContent(c, reference.AcquisitionMethod, rpdbKeys)
	}
	for _, media := range plan.rpdbMedia {
		collectUploadKeysFromValue(c, media.URL, rpdbKeys)
		collectUploadKeysFromValue(c, media.ThumbnailURL, rpdbKeys)
		collectRPDBStructuredUploadKeys(c, media.Meta, rpdbKeys)
	}
	for _, slot := range plan.rpdbTransmogSlots {
		collectUploadKeysFromValue(c, slot.WowheadURL, rpdbKeys)
		collectUploadKeysFromContent(c, slot.Description, rpdbKeys)
		collectUploadKeysFromContent(c, slot.Note, rpdbKeys)
	}
	for _, step := range plan.rpdbGuideSteps {
		collectUploadKeysFromContent(c, step.Body, rpdbKeys)
		collectRPDBStructuredUploadKeys(c, step.Meta, rpdbKeys)
	}
	for _, revision := range plan.rpdbRevisions {
		collectRPDBStructuredUploadKeys(c, revision.Payload, rpdbKeys)
		collectUploadKeysFromValue(c, revision.ChangeSummary, rpdbKeys)
		collectUploadKeysFromContent(c, revision.ChangeSummary, rpdbKeys)
		collectUploadKeysFromValue(c, revision.ReviewComment, rpdbKeys)
		collectUploadKeysFromContent(c, revision.ReviewComment, rpdbKeys)
	}
	for _, set := range plan.rpdbSets {
		collectUploadKeysFromValue(c, set.CoverImage, rpdbKeys)
		collectUploadKeysFromContent(c, set.Description, rpdbKeys)
	}

	for _, entry := range plan.storyEntries {
		collectUploadKeysFromValue(c, entry.Content, keys)
		collectUploadKeysFromContent(c, entry.Content, keys)
	}

	for _, track := range plan.storyMusicTracks {
		collectUploadKeysFromValue(c, track.URL, keys)
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

	for _, card := range plan.characterCards {
		collectUploadKeysFromValue(c, card.PortraitImage, keys)
	}
	for _, impression := range plan.cardImpressions {
		collectUploadKeysFromValue(c, impression.IconImage, keys)
		collectUploadKeysFromValue(c, impression.Image, keys)
	}

	s.deleteUploadKeys(keys)
	s.deleteUnreferencedRPDBUploadKeys(rpdbKeys)
	_ = s.cleanupCharacterCardUserStorage(plan.user.ID)
	s.cleanupCommentImageURLs(nil, commentImageURLs...)
}

func collectRPDBStructuredUploadKeys(c *gin.Context, raw string, keys map[string]struct{}) {
	if strings.TrimSpace(raw) == "" {
		return
	}
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		collectUploadKeysFromContent(c, raw, keys)
		collectUploadKeysFromValue(c, raw, keys)
		return
	}
	collectRPDBStructuredUploadValue(c, value, keys)
}

func collectRPDBStructuredUploadValue(c *gin.Context, value interface{}, keys map[string]struct{}) {
	switch typed := value.(type) {
	case string:
		collectUploadKeysFromValue(c, typed, keys)
		collectUploadKeysFromContent(c, typed, keys)
	case []interface{}:
		for _, entry := range typed {
			collectRPDBStructuredUploadValue(c, entry, keys)
		}
	case map[string]interface{}:
		for _, entry := range typed {
			collectRPDBStructuredUploadValue(c, entry, keys)
		}
	}
}

func (s *Server) deleteUnreferencedRPDBUploadKeys(keys map[string]struct{}) {
	for key := range keys {
		if isRPDBUploadKeyReferenced(key) {
			continue
		}
		s.deleteUploadKey(key)
	}
}

func isRPDBUploadKeyReferenced(key string) bool {
	if key == "" || database.DB == nil {
		return true
	}
	patterns := uploadKeyMatchPatterns(key)
	if len(patterns) == 0 {
		return true
	}

	targets := []struct {
		model   interface{}
		columns []string
	}{
		{&model.RPDBWork{}, []string{"cover_image", "content", "rp_use_cases", "effect_description", "restrictions", "extra"}},
		{&model.RPDBDraft{}, []string{"cover_image", "payload"}},
		{&model.RPDBReference{}, []string{"icon", "url", "description", "acquisition_method"}},
		{&model.RPDBMedia{}, []string{"url", "thumbnail_url", "meta"}},
		{&model.RPDBTransmogSlot{}, []string{"wowhead_url", "description", "note"}},
		{&model.RPDBGuideStep{}, []string{"body", "meta"}},
		{&model.RPDBRevision{}, []string{"payload", "change_summary", "review_comment"}},
		{&model.RPDBSet{}, []string{"cover_image", "description"}},
		{&model.RPDBComment{}, []string{"image_url"}},
	}
	for _, target := range targets {
		clauses := make([]string, 0, len(target.columns)*len(patterns))
		args := make([]interface{}, 0, len(target.columns)*len(patterns))
		for _, pattern := range patterns {
			for _, column := range target.columns {
				clauses = append(clauses, "CAST("+column+" AS TEXT) LIKE ?")
				args = append(args, pattern)
			}
		}
		var count int64
		if err := database.DB.Model(target.model).
			Where("("+strings.Join(clauses, " OR ")+")", args...).
			Limit(1).Count(&count).Error; err != nil {
			return true
		}
		if count > 0 {
			return true
		}
	}
	return false
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
		if err := visibleCommentImages(tx.Model(&model.Comment{})).Where("post_id = ?", postID).Count(&commentCount).Error; err != nil {
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
		if err := visibleCommentImages(tx.Model(&model.ItemComment{})).Where("item_id = ? AND rating > 0 AND parent_id IS NULL", itemID).Count(&ratingCount).Error; err != nil {
			return err
		}
		var avgRating float64
		if ratingCount > 0 {
			if err := visibleCommentImages(tx.Model(&model.ItemComment{})).Where("item_id = ? AND rating > 0 AND parent_id IS NULL", itemID).Select("AVG(rating)").Scan(&avgRating).Error; err != nil {
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
