package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	authpkg "github.com/rpbox/server/pkg/auth"
)

func TestDeleteAccount(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Profile{},
		&model.ProfileVersion{},
		&model.AccountBackup{},
		&model.AccountBackupVersion{},
		&model.Story{},
		&model.StoryEntry{},
		&model.StoryBookmark{},
		&model.StoryMusicTrack{},
		&model.StoryMusicPlaylist{},
		&model.StoryMusicPlaylistTrack{},
		&model.StoryMusicTrackStory{},
		&model.StoryMusicSegment{},
		&model.Character{},
		&model.CharacterCard{},
		&model.CharacterCardPortrait{},
		&model.CharacterCardImpression{},
		&model.CharacterCardPublication{},
		&model.CharacterCardSubmission{},
		&model.Tag{},
		&model.StoryTag{},
		&model.Guild{},
		&model.GuildMember{},
		&model.GuildApplication{},
		&model.StoryGuild{},
		&model.Item{},
		&model.ItemTag{},
		&model.ItemRating{},
		&model.ItemComment{},
		&model.ItemLike{},
		&model.ItemFavorite{},
		&model.ItemView{},
		&model.ItemDownload{},
		&model.ItemPendingEdit{},
		&model.ItemImage{},
		&model.Post{},
		&model.PostEditRequest{},
		&model.PostTag{},
		&model.Comment{},
		&model.PostLike{},
		&model.PostFavorite{},
		&model.PostView{},
		&model.CommentLike{},
		&model.ContentModerationViolation{},
		&model.Notification{},
		&model.UserDailyActivity{},
		&model.UserActivityLog{},
		&model.Collection{},
		&model.CollectionPost{},
		&model.CollectionItem{},
		&model.CollectionFavorite{},
		&model.RPDBWork{},
		&model.RPDBDraft{},
		&model.RPDBReference{},
		&model.RPDBMedia{},
		&model.RPDBTransmogSlot{},
		&model.RPDBGuideStep{},
		&model.RPDBTag{},
		&model.RPDBLike{},
		&model.RPDBFavorite{},
		&model.RPDBView{},
		&model.RPDBViewEvent{},
		&model.RPDBComment{},
		&model.RPDBCommentLike{},
		&model.RPDBList{},
		&model.RPDBListEntry{},
		&model.RPDBRevision{},
		&model.RPDBVerification{},
		&model.RPDBSet{},
		&model.RPDBSetWork{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
		&model.ContentReport{},
	)
	database.DB = db

	passHash, err := authpkg.HashPassword("secret123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := model.User{Username: "deleter", Email: "deleter@example.com", PassHash: passHash}
	otherUser := model.User{Username: "other", Email: "other@example.com", PassHash: passHash}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}

	otherPost := model.Post{AuthorID: otherUser.ID, Title: "Other Post", Content: "body", CommentCount: 1, LikeCount: 1, FavoriteCount: 1, ViewCount: 1}
	ownedPost := model.Post{AuthorID: user.ID, Title: "Owned Post", Content: "owned"}
	if err := db.Create(&otherPost).Error; err != nil {
		t.Fatalf("create other post: %v", err)
	}
	if err := db.Create(&ownedPost).Error; err != nil {
		t.Fatalf("create owned post: %v", err)
	}

	otherComment := model.Comment{PostID: otherPost.ID, AuthorID: otherUser.ID, Content: "other comment", LikeCount: 1}
	userComment := model.Comment{PostID: otherPost.ID, AuthorID: user.ID, Content: "user comment"}
	if err := db.Create(&otherComment).Error; err != nil {
		t.Fatalf("create other comment: %v", err)
	}
	if err := db.Create(&userComment).Error; err != nil {
		t.Fatalf("create user comment: %v", err)
	}
	if err := db.Create(&model.CommentLike{CommentID: otherComment.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create comment like: %v", err)
	}
	if err := db.Create(&model.PostLike{PostID: otherPost.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create post like: %v", err)
	}
	if err := db.Create(&model.PostFavorite{PostID: otherPost.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create post favorite: %v", err)
	}
	if err := db.Create(&model.PostView{PostID: otherPost.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create post view: %v", err)
	}

	otherItem := model.Item{
		AuthorID:      otherUser.ID,
		Name:          "Other Item",
		Type:          "item",
		Downloads:     1,
		Rating:        4,
		RatingCount:   1,
		LikeCount:     1,
		FavoriteCount: 1,
	}
	ownedItem := model.Item{AuthorID: user.ID, Name: "Owned Item", Type: "item"}
	if err := db.Create(&otherItem).Error; err != nil {
		t.Fatalf("create other item: %v", err)
	}
	if err := db.Create(&ownedItem).Error; err != nil {
		t.Fatalf("create owned item: %v", err)
	}

	userItemComment := model.ItemComment{ItemID: otherItem.ID, UserID: user.ID, Rating: 4, Content: "great"}
	if err := db.Create(&userItemComment).Error; err != nil {
		t.Fatalf("create item comment: %v", err)
	}
	if err := db.Create(&model.ItemLike{ItemID: otherItem.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create item like: %v", err)
	}
	if err := db.Create(&model.ItemFavorite{ItemID: otherItem.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create item favorite: %v", err)
	}
	if err := db.Create(&model.ItemView{ItemID: otherItem.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create item view: %v", err)
	}
	if err := db.Create(&model.ItemDownload{ItemID: otherItem.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("create item download: %v", err)
	}

	profile := model.Profile{ID: "profile-1", UserID: user.ID, ProfileName: "Profile", Checksum: "abc"}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := db.Create(&model.ProfileVersion{ProfileID: profile.ID, Version: 1, Checksum: "abc"}).Error; err != nil {
		t.Fatalf("create profile version: %v", err)
	}

	backup := model.AccountBackup{UserID: user.ID, AccountID: "acc-1", Checksum: "sum"}
	if err := db.Create(&backup).Error; err != nil {
		t.Fatalf("create backup: %v", err)
	}
	if err := db.Create(&model.AccountBackupVersion{BackupID: backup.ID, Version: 1, Checksum: "sum"}).Error; err != nil {
		t.Fatalf("create backup version: %v", err)
	}
	characterCard := model.CharacterCard{
		UserID: user.ID, DisplayName: "Owned Character Card",
		Status: model.CharacterCardStatusPublished, Visibility: model.CharacterCardVisibilityPublic,
	}
	if err := db.Create(&characterCard).Error; err != nil {
		t.Fatalf("create character card: %v", err)
	}
	cardImpression := model.CharacterCardImpression{CharacterCardID: characterCard.ID, Slot: 1, Active: true, Title: "Owned impression"}
	if err := db.Create(&cardImpression).Error; err != nil {
		t.Fatalf("create character card impression: %v", err)
	}

	story := model.Story{UserID: user.ID, Title: "Story"}
	if err := db.Create(&story).Error; err != nil {
		t.Fatalf("create story: %v", err)
	}
	if err := db.Create(&model.StoryEntry{StoryID: story.ID, Content: "entry"}).Error; err != nil {
		t.Fatalf("create story entry: %v", err)
	}

	collection := model.Collection{AuthorID: user.ID, Name: "Collection"}
	if err := db.Create(&collection).Error; err != nil {
		t.Fatalf("create collection: %v", err)
	}

	guild := model.Guild{Name: "Guild", OwnerID: otherUser.ID, MemberCount: 2}
	if err := db.Create(&guild).Error; err != nil {
		t.Fatalf("create guild: %v", err)
	}
	if err := db.Create(&model.GuildMember{GuildID: guild.ID, UserID: otherUser.ID, Role: "owner"}).Error; err != nil {
		t.Fatalf("create owner guild member: %v", err)
	}
	if err := db.Create(&model.GuildMember{GuildID: guild.ID, UserID: user.ID, Role: "member"}).Error; err != nil {
		t.Fatalf("create user guild member: %v", err)
	}
	ownedGuild := model.Guild{Name: "Owned Guild", OwnerID: user.ID, MemberCount: 1, InviteCode: "owned-guild"}
	if err := db.Create(&ownedGuild).Error; err != nil {
		t.Fatalf("create owned guild: %v", err)
	}
	if err := db.Create(&model.GuildMember{GuildID: ownedGuild.ID, UserID: user.ID, Role: "owner"}).Error; err != nil {
		t.Fatalf("create owned guild member: %v", err)
	}

	const (
		retainedMediaPath             = "/uploads/rpdb/account/retained-media.png"
		contributedMediaPath          = "/uploads/rpdb/account/contributed-media.png"
		privateContributionPath       = "/uploads/rpdb/account/private-contribution.png"
		privateContributionThumbPath  = "/uploads/rpdb/account/private-contribution-thumbnail.png"
		privateContributionMetaPath   = "/uploads/rpdb/account/private-contribution-meta.png"
		otherPrivateMediaPath         = "/uploads/rpdb/account/other-private-media.png"
		privateCoverPath              = "/uploads/rpdb/account/private-cover.png"
		privateContentPath            = "/uploads/rpdb/account/private-content.png"
		privateMediaPath              = "/uploads/rpdb/account/private-media.png"
		privateThumbnailPath          = "/uploads/rpdb/account/private-thumbnail.png"
		privateReferencePath          = "/uploads/rpdb/account/private-reference.png"
		privateGuidePath              = "/uploads/rpdb/account/private-guide.png"
		privateCommentImagePath       = "/uploads/rpdb/account/private-comment.png"
		standaloneDraftPath           = "/uploads/rpdb/account/standalone-draft.png"
		retainedDraftCoverPath        = "/uploads/rpdb/account/retained-draft-cover.png"
		userRevisionUploadPath        = "/uploads/rpdb/account/user-revision.png"
		deletedWorkRevisionUploadPath = "/uploads/rpdb/account/deleted-work-revision.png"
		setCoverPath                  = "/uploads/rpdb/account/set-cover.png"
	)

	retainedWork := model.RPDBWork{
		AuthorID: user.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "Retained public knowledge",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true, MediaCount: 1, ListCount: 1,
	}
	privateWork := model.RPDBWork{
		AuthorID: user.ID, Type: model.RPDBWorkTypeHomeShowcase, Title: "Private pending work",
		Content: `<p><img src="` + privateContentPath + `"></p>`, CoverImage: privateCoverPath,
		Status: model.RPDBStatusPending, ReviewStatus: model.RPDBReviewPending,
		Visibility: model.RPDBVisibilityPrivate, IsPublic: false, MediaCount: 1, CommentCount: 1, ListCount: 1,
	}
	otherRPDBWork := model.RPDBWork{
		AuthorID: otherUser.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "Other retained work",
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved,
		Visibility: model.RPDBVisibilityPublic, IsPublic: true,
		LikeCount: 2, FavoriteCount: 2, ViewCount: 2, CommentCount: 2, ListCount: 2,
		VerifiedCount: 1, OutdatedCount: 1, VerificationStatus: model.RPDBVerificationDisputed,
	}
	otherPrivateRPDBWork := model.RPDBWork{
		AuthorID: otherUser.ID, Type: model.RPDBWorkTypeHomeShowcase, Title: "Other user's private pending work",
		Status: model.RPDBStatusPending, ReviewStatus: model.RPDBReviewPending,
		Visibility: model.RPDBVisibilityPrivate, IsPublic: false, MediaCount: 2,
	}
	for name, work := range map[string]*model.RPDBWork{
		"retained RPDB work":      &retainedWork,
		"private RPDB work":       &privateWork,
		"other RPDB work":         &otherRPDBWork,
		"other private RPDB work": &otherPrivateRPDBWork,
	} {
		if err := db.Create(work).Error; err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	retainedMediaAuthorID := user.ID
	retainedMedia := model.RPDBMedia{
		WorkID: retainedWork.ID, AuthorID: &retainedMediaAuthorID, Type: "image", URL: retainedMediaPath,
		ReviewStatus: model.RPDBReviewApproved,
	}
	contributedMediaAuthorID := user.ID
	contributedMedia := model.RPDBMedia{
		WorkID: otherRPDBWork.ID, AuthorID: &contributedMediaAuthorID, Type: "image", URL: contributedMediaPath,
		ReviewStatus: model.RPDBReviewApproved,
	}
	privateMediaAuthorID := user.ID
	privateMedia := model.RPDBMedia{
		WorkID: privateWork.ID, AuthorID: &privateMediaAuthorID, Type: "image", URL: privateMediaPath,
		ThumbnailURL: privateThumbnailPath, Meta: `{"original":"` + privateMediaPath + `"}`,
		ReviewStatus: model.RPDBReviewPending,
	}
	privateContributionAuthorID := user.ID
	privateContribution := model.RPDBMedia{
		WorkID: otherPrivateRPDBWork.ID, AuthorID: &privateContributionAuthorID, Type: "image",
		URL: privateContributionPath, ThumbnailURL: privateContributionThumbPath,
		Meta: `{"proof":"` + privateContributionMetaPath + `"}`, ReviewStatus: model.RPDBReviewPending,
	}
	otherPrivateMediaAuthorID := otherUser.ID
	otherPrivateMedia := model.RPDBMedia{
		WorkID: otherPrivateRPDBWork.ID, AuthorID: &otherPrivateMediaAuthorID, Type: "image",
		URL: otherPrivateMediaPath, ReviewStatus: model.RPDBReviewPending,
	}
	if err := db.Create(&[]*model.RPDBMedia{
		&retainedMedia,
		&contributedMedia,
		&privateMedia,
		&privateContribution,
		&otherPrivateMedia,
	}).Error; err != nil {
		t.Fatalf("create RPDB media: %v", err)
	}
	if err := db.Create(&model.RPDBReference{
		WorkID: privateWork.ID, ExternalType: "item", ExternalID: "private-item", Name: "Private ref",
		Icon: privateReferencePath, IsPrimary: true,
	}).Error; err != nil {
		t.Fatalf("create private RPDB reference: %v", err)
	}
	if err := db.Create(&model.RPDBTransmogSlot{
		WorkID: privateWork.ID, Slot: "HEAD", Name: "Private slot", Note: `![private](` + privateGuidePath + `)`,
	}).Error; err != nil {
		t.Fatalf("create private RPDB transmog slot: %v", err)
	}
	if err := db.Create(&model.RPDBGuideStep{
		WorkID: privateWork.ID, SortOrder: 1, Title: "Private step", Meta: `{"image":"` + privateGuidePath + `"}`,
	}).Error; err != nil {
		t.Fatalf("create private RPDB guide step: %v", err)
	}
	deletedTag := model.Tag{Name: "Deleted RPDB tag", Category: "rpdb", Type: "custom", CreatorID: user.ID, IsPublic: true}
	if err := db.Create(&deletedTag).Error; err != nil {
		t.Fatalf("create authored RPDB tag: %v", err)
	}
	if err := db.Create(&model.RPDBTag{WorkID: retainedWork.ID, TagID: deletedTag.ID, AddedBy: user.ID}).Error; err != nil {
		t.Fatalf("create authored RPDB tag link: %v", err)
	}
	sharedTag := model.Tag{Name: "Other RPDB tag", Category: "rpdb", Type: "custom", CreatorID: otherUser.ID, IsPublic: true}
	if err := db.Create(&sharedTag).Error; err != nil {
		t.Fatalf("create shared RPDB tag: %v", err)
	}
	if err := db.Create(&model.RPDBTag{WorkID: privateWork.ID, TagID: sharedTag.ID, AddedBy: otherUser.ID}).Error; err != nil {
		t.Fatalf("create private-work RPDB tag link: %v", err)
	}

	retainedWorkID := retainedWork.ID
	standaloneDraft := model.RPDBDraft{
		AuthorID: user.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "Standalone private draft",
		CoverImage: standaloneDraftPath, Payload: `{"media":[{"url":"` + standaloneDraftPath + `"}]}`,
		Status: model.RPDBDraftStatusActive,
	}
	retainedDraft := model.RPDBDraft{
		AuthorID: user.ID, WorkID: &retainedWorkID, Type: retainedWork.Type, Title: "Retained work private draft",
		CoverImage: retainedDraftCoverPath, Payload: `{"media":[{"url":"` + retainedMediaPath + `"}]}`,
		Status: model.RPDBDraftStatusActive,
	}
	otherDraft := model.RPDBDraft{
		AuthorID: otherUser.ID, WorkID: &retainedWorkID, Type: retainedWork.Type, Title: "Other user's draft",
		Payload: `{}`, Status: model.RPDBDraftStatusActive,
	}
	privateDraftWorkID := privateWork.ID
	otherPrivateDraft := model.RPDBDraft{
		AuthorID: otherUser.ID, WorkID: &privateDraftWorkID, Type: privateWork.Type, Title: "Other user's detached draft",
		Payload: `{}`, BaseVersion: 1, Status: model.RPDBDraftStatusActive,
	}
	if err := db.Create(&[]*model.RPDBDraft{&standaloneDraft, &retainedDraft, &otherDraft, &otherPrivateDraft}).Error; err != nil {
		t.Fatalf("create RPDB drafts: %v", err)
	}

	userList := model.RPDBList{UserID: user.ID, Name: "Private account list", ItemCount: 2}
	otherList := model.RPDBList{UserID: otherUser.ID, Name: "Other list", ItemCount: 2}
	if err := db.Create(&[]*model.RPDBList{&userList, &otherList}).Error; err != nil {
		t.Fatalf("create RPDB lists: %v", err)
	}
	if err := db.Create(&[]model.RPDBListEntry{
		{ListID: userList.ID, WorkID: retainedWork.ID, Status: model.RPDBListStatusWanted, Quantity: 1},
		{ListID: userList.ID, WorkID: otherRPDBWork.ID, Status: model.RPDBListStatusWanted, Quantity: 1},
		{ListID: otherList.ID, WorkID: privateWork.ID, Status: model.RPDBListStatusWanted, Quantity: 1},
		{ListID: otherList.ID, WorkID: otherRPDBWork.ID, Status: model.RPDBListStatusWanted, Quantity: 1},
	}).Error; err != nil {
		t.Fatalf("create RPDB list entries: %v", err)
	}

	userSet := model.RPDBSet{
		AuthorID: user.ID, Name: "Authored private set", CoverImage: setCoverPath,
		Status: model.RPDBStatusPublished, ReviewStatus: model.RPDBReviewApproved, IsPublic: true, ItemCount: 2,
	}
	otherSet := model.RPDBSet{
		AuthorID: otherUser.ID, Name: "Other set", Status: model.RPDBStatusPublished,
		ReviewStatus: model.RPDBReviewApproved, IsPublic: true, ItemCount: 2,
	}
	if err := db.Create(&[]*model.RPDBSet{&userSet, &otherSet}).Error; err != nil {
		t.Fatalf("create RPDB sets: %v", err)
	}
	if err := db.Create(&[]model.RPDBSetWork{
		{SetID: userSet.ID, WorkID: retainedWork.ID},
		{SetID: userSet.ID, WorkID: otherRPDBWork.ID},
		{SetID: otherSet.ID, WorkID: privateWork.ID},
		{SetID: otherSet.ID, WorkID: otherRPDBWork.ID},
	}).Error; err != nil {
		t.Fatalf("create RPDB set entries: %v", err)
	}

	privateWorkID := privateWork.ID
	userRetainedRevision := model.RPDBRevision{
		WorkID: retainedWork.ID, ProposerID: user.ID, BaseVersion: 1,
		Payload: `{"media":[{"url":"` + userRevisionUploadPath + `"}]}`, Status: model.RPDBReviewPending,
	}
	userHistoricalRevision := model.RPDBRevision{
		WorkID: otherRPDBWork.ID, ProposerID: user.ID, BaseVersion: 1, Payload: `{}`, Status: model.RPDBReviewRejected,
	}
	userDeletedWorkRevision := model.RPDBRevision{
		WorkID: privateWorkID, ProposerID: user.ID, BaseVersion: 1,
		Payload: `{"media":[{"url":"` + deletedWorkRevisionUploadPath + `"}]}`, Status: model.RPDBReviewPending,
	}
	otherRetainedRevision := model.RPDBRevision{
		WorkID: retainedWork.ID, ProposerID: otherUser.ID, BaseVersion: 1, Payload: `{}`, Status: model.RPDBReviewPending,
	}
	otherDeletedRevision := model.RPDBRevision{
		WorkID: privateWorkID, ProposerID: otherUser.ID, BaseVersion: 1,
		Payload: `{"media":[{"url":"` + deletedWorkRevisionUploadPath + `"}]}`, Status: model.RPDBReviewPending,
	}
	if err := db.Create(&[]*model.RPDBRevision{
		&userRetainedRevision,
		&userHistoricalRevision,
		&userDeletedWorkRevision,
		&otherRetainedRevision,
		&otherDeletedRevision,
	}).Error; err != nil {
		t.Fatalf("create RPDB revisions: %v", err)
	}

	if err := db.Create(&[]model.RPDBLike{
		{WorkID: otherRPDBWork.ID, UserID: user.ID},
		{WorkID: otherRPDBWork.ID, UserID: otherUser.ID},
	}).Error; err != nil {
		t.Fatalf("create RPDB likes: %v", err)
	}
	if err := db.Create(&[]model.RPDBFavorite{
		{WorkID: otherRPDBWork.ID, UserID: user.ID},
		{WorkID: otherRPDBWork.ID, UserID: otherUser.ID},
	}).Error; err != nil {
		t.Fatalf("create RPDB favorites: %v", err)
	}
	if err := db.Create(&[]model.RPDBView{
		{WorkID: otherRPDBWork.ID, UserID: user.ID},
		{WorkID: otherRPDBWork.ID, UserID: otherUser.ID},
	}).Error; err != nil {
		t.Fatalf("create RPDB views: %v", err)
	}
	viewDate := time.Now().Format("2006-01-02")
	if err := db.Create(&[]model.RPDBViewEvent{
		{WorkID: otherRPDBWork.ID, UserID: user.ID, ViewDate: viewDate},
		{WorkID: otherRPDBWork.ID, UserID: otherUser.ID, ViewDate: viewDate},
	}).Error; err != nil {
		t.Fatalf("create RPDB view events: %v", err)
	}
	if err := db.Create(&[]model.RPDBVerification{
		{WorkID: otherRPDBWork.ID, UserID: user.ID, WorkVersion: 1, Result: "valid"},
		{WorkID: otherRPDBWork.ID, UserID: otherUser.ID, WorkVersion: 1, Result: "outdated"},
	}).Error; err != nil {
		t.Fatalf("create RPDB verifications: %v", err)
	}

	otherRPDBComment := model.RPDBComment{
		WorkID: otherRPDBWork.ID, AuthorID: otherUser.ID, Content: "Other RPDB comment", LikeCount: 2,
		Status: model.RPDBStatusPublished,
	}
	userRPDBComment := model.RPDBComment{
		WorkID: otherRPDBWork.ID, AuthorID: user.ID, Content: "Deleted user's RPDB comment",
		Status: model.RPDBStatusPublished,
	}
	privateRPDBComment := model.RPDBComment{
		WorkID: privateWork.ID, AuthorID: otherUser.ID, Content: "Comment on private work",
		ImageURL: privateCommentImagePath, ImageReviewStatus: "approved", Status: model.RPDBStatusPublished,
	}
	if err := db.Create(&otherRPDBComment).Error; err != nil {
		t.Fatalf("create other RPDB comment: %v", err)
	}
	userRPDBComment.ParentID = &otherRPDBComment.ID
	if err := db.Create(&userRPDBComment).Error; err != nil {
		t.Fatalf("create user RPDB comment: %v", err)
	}
	userRPDBReply := model.RPDBComment{
		WorkID: otherRPDBWork.ID, AuthorID: user.ID, ParentID: &userRPDBComment.ID,
		Content: "Deleted user's nested RPDB reply", Status: model.RPDBStatusPublished,
	}
	if err := db.Create(&userRPDBReply).Error; err != nil {
		t.Fatalf("create user nested RPDB reply: %v", err)
	}
	otherRPDBGrandchild := model.RPDBComment{
		WorkID: otherRPDBWork.ID, AuthorID: otherUser.ID, ParentID: &userRPDBReply.ID,
		Content: "Surviving nested RPDB reply", Status: model.RPDBStatusPublished,
	}
	if err := db.Create(&otherRPDBGrandchild).Error; err != nil {
		t.Fatalf("create surviving nested RPDB reply: %v", err)
	}
	if err := db.Create(&privateRPDBComment).Error; err != nil {
		t.Fatalf("create private-work RPDB comment: %v", err)
	}
	if err := db.Create(&[]model.RPDBCommentLike{
		{CommentID: otherRPDBComment.ID, UserID: user.ID},
		{CommentID: otherRPDBComment.ID, UserID: otherUser.ID},
		{CommentID: privateRPDBComment.ID, UserID: user.ID},
	}).Error; err != nil {
		t.Fatalf("create RPDB comment likes: %v", err)
	}

	if err := db.Create(&model.UserBlock{BlockerID: user.ID, BlockedUserID: otherUser.ID, Reason: "private"}).Error; err != nil {
		t.Fatalf("create user block: %v", err)
	}
	if err := db.Create(&model.UserHiddenContent{
		UserID: user.ID, TargetType: "rpdb_work", TargetID: otherRPDBWork.ID, Reason: "private",
	}).Error; err != nil {
		t.Fatalf("create hidden RPDB work: %v", err)
	}
	if err := db.Create(&model.ContentReport{
		ReporterID: user.ID, TargetType: "rpdb_work", TargetID: otherRPDBWork.ID,
		TargetUserID: otherUser.ID, Reason: "other", Detail: "private report detail",
	}).Error; err != nil {
		t.Fatalf("create RPDB report: %v", err)
	}
	deletedWorkReport := model.ContentReport{
		ReporterID: otherUser.ID, TargetType: "rpdb_work", TargetID: privateWork.ID,
		TargetUserID: user.ID, Reason: "other", Detail: "report against deleted private work",
	}
	deletedCommentReport := model.ContentReport{
		ReporterID: otherUser.ID, TargetType: "rpdb_comment", TargetID: privateRPDBComment.ID,
		TargetUserID: otherUser.ID, Reason: "other", Detail: "report against deleted private-work comment",
	}
	retainedWorkReport := model.ContentReport{
		ReporterID: otherUser.ID, TargetType: "rpdb_work", TargetID: retainedWork.ID,
		TargetUserID: user.ID, Reason: "other", Detail: "historical report against retained public work",
	}
	deletedCharacterCardReport := model.ContentReport{
		ReporterID: otherUser.ID, TargetType: "character_card", TargetID: characterCard.ID,
		TargetUserID: user.ID, Reason: "other", Detail: "report against deleted character card",
	}
	deletedGuildReport := model.ContentReport{
		ReporterID: otherUser.ID, TargetType: "guild", TargetID: ownedGuild.ID,
		TargetUserID: user.ID, Reason: "other", Detail: "report against deleted guild",
	}
	if err := db.Create(&[]*model.ContentReport{
		&deletedWorkReport,
		&deletedCommentReport,
		&retainedWorkReport,
		&deletedCharacterCardReport,
		&deletedGuildReport,
	}).Error; err != nil {
		t.Fatalf("create other user's RPDB reports: %v", err)
	}

	if err := db.Create(&model.Notification{UserID: user.ID, Type: "system", Content: "hello"}).Error; err != nil {
		t.Fatalf("create notification: %v", err)
	}
	if err := db.Create(&model.UserDailyActivity{UserID: user.ID}).Error; err != nil {
		t.Fatalf("create daily activity: %v", err)
	}
	if err := db.Create(&model.UserActivityLog{UserID: user.ID, Action: "a", ReferenceKey: "b"}).Error; err != nil {
		t.Fatalf("create activity log: %v", err)
	}

	server := newTestServer(t, db)
	portraitPath := writeCharacterCardTestPNG(t, server, fmt.Sprintf("character-cards/%d/portrait/account.png", user.ID))
	iconPath := writeCharacterCardTestPNG(t, server, fmt.Sprintf("character-cards/%d/impression-icon/account.png", user.ID))
	imagePath := writeCharacterCardTestPNG(t, server, fmt.Sprintf("character-cards/%d/impression-image/account.png", user.ID))
	abandonedPendingPath := writeCharacterCardTestPNG(t, server, fmt.Sprintf("character-cards/%d/pending/icon/abandoned.png", user.ID))
	for _, reference := range []string{
		retainedMediaPath,
		contributedMediaPath,
		privateContributionPath,
		privateContributionThumbPath,
		privateContributionMetaPath,
		otherPrivateMediaPath,
		privateCoverPath,
		privateContentPath,
		privateMediaPath,
		privateThumbnailPath,
		privateReferencePath,
		privateGuidePath,
		privateCommentImagePath,
		standaloneDraftPath,
		retainedDraftCoverPath,
		userRevisionUploadPath,
		deletedWorkRevisionUploadPath,
		setCoverPath,
	} {
		key := uploadsKeyFromPath(reference)
		if created := writeCharacterCardTestPNG(t, server, key); created != reference {
			t.Fatalf("unexpected RPDB upload reference: got %q want %q", created, reference)
		}
	}
	if err := db.Model(&model.CharacterCard{}).Where("id = ?", characterCard.ID).Update("portrait_image", portraitPath).Error; err != nil {
		t.Fatalf("set character card portrait: %v", err)
	}
	if err := db.Model(&model.CharacterCardImpression{}).Where("id = ?", cardImpression.ID).Updates(map[string]interface{}{
		"icon_image": iconPath,
		"image":      imagePath,
	}).Error; err != nil {
		t.Fatalf("set impression images: %v", err)
	}
	token := newTestToken(t, user)

	resp := performRequest(server.router, http.MethodDelete, "/api/v1/user/account", map[string]string{
		"password": "secret123",
	}, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var deletedUser model.User
	if err := db.First(&deletedUser, user.ID).Error; err != nil {
		t.Fatalf("reload deleted user: %v", err)
	}
	if deletedUser.AccountDeletedAt == nil {
		t.Fatalf("expected account_deleted_at to be set")
	}
	expectedUsername := fmt.Sprintf("deleted-user-%d", user.ID)
	if deletedUser.Username != expectedUsername {
		t.Fatalf("expected anonymized username, got %q", deletedUser.Username)
	}
	expectedEmail := fmt.Sprintf("deleted+%d@rpbox.invalid", user.ID)
	if deletedUser.Email != expectedEmail {
		t.Fatalf("expected anonymized email, got %q", deletedUser.Email)
	}

	assertCount := func(name string, target interface{}, expected int64, query string, args ...interface{}) {
		t.Helper()
		var count int64
		if err := db.Model(target).Where(query, args...).Count(&count).Error; err != nil {
			t.Fatalf("%s count failed: %v", name, err)
		}
		if count != expected {
			t.Fatalf("%s expected %d, got %d", name, expected, count)
		}
	}

	assertCount("profiles", &model.Profile{}, 0, "user_id = ?", user.ID)
	assertCount("profile_versions", &model.ProfileVersion{}, 0, "profile_id = ?", profile.ID)
	assertCount("account_backups", &model.AccountBackup{}, 0, "user_id = ?", user.ID)
	assertCount("account_backup_versions", &model.AccountBackupVersion{}, 0, "backup_id = ?", backup.ID)
	assertCount("character_cards", &model.CharacterCard{}, 0, "user_id = ?", user.ID)
	assertCount("character_card_impressions", &model.CharacterCardImpression{}, 0, "character_card_id = ?", characterCard.ID)
	assertCount("owned_posts", &model.Post{}, 0, "author_id = ?", user.ID)
	assertCount("owned_items", &model.Item{}, 0, "author_id = ?", user.ID)
	assertCount("stories", &model.Story{}, 0, "user_id = ?", user.ID)
	assertCount("collections", &model.Collection{}, 0, "author_id = ?", user.ID)
	assertCount("owned_guilds", &model.Guild{}, 0, "id = ?", ownedGuild.ID)
	assertCount("notifications", &model.Notification{}, 0, "user_id = ? OR actor_id = ?", user.ID, user.ID)
	assertCount("retained_public_rpdb_work", &model.RPDBWork{}, 1, "id = ? AND author_id = ?", retainedWork.ID, user.ID)
	assertCount("private_rpdb_work", &model.RPDBWork{}, 0, "id = ?", privateWork.ID)
	assertCount("other_private_rpdb_work", &model.RPDBWork{}, 1, "id = ? AND author_id = ?", otherPrivateRPDBWork.ID, otherUser.ID)
	assertCount("authored_rpdb_drafts", &model.RPDBDraft{}, 0, "author_id = ?", user.ID)
	assertCount("other_rpdb_draft", &model.RPDBDraft{}, 1, "id = ?", otherDraft.ID)
	var refreshedOtherPrivateDraft model.RPDBDraft
	if err := db.First(&refreshedOtherPrivateDraft, otherPrivateDraft.ID).Error; err != nil {
		t.Fatalf("reload other user's detached RPDB draft: %v", err)
	}
	if refreshedOtherPrivateDraft.WorkID != nil || refreshedOtherPrivateDraft.BaseVersion != 0 {
		t.Fatalf("expected other user's RPDB draft to survive detached, got %+v", refreshedOtherPrivateDraft)
	}
	assertCount("private_rpdb_references", &model.RPDBReference{}, 0, "work_id = ?", privateWork.ID)
	assertCount("private_rpdb_media", &model.RPDBMedia{}, 0, "work_id = ?", privateWork.ID)
	assertCount("nonpublic_contributed_rpdb_media", &model.RPDBMedia{}, 0, "id = ?", privateContribution.ID)
	assertCount("other_users_private_rpdb_media", &model.RPDBMedia{}, 1, "id = ?", otherPrivateMedia.ID)
	assertCount("private_rpdb_transmog_slots", &model.RPDBTransmogSlot{}, 0, "work_id = ?", privateWork.ID)
	assertCount("private_rpdb_guide_steps", &model.RPDBGuideStep{}, 0, "work_id = ?", privateWork.ID)
	assertCount("authored_rpdb_likes", &model.RPDBLike{}, 0, "user_id = ?", user.ID)
	assertCount("authored_rpdb_favorites", &model.RPDBFavorite{}, 0, "user_id = ?", user.ID)
	assertCount("authored_rpdb_views", &model.RPDBView{}, 0, "user_id = ?", user.ID)
	assertCount("authored_rpdb_view_events", &model.RPDBViewEvent{}, 0, "user_id = ?", user.ID)
	assertCount("authored_rpdb_verifications", &model.RPDBVerification{}, 0, "user_id = ?", user.ID)
	assertCount("authored_rpdb_comments", &model.RPDBComment{}, 0, "author_id = ?", user.ID)
	assertCount("private_rpdb_comments", &model.RPDBComment{}, 0, "work_id = ?", privateWork.ID)
	assertCount("authored_rpdb_comment_likes", &model.RPDBCommentLike{}, 0, "user_id = ?", user.ID)
	assertCount("authored_rpdb_lists", &model.RPDBList{}, 0, "user_id = ?", user.ID)
	assertCount("authored_rpdb_list_entries", &model.RPDBListEntry{}, 0, "list_id = ?", userList.ID)
	assertCount("deleted_work_rpdb_list_entries", &model.RPDBListEntry{}, 0, "work_id = ?", privateWork.ID)
	assertCount("authored_rpdb_sets", &model.RPDBSet{}, 0, "author_id = ?", user.ID)
	assertCount("authored_rpdb_set_entries", &model.RPDBSetWork{}, 0, "set_id = ?", userSet.ID)
	assertCount("deleted_work_rpdb_set_entries", &model.RPDBSetWork{}, 0, "work_id = ?", privateWork.ID)
	assertCount("authored_rpdb_revisions", &model.RPDBRevision{}, 0, "proposer_id = ?", user.ID)
	assertCount("authored_deleted_work_rpdb_revision", &model.RPDBRevision{}, 0, "id = ?", userDeletedWorkRevision.ID)
	assertCount("other_retained_rpdb_revision", &model.RPDBRevision{}, 1, "id = ?", otherRetainedRevision.ID)
	assertCount("deleted_work_rpdb_revision", &model.RPDBRevision{}, 0, "id = ?", otherDeletedRevision.ID)
	assertCount("authored_rpdb_tags", &model.Tag{}, 0, "id = ?", deletedTag.ID)
	assertCount("authored_rpdb_tag_links", &model.RPDBTag{}, 0, "tag_id = ?", deletedTag.ID)
	assertCount("shared_rpdb_tag", &model.Tag{}, 1, "id = ?", sharedTag.ID)
	assertCount("deleted_work_shared_tag_link", &model.RPDBTag{}, 0, "work_id = ? AND tag_id = ?", privateWork.ID, sharedTag.ID)
	assertCount("user_blocks", &model.UserBlock{}, 0, "blocker_id = ? OR blocked_user_id = ?", user.ID, user.ID)
	assertCount("hidden_content", &model.UserHiddenContent{}, 0, "user_id = ?", user.ID)
	assertCount("content_reports", &model.ContentReport{}, 0, "reporter_id = ?", user.ID)
	assertCount("deleted_work_reports", &model.ContentReport{}, 0, "id = ?", deletedWorkReport.ID)
	assertCount("deleted_comment_reports", &model.ContentReport{}, 0, "id = ?", deletedCommentReport.ID)
	assertCount("retained_work_reports", &model.ContentReport{}, 1, "id = ?", retainedWorkReport.ID)
	assertCount("deleted_character_card_reports", &model.ContentReport{}, 0, "id = ?", deletedCharacterCardReport.ID)
	assertCount("deleted_guild_reports", &model.ContentReport{}, 0, "id = ?", deletedGuildReport.ID)
	for _, reference := range []string{portraitPath, iconPath, imagePath, abandonedPendingPath} {
		if _, err := os.Stat(characterCardTestUploadFile(server, reference)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("account deletion left character-card image %q: %v", reference, err)
		}
	}
	for _, reference := range []string{
		privateCoverPath,
		privateContentPath,
		privateMediaPath,
		privateThumbnailPath,
		privateReferencePath,
		privateGuidePath,
		privateCommentImagePath,
		standaloneDraftPath,
		retainedDraftCoverPath,
		privateContributionPath,
		privateContributionThumbPath,
		privateContributionMetaPath,
		userRevisionUploadPath,
		deletedWorkRevisionUploadPath,
		setCoverPath,
	} {
		if _, err := os.Stat(characterCardTestUploadFile(server, reference)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("account deletion left private RPDB upload %q: %v", reference, err)
		}
	}
	for _, reference := range []string{retainedMediaPath, contributedMediaPath, otherPrivateMediaPath} {
		if _, err := os.Stat(characterCardTestUploadFile(server, reference)); err != nil {
			t.Fatalf("account deletion removed retained RPDB upload %q: %v", reference, err)
		}
	}
	var refreshedOtherPrivateWork model.RPDBWork
	if err := db.First(&refreshedOtherPrivateWork, otherPrivateRPDBWork.ID).Error; err != nil {
		t.Fatalf("reload other user's private RPDB work: %v", err)
	}
	if refreshedOtherPrivateWork.MediaCount != 1 {
		t.Fatalf("expected other user's private RPDB work media count 1, got %d", refreshedOtherPrivateWork.MediaCount)
	}
	var refreshedOtherPrivateMedia model.RPDBMedia
	if err := db.First(&refreshedOtherPrivateMedia, otherPrivateMedia.ID).Error; err != nil {
		t.Fatalf("reload other user's private RPDB media: %v", err)
	}
	if refreshedOtherPrivateMedia.AuthorID == nil || *refreshedOtherPrivateMedia.AuthorID != otherUser.ID {
		t.Fatalf("unexpected other user's private RPDB media attribution: %+v", refreshedOtherPrivateMedia.AuthorID)
	}

	var refreshedRetainedWork model.RPDBWork
	if err := db.First(&refreshedRetainedWork, retainedWork.ID).Error; err != nil {
		t.Fatalf("reload retained RPDB work: %v", err)
	}
	if refreshedRetainedWork.AuthorID != user.ID || refreshedRetainedWork.MediaCount != 1 || refreshedRetainedWork.ListCount != 0 {
		t.Fatalf("unexpected retained RPDB work after deletion: %+v", refreshedRetainedWork)
	}
	for name, mediaID := range map[string]uint{
		"retained authored media":    retainedMedia.ID,
		"contributed authored media": contributedMedia.ID,
	} {
		var media model.RPDBMedia
		if err := db.First(&media, mediaID).Error; err != nil {
			t.Fatalf("reload %s: %v", name, err)
		}
		if media.AuthorID != nil {
			t.Fatalf("expected %s author attribution to be cleared, got %d", name, *media.AuthorID)
		}
	}

	var refreshedOtherRPDBWork model.RPDBWork
	if err := db.First(&refreshedOtherRPDBWork, otherRPDBWork.ID).Error; err != nil {
		t.Fatalf("reload other RPDB work: %v", err)
	}
	if refreshedOtherRPDBWork.LikeCount != 1 ||
		refreshedOtherRPDBWork.FavoriteCount != 1 ||
		refreshedOtherRPDBWork.ViewCount != 1 ||
		refreshedOtherRPDBWork.CommentCount != 2 ||
		refreshedOtherRPDBWork.ListCount != 1 ||
		refreshedOtherRPDBWork.MediaCount != 1 ||
		refreshedOtherRPDBWork.VerifiedCount != 0 ||
		refreshedOtherRPDBWork.OutdatedCount != 1 ||
		refreshedOtherRPDBWork.VerificationStatus != model.RPDBVerificationStale {
		t.Fatalf("unexpected RPDB counters after deletion: %+v", refreshedOtherRPDBWork)
	}
	var refreshedOtherRPDBComment model.RPDBComment
	if err := db.First(&refreshedOtherRPDBComment, otherRPDBComment.ID).Error; err != nil {
		t.Fatalf("reload other RPDB comment: %v", err)
	}
	if refreshedOtherRPDBComment.LikeCount != 1 {
		t.Fatalf("expected other RPDB comment like count 1, got %d", refreshedOtherRPDBComment.LikeCount)
	}
	var refreshedRPDBGrandchild model.RPDBComment
	if err := db.First(&refreshedRPDBGrandchild, otherRPDBGrandchild.ID).Error; err != nil {
		t.Fatalf("reload surviving nested RPDB reply: %v", err)
	}
	if refreshedRPDBGrandchild.ParentID == nil || *refreshedRPDBGrandchild.ParentID != otherRPDBComment.ID {
		t.Fatalf("expected surviving RPDB reply to reparent to %d, got %+v", otherRPDBComment.ID, refreshedRPDBGrandchild.ParentID)
	}
	var refreshedOtherList model.RPDBList
	if err := db.First(&refreshedOtherList, otherList.ID).Error; err != nil {
		t.Fatalf("reload other RPDB list: %v", err)
	}
	if refreshedOtherList.ItemCount != 1 {
		t.Fatalf("expected other RPDB list item count 1, got %d", refreshedOtherList.ItemCount)
	}
	var refreshedOtherSet model.RPDBSet
	if err := db.First(&refreshedOtherSet, otherSet.ID).Error; err != nil {
		t.Fatalf("reload other RPDB set: %v", err)
	}
	if refreshedOtherSet.ItemCount != 1 {
		t.Fatalf("expected other RPDB set item count 1, got %d", refreshedOtherSet.ItemCount)
	}

	rpdbResp := performRequest(
		server.router,
		http.MethodGet,
		fmt.Sprintf("/api/v1/rpdb/works/%d", retainedWork.ID),
		nil,
		"",
	)
	if rpdbResp.Code != http.StatusOK {
		t.Fatalf("expected retained RPDB work to remain readable, got %d: %s", rpdbResp.Code, rpdbResp.Body.String())
	}
	var retainedPayload struct {
		Work struct {
			AuthorID     uint   `json:"author_id"`
			AuthorName   string `json:"author_name"`
			AuthorAvatar string `json:"author_avatar"`
		} `json:"work"`
	}
	if err := json.Unmarshal(rpdbResp.Body.Bytes(), &retainedPayload); err != nil {
		t.Fatalf("decode retained RPDB work: %v", err)
	}
	if retainedPayload.Work.AuthorID != user.ID ||
		retainedPayload.Work.AuthorName != expectedUsername ||
		retainedPayload.Work.AuthorAvatar != "" {
		t.Fatalf("retained RPDB work exposed non-anonymized author data: %+v", retainedPayload.Work)
	}

	var refreshedPost model.Post
	if err := db.First(&refreshedPost, otherPost.ID).Error; err != nil {
		t.Fatalf("reload other post: %v", err)
	}
	if refreshedPost.CommentCount != 1 || refreshedPost.LikeCount != 0 || refreshedPost.FavoriteCount != 0 || refreshedPost.ViewCount != 0 {
		t.Fatalf("unexpected post counters after deletion: %+v", refreshedPost)
	}

	var refreshedComment model.Comment
	if err := db.First(&refreshedComment, otherComment.ID).Error; err != nil {
		t.Fatalf("reload other comment: %v", err)
	}
	if refreshedComment.LikeCount != 0 {
		t.Fatalf("expected comment like count 0, got %d", refreshedComment.LikeCount)
	}

	var refreshedItem model.Item
	if err := db.First(&refreshedItem, otherItem.ID).Error; err != nil {
		t.Fatalf("reload other item: %v", err)
	}
	if refreshedItem.RatingCount != 0 || refreshedItem.Rating != 0 || refreshedItem.LikeCount != 0 || refreshedItem.FavoriteCount != 0 || refreshedItem.Downloads != 0 {
		t.Fatalf("unexpected item counters after deletion: %+v", refreshedItem)
	}

	var refreshedGuild model.Guild
	if err := db.First(&refreshedGuild, guild.ID).Error; err != nil {
		t.Fatalf("reload guild: %v", err)
	}
	if refreshedGuild.MemberCount != 1 {
		t.Fatalf("expected guild member_count 1, got %d", refreshedGuild.MemberCount)
	}

	unauthorizedResp := performRequest(server.router, http.MethodGet, "/api/v1/user/info", nil, token)
	if unauthorizedResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after deletion, got %d: %s", unauthorizedResp.Code, unauthorizedResp.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["message"] != "账号已删除" {
		t.Fatalf("unexpected response message: %+v", payload)
	}
}
