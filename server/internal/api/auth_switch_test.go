package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"github.com/rpbox/server/pkg/auth"
)

func TestSwitchLoginUsesSixtyDayDeviceToken(t *testing.T) {
	db := testutil.NewTestDB(t, &model.User{})

	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := model.User{Username: "alice", Email: "alice@example.com", PassHash: hash}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	server := newTestServer(t, db)
	loginResp := performRequest(server.router, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": "alice",
		"password": "secret123",
	}, "")
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d body=%s", loginResp.Code, loginResp.Body.String())
	}

	var loginPayload struct {
		Token                string `json:"token"`
		SwitchToken          string `json:"switch_token"`
		SwitchTokenExpiresAt string `json:"switch_token_expires_at"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginPayload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginPayload.Token == "" {
		t.Fatalf("expected login token")
	}
	if loginPayload.SwitchToken == "" {
		t.Fatalf("expected switch token")
	}
	if loginPayload.SwitchTokenExpiresAt == "" {
		t.Fatalf("expected switch token expiry")
	}

	switchResp := performRequest(server.router, http.MethodPost, "/api/v1/auth/switch", map[string]string{
		"switch_token": loginPayload.SwitchToken,
	}, "")
	if switchResp.Code != http.StatusOK {
		t.Fatalf("expected switch 200, got %d body=%s", switchResp.Code, switchResp.Body.String())
	}

	var switchPayload struct {
		Token       string `json:"token"`
		SwitchToken string `json:"switch_token"`
		User        struct {
			ID       uint   `json:"id"`
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.Unmarshal(switchResp.Body.Bytes(), &switchPayload); err != nil {
		t.Fatalf("decode switch response: %v", err)
	}
	if switchPayload.Token == "" || switchPayload.SwitchToken == "" {
		t.Fatalf("expected renewed auth and switch tokens")
	}
	if switchPayload.User.ID != user.ID || switchPayload.User.Username != user.Username {
		t.Fatalf("unexpected switch user: %+v", switchPayload.User)
	}
}
