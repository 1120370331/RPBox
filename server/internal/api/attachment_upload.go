package api

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxAttachmentBytes int64 = 25 << 20

var errAttachmentTooLarge = errors.New("attachment too large")

// uploadAttachment 上传附件（最大 25MB）
func (s *Server) uploadAttachment(c *gin.Context) {
	header, err := c.FormFile("file")
	if err != nil {
		header, err = c.FormFile("attachment")
	}
	if err != nil || header == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择附件文件"})
		return
	}

	if header.Size > maxAttachmentBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "附件不能超过25MB"})
		return
	}

	attachmentURL, err := s.saveUploadedAttachment(c, header, "attachments")
	if err != nil {
		if errors.Is(err, errAttachmentTooLarge) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "附件不能超过25MB"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存附件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"url":  attachmentURL,
			"name": filepath.Base(header.Filename),
			"size": header.Size,
		},
	})
}

func (s *Server) saveUploadedAttachment(c *gin.Context, header *multipart.FileHeader, subdir string) (string, error) {
	if header == nil {
		return "", errors.New("empty attachment")
	}

	file, err := header.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	limit := maxAttachmentBytes + 1
	data, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxAttachmentBytes {
		return "", errAttachmentTooLarge
	}
	if len(data) == 0 {
		return "", errors.New("empty attachment")
	}

	contentType := header.Header.Get("Content-Type")
	if contentType != "" {
		contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}

	cleanSubdir := cleanUploadSubdir(subdir)
	name, err := randomHex(16)
	if err != nil {
		return "", err
	}
	baseName := filepath.Base(header.Filename)
	ext := strings.ToLower(filepath.Ext(baseName))
	if len(ext) > 16 {
		ext = ""
	}
	filename := name + ext
	relativePath := path.Join(cleanSubdir, filename)

	if s.ossEnabled() {
		objectKey := s.buildOSSKey(cleanSubdir, filename)
		if err := s.uploadToOSS(objectKey, data, contentType); err != nil {
			return "", err
		}
		urlPath := path.Join("/", uploadDirName, relativePath)
		return buildPublicURL(c, urlPath), nil
	}

	baseDir := filepath.Join(s.cfg.Storage.Path, uploadDirName)
	targetDir := baseDir
	if cleanSubdir != "" {
		targetDir = filepath.Join(baseDir, filepath.FromSlash(cleanSubdir))
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	targetPath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return "", err
	}

	urlPath := path.Join("/", uploadDirName, relativePath)
	return buildPublicURL(c, urlPath), nil
}
