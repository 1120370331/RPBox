package api

import (
	"net/http"
	"testing"

	"github.com/rpbox/server/internal/config"
)

func TestWebSocketOriginPolicy(t *testing.T) {
	server := &Server{cfg: &config.Config{CORS: config.CORSConfig{
		AllowedOrigins: []string{"https://app.rpbox.test", "http://localhost:1420"},
	}}}

	tests := []struct {
		name    string
		origins []string
		want    bool
	}{
		{name: "trusted production origin", origins: []string{"https://app.rpbox.test"}, want: true},
		{name: "trusted local origin", origins: []string{"http://localhost:1420"}, want: true},
		{name: "native client without origin", want: true},
		{name: "malicious suffix", origins: []string{"https://app.rpbox.test.evil.example"}, want: false},
		{name: "malicious prefix", origins: []string{"https://evil-app.rpbox.test"}, want: false},
		{name: "different scheme", origins: []string{"http://app.rpbox.test"}, want: false},
		{name: "different port", origins: []string{"https://app.rpbox.test:444"}, want: false},
		{name: "duplicate origin headers", origins: []string{"https://app.rpbox.test", "https://evil.example"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "http://server.test/api/v1/ws/notifications", nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}
			for _, origin := range tt.origins {
				req.Header.Add("Origin", origin)
			}

			if got := server.webSocketUpgrader().CheckOrigin(req); got != tt.want {
				t.Fatalf("origin accepted = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebSocketOriginPolicyIsServerScopedAndFailClosed(t *testing.T) {
	first := &Server{cfg: &config.Config{CORS: config.CORSConfig{
		AllowedOrigins: []string{"https://first.rpbox.test"},
	}}}
	second := &Server{cfg: &config.Config{CORS: config.CORSConfig{
		AllowedOrigins: []string{"https://second.rpbox.test"},
	}}}
	empty := &Server{cfg: &config.Config{}}
	wildcard := &Server{cfg: &config.Config{CORS: config.CORSConfig{
		AllowedOrigins: []string{"*"},
	}}}

	requestWithOrigin := func(origin string) *http.Request {
		req, err := http.NewRequest(http.MethodGet, "http://server.test/api/v1/ws/notifications", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		req.Header.Set("Origin", origin)
		return req
	}

	if !first.webSocketUpgrader().CheckOrigin(requestWithOrigin("https://first.rpbox.test")) {
		t.Fatal("first server rejected its configured origin")
	}
	if first.webSocketUpgrader().CheckOrigin(requestWithOrigin("https://second.rpbox.test")) {
		t.Fatal("first server accepted the second server's configured origin")
	}
	if !second.webSocketUpgrader().CheckOrigin(requestWithOrigin("https://second.rpbox.test")) {
		t.Fatal("second server rejected its configured origin")
	}
	if empty.webSocketUpgrader().CheckOrigin(requestWithOrigin("https://any.example")) {
		t.Fatal("empty production allowlist accepted a non-empty Origin")
	}
	if wildcard.webSocketUpgrader().CheckOrigin(requestWithOrigin("https://any.example")) {
		t.Fatal("wildcard allowlist accepted an arbitrary Origin")
	}
	if wildcard.webSocketUpgrader().CheckOrigin(requestWithOrigin("*")) {
		t.Fatal("wildcard allowlist accepted a literal wildcard Origin")
	}
}
