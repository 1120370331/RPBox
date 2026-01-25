package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"github.com/rpbox/server/pkg/auth"
	"gorm.io/gorm"
)

func TestHealthEndpoint(t *testing.T) {
	server := newTestServer(t, testutil.NewTestDB(t))

	resp := performRequest(server.router, http.MethodGet, "/health", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestPresetTagsEndpoint(t *testing.T) {
	db := testutil.NewTestDB(t, &model.Tag{})
	database.DB = db
	db.Create(&model.Tag{Name: "tag-a", Type: "preset", Category: "story", IsPublic: true})
	db.Create(&model.Tag{Name: "tag-b", Type: "preset", Category: "item", IsPublic: true})
	db.Create(&model.Tag{Name: "tag-c", Type: "custom", Category: "story", IsPublic: false})

	server := newTestServer(t, db)

	resp := performRequest(server.router, http.MethodGet, "/api/v1/tags/preset", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var payload struct {
		Tags []model.Tag `json:"tags"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Tags) != 2 {
		t.Fatalf("expected 2 preset tags, got %d", len(payload.Tags))
	}
}

func TestProfileVersionFlow(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{}, &model.Profile{}, &model.ProfileVersion{})
	database.DB = db

	user := model.User{Username: "tester", Email: "tester@example.com", PassHash: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	server := newTestServer(t, db)
	token := newTestToken(t, user)

	createReq := map[string]string{
		"id":           "profile-1",
		"profile_name": "First",
		"checksum":     "abc",
	}
	resp := performRequest(server.router, http.MethodPost, "/api/v1/profiles", createReq, token)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}

	updateReq := map[string]string{
		"profile_name": "Second",
		"checksum":     "def",
	}
	resp = performRequest(server.router, http.MethodPut, "/api/v1/profiles/profile-1", updateReq, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	resp = performRequest(server.router, http.MethodGet, "/api/v1/profiles/profile-1/versions", nil, token)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var versions struct {
		Versions []model.ProfileVersion `json:"versions"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &versions); err != nil {
		t.Fatalf("decode versions: %v", err)
	}
	if len(versions.Versions) != 1 {
		t.Fatalf("expected 1 version entry, got %d", len(versions.Versions))
	}
}

func newTestServer(t *testing.T, db *gorm.DB) *Server {
	t.Helper()

	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	cfg := testutil.NewTestConfig(t)
	auth.Init(cfg.JWT.Secret)
	database.DB = db

	return NewServer(cfg)
}

func newTestToken(t *testing.T, user model.User) string {
	t.Helper()

	token, err := auth.GenerateToken(user.ID, user.Username, 1)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}

func performRequest(router http.Handler, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		payload, _ := json.Marshal(body)
		reader = bytes.NewBuffer(payload)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}
