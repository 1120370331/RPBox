package api

import (
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// getUploadObject serves uploaded images from local storage or OSS.
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

	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("Content-Type", contentType)
	c.Data(http.StatusOK, contentType, data)
}
