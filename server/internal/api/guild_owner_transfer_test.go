package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestTransferGuildOwnerUpdatesRolesAndPermissions(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Guild{}, &model.GuildMember{}, &model.AdminActionLog{})
	database.DB = db

	oldOwner := model.User{Username: "old-owner", Email: "old@example.com", EmailVerified: true, PassHash: "hash", Role: "user"}
	newOwner := model.User{Username: "new-owner", Email: "new@example.com", EmailVerified: true, PassHash: "hash", Role: "user"}
	if err := db.Create(&[]*model.User{&oldOwner, &newOwner}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	guild := model.Guild{Name: "Test Guild", OwnerID: oldOwner.ID, MemberCount: 2, InviteCode: "invite", Status: "approved"}
	if err := db.Create(&guild).Error; err != nil {
		t.Fatalf("create guild: %v", err)
	}
	if err := db.Create(&[]model.GuildMember{
		{GuildID: guild.ID, UserID: oldOwner.ID, Role: "owner"},
		{GuildID: guild.ID, UserID: newOwner.ID, Role: "member"},
	}).Error; err != nil {
		t.Fatalf("create members: %v", err)
	}

	server := newTestServer(t, db)
	resp := performRequest(
		server.router,
		http.MethodPut,
		fmt.Sprintf("/api/v1/guilds/%d/owner", guild.ID),
		map[string]uint{"new_owner_id": newOwner.ID},
		newTestToken(t, oldOwner),
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var refreshed model.Guild
	if err := db.First(&refreshed, guild.ID).Error; err != nil {
		t.Fatalf("load guild: %v", err)
	}
	if refreshed.OwnerID != newOwner.ID {
		t.Fatalf("expected owner_id %d, got %d", newOwner.ID, refreshed.OwnerID)
	}

	var oldMember, newMember model.GuildMember
	if err := db.Where("guild_id = ? AND user_id = ?", guild.ID, oldOwner.ID).First(&oldMember).Error; err != nil {
		t.Fatalf("load old owner member: %v", err)
	}
	if oldMember.Role != "admin" {
		t.Fatalf("expected old owner role admin, got %q", oldMember.Role)
	}
	if err := db.Where("guild_id = ? AND user_id = ?", guild.ID, newOwner.ID).First(&newMember).Error; err != nil {
		t.Fatalf("load new owner member: %v", err)
	}
	if newMember.Role != "owner" {
		t.Fatalf("expected new owner role owner, got %q", newMember.Role)
	}

	detailResp := performRequest(
		server.router,
		http.MethodGet,
		fmt.Sprintf("/api/v1/guilds/%d", guild.ID),
		nil,
		newTestToken(t, newOwner),
	)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("expected get guild 200, got %d body=%s", detailResp.Code, detailResp.Body.String())
	}
	var detail struct {
		MyRole string `json:"my_role"`
	}
	if err := json.Unmarshal(detailResp.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.MyRole != "owner" {
		t.Fatalf("expected my_role owner, got %q", detail.MyRole)
	}

	updateResp := performRequest(
		server.router,
		http.MethodPut,
		fmt.Sprintf("/api/v1/guilds/%d", guild.ID),
		map[string]string{"description": "updated by new owner"},
		newTestToken(t, newOwner),
	)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("expected update guild 200, got %d body=%s", updateResp.Code, updateResp.Body.String())
	}
}
