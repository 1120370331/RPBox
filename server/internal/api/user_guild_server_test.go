package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rpbox/server/internal/model"
)

func TestUserGuildServerSummaryPreservesFilters(t *testing.T) {
	db, s, user := newGuildServerFixture(t)
	joinedAt := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	type expectedGuild struct {
		server, role, status string
	}
	want := make(map[uint]expectedGuild)
	for i, tc := range []struct {
		server, status, role string
		visible              bool
	}{
		{"数据库服务器", "approved", "member", true},
		{"", "approved", "admin", true},
		{"待审核服", "pending", "owner", true},
		{"隐藏待审核服", "pending", "member", false},
		{"隐藏拒绝服", "rejected", "admin", false},
	} {
		guild := model.Guild{Name: fmt.Sprintf("Summary Guild %d", i), Server: tc.server, OwnerID: user.ID, Status: tc.status, InviteCode: fmt.Sprintf("summary-%d", i), MemberCount: 3}
		if err := db.Create(&guild).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&model.GuildMember{GuildID: guild.ID, UserID: user.ID, Role: tc.role, JoinedAt: joinedAt}).Error; err != nil {
			t.Fatal(err)
		}
		if tc.visible {
			want[guild.ID] = expectedGuild{tc.server, tc.role, tc.status}
		}
	}
	// Unrelated membership must not appear in this user's summary.
	otherGuild := model.Guild{Name: "Other User Guild", Server: "其他服", OwnerID: user.ID, Status: "approved", InviteCode: "summary-other"}
	if err := db.Create(&otherGuild).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.GuildMember{GuildID: otherGuild.ID, UserID: user.ID + 1, Role: "member", JoinedAt: joinedAt}).Error; err != nil {
		t.Fatal(err)
	}
	resp := performRequest(s.router, http.MethodGet, fmt.Sprintf("/api/v1/users/%d/guilds", user.ID), nil, newTestToken(t, user))
	if resp.Code != http.StatusOK {
		t.Fatalf("summary status=%d body=%s", resp.Code, resp.Body.String())
	}
	var payload struct {
		Guilds []struct {
			ID          uint      `json:"id"`
			Server      *string   `json:"server"`
			Role        string    `json:"role"`
			Status      string    `json:"status"`
			MemberCount int       `json:"member_count"`
			JoinedAt    time.Time `json:"joined_at"`
		} `json:"guilds"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Guilds) != len(want) {
		t.Fatalf("existing visibility filter changed: %s", resp.Body.String())
	}
	for _, guild := range payload.Guilds {
		expected, ok := want[guild.ID]
		if !ok || guild.Server == nil || *guild.Server != expected.server || guild.Role != expected.role || guild.Status != expected.status || guild.MemberCount != 3 || !guild.JoinedAt.Equal(joinedAt) {
			t.Fatalf("summary mismatch: %#v; body=%s", guild, resp.Body.String())
		}
		assertStoredGuildServer(t, db, guild.ID, expected.server)
		delete(want, guild.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing visible guilds: %#v", want)
	}
}
