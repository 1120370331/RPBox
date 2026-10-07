package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// An explicit test DSN enables PostgreSQL verification in a disposable schema.
// No application configuration or production credentials are read.
func newGuildServerDB(t *testing.T) *gorm.DB {
	t.Helper()
	models := []interface{}{&model.User{}, &model.Guild{}, &model.GuildMember{}, &model.Story{}, &model.StoryGuild{}}
	dsn := os.Getenv("RPBOX_GUILD_TEST_POSTGRES_DSN")
	if dsn == "" {
		return testutil.NewTestDB(t, models...)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	schema := fmt.Sprintf("guild_server_api_%d", time.Now().UnixNano())
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Errorf("cleanup test schema: %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	if err := db.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	return db
}

func newGuildServerFixture(t *testing.T) (*gorm.DB, *Server, model.User) {
	t.Helper()
	db := newGuildServerDB(t)
	previousDB := database.DB
	t.Cleanup(func() { database.DB = previousDB })
	s := newTestServer(t, db)
	owner := model.User{Username: "server-owner", Email: "server-owner@example.com", EmailVerified: true, PassHash: "hash", Role: "user"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	return db, s, owner
}

func assertStoredGuildServer(t *testing.T, db *gorm.DB, id uint, want string) model.Guild {
	t.Helper()
	var stored model.Guild
	if err := db.First(&stored, id).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Server != want {
		t.Fatalf("fresh database query server=%q want=%q", stored.Server, want)
	}
	return stored
}

func TestGuildServerPersistence(t *testing.T) {
	db, s, owner := newGuildServerFixture(t)
	token := newTestToken(t, owner)
	created := performRequest(s.router, http.MethodPost, "/api/v1/guilds", map[string]string{"name": "Contract Guild", "server": "  月神 服  "}, token)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var guild model.Guild
	if err := json.Unmarshal(created.Body.Bytes(), &guild); err != nil {
		t.Fatal(err)
	}
	if guild.ID == 0 || guild.Server != "月神 服" {
		t.Fatalf("create response=%s", created.Body.String())
	}
	assertStoredGuildServer(t, db, guild.ID, "月神 服")
	for _, tc := range []struct {
		name   string
		body   map[string]interface{}
		status int
		want   string
	}{
		{"update", map[string]interface{}{"server": " \t 银月 \n"}, http.StatusOK, "银月"},
		{"omitted", map[string]interface{}{"description": "Description"}, http.StatusOK, "银月"},
		{"empty object", map[string]interface{}{}, http.StatusOK, "银月"},
		{"reject oversized", map[string]interface{}{"server": strings.Repeat("服", 129), "name": "Must not save"}, http.StatusBadRequest, "银月"},
		{"reject non-string", map[string]interface{}{"server": 42}, http.StatusBadRequest, "银月"},
		{"explicit clear", map[string]interface{}{"server": ""}, http.StatusOK, ""},
		{"unicode boundary", map[string]interface{}{"server": strings.Repeat("服", 128)}, http.StatusOK, strings.Repeat("服", 128)},
		{"whitespace clear", map[string]interface{}{"server": " \n\t"}, http.StatusOK, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := performRequest(s.router, http.MethodPut, fmt.Sprintf("/api/v1/guilds/%d", guild.ID), tc.body, token)
			if resp.Code != tc.status {
				t.Fatalf("update status=%d body=%s", resp.Code, resp.Body.String())
			}
			stored := assertStoredGuildServer(t, db, guild.ID, tc.want)
			if stored.Name != "Contract Guild" || stored.OwnerID != owner.ID || stored.MemberCount != 1 {
				t.Fatalf("unrelated guild fields changed: %#v", stored)
			}
			if tc.status == http.StatusOK {
				var returned model.Guild
				if err := json.Unmarshal(resp.Body.Bytes(), &returned); err != nil || returned.Server != stored.Server {
					t.Fatalf("response differs from persisted guild: %s err=%v", resp.Body.String(), err)
				}
			}
		})
	}
	// Seed stored state independently of requests to detect projections omitting server.
	if err := db.Model(&model.Guild{}).Where("id = ?", guild.ID).Updates(map[string]interface{}{"server": "持久服", "status": "approved"}).Error; err != nil {
		t.Fatal(err)
	}
	story := model.Story{Title: "Contract Story", UserID: owner.ID}
	if err := db.Create(&story).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.StoryGuild{StoryID: story.ID, GuildID: guild.ID, AddedBy: owner.ID}).Error; err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{fmt.Sprintf("/api/v1/guilds/%d", guild.ID), "/api/v1/guilds", "/api/v1/public/guilds", fmt.Sprintf("/api/v1/stories/%d/guilds", story.ID)} {
		t.Run("read "+path, func(t *testing.T) {
			resp := performRequest(s.router, http.MethodGet, path, nil, token)
			if resp.Code != http.StatusOK {
				t.Fatalf("read status=%d body=%s", resp.Code, resp.Body.String())
			}
			var payload struct {
				Guild  model.Guild   `json:"guild"`
				Guilds []model.Guild `json:"guilds"`
			}
			if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if len(payload.Guilds) > 0 {
				payload.Guild = payload.Guilds[0]
			}
			if payload.Guild.ID != guild.ID || payload.Guild.Server != "持久服" {
				t.Fatalf("read must return persisted server: %s", resp.Body.String())
			}
		})
	}
}

func TestGuildServerCreateOptionalAndRejectOversized(t *testing.T) {
	for _, tc := range []struct {
		name    string
		server  interface{}
		include bool
		status  int
		want    string
	}{
		{"omitted", nil, false, http.StatusCreated, ""},
		{"empty", "", true, http.StatusCreated, ""},
		{"boundary", strings.Repeat("服", 128), true, http.StatusCreated, strings.Repeat("服", 128)},
		{"oversized", strings.Repeat("服", 129), true, http.StatusBadRequest, ""},
		{"wrong type", 42, true, http.StatusBadRequest, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, s, owner := newGuildServerFixture(t)
			body := map[string]interface{}{"name": "Optional"}
			if tc.include {
				body["server"] = tc.server
			}
			resp := performRequest(s.router, http.MethodPost, "/api/v1/guilds", body, newTestToken(t, owner))
			if resp.Code != tc.status {
				t.Fatalf("create status=%d body=%s", resp.Code, resp.Body.String())
			}
			var guilds []model.Guild
			if err := db.Find(&guilds).Error; err != nil {
				t.Fatal(err)
			}
			if tc.status == http.StatusCreated {
				if len(guilds) != 1 || guilds[0].Server != tc.want {
					t.Fatalf("fresh stored guilds=%#v", guilds)
				}
			} else if len(guilds) != 0 {
				t.Fatalf("rejected create persisted guilds=%#v", guilds)
			}
		})
	}
}

func TestGuildServerWriteErrors(t *testing.T) {
	for _, target := range []string{"guild create", "member create", "guild update"} {
		t.Run(target, func(t *testing.T) {
			db, s, owner := newGuildServerFixture(t)
			token := newTestToken(t, owner)
			if target != "guild update" {
				table := "guilds"
				if target == "member create" {
					table = "guild_members"
				}
				if err := db.Callback().Create().Before("gorm:create").Register("reject_guild_server_write", func(tx *gorm.DB) {
					if tx.Statement.Table == table {
						tx.AddError(errors.New("injected write failure"))
					}
				}); err != nil {
					t.Fatal(err)
				}
				resp := performRequest(s.router, http.MethodPost, "/api/v1/guilds", map[string]string{"name": "Fail", "server": "失败服"}, token)
				if resp.Code != http.StatusInternalServerError {
					t.Fatalf("failed create status=%d body=%s", resp.Code, resp.Body.String())
				}
				for _, m := range []interface{}{&model.Guild{}, &model.GuildMember{}} {
					var count int64
					if err := db.Model(m).Count(&count).Error; err != nil || count != 0 {
						t.Fatalf("failed create must rollback: count=%d err=%v", count, err)
					}
				}
				return
			}
			guild := model.Guild{Name: "Preserved", Server: "原服", OwnerID: owner.ID, InviteCode: "error-test"}
			if err := db.Create(&guild).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Callback().Update().Before("gorm:update").Register("reject_guild_server_update", func(tx *gorm.DB) {
				if tx.Statement.Table == "guilds" {
					tx.AddError(errors.New("injected update failure"))
				}
			}); err != nil {
				t.Fatal(err)
			}
			resp := performRequest(s.router, http.MethodPut, fmt.Sprintf("/api/v1/guilds/%d", guild.ID), map[string]string{"server": "新服"}, token)
			if resp.Code != http.StatusInternalServerError {
				t.Fatalf("failed update status=%d body=%s", resp.Code, resp.Body.String())
			}
			assertStoredGuildServer(t, db, guild.ID, "原服")
		})
	}
}

func TestGuildServerOmittedUpdatePreservesLatestStoredValue(t *testing.T) {
	db, s, owner := newGuildServerFixture(t)
	guild := model.Guild{Name: "Concurrent", Server: "原服", OwnerID: owner.ID, InviteCode: "latest-test"}
	if err := db.Create(&guild).Error; err != nil {
		t.Fatal(err)
	}
	// Change the stored name after the handler's initial read, just before its
	// write, to deterministically expose a stale whole-row save.
	if err := db.Callback().Update().Before("gorm:update").Register("change_server_before_omitted_update", func(tx *gorm.DB) {
		if tx.Statement.Table == "guilds" {
			if err := tx.Exec("UPDATE guilds SET server = ? WHERE id = ?", "最新服", guild.ID).Error; err != nil {
				tx.AddError(err)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	resp := performRequest(s.router, http.MethodPut, fmt.Sprintf("/api/v1/guilds/%d", guild.ID), map[string]string{"description": "New description"}, newTestToken(t, owner))
	if resp.Code != http.StatusOK {
		t.Fatalf("omitted update status=%d body=%s", resp.Code, resp.Body.String())
	}
	stored := assertStoredGuildServer(t, db, guild.ID, "最新服")
	if stored.Description != "New description" {
		t.Fatal("requested description was not saved")
	}
}
