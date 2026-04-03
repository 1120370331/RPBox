package service

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

const (
	defaultSensitiveKeywordFile       = "storage/moderation/sensitive_keywords_cn.txt"
	defaultStrictPoliticalKeywordFile = "storage/moderation/sensitive_keywords_political_strict.txt"
)

var (
	sensitiveKeywordsOnce       sync.Once
	sensitiveKeywords           []string
	strictPoliticalKeywordsOnce sync.Once
	strictPoliticalKeywords     map[string]struct{}
)

var politicalKeywordMarkers = []string{
	"政府",
	"中共",
	"共产党",
	"民主",
	"政权",
	"政治",
	"主席",
	"总理",
	"书记",
	"国家",
	"中央",
	"人大",
	"政协",
	"习近平",
	"毛泽东",
	"邓小平",
	"江泽民",
	"胡锦涛",
	"温家宝",
	"李克强",
	"胡耀邦",
	"赵紫阳",
	"天安门",
	"六四",
	"台独",
	"港独",
	"藏独",
	"疆独",
}

var shortSensitiveKeywordAllowlist = map[string]struct{}{
	"开盒":  {},
	"法轮功": {},
	"六四":  {},
	"东突":  {},
	"台独":  {},
	"港独":  {},
	"藏独":  {},
	"疆独":  {},
	"强奸":  {},
	"轮奸":  {},
	"迷奸":  {},
	"嫖娼":  {},
	"卖淫":  {},
	"约炮":  {},
	"吸毒":  {},
	"贩毒":  {},
	"冰毒":  {},
	"毒品":  {},
	"枪支":  {},
	"炸药":  {},
	"爆炸":  {},
}

var asciiSensitiveKeywordAllowlist = map[string]struct{}{
	"fuck":         {},
	"fucking":      {},
	"motherfucker": {},
	"shit":         {},
	"bitch":        {},
	"nigger":       {},
	"terrorist":    {},
	"terrorism":    {},
	"isis":         {},
}

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

		strictPolitical := getStrictPoliticalKeywords()
		normalized := make([]string, 0, len(fileKeywords))
		seen := make(map[string]struct{})
		for _, kw := range fileKeywords {
			k := normalizeSensitiveText(kw)
			if !isUsableKeyword(k) {
				continue
			}
			if shouldSkipPoliticalKeyword(k, strictPolitical) {
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

func getStrictPoliticalKeywords() map[string]struct{} {
	strictPoliticalKeywordsOnce.Do(func() {
		strictPoliticalKeywords = make(map[string]struct{})
		for _, path := range strictPoliticalKeywordFileCandidates() {
			keywords := loadSensitiveKeywords(path)
			if len(keywords) == 0 {
				continue
			}
			for _, kw := range keywords {
				k := normalizeSensitiveText(kw)
				if !isUsableKeyword(k) {
					continue
				}
				strictPoliticalKeywords[k] = struct{}{}
			}
			break
		}
	})
	return strictPoliticalKeywords
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

func strictPoliticalKeywordFileCandidates() []string {
	envPath := strings.TrimSpace(os.Getenv("RPBOX_SENSITIVE_STRICT_POLITICAL_KEYWORDS_FILE"))
	if envPath != "" {
		return []string{envPath}
	}
	return []string{
		defaultStrictPoliticalKeywordFile,
		filepath.Join("..", "..", defaultStrictPoliticalKeywordFile),
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

	// 纯英文关键词误伤极高（例如 test/http），改为白名单制。
	if asciiOnly {
		_, ok := asciiSensitiveKeywordAllowlist[keyword]
		return ok
	}

	// 中文短词（2/3字）误伤非常高，只允许明确高风险词保留。
	if hasHan && len(runes) < 4 {
		_, ok := shortSensitiveKeywordAllowlist[keyword]
		return ok
	}

	// 既非纯ASCII又不含汉字（例如少量符号组合），长度至少3
	if !hasHan && !asciiOnly && len(runes) < 3 {
		return false
	}

	return true
}

func shouldSkipPoliticalKeyword(keyword string, strict map[string]struct{}) bool {
	if len(strict) == 0 || !isLikelyPoliticalKeyword(keyword) {
		return false
	}
	_, ok := strict[keyword]
	return !ok
}

func isLikelyPoliticalKeyword(keyword string) bool {
	for _, marker := range politicalKeywordMarkers {
		if strings.Contains(keyword, marker) {
			return true
		}
	}
	return false
}
