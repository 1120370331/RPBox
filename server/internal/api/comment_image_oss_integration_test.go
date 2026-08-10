package api

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rpbox/server/internal/config"
	"github.com/spf13/viper"
)

// TestCommentImageOSSIntegration is an opt-in smoke test against the configured
// real OSS bucket. It uploads, reads, byte-compares, and deletes a tiny GIF.
//
// Run with:
//
//	RPBOX_RUN_OSS_INTEGRATION=1 go test ./internal/api -run TestCommentImageOSSIntegration -count=1
func TestCommentImageOSSIntegration(t *testing.T) {
	if os.Getenv("RPBOX_RUN_OSS_INTEGRATION") != "1" {
		t.Skip("set RPBOX_RUN_OSS_INTEGRATION=1 to run the real OSS smoke test")
	}

	ossConfig := loadCommentImageIntegrationOSSConfig(t)
	server := &Server{cfg: &config.Config{OSS: ossConfig}}
	if !server.ossEnabled() {
		t.Fatal("real OSS smoke test requires a complete enabled OSS configuration")
	}

	randomName, err := randomHex(16)
	if err != nil {
		t.Fatalf("generate integration object name: %v", err)
	}
	relativeKey := path.Join(commentImageUploadSubdir, "integration-tests", randomName+".gif")
	objectKey := server.buildOSSKey(relativeKey, "")
	uploaded := false
	defer func() {
		if uploaded {
			if cleanupErr := server.deleteFromOSS(objectKey); cleanupErr != nil {
				t.Errorf("clean up real OSS smoke object: %v", cleanupErr)
			}
		}
	}()

	if err := server.uploadToOSS(objectKey, minimalGIF, "image/gif"); err != nil {
		t.Fatalf("upload GIF to real OSS: %v", err)
	}
	uploaded = true

	data, contentType, err := server.readImageFromOSS(objectKey)
	if err != nil {
		t.Fatalf("read GIF from real OSS: %v", err)
	}
	if !bytes.Equal(data, minimalGIF) {
		t.Fatal("real OSS returned different GIF bytes")
	}
	if strings.TrimSpace(strings.Split(contentType, ";")[0]) != "image/gif" {
		t.Fatalf("real OSS returned unexpected content type %q", contentType)
	}

	if err := server.deleteFromOSS(objectKey); err != nil {
		t.Fatalf("delete GIF from real OSS: %v", err)
	}
	uploaded = false
	if _, _, err := server.readImageFromOSS(objectKey); err == nil {
		t.Fatal("real OSS object still exists after deletion")
	}
}

func loadCommentImageIntegrationOSSConfig(t *testing.T) config.OSSConfig {
	t.Helper()

	if endpoint := strings.TrimSpace(os.Getenv("RPBOX_OSS_ENDPOINT")); endpoint != "" {
		return config.OSSConfig{
			Enabled:         true,
			Endpoint:        endpoint,
			Bucket:          strings.TrimSpace(os.Getenv("RPBOX_OSS_BUCKET")),
			AccessKeyID:     strings.TrimSpace(os.Getenv("RPBOX_OSS_ACCESS_KEY_ID")),
			AccessKeySecret: strings.TrimSpace(os.Getenv("RPBOX_OSS_ACCESS_KEY_SECRET")),
			Prefix:          strings.Trim(strings.TrimSpace(os.Getenv("RPBOX_OSS_PREFIX")), "/"),
		}
	}

	configFile := strings.TrimSpace(os.Getenv("RPBOX_OSS_CONFIG_FILE"))
	if configFile == "" {
		configFile = filepath.Join("..", "..", "config.local.yaml")
	}
	v := viper.New()
	v.SetConfigFile(configFile)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("read OSS integration config %s: %v", configFile, err)
	}
	var ossConfig config.OSSConfig
	if err := v.UnmarshalKey("oss", &ossConfig); err != nil {
		t.Fatalf("decode OSS integration config %s: %v", configFile, err)
	}
	return ossConfig
}
