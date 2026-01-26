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

const iconCacheDir = "./data/icons"

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

// getIcon 获取图标（带缓存）
func (s *Server) getIcon(c *gin.Context) {
	iconName, err := normalizeIconName(c.Param("name"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的图标名称"})
		return
	}
	cachePath := filepath.Join(iconCacheDir, iconName+".jpg")

	// 检查缓存
	if data, err := os.ReadFile(cachePath); err == nil {
		contentType := http.DetectContentType(data)
		c.Header("Content-Type", contentType)
		c.Header("Cache-Control", "public, max-age=31536000")
		c.Data(http.StatusOK, contentType, data)
		return
	}

	// 从暴雪 CDN 拉取
	url := fmt.Sprintf("https://render.worldofwarcraft.com/us/icons/56/%s.jpg", iconName)
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取图标失败"})
		return
	}
	req.Header.Set("User-Agent", "RPBox/1.0")
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取图标失败"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "图标不存在"})
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取图标失败"})
		return
	}
	contentType := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	// 保存到缓存
	os.MkdirAll(iconCacheDir, 0755)
	os.WriteFile(cachePath, data, 0644)

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Data(http.StatusOK, contentType, data)
}
