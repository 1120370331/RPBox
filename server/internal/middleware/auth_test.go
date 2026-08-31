package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"github.com/rpbox/server/pkg/auth"
)

func TestJWTAuthQueryTokenIsRestrictedToNotificationWebSocketUpgrade(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.Init("middleware-auth-test-secret")

	db := testutil.NewTestDB(t, &model.User{})
	database.DB = db
	user := model.User{Username: "websocket-user", Email: "websocket@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, err := auth.GenerateToken(user.ID, user.Username, 1)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	router := gin.New()
	protected := router.Group("/api/v1")
	protected.Use(JWTAuth())
	protected.GET("/profile", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	protected.GET("/ws/notifications", func(c *gin.Context) {
		if _, exists := c.GetQuery("token"); exists {
			t.Error("query token remained visible to the WebSocket handler")
		}
		if got := c.Query("client"); got != "desktop" {
			t.Errorf("client query parameter = %q, want desktop", got)
		}
		c.Status(http.StatusNoContent)
	})
	protected.POST("/ws/notifications", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	tests := []struct {
		name         string
		method       string
		path         string
		bearer       string
		upgrade      bool
		partial      bool
		wantStatus   int
		wantRawQuery string
	}{
		{
			name:       "ordinary API rejects query token",
			method:     http.MethodGet,
			path:       "/api/v1/profile?token=" + token,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "same GET path without upgrade rejects query token",
			method:     http.MethodGet,
			path:       "/api/v1/ws/notifications?token=" + token,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "POST upgrade headers still reject query token",
			method:     http.MethodPost,
			path:       "/api/v1/ws/notifications?token=" + token,
			upgrade:    true,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "partial upgrade rejects query token",
			method:     http.MethodGet,
			path:       "/api/v1/ws/notifications?token=" + token,
			partial:    true,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:         "real notification upgrade accepts query token",
			method:       http.MethodGet,
			path:         "/api/v1/ws/notifications?token=" + token + "&client=desktop",
			upgrade:      true,
			wantStatus:   http.StatusNoContent,
			wantRawQuery: "client=desktop",
		},
		{
			name:         "bearer remains valid and query credential is stripped",
			method:       http.MethodGet,
			path:         "/api/v1/profile?token=query-secret&view=summary",
			bearer:       token,
			wantStatus:   http.StatusNoContent,
			wantRawQuery: "view=summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.bearer != "" {
				req.Header.Set("Authorization", "Bearer "+tt.bearer)
			}
			if tt.upgrade {
				req.Header.Set("Connection", "keep-alive, Upgrade")
				req.Header.Set("Upgrade", "websocket")
			} else if tt.partial {
				req.Header.Set("Upgrade", "websocket")
			}

			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, tt.wantStatus, response.Body.String())
			}
			if got := req.URL.RawQuery; got != tt.wantRawQuery {
				t.Fatalf("downstream raw query = %q, want %q", got, tt.wantRawQuery)
			}
		})
	}
}
