package database

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Complete Init requires PostgreSQL-specific migrations. A fresh database is
// created for each scenario on an explicitly supplied local disposable cluster.
func newGuildServerInitDatabase(t *testing.T) (*config.DatabaseConfig, *gorm.DB) {
	t.Helper()
	dsn := os.Getenv("RPBOX_GUILD_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("complete Init verification requires explicit disposable PostgreSQL DSN")
	}
	connection, err := pgconn.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if connection.Host != "127.0.0.1" || connection.User != "guild_test" || connection.Database != "postgres" || connection.Password != "" || connection.Port == 5432 {
		t.Fatal("Init tests require the local disposable guild_test cluster on a nondefault port")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("guild_server_init_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE DATABASE " + name).Error; err != nil {
		adminSQL.Close()
		t.Fatal(err)
	}
	previousDB := DB
	t.Cleanup(func() {
		DB = previousDB
		if err := admin.Exec("DROP DATABASE " + name + " WITH (FORCE)").Error; err != nil {
			t.Errorf("cleanup Init database: %v", err)
		}
		if err := adminSQL.Close(); err != nil {
			t.Errorf("close Init control connection: %v", err)
		}
	})
	// The disposable cluster uses trust authentication. Supply a non-secret test
	// password because Init's existing DSN formatter does not quote empty values.
	cfg := &config.DatabaseConfig{Host: "127.0.0.1", Port: strconv.Itoa(int(connection.Port)), User: "guild_test", Password: "isolated-test-only", DBName: name, SSLMode: "disable"}
	fixtureDSN := fmt.Sprintf("host=127.0.0.1 port=%s user=guild_test dbname=%s sslmode=disable", cfg.Port, name)
	fixture, err := gorm.Open(postgres.Open(fixtureDSN), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	fixtureSQL, err := fixture.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixtureSQL.Close(); err != nil {
			t.Errorf("close fixture database: %v", err)
		}
	})
	return cfg, fixture
}

func runCheckedGuildServerInit(t *testing.T, cfg *config.DatabaseConfig) *gorm.DB {
	t.Helper()
	var captured bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(io.MultiWriter(previousOutput, &captured))
	err := Init(cfg)
	log.SetOutput(previousOutput)
	if err != nil {
		t.Fatalf("complete database.Init failed: %v", err)
	}
	// Existing Init logs some manual/index/security failures instead of returning
	// them; these must also fail this verification rather than produce a false pass.
	if strings.Contains(captured.String(), "[DB ") {
		t.Fatalf("complete Init suppressed a migration failure: %s", captured.String())
	}
	db := DB
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close initialized database: %v", err)
		}
	})
	return db
}

func TestGuildServerCompleteInitPreservesRecords(t *testing.T) {
	for _, scenario := range []string{"legacy column missing", "existing names and null"} {
		t.Run(scenario, func(t *testing.T) {
			cfg, fixture := newGuildServerInitDatabase(t)
			if err := fixture.AutoMigrate(&model.User{}, &model.Guild{}, &model.GuildMember{}); err != nil {
				t.Fatal(err)
			}
			owner := model.User{Username: "init-owner", Email: "init-owner@example.com", EmailVerified: true, PassHash: "preserved-hash"}
			if err := fixture.Create(&owner).Error; err != nil {
				t.Fatal(err)
			}
			guilds := []model.Guild{
				{Name: "Legacy Guild", Server: "", OwnerID: owner.ID, Description: "Preserved description", Status: "approved", InviteCode: "init-legacy"},
				{Name: "Named Guild", Server: "已有服务器", OwnerID: owner.ID, Description: "Named description", Status: "approved", InviteCode: "init-named"},
			}
			if err := fixture.Create(&guilds).Error; err != nil {
				t.Fatal(err)
			}
			// Compare against database values, including PostgreSQL timestamp precision.
			if err := fixture.Order("id").Find(&guilds).Error; err != nil {
				t.Fatal(err)
			}
			for _, guild := range guilds {
				if err := fixture.Create(&model.GuildMember{GuildID: guild.ID, UserID: owner.ID, Role: "owner"}).Error; err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "legacy column missing" {
				if err := fixture.Migrator().DropColumn(&model.Guild{}, "Server"); err != nil {
					t.Fatal(err)
				}
				if fixture.Migrator().HasColumn(&model.Guild{}, "Server") {
					t.Fatal("legacy fixture must lack server column")
				}
				guilds[1].Server = ""
			} else if err := fixture.Model(&model.Guild{}).Where("id = ?", guilds[0].ID).UpdateColumn("server", nil).Error; err != nil {
				t.Fatal(err)
			}
			var tagCount int64
			for pass := 1; pass <= 2; pass++ {
				db := runCheckedGuildServerInit(t, cfg)
				var stored []model.Guild
				if err := db.Order("id").Find(&stored).Error; err != nil {
					t.Fatal(err)
				}
				if len(stored) != len(guilds) {
					t.Fatalf("Init changed guild count: %d", len(stored))
				}
				for i, before := range guilds {
					after := stored[i]
					if after.ID != before.ID || after.Name != before.Name || after.Server != before.Server || after.OwnerID != before.OwnerID || after.Description != before.Description || after.InviteCode != before.InviteCode || after.Status != before.Status || !after.CreatedAt.Equal(before.CreatedAt) || !after.UpdatedAt.Equal(before.UpdatedAt) {
						t.Fatalf("Init pass %d changed guild: before=%#v after=%#v", pass, before, after)
					}
				}
				var members, nullServers int64
				if err := db.Model(&model.GuildMember{}).Count(&members).Error; err != nil || members != 2 {
					t.Fatalf("members=%d err=%v", members, err)
				}
				if err := db.Model(&model.Guild{}).Where("server IS NULL").Count(&nullServers).Error; err != nil || nullServers != 0 {
					t.Fatalf("NULL servers=%d err=%v", nullServers, err)
				}
				var tags int64
				if err := db.Model(&model.Tag{}).Count(&tags).Error; err != nil || tags == 0 || (pass == 2 && tags != tagCount) {
					t.Fatalf("preset tags=%d previous=%d err=%v", tags, tagCount, err)
				}
				tagCount = tags
				for _, table := range []string{"account_backups", "character_cards", "story_entries", "posts", "items", "rpdb_works", "rpdb_sets"} {
					if !db.Migrator().HasTable(table) {
						t.Fatalf("complete Init did not create %s", table)
					}
				}
				var triggerCount, constraintCount, pendingIndexCount int64
				if err := db.Raw("SELECT count(*) FROM pg_trigger WHERE tgrelid = 'guilds'::regclass AND tgname = 'guild_owner_email_verified' AND NOT tgisinternal").Scan(&triggerCount).Error; err != nil || triggerCount != 1 {
					t.Fatalf("owner trigger=%d err=%v", triggerCount, err)
				}
				if err := db.Raw("SELECT count(*) FROM pg_constraint WHERE conrelid = 'guilds'::regclass AND conname = 'guilds_faction_allowed'").Scan(&constraintCount).Error; err != nil || constraintCount != 1 {
					t.Fatalf("faction constraint=%d err=%v", constraintCount, err)
				}
				if err := db.Raw("SELECT count(*) FROM pg_indexes WHERE schemaname = 'public' AND tablename = 'guilds' AND indexname = 'idx_guilds_owner_pending_unique'").Scan(&pendingIndexCount).Error; err != nil || pendingIndexCount != 1 {
					t.Fatalf("pending index=%d err=%v", pendingIndexCount, err)
				}
				var maxLength int
				if err := db.Raw("SELECT character_maximum_length FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'guilds' AND column_name = 'server'").Scan(&maxLength).Error; err != nil || maxLength != 128 {
					t.Fatalf("server length=%d err=%v", maxLength, err)
				}
				var retainedOwner model.User
				if err := db.First(&retainedOwner, owner.ID).Error; err != nil || retainedOwner.PassHash != owner.PassHash || !retainedOwner.EmailVerified {
					t.Fatalf("Init changed owner: %#v err=%v", retainedOwner, err)
				}
				t.Logf("complete Init pass %d preserved 2 guilds, 2 memberships; tags=%d; security trigger/constraint/index present; server varchar(128)", pass, tags)
			}
		})
	}
}
