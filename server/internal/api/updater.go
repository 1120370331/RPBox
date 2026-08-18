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
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/pkg/auth"
)

const (
	desktopUpdateChannelHeader = "X-RPBox-Update-Channel"
	updateChannelStable        = "stable"
	updateChannelBeta          = "beta"
)

// UpdateResponse Tauri updater 响应格式
type UpdateResponse struct {
	Version   string              `json:"version"`
	Notes     string              `json:"notes,omitempty"`
	PubDate   string              `json:"pub_date,omitempty"`
	Channel   string              `json:"channel,omitempty"`
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
	Channel       string `json:"channel,omitempty"`
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

	latest := s.resolveDesktopReleaseForCheck(c, currentVersion)
	latestVersion := latest.LatestVersion

	// 调试日志
	fmt.Printf("checkUpdate: channel=%s current=%s latest=%s\n", latest.Channel, currentVersion, latestVersion)

	// 当前版本已是最新（或比服务端高），返回 204 No Content
	if !isNewerVersion(latestVersion, currentVersion) {
		c.Status(http.StatusNoContent)
		return
	}

	// 构建平台标识
	platformKey := target + "-" + arch

	// 获取更新包信息
	baseURL := s.desktopReleaseBaseURL(latest.Channel)
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
		Notes:     latest.Notes,
		PubDate:   latest.PubDate,
		Channel:   latest.Channel,
		URL:       url,
		Signature: signature,
	}

	c.JSON(http.StatusOK, response)
}

// getDesktopLatest 返回桌面端最新版本元信息。
func (s *Server) getDesktopLatest(c *gin.Context) {
	latest := s.resolveDesktopReleaseForLatest(c)
	c.JSON(http.StatusOK, latest)
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

// getIOSUpdateLink 返回 iOS 的稳定更新跳转信息（用于 TestFlight / App Store）。
func (s *Server) getIOSUpdateLink(c *gin.Context) {
	latest, ok := s.resolveMobileLatestRelease("ios")
	if !ok || latest.URL == "" || latest.LatestVersion == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "ios latest release not found",
		})
		return
	}
	if _, err := url.ParseRequestURI(latest.URL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "ios update url is invalid",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"target":         "ios",
		"latest_version": latest.LatestVersion,
		"version":        latest.LatestVersion,
		"url":            latest.URL,
		"notes":          latest.Notes,
		"pub_date":       latest.PubDate,
		"mandatory":      latest.Mandatory,
	})
}

func (s *Server) resolveMobileLatestRelease(target string) (*MobileLatestRelease, bool) {
	var platformCfg config.MobilePlatformUpdaterConfig
	updater := s.cfg.Updater
	switch target {
	case "android":
		platformCfg = updater.Mobile.Android
	case "ios":
		platformCfg = updater.Mobile.IOS
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

func (s *Server) getReleaseFile(c *gin.Context) {
	relPath := strings.TrimPrefix(c.Param("filepath"), "/")
	cleanPath := filepath.Clean(relPath)
	if cleanPath == "." || strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid release path"})
		return
	}

	if isBetaReleasePath(cleanPath) && !s.isDesktopBetaEligible(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要测试版更新权限"})
		return
	}

	fullPath := filepath.Join("releases", cleanPath)
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "release file not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read release file"})
		return
	}

	c.File(fullPath)
}

func isBetaReleasePath(relPath string) bool {
	firstSegment := strings.ToLower(strings.Split(filepath.ToSlash(relPath), "/")[0])
	if firstSegment == "latest-beta.json" {
		return true
	}
	return strings.Contains(firstSegment, "-beta") ||
		strings.Contains(firstSegment, "-alpha") ||
		strings.Contains(firstSegment, "-rc")
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

func (s *Server) desktopReleaseBaseURL(channel string) string {
	updater := s.cfg.Updater
	if channel == updateChannelBeta && updater.Beta.BaseURL != "" {
		return updater.Beta.BaseURL
	}
	return updater.BaseURL
}

func readLatestRelease() (*LatestRelease, error) {
	return readLatestReleaseFile(filepath.Join("releases", "latest.json"))
}

func readBetaLatestRelease() (*LatestRelease, error) {
	return readLatestReleaseFile(filepath.Join("releases", "latest-beta.json"))
}

func readLatestReleaseFile(latestPath string) (*LatestRelease, error) {
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

func (s *Server) resolveDesktopLatestRelease() *LatestRelease {
	updater := s.cfg.Updater
	latest := &LatestRelease{
		LatestVersion: updater.LatestVersion,
		Notes:         updater.ReleaseNotes,
		PubDate:       updater.PubDate,
		Channel:       updateChannelStable,
	}
	latest.Version = latest.LatestVersion

	if metadata, err := readLatestRelease(); err == nil {
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
		if metadata.Channel != "" {
			latest.Channel = metadata.Channel
		}
	} else if !os.IsNotExist(err) {
		fmt.Printf("resolveDesktopLatestRelease: failed to read latest.json: %v\n", err)
	}

	if latest.LatestVersion == "" && latest.Version != "" {
		latest.LatestVersion = latest.Version
	}
	if latest.LatestVersion == "" {
		latest.LatestVersion = "0.1.0"
	}
	latest.Version = latest.LatestVersion
	if latest.Channel == "" {
		latest.Channel = updateChannelStable
	}

	return latest
}

func (s *Server) resolveDesktopBetaRelease() (*LatestRelease, bool) {
	betaCfg := s.cfg.Updater.Beta
	latest := &LatestRelease{
		LatestVersion: betaCfg.LatestVersion,
		Notes:         betaCfg.ReleaseNotes,
		PubDate:       betaCfg.PubDate,
		Channel:       updateChannelBeta,
	}
	latest.Version = latest.LatestVersion

	hasRelease := latest.LatestVersion != "" || latest.Version != ""
	if metadata, err := readBetaLatestRelease(); err == nil {
		hasRelease = true
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
		if metadata.Channel != "" {
			latest.Channel = metadata.Channel
		}
	} else if !os.IsNotExist(err) {
		fmt.Printf("resolveDesktopBetaRelease: failed to read latest-beta.json: %v\n", err)
	}

	if latest.LatestVersion == "" && latest.Version != "" {
		latest.LatestVersion = latest.Version
	}
	if latest.LatestVersion == "" || !hasRelease {
		return nil, false
	}
	latest.Version = latest.LatestVersion
	latest.Channel = updateChannelBeta

	return latest, true
}

func (s *Server) resolveDesktopReleaseForCheck(c *gin.Context, currentVersion string) *LatestRelease {
	stable := s.resolveDesktopLatestRelease()
	if !s.canServeDesktopBeta(c) {
		return stable
	}

	beta, ok := s.resolveDesktopBetaRelease()
	if !ok || !isNewerVersion(beta.LatestVersion, currentVersion) {
		return stable
	}
	if stable.LatestVersion == "" {
		return beta
	}
	if cmp, ok := compareVersions(beta.LatestVersion, stable.LatestVersion); ok {
		if cmp > 0 {
			return beta
		}
		return stable
	}
	if beta.LatestVersion != stable.LatestVersion {
		return beta
	}
	return stable
}

func (s *Server) resolveDesktopReleaseForLatest(c *gin.Context) *LatestRelease {
	stable := s.resolveDesktopLatestRelease()
	if !s.canServeDesktopBeta(c) {
		return stable
	}

	beta, ok := s.resolveDesktopBetaRelease()
	if !ok {
		return stable
	}
	if stable.LatestVersion == "" {
		return beta
	}
	if cmp, ok := compareVersions(beta.LatestVersion, stable.LatestVersion); ok {
		if cmp > 0 {
			return beta
		}
		return stable
	}
	if beta.LatestVersion != stable.LatestVersion {
		return beta
	}
	return stable
}

func (s *Server) canServeDesktopBeta(c *gin.Context) bool {
	if !isDesktopBetaRequested(c) {
		return false
	}
	return s.isDesktopBetaEligible(c)
}

func isDesktopBetaRequested(c *gin.Context) bool {
	channel := strings.ToLower(strings.TrimSpace(c.GetHeader(desktopUpdateChannelHeader)))
	if channel == "" {
		channel = strings.ToLower(strings.TrimSpace(c.Query("channel")))
	}
	if channel == updateChannelBeta {
		return true
	}
	beta := strings.ToLower(strings.TrimSpace(c.Query("beta")))
	return beta == "1" || beta == "true"
}

func (s *Server) isDesktopBetaEligible(c *gin.Context) bool {
	token := bearerTokenFromHeader(c.GetHeader("Authorization"))
	if token == "" {
		return false
	}

	claims, err := auth.ParseToken(token)
	if err != nil {
		return false
	}

	var user model.User
	if err := database.DB.
		Select("id", "role", "is_sponsor", "sponsor_level", "sponsor_expires_at", "account_deleted_at").
		First(&user, claims.UserID).Error; err != nil {
		return false
	}
	if user.AccountDeletedAt != nil {
		return false
	}
	return user.Role == "admin" || resolveSponsorLevel(user) >= 1
}

func bearerTokenFromHeader(authHeader string) string {
	parts := strings.SplitN(strings.TrimSpace(authHeader), " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
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
	pa, ok := parseComparableVersion(a)
	if !ok {
		return 0, false
	}
	pb, ok := parseComparableVersion(b)
	if !ok {
		return 0, false
	}

	maxLen := len(pa.core)
	if len(pb.core) > maxLen {
		maxLen = len(pb.core)
	}

	for i := 0; i < maxLen; i++ {
		ai := 0
		bi := 0
		if i < len(pa.core) {
			ai = pa.core[i]
		}
		if i < len(pb.core) {
			bi = pb.core[i]
		}
		if ai > bi {
			return 1, true
		}
		if ai < bi {
			return -1, true
		}
	}

	if !pa.hasPreRelease && !pb.hasPreRelease {
		return 0, true
	}
	if !pa.hasPreRelease {
		return 1, true
	}
	if !pb.hasPreRelease {
		return -1, true
	}

	maxPreLen := len(pa.preRelease)
	if len(pb.preRelease) > maxPreLen {
		maxPreLen = len(pb.preRelease)
	}
	for i := 0; i < maxPreLen; i++ {
		if i >= len(pa.preRelease) {
			return -1, true
		}
		if i >= len(pb.preRelease) {
			return 1, true
		}

		cmp := comparePrereleaseIdentifier(pa.preRelease[i], pb.preRelease[i])
		if cmp != 0 {
			return cmp, true
		}
	}

	return 0, true
}

type comparableVersion struct {
	core          []int
	preRelease    []string
	hasPreRelease bool
}

func parseComparableVersion(version string) (comparableVersion, bool) {
	normalized := normalizeVersion(version)
	if normalized == "" {
		return comparableVersion{}, false
	}

	if idx := strings.Index(normalized, "+"); idx >= 0 {
		normalized = normalized[:idx]
	}

	var preRelease []string
	hasPreRelease := false
	if idx := strings.Index(normalized, "-"); idx >= 0 {
		hasPreRelease = true
		if idx < len(normalized)-1 {
			preRelease = strings.Split(normalized[idx+1:], ".")
		}
		normalized = normalized[:idx]
	}

	parts := strings.Split(normalized, ".")
	core := make([]int, len(parts))
	for i, part := range parts {
		if part == "" {
			return comparableVersion{}, false
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return comparableVersion{}, false
		}
		core[i] = n
	}

	return comparableVersion{
		core:          core,
		preRelease:    preRelease,
		hasPreRelease: hasPreRelease,
	}, true
}

func comparePrereleaseIdentifier(a, b string) int {
	aNum, aIsNum := parsePrereleaseNumber(a)
	bNum, bIsNum := parsePrereleaseNumber(b)

	switch {
	case aIsNum && bIsNum:
		if aNum > bNum {
			return 1
		}
		if aNum < bNum {
			return -1
		}
		return 0
	case aIsNum:
		return -1
	case bIsNum:
		return 1
	default:
		return strings.Compare(a, b)
	}
}

func parsePrereleaseNumber(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return n, true
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
