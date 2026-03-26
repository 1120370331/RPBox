package service

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestDetectSensitiveKeywords(t *testing.T) {
	tmpDir := t.TempDir()
	keywordFile := filepath.Join(tmpDir, "keywords.txt")
	if err := os.WriteFile(keywordFile, []byte("法轮功\n开盒\n"), 0o644); err != nil {
		t.Fatalf("write keyword file: %v", err)
	}
	t.Setenv("RPBOX_SENSITIVE_KEYWORDS_FILE", keywordFile)
	sensitiveKeywords = nil
	sensitiveKeywordsOnce = sync.Once{}

	t.Run("match normalized text", func(t *testing.T) {
		got := DetectSensitiveKeywords("这是一起法 轮 功 相关话题")
		if len(got) == 0 {
			t.Fatalf("expected at least one keyword")
		}
	})

	t.Run("no match", func(t *testing.T) {
		got := DetectSensitiveKeywords("今天公会活动很开心")
		if len(got) != 0 {
			t.Fatalf("expected no keyword, got %v", got)
		}
	})
}
