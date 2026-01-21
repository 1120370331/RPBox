package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	uploadDirName       = "uploads"
	remoteImageMaxBytes = 30 << 20
)

var imageExtMap = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

func (s *Server) saveUploadedImage(c *gin.Context, header *multipart.FileHeader, subdir string) (string, error) {
	file, err := header.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	contentType := header.Header.Get("Content-Type")
	if contentType != "" {
		contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("unsupported file type")
	}

	ext := imageExtension(contentType, header.Filename)
	if ext == "" {
		return "", fmt.Errorf("unsupported image format")
	}

	cleanSubdir := cleanUploadSubdir(subdir)
	baseDir := filepath.Join(s.cfg.Storage.Path, uploadDirName)
	targetDir := baseDir
	if cleanSubdir != "" {
		targetDir = filepath.Join(baseDir, filepath.FromSlash(cleanSubdir))
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	name, err := randomHex(16)
	if err != nil {
		return "", err
	}
	filename := name + ext
	targetPath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return "", err
	}

	urlPath := path.Join("/", uploadDirName, cleanSubdir, filename)
	return buildPublicURL(c, urlPath), nil
}

func buildPublicURL(c *gin.Context, urlPath string) string {
	if urlPath == "" {
		return urlPath
	}
	if strings.HasPrefix(urlPath, "http://") || strings.HasPrefix(urlPath, "https://") {
		return urlPath
	}
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := c.GetHeader("X-Forwarded-Proto"); forwarded != "" {
		scheme = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}

	host := c.Request.Host
	if forwardedHost := c.GetHeader("X-Forwarded-Host"); forwardedHost != "" {
		host = strings.TrimSpace(strings.Split(forwardedHost, ",")[0])
	}
	if host == "" {
		return urlPath
	}

	return scheme + "://" + host + urlPath
}

func (s *Server) loadImageBytes(c *gin.Context, value string) ([]byte, string, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, "", errors.New("empty image value")
	}

	if strings.HasPrefix(raw, "data:") {
		return decodeDataURI(raw)
	}

	if isImageURL(raw) {
		return s.readImageFromURL(c, raw)
	}

	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, "", err
	}
	return data, http.DetectContentType(data), nil
}

func decodeDataURI(dataURI string) ([]byte, string, error) {
	parts := strings.SplitN(dataURI, ",", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("invalid data uri")
	}
	meta := parts[0]
	dataPart := parts[1]

	contentType := "application/octet-stream"
	if strings.HasPrefix(meta, "data:") {
		meta = strings.TrimPrefix(meta, "data:")
		metaParts := strings.Split(meta, ";")
		if len(metaParts) > 0 && metaParts[0] != "" {
			contentType = metaParts[0]
		}
	}

	data, err := base64.StdEncoding.DecodeString(dataPart)
	if err != nil {
		return nil, "", err
	}

	if contentType == "application/octet-stream" {
		contentType = http.DetectContentType(data)
	}

	return data, contentType, nil
}

func (s *Server) readImageFromURL(c *gin.Context, raw string) ([]byte, string, error) {
	if strings.HasPrefix(raw, "/uploads/") || strings.HasPrefix(raw, "uploads/") {
		return s.readImageFromLocalPath(raw)
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return nil, "", err
		}
		if parsed.Path != "" && strings.HasPrefix(parsed.Path, "/uploads/") && isSameHost(c, parsed.Host) {
			return s.readImageFromLocalPath(parsed.Path)
		}

		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Get(raw)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("image fetch failed: %s", resp.Status)
		}

		limit := io.LimitReader(resp.Body, remoteImageMaxBytes)
		data, err := io.ReadAll(limit)
		if err != nil {
			return nil, "", err
		}
		contentType := resp.Header.Get("Content-Type")
		if contentType != "" {
			contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
		}
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
		return data, contentType, nil
	}

	return s.readImageFromLocalPath(raw)
}

func (s *Server) readImageFromLocalPath(raw string) ([]byte, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, "", fmt.Errorf("empty local image path")
	}

	clean := path.Clean(trimmed)
	if strings.HasPrefix(clean, "uploads/") {
		clean = "/" + clean
	}
	if !strings.HasPrefix(clean, "/uploads/") {
		return nil, "", fmt.Errorf("unsupported local image path")
	}

	relative := strings.TrimPrefix(clean, "/uploads/")
	baseDir := filepath.Clean(filepath.Join(s.cfg.Storage.Path, uploadDirName))
	targetPath := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(relative)))

	if targetPath != baseDir && !strings.HasPrefix(targetPath, baseDir+string(os.PathSeparator)) {
		return nil, "", fmt.Errorf("invalid local image path")
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, "", err
	}
	return data, http.DetectContentType(data), nil
}

func imageExtension(contentType, filename string) string {
	if ext, ok := imageExtMap[contentType]; ok {
		return ext
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext != "" {
		return ext
	}

	return ""
}

func cleanUploadSubdir(subdir string) string {
	cleaned := path.Clean("/" + subdir)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func isImageURL(value string) bool {
	return strings.HasPrefix(value, "http://") ||
		strings.HasPrefix(value, "https://") ||
		strings.HasPrefix(value, "/uploads/") ||
		strings.HasPrefix(value, "uploads/")
}

func isSameHost(c *gin.Context, host string) bool {
	if c == nil {
		return false
	}
	requestHost := normalizedHost(c.GetHeader("X-Forwarded-Host"))
	if requestHost == "" {
		requestHost = normalizedHost(c.Request.Host)
	}
	compareHost := normalizedHost(host)
	return requestHost != "" && compareHost != "" && requestHost == compareHost
}

func normalizedHost(host string) string {
	if host == "" {
		return ""
	}
	host = strings.TrimSpace(strings.Split(host, ",")[0])
	return strings.ToLower(host)
}

func randomHex(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("invalid random length")
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
