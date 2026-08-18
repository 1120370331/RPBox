package api

import (
	"archive/zip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rpbox/server/internal/config"
)

func TestGitHubReleasePageLatestBuildsDownloadFromPublicReleaseRedirect(t *testing.T) {
	releaseURL, err := url.Parse("https://github.com/Total-RP/Total-RP-3/releases/tag/3.4.1")
	if err != nil {
		t.Fatalf("parse release URL: %v", err)
	}

	latest, err := gitHubReleasePageLatest(trp3GitHubProjects[0], releaseURL)
	if err != nil {
		t.Fatalf("build latest info: %v", err)
	}
	if latest.LatestVersion != "3.4.1" {
		t.Fatalf("expected version 3.4.1, got %q", latest.LatestVersion)
	}
	if latest.FileName != "totalRP3-3.4.1.zip" {
		t.Fatalf("unexpected filename %q", latest.FileName)
	}
	if latest.DownloadURL != "https://github.com/Total-RP/Total-RP-3/releases/download/3.4.1/totalRP3-3.4.1.zip" {
		t.Fatalf("unexpected download URL %q", latest.DownloadURL)
	}
}

func TestGitHubReleasePageLatestRejectsUnexpectedRedirect(t *testing.T) {
	releaseURL, err := url.Parse("https://example.com/Total-RP/Total-RP-3/releases/tag/3.4.1")
	if err != nil {
		t.Fatalf("parse release URL: %v", err)
	}

	if _, err := gitHubReleasePageLatest(trp3GitHubProjects[0], releaseURL); err == nil {
		t.Fatal("expected an unexpected redirect to be rejected")
	}
}

func TestParseTRP3ProxyURLSupportsRoxyColonFormat(t *testing.T) {
	proxyURL, err := parseTRP3ProxyURL("192.0.2.10:5782:user:pass")
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}

	if proxyURL.Scheme != "socks5" {
		t.Fatalf("expected socks5 scheme, got %q", proxyURL.Scheme)
	}
	if proxyURL.Host != "192.0.2.10:5782" {
		t.Fatalf("expected host 192.0.2.10:5782, got %q", proxyURL.Host)
	}
	username := proxyURL.User.Username()
	password, ok := proxyURL.User.Password()
	if username != "user" || !ok || password != "pass" {
		t.Fatalf("unexpected proxy credentials: username=%q password=%q ok=%v", username, password, ok)
	}
}

func TestParseTRP3ProxyURLRejectsUnsupportedScheme(t *testing.T) {
	if _, err := parseTRP3ProxyURL("ftp://proxy.example.com:21"); err == nil {
		t.Fatal("expected unsupported scheme error")
	}
}

func TestTRP3AddonCachePathStaysUnderStorageRoot(t *testing.T) {
	storageRoot := t.TempDir()
	server := &Server{
		cfg: &config.Config{
			Storage: config.StorageConfig{Path: storageRoot},
			TRP3Addons: config.TRP3AddonsConfig{
				CacheSubdir: "cache/addons/trp3",
			},
		},
	}

	cacheKey := server.trp3AddonCacheKey(
		trp3GitHubProjects[0],
		&TRP3AddonLatestInfo{LatestVersion: "3.3.6"},
		`..\totalRP3-3.3.6.zip`,
	)
	localPath, err := server.trp3AddonLocalCachePath(cacheKey)
	if err != nil {
		t.Fatalf("local cache path: %v", err)
	}

	rel, err := filepath.Rel(storageRoot, localPath)
	if err != nil {
		t.Fatalf("relative cache path: %v", err)
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		t.Fatalf("cache path escaped storage root: %s", localPath)
	}
	if !strings.Contains(filepath.ToSlash(localPath), "cache/addons/trp3/total-rp-3/3.3.6") {
		t.Fatalf("cache path does not include addon/version partition: %s", localPath)
	}

	if _, err := server.trp3AddonLocalCachePath("../../outside.zip"); err == nil {
		t.Fatal("expected traversal cache key to be rejected")
	}
}

func TestTRP3MirrorOverlayPrefersManualManifest(t *testing.T) {
	storageRoot := t.TempDir()
	server := &Server{
		cfg: &config.Config{
			Storage: config.StorageConfig{Path: storageRoot},
			TRP3Addons: config.TRP3AddonsConfig{
				CacheSubdir: "cache/addons/trp3",
			},
		},
	}

	err := server.upsertTRP3MirrorAddon(TRP3MirrorAddonInfo{
		ID:            "total-rp-3",
		Name:          "Total RP 3",
		ProjectID:     75973,
		Repository:    "Total-RP/Total-RP-3",
		LatestVersion: "9.9.9",
		FileName:      "totalRP3-9.9.9.zip",
		Source:        "mirror",
	})
	if err != nil {
		t.Fatalf("upsert mirror addon: %v", err)
	}

	latest := server.fallbackTRP3Latest("github", "github metadata")
	server.overlayTRP3MirrorLatest(latest)

	var trp TRP3AddonLatestInfo
	for _, addon := range latest.Addons {
		if addon.ID == "total-rp-3" {
			trp = addon
			break
		}
	}
	if trp.LatestVersion != "9.9.9" {
		t.Fatalf("expected mirror version 9.9.9, got %q", trp.LatestVersion)
	}
	if trp.DownloadURL != "mirror://total-rp-3/9.9.9" {
		t.Fatalf("expected mirror download url, got %q", trp.DownloadURL)
	}
	if latest.Source != "mirror" {
		t.Fatalf("expected mirror source, got %q", latest.Source)
	}
}

func TestTRP3MirrorOverlayDoesNotReplaceNewerUpstreamRelease(t *testing.T) {
	storageRoot := t.TempDir()
	server := &Server{
		cfg: &config.Config{
			Storage: config.StorageConfig{Path: storageRoot},
			TRP3Addons: config.TRP3AddonsConfig{
				CacheSubdir: "cache/addons/trp3",
			},
		},
	}

	err := server.upsertTRP3MirrorAddon(TRP3MirrorAddonInfo{
		ID:            "total-rp-3",
		Name:          "Total RP 3",
		ProjectID:     75973,
		Repository:    "Total-RP/Total-RP-3",
		LatestVersion: "3.3.6",
		FileName:      "totalRP3-3.3.6.zip",
	})
	if err != nil {
		t.Fatalf("upsert mirror addon: %v", err)
	}

	latest := server.fallbackTRP3Latest("github", "github metadata")
	server.overlayTRP3MirrorLatest(latest)

	var trp TRP3AddonLatestInfo
	for _, addon := range latest.Addons {
		if addon.ID == "total-rp-3" {
			trp = addon
			break
		}
	}
	if trp.LatestVersion != "3.4.1" {
		t.Fatalf("expected newer upstream version to remain selected, got %q", trp.LatestVersion)
	}
}

func TestValidateTRP3AddonZipPackageRequiresExpectedTOC(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "totalRP3-3.3.6.zip")
	createTestZip(t, zipPath, map[string]string{
		"totalRP3/totalRP3.toc": "## Version: 3.3.6\n",
		"totalRP3/Core.lua":     "print('ok')\n",
	})

	version, err := validateTRP3AddonZipPackage(zipPath, trp3GitHubProjects[0], "3.3.6")
	if err != nil {
		t.Fatalf("validate trp3 zip: %v", err)
	}
	if version != "3.3.6" {
		t.Fatalf("expected toc version 3.3.6, got %q", version)
	}

	badZipPath := filepath.Join(t.TempDir(), "totalRP3-3.3.7.zip")
	createTestZip(t, badZipPath, map[string]string{
		"totalRP3/Core.lua": "print('missing toc')\n",
	})
	if _, err := validateTRP3AddonZipPackage(badZipPath, trp3GitHubProjects[0], "3.3.7"); err == nil {
		t.Fatal("expected missing toc to be rejected")
	}

	wrongLayoutZipPath := filepath.Join(t.TempDir(), "totalRP3-3.3.8.zip")
	createTestZip(t, wrongLayoutZipPath, map[string]string{
		"totalRP3/docs/totalRP3.toc": "## Version: 3.3.8\n",
		"totalRP3/Core.lua":          "print('wrong toc layout')\n",
	})
	if _, err := validateTRP3AddonZipPackage(wrongLayoutZipPath, trp3GitHubProjects[0], "3.3.8"); err == nil {
		t.Fatal("expected nested toc to be rejected")
	}
}

func createTestZip(t *testing.T, zipPath string, files map[string]string) {
	t.Helper()

	out, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer out.Close()

	writer := zip.NewWriter(out)
	defer writer.Close()

	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
}
