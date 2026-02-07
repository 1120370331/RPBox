package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const iconOSSSubdir = "icons"

func normalizeIconName(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("empty icon name")
	}

	if start := strings.Index(trimmed, "|T"); start >= 0 {
		if end := strings.Index(trimmed, "|t"); end > start+2 {
			content := trimmed[start+2 : end]
			if colon := strings.Index(content, ":"); colon >= 0 {
				content = content[:colon]
			}
			trimmed = content
		}
	}

	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "interface/icons/") {
		trimmed = trimmed[len("interface/icons/"):]
	}

	trimmed = path.Base(strings.TrimSpace(trimmed))
	if trimmed == "" || trimmed == "." || trimmed == "/" {
		return "", fmt.Errorf("invalid icon name")
	}

	if ext := strings.ToLower(path.Ext(trimmed)); ext != "" {
		switch ext {
		case ".blp", ".tga", ".png", ".jpg", ".jpeg":
			trimmed = strings.TrimSuffix(trimmed, ext)
		}
	}

	name := strings.ToLower(strings.TrimSpace(trimmed))
	if name == "" {
		return "", fmt.Errorf("invalid icon name")
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return "", fmt.Errorf("invalid icon name")
		}
	}
	return name, nil
}

// iconLocalCacheDir returns the local filesystem path for icon cache.
func (s *Server) iconLocalCacheDir() string {
	return filepath.Join(s.cfg.Storage.Path, "cache", iconOSSSubdir)
}

// getIcon 获取图标（带缓存，支持 OSS）
func (s *Server) getIcon(c *gin.Context) {
	iconName, err := normalizeIconName(c.Param("name"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的图标名称"})
		return
	}
	filename := iconName + ".jpg"

	// 检查缓存
	if data, contentType, ok := s.readIconCache(filename); ok {
		c.Header("Content-Type", contentType)
		c.Header("Cache-Control", "public, max-age=31536000")
		c.Data(http.StatusOK, contentType, data)
		return
	}

	// 从暴雪 CDN 拉取
	data, contentType, err := s.fetchIconFromCDN(c, iconName)
	if err != nil {
		if err == errIconNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "图标不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取图标失败"})
		return
	}

	// 保存到缓存（忽略错误，缓存失败不影响响应）
	s.writeIconCache(filename, data, contentType)

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Data(http.StatusOK, contentType, data)
}

var errIconNotFound = fmt.Errorf("icon not found")

func (s *Server) fetchIconFromCDN(c *gin.Context, iconName string) ([]byte, string, error) {
	url := fmt.Sprintf("https://render.worldofwarcraft.com/us/icons/56/%s.jpg", iconName)
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "RPBox/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", errIconNotFound
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	contentType := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	return data, contentType, nil
}

// readIconCache tries OSS first (if enabled), then local disk.
func (s *Server) readIconCache(filename string) ([]byte, string, bool) {
	if s.ossEnabled() {
		ossKey := s.buildOSSKey(iconOSSSubdir, filename)
		if data, ct, err := s.readImageFromOSS(ossKey); err == nil {
			return data, ct, true
		}
	}

	localPath := filepath.Join(s.iconLocalCacheDir(), filename)
	if data, err := os.ReadFile(localPath); err == nil {
		return data, http.DetectContentType(data), true
	}
	return nil, "", false
}

// writeIconCache writes to OSS (if enabled), otherwise to local disk.
func (s *Server) writeIconCache(filename string, data []byte, contentType string) {
	if s.ossEnabled() {
		ossKey := s.buildOSSKey(iconOSSSubdir, filename)
		_ = s.uploadToOSS(ossKey, data, contentType)
		return
	}

	dir := s.iconLocalCacheDir()
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, filename), data, 0644)
}
