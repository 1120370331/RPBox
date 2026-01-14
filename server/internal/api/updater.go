package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/config"
)

// UpdateResponse Tauri updater 响应格式
type UpdateResponse struct {
	Version   string            `json:"version"`
	Notes     string            `json:"notes,omitempty"`
	PubDate   string            `json:"pub_date,omitempty"`
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

// checkUpdate 检查客户端更新
func (s *Server) checkUpdate(c *gin.Context) {
	target := c.Param("target")
	arch := c.Param("arch")
	currentVersion := c.Param("current_version")

	// 读取最新版本信息
	latestVersion := config.Get().Updater.LatestVersion
	if latestVersion == "" {
		latestVersion = "0.1.0"
	}

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
	var url, signature string
	switch platformKey {
	case "windows-x86_64":
		url = baseURL + "/" + latestVersion + "/RPBox_" + latestVersion + "_x64-setup.nsis.zip"
		signature = getSignature(latestVersion, "windows-x86_64")
	case "windows-aarch64":
		url = baseURL + "/" + latestVersion + "/RPBox_" + latestVersion + "_arm64-setup.nsis.zip"
		signature = getSignature(latestVersion, "windows-aarch64")
	case "darwin-x86_64":
		url = baseURL + "/" + latestVersion + "/RPBox_" + latestVersion + "_x64.app.tar.gz"
		signature = getSignature(latestVersion, "darwin-x86_64")
	case "darwin-aarch64":
		url = baseURL + "/" + latestVersion + "/RPBox_" + latestVersion + "_aarch64.app.tar.gz"
		signature = getSignature(latestVersion, "darwin-aarch64")
	case "linux-x86_64":
		url = baseURL + "/" + latestVersion + "/RPBox_" + latestVersion + "_amd64.AppImage.tar.gz"
		signature = getSignature(latestVersion, "linux-x86_64")
	default:
		c.Status(http.StatusNoContent)
		return
	}

	response := UpdateResponse{
		Version:   latestVersion,
		Notes:     config.Get().Updater.ReleaseNotes,
		PubDate:   config.Get().Updater.PubDate,
		URL:       url,
		Signature: signature,
	}

	c.JSON(http.StatusOK, response)
}

// getSignature 获取签名文件内容
func getSignature(version, platform string) string {
	sigDir := config.Get().Updater.SignatureDir
	if sigDir == "" {
		return ""
	}

	sigFile := filepath.Join(sigDir, version, platform+".sig")
	data, err := os.ReadFile(sigFile)
	if err != nil {
		return ""
	}
	return string(data)
}
