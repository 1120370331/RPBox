package testutil

import (
	"testing"

	"github.com/rpbox/server/internal/config"
)

// NewTestConfig returns a minimal config for API tests.
func NewTestConfig(t *testing.T) *config.Config {
	t.Helper()

	return &config.Config{
		Server: config.ServerConfig{
			Port:          "0",
			Mode:          "test",
			MaxBodySizeMB: 10,
		},
		Storage: config.StorageConfig{
			Path: t.TempDir(),
		},
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Expire: 24,
		},
		Redis: config.RedisConfig{
			Host: "127.0.0.1",
			Port: "6379",
			DB:   0,
		},
		SMTP: config.SMTPConfig{
			Host: "localhost",
			Port: 25,
			From: "test@rpbox.local",
		},
		RateLimit: config.RateLimitConfig{
			Global: config.RateLimitSetting{RPS: 0, Burst: 0},
			Auth:   config.RateLimitSetting{RPS: 0, Burst: 0},
			API:    config.RateLimitSetting{RPS: 0, Burst: 0},
		},
	}
}
