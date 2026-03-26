package service

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

const defaultSensitiveKeywordFile = "storage/moderation/sensitive_keywords_cn.txt"

var (
	sensitiveKeywordsOnce sync.Once
	sensitiveKeywords     []string
)

// DetectSensitiveKeywords returns matched keywords after normalization.
func DetectSensitiveKeywords(contents ...string) []string {
	text := normalizeSensitiveText(strings.Join(contents, " "))
	if text == "" {
		return nil
	}

	keywords := getSensitiveKeywords()
	matched := make([]string, 0, 4)
	seen := make(map[string]struct{})
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			if _, ok := seen[kw]; ok {
				continue
			}
			seen[kw] = struct{}{}
			matched = append(matched, kw)
		}
	}
	return matched
}

func getSensitiveKeywords() []string {
	sensitiveKeywordsOnce.Do(func() {
		fileKeywords := []string{}
		for _, path := range keywordFileCandidates() {
			fileKeywords = loadSensitiveKeywords(path)
			if len(fileKeywords) > 0 {
				break
			}
		}

		normalized := make([]string, 0, len(fileKeywords))
		seen := make(map[string]struct{})
		for _, kw := range fileKeywords {
			k := normalizeSensitiveText(kw)
			if !isUsableKeyword(k) {
				continue
			}
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			normalized = append(normalized, k)
		}
		sensitiveKeywords = normalized
	})
	return sensitiveKeywords
}

func loadSensitiveKeywords(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	res := make([]string, 0, 128)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		res = append(res, line)
	}
	return res
}

func keywordFileCandidates() []string {
	envPath := strings.TrimSpace(os.Getenv("RPBOX_SENSITIVE_KEYWORDS_FILE"))
	if envPath != "" {
		return []string{envPath}
	}
	return []string{
		defaultSensitiveKeywordFile,
		filepath.Join("..", "..", defaultSensitiveKeywordFile),
	}
}

func normalizeSensitiveText(text string) string {
	if text == "" {
		return ""
	}
	text = strings.ToLower(strings.TrimSpace(text))
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isUsableKeyword(keyword string) bool {
	runes := []rune(keyword)
	if len(runes) < 2 {
		return false
	}

	hasHan := false
	asciiOnly := true
	for _, r := range runes {
		if unicode.Is(unicode.Han, r) {
			hasHan = true
		}
		if r > unicode.MaxASCII || !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			asciiOnly = false
		}
	}

	// 纯英文/数字关键词容易误伤，长度至少4
	if asciiOnly && len(runes) < 4 {
		return false
	}

	// 既非纯ASCII又不含汉字（例如少量符号组合），长度至少3
	if !hasHan && !asciiOnly && len(runes) < 3 {
		return false
	}

	return true
}
