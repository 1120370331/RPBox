package api

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"github.com/rpbox/server/pkg/auth"
	"gorm.io/gorm"
)

const testNotificationPath = "/api/v1/test/send-notification"

func TestDevEndpointNotificationRouteModeMatrix(t *testing.T) {
	t.Run("release route is absent", func(t *testing.T) {
		db := testutil.NewTestDB(t, &model.User{}, &model.Notification{})
		target := model.User{
			Username: "release-target",
			Email:    "release-target@example.com",
			PassHash: "hash",
			Role:     "user",
		}
		if err := db.Create(&target).Error; err != nil {
			t.Fatalf("create target: %v", err)
		}

		server := newNotificationRouteTestServer(t, db, gin.ReleaseMode)
		resp := performRequest(server.router, http.MethodPost, testNotificationPath, map[string]interface{}{
			"user_id": target.ID,
			"content": "must not be written",
		}, "")

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected release route to be absent with 404, got %d body=%s", resp.Code, resp.Body.String())
		}
		assertNotificationCount(t, db, 0)
	})

	t.Run("debug route requires an administrator", func(t *testing.T) {
		db := testutil.NewTestDB(t, &model.User{}, &model.Notification{})
		admin := model.User{
			Username: "debug-admin",
			Email:    "debug-admin@example.com",
			PassHash: "hash",
			Role:     "admin",
		}
		regularUser := model.User{
			Username: "debug-user",
			Email:    "debug-user@example.com",
			PassHash: "hash",
			Role:     "user",
		}
		target := model.User{
			Username: "debug-target",
			Email:    "debug-target@example.com",
			PassHash: "hash",
			Role:     "user",
		}
		if err := db.Create(&[]*model.User{&admin, &regularUser, &target}).Error; err != nil {
			t.Fatalf("create users: %v", err)
		}

		server := newNotificationRouteTestServer(t, db, gin.DebugMode)
		body := map[string]interface{}{
			"user_id": target.ID,
			"content": "debug notification",
		}

		anonymousResp := performRequest(server.router, http.MethodPost, testNotificationPath, body, "")
		if anonymousResp.Code != http.StatusUnauthorized {
			t.Fatalf("expected anonymous debug request to return 401, got %d body=%s", anonymousResp.Code, anonymousResp.Body.String())
		}

		userResp := performRequest(server.router, http.MethodPost, testNotificationPath, body, newTestToken(t, regularUser))
		if userResp.Code != http.StatusForbidden {
			t.Fatalf("expected non-admin debug request to return 403, got %d body=%s", userResp.Code, userResp.Body.String())
		}
		assertNotificationCount(t, db, 0)

		adminResp := performRequest(server.router, http.MethodPost, testNotificationPath, body, newTestToken(t, admin))
		if adminResp.Code != http.StatusOK {
			t.Fatalf("expected admin debug request to return 200, got %d body=%s", adminResp.Code, adminResp.Body.String())
		}

		assertNotificationCount(t, db, 1)
		var notification model.Notification
		if err := db.First(&notification).Error; err != nil {
			t.Fatalf("load notification: %v", err)
		}
		if notification.UserID != target.ID || notification.Content != "debug notification" || notification.Type != "system" {
			t.Fatalf("unexpected stored notification: %#v", notification)
		}
	})
}

func newNotificationRouteTestServer(t *testing.T, db *gorm.DB, mode string) *Server {
	t.Helper()

	cfg := testutil.NewTestConfig(t)
	cfg.Server.Mode = mode
	previousMode := gin.Mode()
	gin.SetMode(mode)
	t.Cleanup(func() { gin.SetMode(previousMode) })
	auth.Init(cfg.JWT.Secret)
	database.DB = db

	server := &Server{
		cfg:    cfg,
		router: gin.New(),
	}
	server.setupRoutes()
	return server
}

func assertNotificationCount(t *testing.T, db *gorm.DB, expected int64) {
	t.Helper()

	var count int64
	if err := db.Model(&model.Notification{}).Count(&count).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if count != expected {
		t.Fatalf("expected %d notifications, got %d", expected, count)
	}
}
