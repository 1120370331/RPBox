package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const iconCacheDir = "./data/icons"

// getIcon 获取图标（带缓存）
func (s *Server) getIcon(c *gin.Context) {
	iconName := c.Param("name")

	// 安全检查：只允许字母数字和下划线
	for _, r := range iconName {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的图标名称"})
			return
		}
	}

	iconName = strings.ToLower(iconName)
	cachePath := filepath.Join(iconCacheDir, iconName+".jpg")

	// 检查缓存
	if data, err := os.ReadFile(cachePath); err == nil {
		c.Header("Content-Type", "image/jpeg")
		c.Header("Cache-Control", "public, max-age=31536000")
		c.Data(http.StatusOK, "image/jpeg", data)
		return
	}

	// 从暴雪 CDN 拉取
	url := fmt.Sprintf("https://render.worldofwarcraft.com/us/icons/56/%s.jpg", iconName)
	resp, err := http.Get(url)
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

	// 保存到缓存
	os.MkdirAll(iconCacheDir, 0755)
	os.WriteFile(cachePath, data, 0644)

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Data(http.StatusOK, "image/jpeg", data)
}
