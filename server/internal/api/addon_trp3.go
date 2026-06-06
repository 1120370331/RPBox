package api

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type TRP3AddonLatestInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ProjectID     int    `json:"projectId"`
	Repository    string `json:"repository"`
	LatestVersion string `json:"latestVersion"`
	DownloadURL   string `json:"downloadUrl"`
	FileName      string `json:"fileName"`
	FileDate      string `json:"fileDate,omitempty"`
	SourceURL     string `json:"sourceUrl"`
	CurseForgeURL string `json:"curseforgeUrl"`
	License       string `json:"license"`
}

type TRP3LatestResponse struct {
	Source      string                `json:"source"`
	Note        string                `json:"note"`
	CachedUntil string                `json:"cachedUntil,omitempty"`
	Addons      []TRP3AddonLatestInfo `json:"addons"`
}

type TRP3MirrorAddonInfo struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ProjectID        int    `json:"projectId"`
	Repository       string `json:"repository"`
	LatestVersion    string `json:"latestVersion"`
	FileName         string `json:"fileName"`
	FileDate         string `json:"fileDate,omitempty"`
	Source           string `json:"source"`
	UploadedBy       uint   `json:"uploadedBy,omitempty"`
	UploadedByName   string `json:"uploadedByName,omitempty"`
	UploadedAt       string `json:"uploadedAt,omitempty"`
	SizeBytes        int64  `json:"sizeBytes,omitempty"`
	DownloadURL      string `json:"downloadUrl,omitempty"`
	CacheKey         string `json:"cacheKey,omitempty"`
	CacheSource      string `json:"cacheSource,omitempty"`
	ExpectedRoot     string `json:"expectedRoot,omitempty"`
	ExpectedTOC      string `json:"expectedToc,omitempty"`
	OriginalFileName string `json:"originalFileName,omitempty"`
}

type TRP3MirrorManifest struct {
	UpdatedAt string                         `json:"updatedAt"`
	Addons    map[string]TRP3MirrorAddonInfo `json:"addons"`
}

type trp3GitHubProject struct {
	id            string
	name          string
	projectID     int
	owner         string
	repo          string
	filePrefix    string
	curseForgeURL string
	fallback      TRP3AddonLatestInfo
}

type gitHubReleaseResponse struct {
	TagName     string               `json:"tag_name"`
	Name        string               `json:"name"`
	HTMLURL     string               `json:"html_url"`
	PublishedAt string               `json:"published_at"`
	ZipballURL  string               `json:"zipball_url"`
	Assets      []gitHubReleaseAsset `json:"assets"`
}

type gitHubReleaseAsset struct {
	ID                 int64  `json:"id"`
	URL                string `json:"url"`
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

var trp3GitHubProjects = []trp3GitHubProject{
	{
		id:            "total-rp-3",
		name:          "Total RP 3",
		projectID:     75973,
		owner:         "Total-RP",
		repo:          "Total-RP-3",
		filePrefix:    "totalRP3-",
		curseForgeURL: "https://www.curseforge.com/wow/addons/total-rp-3/files",
		fallback: TRP3AddonLatestInfo{
			ID:            "total-rp-3",
			Name:          "Total RP 3",
			ProjectID:     75973,
			Repository:    "Total-RP/Total-RP-3",
			LatestVersion: "3.3.6",
			DownloadURL:   "https://github.com/Total-RP/Total-RP-3/releases/download/3.3.6/totalRP3-3.3.6.zip",
			FileName:      "totalRP3-3.3.6.zip",
			FileDate:      "2026-04-22T00:34:19Z",
			SourceURL:     "https://github.com/Total-RP/Total-RP-3/releases/tag/3.3.6",
			CurseForgeURL: "https://www.curseforge.com/wow/addons/total-rp-3/files",
			License:       "Apache-2.0",
		},
	},
	{
		id:            "total-rp-3-extended",
		name:          "Total RP 3: Extended",
		projectID:     100707,
		owner:         "Total-RP",
		repo:          "Total-RP-3-Extended",
		filePrefix:    "totalRP3_Extended-",
		curseForgeURL: "https://www.curseforge.com/wow/addons/total-rp-3-extended/files",
		fallback: TRP3AddonLatestInfo{
			ID:            "total-rp-3-extended",
			Name:          "Total RP 3: Extended",
			ProjectID:     100707,
			Repository:    "Total-RP/Total-RP-3-Extended",
			LatestVersion: "2.3.3",
			DownloadURL:   "https://github.com/Total-RP/Total-RP-3-Extended/releases/download/2.3.3/totalRP3_Extended-2.3.3.zip",
			FileName:      "totalRP3_Extended-2.3.3.zip",
			FileDate:      "2026-04-21T21:12:58Z",
			SourceURL:     "https://github.com/Total-RP/Total-RP-3-Extended/releases/tag/2.3.3",
			CurseForgeURL: "https://www.curseforge.com/wow/addons/total-rp-3-extended/files",
			License:       "Apache-2.0",
		},
	},
}

func (s *Server) getTRP3Latest(c *gin.Context) {
	latest, err := s.loadTRP3Latest(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	publicLatest := cloneTRP3Latest(latest)
	for i := range publicLatest.Addons {
		publicLatest.Addons[i].DownloadURL = buildPublicURL(
			c,
			"/api/v1/addon/trp3/download/"+url.PathEscape(publicLatest.Addons[i].ID),
		)
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, publicLatest)
}

func (s *Server) downloadTRP3Addon(c *gin.Context) {
	addonID := strings.TrimSpace(c.Param("id"))
	project, ok := findTRP3GitHubProject(addonID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown TRP3 addon"})
		return
	}

	latest, err := s.loadTRP3Latest(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var addon *TRP3AddonLatestInfo
	for i := range latest.Addons {
		if latest.Addons[i].ID == addonID {
			addon = &latest.Addons[i]
			break
		}
	}
	if addon == nil || strings.TrimSpace(addon.DownloadURL) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "TRP3 download metadata unavailable"})
		return
	}

	fileName := sanitizeDownloadFileName(addon.FileName)
	if fileName == "" {
		fileName = project.filePrefix + addon.LatestVersion + ".zip"
	}

	if mirror, ok := s.findTRP3MirrorAddon(addonID); ok &&
		mirror.LatestVersion == addon.LatestVersion &&
		sanitizeDownloadFileName(mirror.FileName) == fileName {
		cacheKey := s.trp3MirrorAddonCacheKey(project, mirror)
		if s.serveTRP3AddonFromLocalCache(c, cacheKey, fileName) {
			return
		}
		if s.serveTRP3AddonFromOSSCache(c, cacheKey, fileName) {
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "TRP3 自有镜像包暂不可用，请重新上传该版本"})
		return
	}

	if err := validateTRP3RemoteDownloadURL(addon.DownloadURL, project); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	cacheKey := s.trp3AddonCacheKey(project, addon, fileName)
	if s.serveTRP3AddonFromCache(c, cacheKey, fileName) {
		return
	}

	if s.trp3AddonCacheEnabled() {
		lock := s.trp3AddonCacheLock(cacheKey)
		lock.Lock()
		defer lock.Unlock()

		if s.serveTRP3AddonFromCache(c, cacheKey, fileName) {
			return
		}
	}

	resp, err := s.fetchTRP3Download(c.Request.Context(), addon.DownloadURL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	if maxBytes := s.trp3AddonMaxDownloadBytes(); maxBytes > 0 && resp.ContentLength > maxBytes {
		resp.Body.Close()
		c.JSON(http.StatusBadGateway, gin.H{"error": "TRP3 插件包超过服务器允许的最大下载大小"})
		return
	}

	body := io.ReadCloser(resp.Body)
	if s.trp3AddonCacheEnabled() {
		body = s.wrapTRP3AddonDownloadCache(body, cacheKey)
	}
	defer body.Close()

	headers := trp3AddonDownloadHeaders(fileName, "remote")
	c.DataFromReader(http.StatusOK, resp.ContentLength, trp3AddonContentType(fileName), body, headers)
}

func (s *Server) listTRP3MirrorAddons(c *gin.Context) {
	manifest, err := s.loadTRP3MirrorManifest()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	latest, _ := s.loadTRP3Latest(c.Request.Context())
	latestByID := make(map[string]TRP3AddonLatestInfo)
	if latest != nil {
		for _, addon := range latest.Addons {
			latestByID[addon.ID] = addon
		}
	}

	addons := make([]TRP3MirrorAddonInfo, 0, len(trp3GitHubProjects))
	for _, project := range trp3GitHubProjects {
		if mirror, ok := manifest.Addons[project.id]; ok {
			mirror.DownloadURL = buildPublicURL(c, "/api/v1/addon/trp3/download/"+url.PathEscape(project.id))
			mirror.CacheKey = s.trp3MirrorAddonCacheKey(project, mirror)
			mirror.CacheSource = s.trp3MirrorAddonCacheSource(project, mirror)
			addons = append(addons, mirror)
			continue
		}

		latestAddon := latestByID[project.id]
		if latestAddon.ID == "" {
			latestAddon = project.fallback
		}
		root, toc := trp3AddonExpectedPackage(project)
		addons = append(addons, TRP3MirrorAddonInfo{
			ID:            project.id,
			Name:          project.name,
			ProjectID:     project.projectID,
			Repository:    project.owner + "/" + project.repo,
			LatestVersion: latestAddon.LatestVersion,
			FileName:      latestAddon.FileName,
			FileDate:      latestAddon.FileDate,
			Source:        "github",
			DownloadURL:   buildPublicURL(c, "/api/v1/addon/trp3/download/"+url.PathEscape(project.id)),
			CacheSource:   "not_mirrored",
			ExpectedRoot:  root,
			ExpectedTOC:   toc,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"updatedAt": manifest.UpdatedAt,
		"addons":    addons,
		"note":      "版主上传的镜像包会优先覆盖公开 TRP3 最新版本接口；未上传的项目继续使用 GitHub Releases 兜底。",
	})
}

func (s *Server) uploadTRP3MirrorAddon(c *gin.Context) {
	addonID := strings.TrimSpace(firstNonEmpty(c.PostForm("addon_id"), c.PostForm("addonId")))
	project, ok := findTRP3GitHubProject(addonID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown TRP3 addon"})
		return
	}

	version := normalizeReleaseVersion(c.PostForm("version"))
	if version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version is required"})
		return
	}
	if sanitized := sanitizeTRP3CacheSegment(version); sanitized == "" || sanitized != version {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version only supports letters, numbers, dots, hyphens, and underscores"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "zip file is required"})
		return
	}
	originalFileName := sanitizeDownloadFileName(fileHeader.Filename)
	if !strings.HasSuffix(strings.ToLower(originalFileName), ".zip") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only zip addon packages are supported"})
		return
	}
	maxBytes := s.trp3AddonMaxDownloadBytes()
	if maxBytes > 0 && fileHeader.Size > maxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TRP3 插件包超过服务器允许的最大上传大小"})
		return
	}

	fileName := project.filePrefix + version + ".zip"
	addonInfo := TRP3MirrorAddonInfo{
		ID:               project.id,
		Name:             project.name,
		ProjectID:        project.projectID,
		Repository:       project.owner + "/" + project.repo,
		LatestVersion:    version,
		FileName:         fileName,
		FileDate:         time.Now().UTC().Format(time.RFC3339),
		Source:           "mirror",
		UploadedBy:       c.GetUint("userID"),
		UploadedByName:   strings.TrimSpace(c.GetString("username")),
		UploadedAt:       time.Now().UTC().Format(time.RFC3339),
		SizeBytes:        fileHeader.Size,
		OriginalFileName: originalFileName,
	}
	addonInfo.ExpectedRoot, addonInfo.ExpectedTOC = trp3AddonExpectedPackage(project)
	cacheKey := s.trp3MirrorAddonCacheKey(project, addonInfo)

	localPath, err := s.trp3AddonLocalCachePath(cacheKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create addon mirror directory"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer src.Close()

	tmpFile, err := os.CreateTemp(filepath.Dir(localPath), "."+filepath.Base(localPath)+".*.tmp")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp addon package"})
		return
	}
	tmpPath := tmpFile.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	written, err := io.Copy(tmpFile, io.LimitReader(src, maxBytes+1))
	closeErr := tmpFile.Close()
	if err != nil || closeErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store uploaded addon package"})
		return
	}
	if maxBytes > 0 && written > maxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TRP3 插件包超过服务器允许的最大上传大小"})
		return
	}
	addonInfo.SizeBytes = written

	tocVersion, err := validateTRP3AddonZipPackage(tmpPath, project, version)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if tocVersion != "" {
		addonInfo.LatestVersion = normalizeReleaseVersion(tocVersion)
		if addonInfo.LatestVersion != version {
			addonInfo.FileName = project.filePrefix + addonInfo.LatestVersion + ".zip"
			cacheKey = s.trp3MirrorAddonCacheKey(project, addonInfo)
			nextLocalPath, pathErr := s.trp3AddonLocalCachePath(cacheKey)
			if pathErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": pathErr.Error()})
				return
			}
			if err := os.MkdirAll(filepath.Dir(nextLocalPath), 0755); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create addon mirror directory"})
				return
			}
			localPath = nextLocalPath
		}
	}

	_ = os.Remove(localPath)
	if err := os.Rename(tmpPath, localPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish uploaded addon package"})
		return
	}
	committed = true

	if s.ossEnabled() {
		if err := s.uploadFileToOSS(s.buildOSSKey(cacheKey, ""), localPath, trp3AddonContentType(fileName)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "uploaded locally but failed to sync addon package to OSS: " + err.Error()})
			return
		}
	}

	if err := s.upsertTRP3MirrorAddon(addonInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.invalidateTRP3LatestCache()
	addonInfo.DownloadURL = buildPublicURL(c, "/api/v1/addon/trp3/download/"+url.PathEscape(project.id))
	addonInfo.CacheKey = cacheKey
	addonInfo.CacheSource = s.trp3MirrorAddonCacheSource(project, addonInfo)
	c.JSON(http.StatusOK, gin.H{"message": "TRP3 镜像包已发布", "addon": addonInfo})
}

func (s *Server) loadTRP3Latest(ctx context.Context) (*TRP3LatestResponse, error) {
	now := time.Now()

	s.trp3LatestMu.Lock()
	if s.trp3LatestCache != nil && now.Before(s.trp3LatestCacheUntil) {
		cached := *s.trp3LatestCache
		s.trp3LatestMu.Unlock()
		return &cached, nil
	}
	s.trp3LatestMu.Unlock()

	var result *TRP3LatestResponse
	if !s.cfg.TRP3Addons.Enabled {
		result = s.fallbackTRP3Latest("fallback", "TRP3 GitHub 查询未启用，当前使用服务器内置版本信息。")
	} else {
		result = s.fetchTRP3LatestFromGitHub(ctx)
	}

	cacheTTL := time.Duration(s.cfg.TRP3Addons.CacheTTLMinutes) * time.Minute
	if cacheTTL <= 0 {
		cacheTTL = 6 * time.Hour
	}
	cachedUntil := now.Add(cacheTTL)
	s.overlayTRP3MirrorLatest(result)
	result.CachedUntil = cachedUntil.UTC().Format(time.RFC3339)

	s.trp3LatestMu.Lock()
	s.trp3LatestCache = result
	s.trp3LatestCacheUntil = cachedUntil
	s.trp3LatestMu.Unlock()

	cached := *result
	return &cached, nil
}

func cloneTRP3Latest(latest *TRP3LatestResponse) *TRP3LatestResponse {
	if latest == nil {
		return nil
	}
	cloned := *latest
	cloned.Addons = append([]TRP3AddonLatestInfo(nil), latest.Addons...)
	return &cloned
}

func (s *Server) fallbackTRP3Latest(source string, note string) *TRP3LatestResponse {
	addons := make([]TRP3AddonLatestInfo, 0, len(trp3GitHubProjects))
	for _, project := range trp3GitHubProjects {
		addon := project.fallback
		addon.DownloadURL = s.rewriteGitHubDownloadURL(addon.DownloadURL)
		addons = append(addons, addon)
	}
	return &TRP3LatestResponse{
		Source: source,
		Note:   note,
		Addons: addons,
	}
}

func (s *Server) overlayTRP3MirrorLatest(latest *TRP3LatestResponse) {
	if latest == nil {
		return
	}
	manifest, err := s.loadTRP3MirrorManifest()
	if err != nil || len(manifest.Addons) == 0 {
		return
	}

	mirrored := make([]string, 0, len(manifest.Addons))
	for i := range latest.Addons {
		mirror, ok := manifest.Addons[latest.Addons[i].ID]
		if !ok || mirror.LatestVersion == "" || mirror.FileName == "" {
			continue
		}
		project, projectOK := findTRP3GitHubProject(mirror.ID)
		if !projectOK {
			continue
		}
		latest.Addons[i] = mirror.toLatestInfo(project)
		mirrored = append(mirrored, project.name)
	}

	if len(mirrored) == 0 {
		return
	}
	if latest.Source == "" || latest.Source == "github" || latest.Source == "fallback" {
		latest.Source = "mirror"
	} else if latest.Source != "mirror" && latest.Source != "mixed_mirror" {
		latest.Source = "mixed_mirror"
	}
	mirrorNote := "已优先使用 RPBox 自有镜像分发：" + strings.Join(mirrored, "、") + "。"
	if strings.TrimSpace(latest.Note) == "" {
		latest.Note = mirrorNote
	} else if !strings.Contains(latest.Note, "RPBox 自有镜像") {
		latest.Note = mirrorNote + latest.Note
	}
}

func (m TRP3MirrorAddonInfo) toLatestInfo(project trp3GitHubProject) TRP3AddonLatestInfo {
	sourceURL := strings.TrimSpace(project.fallback.SourceURL)
	if m.LatestVersion != "" {
		sourceURL = fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", project.owner, project.repo, m.LatestVersion)
	}
	return TRP3AddonLatestInfo{
		ID:            project.id,
		Name:          project.name,
		ProjectID:     project.projectID,
		Repository:    project.owner + "/" + project.repo,
		LatestVersion: m.LatestVersion,
		DownloadURL:   "mirror://" + project.id + "/" + m.LatestVersion,
		FileName:      m.FileName,
		FileDate:      firstNonEmpty(m.FileDate, m.UploadedAt),
		SourceURL:     sourceURL,
		CurseForgeURL: project.curseForgeURL,
		License:       "Apache-2.0",
	}
}

func (s *Server) findTRP3MirrorAddon(addonID string) (TRP3MirrorAddonInfo, bool) {
	manifest, err := s.loadTRP3MirrorManifest()
	if err != nil || manifest == nil || manifest.Addons == nil {
		return TRP3MirrorAddonInfo{}, false
	}
	mirror, ok := manifest.Addons[addonID]
	if !ok || strings.TrimSpace(mirror.LatestVersion) == "" || strings.TrimSpace(mirror.FileName) == "" {
		return TRP3MirrorAddonInfo{}, false
	}
	return mirror, true
}

func (s *Server) loadTRP3MirrorManifest() (*TRP3MirrorManifest, error) {
	s.trp3MirrorManifestMu.Lock()
	defer s.trp3MirrorManifestMu.Unlock()
	return s.loadTRP3MirrorManifestLocked()
}

func (s *Server) loadTRP3MirrorManifestLocked() (*TRP3MirrorManifest, error) {
	localPath, err := s.trp3MirrorManifestLocalPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read TRP3 mirror manifest: %w", err)
		}
		if s.ossEnabled() {
			body, _, _, ossErr := s.readObjectFromOSS(s.buildOSSKey(s.trp3MirrorManifestCacheKey(), ""))
			if ossErr == nil {
				defer body.Close()
				data, err = io.ReadAll(io.LimitReader(body, 4<<20))
				if err != nil {
					return nil, fmt.Errorf("failed to read TRP3 mirror manifest from OSS: %w", err)
				}
				_ = os.MkdirAll(filepath.Dir(localPath), 0755)
				_ = os.WriteFile(localPath, data, 0644)
			}
		}
		if len(data) == 0 {
			return &TRP3MirrorManifest{
				UpdatedAt: "",
				Addons:    map[string]TRP3MirrorAddonInfo{},
			}, nil
		}
	}

	var manifest TRP3MirrorManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse TRP3 mirror manifest: %w", err)
	}
	if manifest.Addons == nil {
		manifest.Addons = map[string]TRP3MirrorAddonInfo{}
	}
	return &manifest, nil
}

func (s *Server) upsertTRP3MirrorAddon(addon TRP3MirrorAddonInfo) error {
	s.trp3MirrorManifestMu.Lock()
	defer s.trp3MirrorManifestMu.Unlock()

	manifest, err := s.loadTRP3MirrorManifestLocked()
	if err != nil {
		return err
	}
	if manifest.Addons == nil {
		manifest.Addons = map[string]TRP3MirrorAddonInfo{}
	}
	addon.Source = "mirror"
	addon.CacheKey = ""
	addon.CacheSource = ""
	addon.DownloadURL = ""
	manifest.Addons[addon.ID] = addon
	manifest.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return s.writeTRP3MirrorManifestLocked(manifest)
}

func (s *Server) writeTRP3MirrorManifestLocked(manifest *TRP3MirrorManifest) error {
	localPath, err := s.trp3MirrorManifestLocalPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create TRP3 mirror manifest directory: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode TRP3 mirror manifest: %w", err)
	}
	data = append(data, '\n')

	tmpFile, err := os.CreateTemp(filepath.Dir(localPath), ".manifest.*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp TRP3 mirror manifest: %w", err)
	}
	tmpPath := tmpFile.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp TRP3 mirror manifest: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp TRP3 mirror manifest: %w", err)
	}
	_ = os.Remove(localPath)
	if err := os.Rename(tmpPath, localPath); err != nil {
		return fmt.Errorf("failed to publish TRP3 mirror manifest: %w", err)
	}
	committed = true

	if s.ossEnabled() {
		if err := s.uploadToOSS(s.buildOSSKey(s.trp3MirrorManifestCacheKey(), ""), data, "application/json"); err != nil {
			return fmt.Errorf("failed to sync TRP3 mirror manifest to OSS: %w", err)
		}
	}
	return nil
}

func (s *Server) trp3MirrorManifestCacheKey() string {
	return path.Join(s.trp3AddonCacheSubdir(), "manifest.json")
}

func (s *Server) trp3MirrorManifestLocalPath() (string, error) {
	return s.trp3AddonLocalCachePath(s.trp3MirrorManifestCacheKey())
}

func (s *Server) trp3MirrorAddonCacheKey(project trp3GitHubProject, addon TRP3MirrorAddonInfo) string {
	version := sanitizeTRP3CacheSegment(addon.LatestVersion)
	if version == "" {
		version = "unknown"
	}
	fileName := sanitizeDownloadFileName(addon.FileName)
	if fileName == "" {
		fileName = project.filePrefix + version + ".zip"
	}
	return path.Join(s.trp3AddonCacheSubdir(), sanitizeTRP3CacheSegment(project.id), version, fileName)
}

func (s *Server) trp3MirrorAddonCacheSource(project trp3GitHubProject, addon TRP3MirrorAddonInfo) string {
	cacheKey := s.trp3MirrorAddonCacheKey(project, addon)
	if localPath, err := s.trp3AddonLocalCachePath(cacheKey); err == nil {
		if info, statErr := os.Stat(localPath); statErr == nil && !info.IsDir() && info.Size() > 0 {
			return "local"
		}
	}
	if s.ossEnabled() {
		body, _, _, err := s.readObjectFromOSS(s.buildOSSKey(cacheKey, ""))
		if err == nil {
			_ = body.Close()
			return "oss"
		}
	}
	return "missing"
}

func (s *Server) invalidateTRP3LatestCache() {
	s.trp3LatestMu.Lock()
	s.trp3LatestCache = nil
	s.trp3LatestCacheUntil = time.Time{}
	s.trp3LatestMu.Unlock()
}

func trp3AddonExpectedPackage(project trp3GitHubProject) (string, string) {
	switch project.id {
	case "total-rp-3":
		return "totalRP3", "totalRP3.toc"
	case "total-rp-3-extended":
		return "totalRP3_Extended", "totalRP3_Extended.toc"
	default:
		return "", ""
	}
}

func validateTRP3AddonZipPackage(filePath string, project trp3GitHubProject, expectedVersion string) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("无法打开 zip 插件包")
	}
	defer reader.Close()

	expectedRoot, expectedTOC := trp3AddonExpectedPackage(project)
	if expectedRoot == "" || expectedTOC == "" {
		return "", fmt.Errorf("unknown TRP3 addon")
	}

	foundRoot := false
	foundTOC := false
	tocVersion := ""
	expectedRootLower := strings.ToLower(expectedRoot)
	expectedTOCLower := strings.ToLower(expectedTOC)
	for _, file := range reader.File {
		cleanedName := path.Clean(strings.ReplaceAll(file.Name, "\\", "/"))
		if cleanedName == "." || strings.HasPrefix(cleanedName, "../") || strings.HasPrefix(cleanedName, "/") {
			return "", fmt.Errorf("插件包中包含不安全路径")
		}
		parts := strings.Split(cleanedName, "/")
		for index, part := range parts {
			if strings.ToLower(part) != expectedRootLower {
				continue
			}
			foundRoot = true
			if index+1 < len(parts) && strings.ToLower(parts[index+1]) == expectedTOCLower {
				foundTOC = true
				if tocVersion == "" {
					tocVersion = readTOCVersionFromZip(file)
				}
			}
		}
	}
	if !foundRoot {
		return "", fmt.Errorf("插件包中未找到 %s 目录", expectedRoot)
	}
	if !foundTOC {
		return "", fmt.Errorf("插件包中未找到 %s", expectedTOC)
	}
	if tocVersion != "" && normalizeReleaseVersion(tocVersion) != normalizeReleaseVersion(expectedVersion) {
		return "", fmt.Errorf("上传版本 %s 与 toc 版本 %s 不一致", expectedVersion, tocVersion)
	}
	return tocVersion, nil
}

func readTOCVersionFromZip(file *zip.File) string {
	body, err := file.Open()
	if err != nil {
		return ""
	}
	defer body.Close()

	data, err := io.ReadAll(io.LimitReader(body, 256<<10))
	if err != nil {
		return ""
	}
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "## Version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "## Version:"))
		}
	}
	return ""
}

func (s *Server) fetchTRP3LatestFromGitHub(ctx context.Context) *TRP3LatestResponse {
	addons := make([]TRP3AddonLatestInfo, 0, len(trp3GitHubProjects))
	fallbacks := make([]string, 0)

	for _, project := range trp3GitHubProjects {
		addon, err := s.fetchGitHubAddonLatest(ctx, project)
		if err != nil {
			fallbacks = append(fallbacks, fmt.Sprintf("%s: %v", project.name, err))
			fallback := project.fallback
			fallback.DownloadURL = s.rewriteGitHubDownloadURL(fallback.DownloadURL)
			addons = append(addons, fallback)
			continue
		}
		addons = append(addons, addon)
	}

	if len(fallbacks) == 0 {
		return &TRP3LatestResponse{
			Source: "github",
			Note:   "已从 Total RP GitHub Releases 获取 Total RP 3 与 Total RP 3: Extended 最新版本信息。",
			Addons: addons,
		}
	}

	source := "mixed"
	if len(fallbacks) == len(trp3GitHubProjects) {
		source = "fallback"
	}
	return &TRP3LatestResponse{
		Source: source,
		Note:   "部分 TRP3 GitHub 元数据暂时无法获取，已对失败项使用服务器内置兜底信息：" + strings.Join(fallbacks, "；"),
		Addons: addons,
	}
}

func (s *Server) fetchGitHubAddonLatest(ctx context.Context, project trp3GitHubProject) (TRP3AddonLatestInfo, error) {
	timeout := time.Duration(s.cfg.TRP3Addons.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	baseURL := strings.TrimRight(s.cfg.TRP3Addons.GitHubAPIBaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	requestURL := fmt.Sprintf("%s/repos/%s/%s/releases/latest", baseURL, project.owner, project.repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return TRP3AddonLatestInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "RPBox/0.2 TRP3 addon metadata")
	if token := strings.TrimSpace(s.cfg.TRP3Addons.GitHubToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client, err := s.trp3HTTPClient(timeout)
	if err != nil {
		return TRP3AddonLatestInfo{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return TRP3AddonLatestInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TRP3AddonLatestInfo{}, fmt.Errorf("GitHub HTTP %d", resp.StatusCode)
	}

	var release gitHubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return TRP3AddonLatestInfo{}, err
	}

	downloadURL, fileName := selectGitHubReleaseDownload(release, project)
	if strings.TrimSpace(downloadURL) == "" {
		return TRP3AddonLatestInfo{}, fmt.Errorf("GitHub release 未返回可用 zip")
	}
	if fileName == "" {
		fileName = fmt.Sprintf("%s-%s-source.zip", project.repo, release.TagName)
	}

	latestVersion := normalizeReleaseVersion(release.TagName)
	if latestVersion == "" {
		latestVersion = normalizeReleaseVersion(release.Name)
	}
	if latestVersion == "" {
		latestVersion = "unknown"
	}

	sourceURL := strings.TrimSpace(release.HTMLURL)
	if sourceURL == "" && release.TagName != "" {
		sourceURL = fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", project.owner, project.repo, release.TagName)
	}

	return TRP3AddonLatestInfo{
		ID:            project.id,
		Name:          project.name,
		ProjectID:     project.projectID,
		Repository:    project.owner + "/" + project.repo,
		LatestVersion: latestVersion,
		DownloadURL:   s.rewriteGitHubDownloadURL(downloadURL),
		FileName:      fileName,
		FileDate:      firstNonEmpty(release.PublishedAt, releaseAssetDate(release, fileName)),
		SourceURL:     sourceURL,
		CurseForgeURL: project.curseForgeURL,
		License:       "Apache-2.0",
	}, nil
}

func (s *Server) fetchTRP3Download(ctx context.Context, downloadURL string) (*http.Response, error) {
	timeout := time.Duration(s.cfg.TRP3Addons.TimeoutSeconds) * time.Second
	if timeout < 2*time.Minute {
		timeout = 2 * time.Minute
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/zip, application/octet-stream, */*")
	req.Header.Set("User-Agent", "RPBox/0.2 TRP3 addon download proxy")
	if parsed, err := url.Parse(downloadURL); err == nil &&
		strings.EqualFold(parsed.Hostname(), "api.github.com") &&
		strings.Contains(parsed.EscapedPath(), "/releases/assets/") {
		req.Header.Set("Accept", "application/octet-stream")
	}

	if token := strings.TrimSpace(s.cfg.TRP3Addons.GitHubToken); token != "" {
		if parsed, err := url.Parse(downloadURL); err == nil && strings.EqualFold(parsed.Hostname(), "api.github.com") {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	client, err := s.trp3HTTPClient(timeout)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载 TRP3 插件失败: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		detail := strings.TrimSpace(string(body))
		if detail != "" {
			return nil, fmt.Errorf("下载 TRP3 插件失败，HTTP %d: %s", resp.StatusCode, detail)
		}
		return nil, fmt.Errorf("下载 TRP3 插件失败，HTTP %d", resp.StatusCode)
	}

	return resp, nil
}

func selectGitHubReleaseDownload(release gitHubReleaseResponse, project trp3GitHubProject) (string, string) {
	for _, asset := range release.Assets {
		fileName := strings.TrimSpace(asset.Name)
		if fileName == "" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(fileName), ".zip") {
			continue
		}
		if project.filePrefix != "" && !strings.HasPrefix(fileName, project.filePrefix) {
			continue
		}
		downloadURL := firstNonEmpty(asset.BrowserDownloadURL, asset.URL)
		if strings.TrimSpace(downloadURL) == "" {
			continue
		}
		return downloadURL, fileName
	}

	if strings.TrimSpace(release.ZipballURL) == "" {
		return "", ""
	}
	return release.ZipballURL, fmt.Sprintf("%s-%s-source.zip", project.repo, release.TagName)
}

func (s *Server) trp3HTTPClient(timeout time.Duration) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	proxyValue := strings.TrimSpace(s.cfg.TRP3Addons.ProxyURL)
	if proxyValue != "" {
		proxyURL, err := parseTRP3ProxyURL(proxyValue)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	} else {
		transport.Proxy = http.ProxyFromEnvironment
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}

func parseTRP3ProxyURL(raw string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("empty proxy url")
	}

	if !strings.Contains(value, "://") {
		parts := strings.SplitN(value, ":", 4)
		if len(parts) == 4 && parts[0] != "" && parts[1] != "" && parts[2] != "" {
			value = (&url.URL{
				Scheme: "socks5",
				Host:   net.JoinHostPort(parts[0], parts[1]),
				User:   url.UserPassword(parts[2], parts[3]),
			}).String()
		} else {
			value = "http://" + value
		}
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return nil, errors.New("TRP3 proxy url is invalid")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "socks5", "socks5h":
	default:
		return nil, errors.New("TRP3 proxy url only supports http, https, socks5, or socks5h")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return nil, errors.New("TRP3 proxy url host is empty")
	}
	return parsed, nil
}

func (s *Server) trp3AddonCacheEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.TRP3Addons.CacheEnabled
}

func (s *Server) trp3AddonCacheSubdir() string {
	if s == nil || s.cfg == nil {
		return "cache/addons/trp3"
	}
	subdir := strings.TrimSpace(s.cfg.TRP3Addons.CacheSubdir)
	if subdir == "" {
		subdir = "cache/addons/trp3"
	}
	cleaned := path.Clean("/" + strings.Trim(subdir, "/"))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." || cleaned == "" {
		return "cache/addons/trp3"
	}
	return cleaned
}

func (s *Server) trp3AddonCacheKey(project trp3GitHubProject, addon *TRP3AddonLatestInfo, fileName string) string {
	version := "unknown"
	if addon != nil {
		if cleaned := sanitizeTRP3CacheSegment(addon.LatestVersion); cleaned != "" {
			version = cleaned
		}
	}
	name := sanitizeDownloadFileName(fileName)
	if name == "" {
		name = project.filePrefix + version + ".zip"
	}
	return path.Join(s.trp3AddonCacheSubdir(), sanitizeTRP3CacheSegment(project.id), version, name)
}

func sanitizeTRP3CacheSegment(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), ".")
}

func (s *Server) trp3AddonLocalCachePath(cacheKey string) (string, error) {
	if s == nil || s.cfg == nil {
		return "", errors.New("server config is unavailable")
	}
	storageRoot := strings.TrimSpace(s.cfg.Storage.Path)
	if storageRoot == "" {
		storageRoot = "storage"
	}
	absRoot, err := filepath.Abs(filepath.Clean(storageRoot))
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(filepath.Clean(filepath.Join(absRoot, filepath.FromSlash(cacheKey))))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return "", err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", errors.New("TRP3 cache path escapes storage root")
	}
	return absPath, nil
}

func (s *Server) serveTRP3AddonFromCache(c *gin.Context, cacheKey string, fileName string) bool {
	if !s.trp3AddonCacheEnabled() {
		return false
	}
	if s.serveTRP3AddonFromLocalCache(c, cacheKey, fileName) {
		return true
	}
	return s.serveTRP3AddonFromOSSCache(c, cacheKey, fileName)
}

func (s *Server) serveTRP3AddonFromLocalCache(c *gin.Context, cacheKey string, fileName string) bool {
	localPath, err := s.trp3AddonLocalCachePath(cacheKey)
	if err != nil {
		return false
	}

	file, err := os.Open(localPath)
	if err != nil {
		return false
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.IsDir() || info.Size() <= 0 {
		return false
	}

	for key, value := range trp3AddonDownloadHeaders(fileName, "local") {
		c.Header(key, value)
	}
	c.Header("Content-Type", trp3AddonContentType(fileName))
	http.ServeContent(c.Writer, c.Request, fileName, info.ModTime(), file)
	return true
}

func (s *Server) serveTRP3AddonFromOSSCache(c *gin.Context, cacheKey string, fileName string) bool {
	if !s.ossEnabled() {
		return false
	}

	body, contentType, contentLength, err := s.readObjectFromOSS(s.buildOSSKey(cacheKey, ""))
	if err != nil {
		return false
	}
	defer body.Close()

	if contentType == "" || contentType == "application/octet-stream" {
		contentType = trp3AddonContentType(fileName)
	}
	headers := trp3AddonDownloadHeaders(fileName, "oss")
	c.DataFromReader(http.StatusOK, contentLength, contentType, body, headers)
	return true
}

func (s *Server) wrapTRP3AddonDownloadCache(source io.ReadCloser, cacheKey string) io.ReadCloser {
	localPath, err := s.trp3AddonLocalCachePath(cacheKey)
	if err != nil {
		log.Printf("[TRP3] cache path unavailable: %v", err)
		return source
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		log.Printf("[TRP3] cache directory unavailable: %v", err)
		return source
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(localPath), "."+filepath.Base(localPath)+".*.tmp")
	if err != nil {
		log.Printf("[TRP3] cache temp file unavailable: %v", err)
		return source
	}

	return &trp3CachingReadCloser{
		source:    source,
		cacheFile: tmpFile,
		tmpPath:   tmpFile.Name(),
		finalPath: localPath,
		maxBytes:  s.trp3AddonMaxDownloadBytes(),
		onCached: func(finalPath string) {
			if s.ossEnabled() {
				ossKey := s.buildOSSKey(cacheKey, "")
				go func() {
					if err := s.uploadFileToOSS(ossKey, finalPath, trp3AddonContentType(filepath.Base(finalPath))); err != nil {
						log.Printf("[TRP3] upload cache to OSS failed: %v", err)
					}
				}()
			}
		},
	}
}

func (s *Server) trp3AddonCacheLock(cacheKey string) *sync.Mutex {
	s.trp3CacheLocksMu.Lock()
	defer s.trp3CacheLocksMu.Unlock()

	if s.trp3CacheLocks == nil {
		s.trp3CacheLocks = make(map[string]*sync.Mutex)
	}
	lock := s.trp3CacheLocks[cacheKey]
	if lock == nil {
		lock = &sync.Mutex{}
		s.trp3CacheLocks[cacheKey] = lock
	}
	return lock
}

func (s *Server) trp3AddonMaxDownloadBytes() int64 {
	if s == nil || s.cfg == nil {
		return 256 << 20
	}
	maxMB := s.cfg.TRP3Addons.MaxDownloadMB
	if maxMB <= 0 {
		maxMB = 256
	}
	return int64(maxMB) << 20
}

func trp3AddonDownloadHeaders(fileName string, cacheSource string) map[string]string {
	return map[string]string{
		"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, sanitizeDownloadFileName(fileName)),
		"Cache-Control":       "private, max-age=300",
		"X-RPBox-Addon-Cache": cacheSource,
	}
}

func trp3AddonContentType(fileName string) string {
	if byExt := mime.TypeByExtension(filepath.Ext(fileName)); byExt != "" {
		return strings.TrimSpace(strings.Split(byExt, ";")[0])
	}
	return "application/zip"
}

type trp3CachingReadCloser struct {
	source    io.ReadCloser
	cacheFile *os.File
	tmpPath   string
	finalPath string
	maxBytes  int64
	bytesRead int64
	finalized bool
	onCached  func(string)
}

func (r *trp3CachingReadCloser) Read(p []byte) (int, error) {
	n, err := r.source.Read(p)
	if n > 0 && r.cacheFile != nil {
		r.bytesRead += int64(n)
		if r.maxBytes > 0 && r.bytesRead > r.maxBytes {
			r.closeCacheFile(false)
			return n, errors.New("TRP3 addon download exceeded max cache size")
		}
		if _, writeErr := r.cacheFile.Write(p[:n]); writeErr != nil {
			log.Printf("[TRP3] write cache failed: %v", writeErr)
			r.closeCacheFile(false)
		}
	}
	if err == io.EOF {
		r.closeCacheFile(true)
	}
	return n, err
}

func (r *trp3CachingReadCloser) Close() error {
	r.closeCacheFile(false)
	return r.source.Close()
}

func (r *trp3CachingReadCloser) closeCacheFile(commit bool) {
	if r.cacheFile == nil {
		return
	}

	tmpPath := r.tmpPath
	finalPath := r.finalPath
	file := r.cacheFile
	r.cacheFile = nil

	if err := file.Close(); err != nil {
		log.Printf("[TRP3] close cache failed: %v", err)
		commit = false
	}

	if !commit || r.finalized {
		_ = os.Remove(tmpPath)
		return
	}

	_ = os.Remove(finalPath)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		log.Printf("[TRP3] commit cache failed: %v", err)
		_ = os.Remove(tmpPath)
		return
	}

	r.finalized = true
	if r.onCached != nil {
		r.onCached(finalPath)
	}
}

func releaseAssetDate(release gitHubReleaseResponse, fileName string) string {
	for _, asset := range release.Assets {
		if asset.Name == fileName {
			return firstNonEmpty(asset.UpdatedAt, asset.CreatedAt)
		}
	}
	return ""
}

func (s *Server) rewriteGitHubDownloadURL(rawURL string) string {
	mirror := strings.TrimSpace(s.cfg.TRP3Addons.GitHubDownloadMirrorBaseURL)
	if mirror == "" || strings.TrimSpace(rawURL) == "" {
		return rawURL
	}

	if strings.Contains(mirror, "{raw_url}") {
		return strings.ReplaceAll(mirror, "{raw_url}", rawURL)
	}
	if strings.Contains(mirror, "{url}") {
		return strings.ReplaceAll(mirror, "{url}", url.QueryEscape(rawURL))
	}
	return strings.TrimRight(mirror, "/") + "/" + rawURL
}

func findTRP3GitHubProject(addonID string) (trp3GitHubProject, bool) {
	for _, project := range trp3GitHubProjects {
		if project.id == addonID {
			return project, true
		}
	}
	return trp3GitHubProject{}, false
}

func validateTRP3RemoteDownloadURL(rawURL string, project trp3GitHubProject) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("TRP3 下载地址格式无效")
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("TRP3 下载必须使用 HTTPS")
	}

	decoded := decodeURLRepeated(rawURL)
	repository := strings.ToLower(project.owner + "/" + project.repo)
	if !strings.Contains(decoded, "github.com/"+repository+"/releases/download/") &&
		!strings.Contains(decoded, "api.github.com/repos/"+repository+"/zipball/") &&
		!strings.Contains(decoded, "api.github.com/repos/"+repository+"/releases/assets/") &&
		!strings.Contains(decoded, "codeload.github.com/"+repository+"/zip/") {
		return fmt.Errorf("TRP3 下载地址必须来自 Total RP 官方 GitHub 仓库或其镜像")
	}

	if !strings.Contains(decoded, ".zip") &&
		!strings.Contains(decoded, "/zipball/") &&
		!strings.Contains(decoded, "/releases/assets/") &&
		!strings.Contains(decoded, "/repos/total-rp/") {
		return fmt.Errorf("TRP3 下载地址必须指向 zip 包或 GitHub zipball")
	}

	return nil
}

func decodeURLRepeated(value string) string {
	decoded := strings.ToLower(strings.TrimSpace(value))
	for i := 0; i < 4; i++ {
		next, err := url.QueryUnescape(decoded)
		if err != nil || next == decoded {
			break
		}
		decoded = next
	}
	return decoded
}

func sanitizeDownloadFileName(fileName string) string {
	cleaned := strings.TrimSpace(fileName)
	cleaned = strings.ReplaceAll(cleaned, `"`, "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "/", "")
	cleaned = strings.ReplaceAll(cleaned, "\\", "")
	return cleaned
}

func normalizeReleaseVersion(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
