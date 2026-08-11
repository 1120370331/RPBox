package api

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/gorm"
)

func newGuildDeletionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.NewTestDB(
		t,
		&model.User{},
		&model.Guild{},
		&model.GuildMember{},
		&model.GuildApplication{},
		&model.Tag{},
		&model.StoryTag{},
		&model.ItemTag{},
		&model.PostTag{},
		&model.StoryGuild{},
		&model.Post{},
		&model.RPDBWork{},
		&model.RPDBTag{},
		&model.Notification{},
	)
}

func createGuildDeletionFixture(t *testing.T, db *gorm.DB) (model.User, model.User, model.Guild, model.Tag, model.Post) {
	t.Helper()

	owner := model.User{Username: "guild-owner", Email: "guild-owner@example.com", PassHash: "hash", Role: "user"}
	member := model.User{Username: "guild-member", Email: "guild-member@example.com", PassHash: "hash", Role: "user"}
	if err := db.Create(&[]*model.User{&owner, &member}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	guild := model.Guild{Name: "Dissolve Me", OwnerID: owner.ID, InviteCode: "dissolve-me", Status: "approved", MemberCount: 2}
	if err := db.Create(&guild).Error; err != nil {
		t.Fatalf("create guild: %v", err)
	}
	if err := db.Create(&[]model.GuildMember{
		{GuildID: guild.ID, UserID: owner.ID, Role: "owner"},
		{GuildID: guild.ID, UserID: member.ID, Role: "member"},
	}).Error; err != nil {
		t.Fatalf("create guild members: %v", err)
	}
	if err := db.Create(&model.GuildApplication{GuildID: guild.ID, UserID: member.ID, Status: "pending"}).Error; err != nil {
		t.Fatalf("create guild application: %v", err)
	}

	guildID := guild.ID
	tag := model.Tag{Name: "Guild Tag", Type: "guild", GuildID: &guildID, CreatorID: owner.ID}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("create guild tag: %v", err)
	}
	if err := db.Create(&model.StoryTag{StoryID: 1001, TagID: tag.ID, AddedBy: owner.ID}).Error; err != nil {
		t.Fatalf("create story tag: %v", err)
	}
	if err := db.Create(&model.ItemTag{ItemID: 1001, TagID: tag.ID, AddedBy: owner.ID}).Error; err != nil {
		t.Fatalf("create item tag: %v", err)
	}
	if err := db.Create(&model.StoryGuild{StoryID: 1001, GuildID: guild.ID, AddedBy: owner.ID}).Error; err != nil {
		t.Fatalf("create story guild: %v", err)
	}

	post := model.Post{
		AuthorID: owner.ID,
		Title:    "Guild post that must survive",
		Content:  "body",
		GuildID:  &guildID,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create guild post: %v", err)
	}
	if err := db.Create(&model.PostTag{PostID: post.ID, TagID: tag.ID, AddedBy: owner.ID}).Error; err != nil {
		t.Fatalf("create post tag: %v", err)
	}

	remainingGuildID := guild.ID + 1000
	soleGuildWork := model.RPDBWork{
		AuthorID: owner.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "Sole guild work",
		Visibility: model.RPDBVisibilityGuild, GuildID: &guild.ID, GuildIDs: []uint{guild.ID},
	}
	multiGuildWork := model.RPDBWork{
		AuthorID: owner.ID, Type: model.RPDBWorkTypeItemShowcase, Title: "Multi guild work",
		Visibility: model.RPDBVisibilityGuild, GuildID: &remainingGuildID, GuildIDs: []uint{remainingGuildID, guild.ID},
	}
	if err := db.Create(&[]*model.RPDBWork{&soleGuildWork, &multiGuildWork}).Error; err != nil {
		t.Fatalf("create guild RPDB works: %v", err)
	}
	if err := db.Create(&model.RPDBTag{WorkID: soleGuildWork.ID, TagID: tag.ID, AddedBy: owner.ID}).Error; err != nil {
		t.Fatalf("create RPDB tag: %v", err)
	}
	if err := db.Create(&model.Notification{
		UserID: member.ID, Type: "guild_invite", TargetType: "guild", TargetID: guild.ID, Content: "guild notification",
	}).Error; err != nil {
		t.Fatalf("create guild notification: %v", err)
	}

	return owner, member, guild, tag, post
}

func assertRecordCount(t *testing.T, db *gorm.DB, value interface{}, query string, args []interface{}, want int64) {
	t.Helper()
	var got int64
	if err := db.Model(value).Where(query, args...).Count(&got).Error; err != nil {
		t.Fatalf("count %T: %v", value, err)
	}
	if got != want {
		t.Fatalf("expected %d %T records, got %d", want, value, got)
	}
}

func TestDeleteGuildCleansAssociationsAndPreservesPosts(t *testing.T) {
	db := newGuildDeletionTestDB(t)
	database.DB = db
	owner, _, guild, tag, post := createGuildDeletionFixture(t, db)

	server := newTestServer(t, db)
	resp := performRequest(
		server.router,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/guilds/%d", guild.ID),
		nil,
		newTestToken(t, owner),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	assertRecordCount(t, db, &model.Guild{}, "id = ?", []interface{}{guild.ID}, 0)
	assertRecordCount(t, db, &model.GuildMember{}, "guild_id = ?", []interface{}{guild.ID}, 0)
	assertRecordCount(t, db, &model.GuildApplication{}, "guild_id = ?", []interface{}{guild.ID}, 0)
	assertRecordCount(t, db, &model.Tag{}, "guild_id = ?", []interface{}{guild.ID}, 0)
	assertRecordCount(t, db, &model.StoryTag{}, "tag_id = ?", []interface{}{tag.ID}, 0)
	assertRecordCount(t, db, &model.ItemTag{}, "tag_id = ?", []interface{}{tag.ID}, 0)
	assertRecordCount(t, db, &model.PostTag{}, "tag_id = ?", []interface{}{tag.ID}, 0)
	assertRecordCount(t, db, &model.RPDBTag{}, "tag_id = ?", []interface{}{tag.ID}, 0)
	assertRecordCount(t, db, &model.StoryGuild{}, "guild_id = ?", []interface{}{guild.ID}, 0)
	assertRecordCount(t, db, &model.Notification{}, "target_type = ? AND target_id = ?", []interface{}{"guild", guild.ID}, 0)

	var preservedPost model.Post
	if err := db.First(&preservedPost, post.ID).Error; err != nil {
		t.Fatalf("load preserved post: %v", err)
	}
	if preservedPost.GuildID != nil {
		t.Fatalf("expected preserved post guild_id to be nil, got %d", *preservedPost.GuildID)
	}

	var soleGuildWork model.RPDBWork
	if err := db.Where("title = ?", "Sole guild work").First(&soleGuildWork).Error; err != nil {
		t.Fatalf("load sole guild work: %v", err)
	}
	if soleGuildWork.Visibility != model.RPDBVisibilityPrivate || soleGuildWork.GuildID != nil || len(soleGuildWork.GuildIDs) != 0 {
		t.Fatalf("expected sole guild work to become private, got visibility=%q guild_id=%v guild_ids=%v", soleGuildWork.Visibility, soleGuildWork.GuildID, soleGuildWork.GuildIDs)
	}

	var multiGuildWork model.RPDBWork
	if err := db.Where("title = ?", "Multi guild work").First(&multiGuildWork).Error; err != nil {
		t.Fatalf("load multi guild work: %v", err)
	}
	if multiGuildWork.Visibility != model.RPDBVisibilityGuild || multiGuildWork.GuildID == nil || *multiGuildWork.GuildID == guild.ID || len(multiGuildWork.GuildIDs) != 1 {
		t.Fatalf("expected multi-guild work to retain only its surviving guild, got visibility=%q guild_id=%v guild_ids=%v", multiGuildWork.Visibility, multiGuildWork.GuildID, multiGuildWork.GuildIDs)
	}
}

func TestDeleteGuildRejectsNonOwner(t *testing.T) {
	db := newGuildDeletionTestDB(t)
	database.DB = db
	_, member, guild, _, _ := createGuildDeletionFixture(t, db)

	server := newTestServer(t, db)
	resp := performRequest(
		server.router,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/guilds/%d", guild.ID),
		nil,
		newTestToken(t, member),
	)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", resp.Code, resp.Body.String())
	}
	assertRecordCount(t, db, &model.Guild{}, "id = ?", []interface{}{guild.ID}, 1)
}

func TestDeleteGuildRollsBackWhenDeletionFails(t *testing.T) {
	db := newGuildDeletionTestDB(t)
	database.DB = db
	owner, _, guild, tag, post := createGuildDeletionFixture(t, db)

	if err := db.Exec(`
		CREATE TRIGGER fail_guild_delete
		BEFORE DELETE ON guilds
		BEGIN
			SELECT RAISE(ABORT, 'forced guild delete failure');
		END;
	`).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	server := newTestServer(t, db)
	resp := performRequest(
		server.router,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/guilds/%d", guild.ID),
		nil,
		newTestToken(t, owner),
	)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", resp.Code, resp.Body.String())
	}

	assertRecordCount(t, db, &model.Guild{}, "id = ?", []interface{}{guild.ID}, 1)
	assertRecordCount(t, db, &model.GuildMember{}, "guild_id = ?", []interface{}{guild.ID}, 2)
	assertRecordCount(t, db, &model.GuildApplication{}, "guild_id = ?", []interface{}{guild.ID}, 1)
	assertRecordCount(t, db, &model.Tag{}, "guild_id = ?", []interface{}{guild.ID}, 1)
	assertRecordCount(t, db, &model.StoryTag{}, "tag_id = ?", []interface{}{tag.ID}, 1)
	assertRecordCount(t, db, &model.ItemTag{}, "tag_id = ?", []interface{}{tag.ID}, 1)
	assertRecordCount(t, db, &model.PostTag{}, "tag_id = ?", []interface{}{tag.ID}, 1)
	assertRecordCount(t, db, &model.RPDBTag{}, "tag_id = ?", []interface{}{tag.ID}, 1)
	assertRecordCount(t, db, &model.StoryGuild{}, "guild_id = ?", []interface{}{guild.ID}, 1)
	assertRecordCount(t, db, &model.Notification{}, "target_type = ? AND target_id = ?", []interface{}{"guild", guild.ID}, 1)

	var rolledBackPost model.Post
	if err := db.First(&rolledBackPost, post.ID).Error; err != nil {
		t.Fatalf("load rolled back post: %v", err)
	}
	if rolledBackPost.GuildID == nil || *rolledBackPost.GuildID != guild.ID {
		t.Fatalf("expected post to remain linked to guild %d, got %#v", guild.ID, rolledBackPost.GuildID)
	}
}
