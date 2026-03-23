package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	Mandatory bool   `json:"mandatory,omitempty"`
}

// Platform 平台信息
type Platform struct {
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

type LatestRelease struct {
	LatestVersion string `json:"latest_version"`
	Version       string `json:"version"`
	Notes         string `json:"notes"`
	PubDate       string `json:"pub_date"`
}

type MobileLatestRelease struct {
	LatestVersion string `json:"latest_version"`
	Version       string `json:"version"`
	Notes         string `json:"notes"`
	PubDate       string `json:"pub_date"`
	URL           string `json:"url"`
	Mandatory     bool   `json:"mandatory"`
}

// checkUpdate 检查客户端更新
func (s *Server) checkUpdate(c *gin.Context) {
	target := strings.ToLower(c.Param("target"))
	arch := c.Param("arch")
	currentVersion := c.Param("current_version")

	if target == "android" || target == "ios" {
		s.checkMobileUpdate(c, target, currentVersion)
		return
	}

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

	// 当前版本已是最新（或比服务端高），返回 204 No Content
	if !isNewerVersion(latestVersion, currentVersion) {
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

func (s *Server) checkMobileUpdate(c *gin.Context, target, currentVersion string) {
	latest, ok := s.resolveMobileLatestRelease(target)
	if !ok {
		c.Status(http.StatusNoContent)
		return
	}

	fmt.Printf("checkMobileUpdate: target=%s current=%s latest=%s\n", target, currentVersion, latest.LatestVersion)

	if latest.LatestVersion == "" || latest.URL == "" || !isNewerVersion(latest.LatestVersion, currentVersion) {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, UpdateResponse{
		Version:   normalizeVersion(latest.LatestVersion),
		Notes:     latest.Notes,
		PubDate:   latest.PubDate,
		URL:       latest.URL,
		Mandatory: latest.Mandatory,
	})
}

// getMobileLatest 返回移动端稳定 latest 元信息。
func (s *Server) getMobileLatest(c *gin.Context) {
	target := strings.ToLower(c.Param("target"))
	latest, ok := s.resolveMobileLatestRelease(target)
	if !ok || latest.LatestVersion == "" || latest.URL == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "latest release not found",
		})
		return
	}

	c.JSON(http.StatusOK, latest)
}

// downloadMobileLatest 稳定下载入口，始终重定向到当前 latest 包地址。
func (s *Server) downloadMobileLatest(c *gin.Context) {
	target := strings.ToLower(c.Param("target"))
	latest, ok := s.resolveMobileLatestRelease(target)
	if !ok || latest.URL == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "latest release not found",
		})
		return
	}
	if _, err := url.ParseRequestURI(latest.URL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "latest release url is invalid",
		})
		return
	}

	c.Redirect(http.StatusFound, latest.URL)
}

func (s *Server) resolveMobileLatestRelease(target string) (*MobileLatestRelease, bool) {
	var platformCfg config.MobilePlatformUpdaterConfig
	switch target {
	case "android":
		platformCfg = config.Get().Updater.Mobile.Android
	case "ios":
		platformCfg = config.Get().Updater.Mobile.IOS
	default:
		return nil, false
	}

	latest := &MobileLatestRelease{
		LatestVersion: platformCfg.LatestVersion,
		Notes:         platformCfg.ReleaseNotes,
		PubDate:       platformCfg.PubDate,
		URL:           platformCfg.URL,
		Mandatory:     platformCfg.Mandatory,
	}
	latest.Version = latest.LatestVersion

	if metadata, err := readMobileLatestRelease(target); err == nil {
		if metadata.LatestVersion != "" {
			latest.LatestVersion = metadata.LatestVersion
		}
		if metadata.Version != "" {
			latest.Version = metadata.Version
		}
		if metadata.Notes != "" {
			latest.Notes = metadata.Notes
		}
		if metadata.PubDate != "" {
			latest.PubDate = metadata.PubDate
		}
		if metadata.URL != "" {
			latest.URL = metadata.URL
		}
		latest.Mandatory = metadata.Mandatory
	} else if !os.IsNotExist(err) {
		fmt.Printf("resolveMobileLatestRelease: failed to read %s metadata: %v\n", target, err)
	}

	if latest.LatestVersion == "" && latest.Version != "" {
		latest.LatestVersion = latest.Version
	}
	// Keep legacy "version" field consistent with latest_version to avoid client misread.
	latest.Version = latest.LatestVersion

	return latest, true
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
	if latest.LatestVersion == "" && latest.Version != "" {
		latest.LatestVersion = latest.Version
	}
	if latest.LatestVersion == "" {
		return nil, fmt.Errorf("latest.json missing latest_version")
	}
	return &latest, nil
}

func readMobileLatestRelease(target string) (*MobileLatestRelease, error) {
	latestPath := filepath.Join("releases", "mobile", "latest-"+target+".json")
	data, err := os.ReadFile(latestPath)
	if err != nil {
		return nil, err
	}
	var latest MobileLatestRelease
	if err := json.Unmarshal(data, &latest); err != nil {
		return nil, err
	}
	if latest.LatestVersion == "" && latest.Version != "" {
		latest.LatestVersion = latest.Version
	}
	return &latest, nil
}

func isNewerVersion(latestVersion, currentVersion string) bool {
	latest := normalizeVersion(latestVersion)
	current := normalizeVersion(currentVersion)
	if latest == "" {
		return false
	}
	if current == "" {
		return true
	}

	if cmp, ok := compareVersions(latest, current); ok {
		return cmp > 0
	}

	return latest != current
}

func compareVersions(a, b string) (int, bool) {
	pa, ok := parseVersionParts(a)
	if !ok {
		return 0, false
	}
	pb, ok := parseVersionParts(b)
	if !ok {
		return 0, false
	}

	maxLen := len(pa)
	if len(pb) > maxLen {
		maxLen = len(pb)
	}

	for i := 0; i < maxLen; i++ {
		ai := 0
		bi := 0
		if i < len(pa) {
			ai = pa[i]
		}
		if i < len(pb) {
			bi = pb[i]
		}
		if ai > bi {
			return 1, true
		}
		if ai < bi {
			return -1, true
		}
	}

	return 0, true
}

func parseVersionParts(version string) ([]int, bool) {
	normalized := normalizeVersion(version)
	if normalized == "" {
		return nil, false
	}

	// 忽略 pre-release/build 元数据，如 1.2.3-beta+001。
	if idx := strings.IndexAny(normalized, "-+"); idx > 0 {
		normalized = normalized[:idx]
	}

	parts := strings.Split(normalized, ".")
	result := make([]int, len(parts))
	for i, part := range parts {
		if part == "" {
			return nil, false
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		result[i] = n
	}
	return result, true
}

func normalizeVersion(version string) string {
	v := strings.TrimSpace(version)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "v") || strings.HasPrefix(v, "V") {
		v = v[1:]
	}
	return strings.TrimSpace(v)
}
