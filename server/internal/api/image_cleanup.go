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
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
)

var (
	htmlImgSrcRegexp = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)
	mdImgRegexp      = regexp.MustCompile(`!\[[^\]]*]\(([^)]+)\)`)
)

func (s *Server) cleanupPostImages(c *gin.Context, post model.Post) {
	keys := make(map[string]struct{})
	collectUploadKeysFromValue(c, post.CoverImage, keys)
	collectUploadKeysFromContent(c, post.Content, keys)
	// 帖子正文图片按用户目录共享；删除草稿/帖子时必须先确认无其它帖子仍引用，否则会清掉已发布内容的图片。
	s.deleteUnreferencedUploadKeys(c, keys, post.ID)
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

// cleanupCommentImageURLs removes OSS-backed comment images after their
// database records have been deleted. A single upload can be referenced by
// more than one comment, so deletion is conservative across all comment
// tables.
func (s *Server) cleanupCommentImageURLs(c *gin.Context, imageURLs ...string) {
	keys := make(map[string]struct{})
	for _, imageURL := range imageURLs {
		key := extractCommentImageStorageKey(c, imageURL)
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
	}

	for key := range keys {
		if isCommentImageKeyReferenced(key) {
			continue
		}
		s.deleteUploadKey(key)
	}
}

func extractCommentImageStorageKey(c *gin.Context, raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() && c != nil && !isSameHost(c, parsed.Host) {
		return ""
	}
	key := uploadsKeyFromPath(parsed.Path)
	if !isValidCommentImageStorageKey(key, 0) {
		return ""
	}
	return strings.TrimPrefix(path.Clean("/"+key), "/")
}

func isCommentImageKeyReferenced(key string) bool {
	if key == "" || database.DB == nil {
		return false
	}
	patterns := uploadKeyMatchPatterns(key)
	if len(patterns) == 0 {
		return false
	}

	for _, target := range []interface{}{&model.Comment{}, &model.ItemComment{}, &model.RPDBComment{}} {
		query := database.DB.Model(target)
		orParts := make([]string, 0, len(patterns))
		args := make([]interface{}, 0, len(patterns))
		for _, pattern := range patterns {
			orParts = append(orParts, "image_url LIKE ?")
			args = append(args, pattern)
		}

		var count int64
		if err := query.Where("("+strings.Join(orParts, " OR ")+")", args...).Limit(1).Count(&count).Error; err != nil {
			// Query failures must never turn into destructive cleanup.
			log.Printf("[comment-image-cleanup] reference check failed for %s: %v", key, err)
			return true
		}
		if count > 0 {
			return true
		}
	}
	return false
}

func loadCommentImageURLs(db *gorm.DB, target interface{}, condition string, args ...interface{}) ([]string, error) {
	if db == nil {
		return nil, nil
	}
	var imageURLs []string
	err := db.Model(target).
		Where(condition, args...).
		Where("COALESCE(TRIM(image_url), '') <> ''").
		Pluck("image_url", &imageURLs).Error
	return imageURLs, err
}

// deleteUnreferencedUploadKeys only deletes keys that are not still referenced by other posts.
func (s *Server) deleteUnreferencedUploadKeys(c *gin.Context, keys map[string]struct{}, excludePostID uint) {
	for key := range keys {
		if isUploadKeyReferencedByOtherPosts(c, key, excludePostID) {
			continue
		}
		s.deleteUploadKey(key)
	}
}

func isUploadKeyReferencedByOtherPosts(c *gin.Context, key string, excludePostID uint) bool {
	if key == "" || database.DB == nil {
		return false
	}
	patterns := uploadKeyMatchPatterns(key)
	if len(patterns) == 0 {
		return false
	}

	query := database.DB.Model(&model.Post{})
	if excludePostID != 0 {
		query = query.Where("id <> ?", excludePostID)
	}

	orParts := make([]string, 0, len(patterns)*2)
	args := make([]interface{}, 0, len(patterns)*2)
	for _, pattern := range patterns {
		orParts = append(orParts, "cover_image LIKE ?", "content LIKE ?")
		args = append(args, pattern, pattern)
	}
	var count int64
	if err := query.Where("("+strings.Join(orParts, " OR ")+")", args...).Limit(1).Count(&count).Error; err != nil {
		// 查询失败时保守不删，避免误清线上图片
		log.Printf("[image-cleanup] reference check failed for %s: %v", key, err)
		return true
	}
	return count > 0
}

func uploadKeyMatchPatterns(key string) []string {
	cleaned := strings.TrimSpace(strings.TrimPrefix(key, "/"))
	if cleaned == "" {
		return nil
	}
	base := cleaned
	if !strings.HasPrefix(base, "uploads/") {
		base = path.Join("uploads", cleaned)
	}
	// 兼容相对路径、绝对路径、完整 URL 中的引用
	return []string{
		"%" + base + "%",
		"%/" + strings.TrimPrefix(base, "uploads/") + "%",
	}
}

func (s *Server) deleteUploadKey(key string) {
	if key == "" || s == nil {
		return
	}
	if isOSSOnlyCommentImagePath(key) && isCommentImageKeyReferenced(key) {
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
