package service

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func resetSensitiveKeywordCaches() {
	sensitiveKeywords = nil
	sensitiveKeywordsOnce = sync.Once{}
	strictPoliticalKeywords = nil
	strictPoliticalKeywordsOnce = sync.Once{}
}

func TestDetectSensitiveKeywords(t *testing.T) {
	tmpDir := t.TempDir()
	keywordFile := filepath.Join(tmpDir, "keywords.txt")
	if err := os.WriteFile(keywordFile, []byte("法轮功\n开盒\n民主\n政府\n共产党\n习近平\n打倒共产党\n台独\n"), 0o644); err != nil {
		t.Fatalf("write keyword file: %v", err)
	}
	strictPoliticalFile := filepath.Join(tmpDir, "strict_political.txt")
	if err := os.WriteFile(strictPoliticalFile, []byte("打倒共产党\n台独\n"), 0o644); err != nil {
		t.Fatalf("write strict political keyword file: %v", err)
	}

	t.Setenv("RPBOX_SENSITIVE_KEYWORDS_FILE", keywordFile)
	t.Setenv("RPBOX_SENSITIVE_STRICT_POLITICAL_KEYWORDS_FILE", strictPoliticalFile)
	resetSensitiveKeywordCaches()

	t.Run("match normalized text", func(t *testing.T) {
		got := DetectSensitiveKeywords("这是一起法 轮 功 相关话题")
		if len(got) == 0 {
			t.Fatalf("expected at least one keyword")
		}
	})

	t.Run("filters broad political keywords", func(t *testing.T) {
		got := DetectSensitiveKeywords("我想讨论一下民主、政府和习近平")
		if len(got) != 0 {
			t.Fatalf("expected broad political keywords to be ignored, got %v", got)
		}
	})

	t.Run("keeps strict political keywords", func(t *testing.T) {
		got := DetectSensitiveKeywords("有人公开喊出打倒共产党这样的口号")
		if len(got) == 0 || got[0] != "打倒共产党" {
			t.Fatalf("expected strict political keyword to match, got %v", got)
		}
	})

	t.Run("no match", func(t *testing.T) {
		got := DetectSensitiveKeywords("今天公会活动很开心")
		if len(got) != 0 {
			t.Fatalf("expected no keyword, got %v", got)
		}
	})
}
