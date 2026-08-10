package api

import (
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	commentImageUploadSubdir = "comment-images"
	maxCommentImageBytes     = int64(20 << 20)
)

var supportedCommentImageTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/gif":  {},
	"image/webp": {},
}

// uploadCommentImage stores a comment image in OSS. Comment images never
// fall back to local storage because they are user-visible, moderated media.
func (s *Server) uploadCommentImage(c *gin.Context) {
	if !s.ossEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "评论配图服务暂不可用：OSS 未配置"})
		return
	}

	header, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择图片文件"})
		return
	}
	if header.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片文件不能为空"})
		return
	}
	if header.Size > maxCommentImageBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论配图不能超过20MB"})
		return
	}

	file, err := header.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取图片失败"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxCommentImageBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取图片失败"})
		return
	}
	if int64(len(data)) > maxCommentImageBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论配图不能超过20MB"})
		return
	}
	contentType := detectCommentImageContentType(data)
	if contentType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 JPG、PNG、GIF 和 WebP 图片"})
		return
	}

	userID := c.GetUint("userID")
	subdir := path.Join(commentImageUploadSubdir, strconv.FormatUint(uint64(userID), 10))
	imageURL, err := s.saveImageBytes(c, data, contentType, subdir)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "评论配图上传到 OSS 失败"})
		return
	}
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "评论配图上传结果无效"})
		return
	}
	uploadKey := uploadsKeyFromPath(parsedURL.Path)
	if !isValidCommentImageStorageKey(uploadKey, userID) {
		c.JSON(http.StatusBadGateway, gin.H{"error": "评论配图上传结果无效"})
		return
	}
	// Persist and return a same-origin relative path. This makes it impossible
	// for callers to smuggle an external host through forwarded-host headers.
	imageURL = path.Join("/", uploadDirName, uploadKey)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"url": imageURL},
	})
}

func detectCommentImageContentType(data []byte) string {
	contentType := strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0])
	if _, ok := supportedCommentImageTypes[contentType]; ok {
		return contentType
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	return ""
}

// requireValidCommentImageReference enforces that every newly submitted
// comment image came from the authenticated user's OSS-only upload namespace.
func (s *Server) requireValidCommentImageReference(c *gin.Context, imageURL string) bool {
	if strings.TrimSpace(imageURL) == "" {
		return true
	}
	if !s.ossEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "评论配图服务暂不可用：OSS 未配置"})
		return false
	}
	if commentImageUploadKey(imageURL, c.GetUint("userID")) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评论配图必须通过评论图片上传服务上传"})
		return false
	}
	return true
}

func commentImageUploadKey(raw string, userID uint) string {
	value := strings.TrimSpace(raw)
	if value == "" || userID == 0 {
		return ""
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || !strings.HasPrefix(value, "/uploads/") {
		return ""
	}

	key := uploadsKeyFromPath(parsed.Path)
	if !isValidCommentImageStorageKey(key, userID) {
		return ""
	}
	return strings.TrimPrefix(path.Clean("/"+key), "/")
}

func isValidCommentImageStorageKey(key string, expectedUserID uint) bool {
	cleaned := strings.TrimPrefix(path.Clean("/"+key), "/")
	parts := strings.Split(cleaned, "/")
	if len(parts) != 3 || parts[0] != commentImageUploadSubdir {
		return false
	}
	ownerID, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || ownerID == 0 || (expectedUserID != 0 && ownerID != uint64(expectedUserID)) {
		return false
	}
	filename := parts[2]
	if !immutableUploadFilePattern.MatchString(filename) {
		return false
	}
	switch strings.ToLower(path.Ext(filename)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func isOSSOnlyCommentImagePath(rawPath string) bool {
	cleaned := strings.TrimPrefix(path.Clean("/"+rawPath), "/")
	return strings.HasPrefix(cleaned, commentImageUploadSubdir+"/")
}
