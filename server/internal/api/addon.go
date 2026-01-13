package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// AddonVersionInfo 插件版本信息
type AddonVersionInfo struct {
	Version          string `json:"version"`
	ReleaseDate      string `json:"releaseDate"`
	MinClientVersion string `json:"minClientVersion"`
	Changelog        string `json:"changelog"`
	DownloadURL      string `json:"downloadUrl"`
}

// AddonManifest 插件版本清单
type AddonManifest struct {
	Name     string             `json:"name"`
	Latest   string             `json:"latest"`
	Versions []AddonVersionInfo `json:"versions"`
}

// getAddonStoragePath 获取插件存储路径
func (s *Server) getAddonStoragePath() string {
	return filepath.Join(s.cfg.Storage.Path, "addons", "RPBox_Addon")
}

// loadManifest 加载版本清单
func (s *Server) loadManifest() (*AddonManifest, error) {
	manifestPath := filepath.Join(s.getAddonStoragePath(), "manifest.json")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 返回默认清单
			return &AddonManifest{
				Name:     "RPBox_Addon",
				Latest:   "1.0.0",
				Versions: []AddonVersionInfo{},
			}, nil
		}
		return nil, err
	}

	var manifest AddonManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// getAddonManifest 获取完整版本清单
func (s *Server) getAddonManifest(c *gin.Context) {
	manifest, err := s.loadManifest()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load manifest"})
		return
	}
	c.JSON(http.StatusOK, manifest)
}

// getAddonLatest 获取最新版本号
func (s *Server) getAddonLatest(c *gin.Context) {
	manifest, err := s.loadManifest()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load manifest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"version":     manifest.Latest,
		"downloadUrl": fmt.Sprintf("/api/v1/addon/download/%s", manifest.Latest),
	})
}

// downloadAddon 下载指定版本插件
func (s *Server) downloadAddon(c *gin.Context) {
	version := c.Param("version")
	if version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version is required"})
		return
	}

	// 尝试从 versions 目录下载
	zipPath := filepath.Join(s.getAddonStoragePath(), "versions", version+".zip")

	// 如果是最新版本，也检查 latest 目录
	manifest, _ := s.loadManifest()
	if manifest != nil && version == manifest.Latest {
		latestZip := filepath.Join(s.getAddonStoragePath(), "latest.zip")
		if _, err := os.Stat(latestZip); err == nil {
			zipPath = latestZip
		}
	}

	file, err := os.Open(zipPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read addon"})
		return
	}
	defer file.Close()

	stat, _ := file.Stat()
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=RPBox_Addon_%s.zip", version))
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size()))

	io.Copy(c.Writer, file)
}
