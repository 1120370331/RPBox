package api

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"mime"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var immutableUploadFilePattern = regexp.MustCompile(`^[0-9a-f]{32}\.[a-z0-9]+$`)

// getUploadObject serves uploaded objects from local storage or OSS.
func (s *Server) getUploadObject(c *gin.Context) {
	rawPath := strings.TrimPrefix(c.Param("filepath"), "/")
	if rawPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	cleanedPath := strings.ReplaceAll(rawPath, `\`, "/")
	cleanedPath = strings.TrimPrefix(path.Clean("/"+cleanedPath), "/")
	// Character-card portraits are served only by the permission-aware image
	// endpoint. Never expose their backing objects through the generic upload
	// route, even when the card itself is public.
	if strings.HasPrefix(strings.ToLower(cleanedPath), "character-cards/") {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	var (
		data        []byte
		contentType string
		err         error
	)

	ossOnlyCommentImage := isOSSOnlyCommentImagePath(rawPath)
	if ossOnlyCommentImage && !s.ossEnabled() {
		err = fmt.Errorf("comment image OSS is unavailable")
	} else if s.ossEnabled() {
		ossKey := s.buildOSSKey(rawPath, "")
		data, contentType, err = s.readImageFromOSS(ossKey)
		if err != nil && !ossOnlyCommentImage {
			data, contentType, err = s.readImageFromLocalPath(path.Join("/uploads", rawPath))
		}
	} else {
		data, contentType, err = s.readImageFromLocalPath(path.Join("/uploads", rawPath))
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "application/octet-stream" {
		if byExt := mime.TypeByExtension(path.Ext(rawPath)); byExt != "" {
			contentType = strings.TrimSpace(strings.Split(byExt, ";")[0])
		}
	}

	cacheControl := "public, max-age=86400"
	if c.Query("v") != "" || immutableUploadFilePattern.MatchString(path.Base(rawPath)) {
		cacheControl = "public, max-age=31536000, immutable"
	}
	etag := fmt.Sprintf(`"%x"`, md5.Sum(data))
	if c.GetHeader("If-None-Match") == etag {
		c.Header("Cache-Control", cacheControl)
		c.Header("ETag", etag)
		c.Status(http.StatusNotModified)
		return
	}

	c.Header("Cache-Control", cacheControl)
	c.Header("ETag", etag)
	c.Header("Content-Type", contentType)
	http.ServeContent(c.Writer, c.Request, path.Base(rawPath), time.Time{}, bytes.NewReader(data))
}
