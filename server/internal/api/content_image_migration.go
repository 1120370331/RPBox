package api

import "strings"

import "github.com/gin-gonic/gin"

// normalizeAndStoreContentImages converts inline/base64 images in rich text or markdown content into uploaded URLs.
func (s *Server) normalizeAndStoreContentImages(c *gin.Context, content, subdir string) string {
	normalized := content
	if strings.TrimSpace(content) == "" {
		return content
	}

	replaceImageRef := func(raw string) (string, bool) {
		value := strings.TrimSpace(raw)
		if value == "" {
			return raw, false
		}
		migrated, err := s.normalizeAndStoreImageValue(c, value, subdir)
		if err != nil {
			return raw, false
		}
		migrated = strings.TrimSpace(migrated)
		if migrated == "" || migrated == value || !isImageURL(migrated) {
			return raw, false
		}
		return migrated, true
	}

	for _, match := range htmlImgSrcRegexp.FindAllStringSubmatch(normalized, -1) {
		if len(match) < 2 {
			continue
		}
		if migrated, ok := replaceImageRef(match[1]); ok {
			normalized = strings.ReplaceAll(normalized, match[1], migrated)
		}
	}

	for _, match := range mdImgRegexp.FindAllStringSubmatch(normalized, -1) {
		if len(match) < 2 {
			continue
		}
		imageURL := cleanMarkdownImageURL(match[1])
		if imageURL == "" {
			continue
		}
		if migrated, ok := replaceImageRef(imageURL); ok {
			normalized = strings.ReplaceAll(normalized, imageURL, migrated)
		}
	}

	return normalized
}
