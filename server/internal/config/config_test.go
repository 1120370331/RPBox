package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadValidatesReleaseJWTSecret(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		secret      string
		wantErr     bool
		wantErrText string
	}{
		{name: "release rejects empty", mode: "release", secret: "", wantErr: true, wantErrText: "at least 32 UTF-8 bytes"},
		{name: "release rejects 31 bytes", mode: "release", secret: strings.Repeat("a", 31), wantErr: true, wantErrText: "at least 32 UTF-8 bytes"},
		{name: "release accepts 32 bytes", mode: "release", secret: strings.Repeat("a", 32)},
		{name: "release rejects repository example", mode: "release", secret: "your-secret-key-change-in-production", wantErr: true, wantErrText: "must not use an example"},
		{name: "release rejects padded test value", mode: "release", secret: strings.Repeat("test", 8), wantErr: true, wantErrText: "must not use an example"},
		{name: "release rejects whitespace padded short value", mode: "release", secret: "short" + strings.Repeat(" ", 32), wantErr: true, wantErrText: "at least 32 UTF-8 bytes"},
		{name: "debug permits existing short values", mode: "debug", secret: "test-secret"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := loadJWTTestConfig(t, test.mode, test.secret, test.secret)
			if test.wantErr {
				if err == nil {
					t.Fatal("Load() error = nil, want validation error")
				}
				if !strings.Contains(err.Error(), "jwt.secret") || !strings.Contains(err.Error(), test.wantErrText) {
					t.Fatalf("Load() error = %q, want jwt.secret requirement", err)
				}
				if test.secret != "" && strings.Contains(err.Error(), test.secret) {
					t.Fatal("Load() error disclosed the configured jwt.secret")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if cfg.JWT.Secret != test.secret {
				t.Fatal("Load() returned an unexpected jwt.secret")
			}
		})
	}
}

func TestLoadJWTSecretEnvironmentOverride(t *testing.T) {
	environmentSecret := strings.Repeat("e", 32)
	cfg, err := loadJWTTestConfig(t, "release", "your-secret-key-change-in-production", environmentSecret)
	if err != nil {
		t.Fatalf("Load() with JWT_SECRET override: %v", err)
	}
	if cfg.JWT.Secret != environmentSecret {
		t.Fatal("JWT_SECRET did not override jwt.secret")
	}
}

func loadJWTTestConfig(t *testing.T, mode, configSecret, environmentSecret string) (*Config, error) {
	t.Helper()

	originalWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	originalGlobal := globalConfig
	temporaryDir := t.TempDir()
	viper.Reset()
	defer func() {
		_ = os.Chdir(originalWorkingDir)
		globalConfig = originalGlobal
		viper.Reset()
	}()

	t.Setenv("SERVER_MODE", mode)
	t.Setenv("JWT_SECRET", environmentSecret)
	configYAML := []byte("server:\n  mode: " + mode + "\njwt:\n  secret: \"" + configSecret + "\"\n")
	if err := os.WriteFile(filepath.Join(temporaryDir, "config.yaml"), configYAML, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.Chdir(temporaryDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	return Load()
}

func TestLoadTrustedProxies(t *testing.T) {
	originalWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	originalGlobal := globalConfig
	temporaryDir := t.TempDir()
	t.Cleanup(func() {
		_ = os.Chdir(originalWorkingDir)
		globalConfig = originalGlobal
		viper.Reset()
	})

	configYAML := []byte("server:\n  trusted_proxies:\n    - 127.0.0.1/32\n    - 10.0.0.0/8\n")
	if err := os.WriteFile(filepath.Join(temporaryDir, "config.yaml"), configYAML, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.Chdir(temporaryDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	viper.Reset()
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	want := []string{"127.0.0.1/32", "10.0.0.0/8"}
	if !reflect.DeepEqual(cfg.Server.TrustedProxies, want) {
		t.Fatalf("trusted proxies = %#v, want %#v", cfg.Server.TrustedProxies, want)
	}
}
