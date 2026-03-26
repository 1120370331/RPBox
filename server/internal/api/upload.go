package api

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"

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

	var (
		data        []byte
		contentType string
		err         error
	)

	if s.ossEnabled() {
		ossKey := s.buildOSSKey(rawPath, "")
		data, contentType, err = s.readImageFromOSS(ossKey)
		if err != nil {
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
	c.Data(http.StatusOK, contentType, data)
}
