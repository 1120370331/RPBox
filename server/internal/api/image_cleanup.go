package api

import (
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/model"
)

var (
	htmlImgSrcRegexp = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)
	mdImgRegexp      = regexp.MustCompile(`!\[[^\]]*]\(([^)]+)\)`)
)

func (s *Server) cleanupPostImages(c *gin.Context, post model.Post) {
	keys := make(map[string]struct{})
	collectUploadKeysFromValue(c, post.CoverImage, keys)
	collectUploadKeysFromContent(c, post.Content, keys)
	s.deleteUploadKeys(keys)
}

func (s *Server) cleanupItemImages(c *gin.Context, item model.Item, images []model.ItemImage) {
	keys := make(map[string]struct{})
	collectUploadKeysFromValue(c, item.PreviewImage, keys)
	collectUploadKeysFromValue(c, item.Icon, keys)
	collectUploadKeysFromContent(c, item.DetailContent, keys)
	for _, img := range images {
		collectUploadKeysFromValue(c, img.ImageData, keys)
	}
	s.deleteUploadKeys(keys)
}

func (s *Server) deleteUploadKeys(keys map[string]struct{}) {
	for key := range keys {
		s.deleteUploadKey(key)
	}
}

func (s *Server) deleteUploadKey(key string) {
	if key == "" || s == nil {
		return
	}
	if s.ossEnabled() {
		ossKey := s.buildOSSKey(key, "")
		if err := s.deleteFromOSS(ossKey); err != nil {
			log.Printf("[OSS] delete failed: %v", err)
		}
	}
	s.deleteLocalUpload(key)
}

func (s *Server) deleteLocalUpload(key string) {
	if s == nil || s.cfg == nil {
		return
	}
	cleaned := path.Clean("/" + key)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || cleaned == "." {
		return
	}
	baseDir := filepath.Clean(filepath.Join(s.cfg.Storage.Path, uploadDirName))
	targetPath := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(cleaned)))
	if targetPath == baseDir || !strings.HasPrefix(targetPath, baseDir+string(os.PathSeparator)) {
		return
	}
	_ = os.Remove(targetPath)
}

func collectUploadKeysFromContent(c *gin.Context, content string, keys map[string]struct{}) {
	if content == "" {
		return
	}
	for _, match := range htmlImgSrcRegexp.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			collectUploadKeysFromValue(c, match[1], keys)
		}
	}
	for _, match := range mdImgRegexp.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			collectUploadKeysFromValue(c, cleanMarkdownImageURL(match[1]), keys)
		}
	}
}

func collectUploadKeysFromValue(c *gin.Context, value string, keys map[string]struct{}) {
	key := extractUploadKey(c, value)
	if key == "" {
		return
	}
	keys[key] = struct{}{}
}

func extractUploadKey(c *gin.Context, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.HasPrefix(trimmed, "data:") {
		return ""
	}
	if strings.HasPrefix(trimmed, "/uploads/") || strings.HasPrefix(trimmed, "uploads/") {
		return uploadsKeyFromPath(stripURLParams(trimmed))
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		parsed, err := url.Parse(trimmed)
		if err != nil || parsed.Path == "" {
			return ""
		}
		if !isSameHost(c, parsed.Host) {
			return ""
		}
		return uploadsKeyFromPath(parsed.Path)
	}
	return ""
}

func cleanMarkdownImageURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.Trim(trimmed, "<>")
	if idx := strings.IndexAny(trimmed, " \t"); idx != -1 {
		trimmed = trimmed[:idx]
	}
	return strings.Trim(trimmed, "\"'")
}

func stripURLParams(raw string) string {
	if idx := strings.IndexAny(raw, "?#"); idx != -1 {
		return raw[:idx]
	}
	return raw
}
