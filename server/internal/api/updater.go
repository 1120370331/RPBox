package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/config"
)

// UpdateResponse Tauri updater 响应格式
type UpdateResponse struct {
	Version   string              `json:"version"`
	Notes     string              `json:"notes,omitempty"`
	PubDate   string              `json:"pub_date,omitempty"`
	Platforms map[string]Platform `json:"platforms,omitempty"`
	// 单平台响应格式
	URL       string `json:"url,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// Platform 平台信息
type Platform struct {
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

type LatestRelease struct {
	LatestVersion string `json:"latest_version"`
	Notes         string `json:"notes"`
	PubDate       string `json:"pub_date"`
}

// checkUpdate 检查客户端更新
func (s *Server) checkUpdate(c *gin.Context) {
	target := c.Param("target")
	arch := c.Param("arch")
	currentVersion := c.Param("current_version")

	// 读取最新版本信息
	latestVersion := config.Get().Updater.LatestVersion
	notes := config.Get().Updater.ReleaseNotes
	pubDate := config.Get().Updater.PubDate
	if latestVersion == "" {
		latestVersion = "0.1.0"
	}
	if latest, err := readLatestRelease(); err == nil {
		if latest.LatestVersion != "" {
			latestVersion = latest.LatestVersion
		}
		if latest.Notes != "" {
			notes = latest.Notes
		}
		if latest.PubDate != "" {
			pubDate = latest.PubDate
		}
	} else if !os.IsNotExist(err) {
		fmt.Printf("checkUpdate: failed to read latest.json: %v\n", err)
	}

	// 调试日志
	fmt.Printf("checkUpdate: current=%s latest=%s\n", currentVersion, latestVersion)

	// 如果当前版本已是最新，返回 204 No Content
	if currentVersion == latestVersion {
		c.Status(http.StatusNoContent)
		return
	}

	// 构建平台标识
	platformKey := target + "-" + arch

	// 获取更新包信息
	baseURL := config.Get().Updater.BaseURL
	if baseURL == "" {
		baseURL = "https://api.rpbox.app/releases"
	}

	// 根据平台返回对应的更新包
	var url, sigFile string
	switch platformKey {
	case "windows-x86_64":
		url = baseURL + "/" + latestVersion + "/RPBox_" + latestVersion + "_x64-setup.exe"
		sigFile = "RPBox_" + latestVersion + "_x64-setup.exe.sig"
	case "windows-aarch64":
		url = baseURL + "/" + latestVersion + "/RPBox_" + latestVersion + "_arm64-setup.exe"
		sigFile = "RPBox_" + latestVersion + "_arm64-setup.exe.sig"
	case "darwin-x86_64":
		url = baseURL + "/" + latestVersion + "/RPBox_" + latestVersion + "_x64.app.tar.gz"
		sigFile = "RPBox_" + latestVersion + "_x64.app.tar.gz.sig"
	case "darwin-aarch64":
		url = baseURL + "/" + latestVersion + "/RPBox_" + latestVersion + "_aarch64.app.tar.gz"
		sigFile = "RPBox_" + latestVersion + "_aarch64.app.tar.gz.sig"
	case "linux-x86_64":
		url = baseURL + "/" + latestVersion + "/RPBox_" + latestVersion + "_amd64.AppImage.tar.gz"
		sigFile = "RPBox_" + latestVersion + "_amd64.AppImage.tar.gz.sig"
	default:
		c.Status(http.StatusNoContent)
		return
	}
	signature := getSignature(latestVersion, sigFile)

	response := UpdateResponse{
		Version:   latestVersion,
		Notes:     notes,
		PubDate:   pubDate,
		URL:       url,
		Signature: signature,
	}

	c.JSON(http.StatusOK, response)
}

// getSignature 获取签名文件内容
func getSignature(version, sigFileName string) string {
	// 默认从 releases 目录读取签名
	sigFile := filepath.Join("releases", version, sigFileName)
	data, err := os.ReadFile(sigFile)
	if err != nil {
		fmt.Printf("getSignature: failed to read %s: %v\n", sigFile, err)
		return ""
	}
	return string(data)
}

func readLatestRelease() (*LatestRelease, error) {
	latestPath := filepath.Join("releases", "latest.json")
	data, err := os.ReadFile(latestPath)
	if err != nil {
		return nil, err
	}
	var latest LatestRelease
	if err := json.Unmarshal(data, &latest); err != nil {
		return nil, err
	}
	if latest.LatestVersion == "" {
		return nil, fmt.Errorf("latest.json missing latest_version")
	}
	return &latest, nil
}
