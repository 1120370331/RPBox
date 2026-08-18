package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain", input: "0.1.0", want: "0.1.0"},
		{name: "prefixed lower", input: "v0.1.0", want: "0.1.0"},
		{name: "prefixed upper", input: "V1.2.3", want: "1.2.3"},
		{name: "trim spaces", input: "  v2.0.1  ", want: "2.0.1"},
		{name: "empty", input: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeVersion(tt.input); got != tt.want {
				t.Fatalf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name    string
		latest  string
		current string
		want    bool
	}{
		{name: "major upgrade", latest: "2.0.0", current: "1.9.9", want: true},
		{name: "minor upgrade", latest: "1.2.0", current: "1.1.9", want: true},
		{name: "patch upgrade", latest: "1.2.4", current: "1.2.3", want: true},
		{name: "same version", latest: "1.2.3", current: "1.2.3", want: false},
		{name: "current newer", latest: "1.2.3", current: "1.2.4", want: false},
		{name: "supports prefix", latest: "v1.2.3", current: "1.2.2", want: true},
		{name: "supports prerelease stripping", latest: "1.2.3-beta", current: "1.2.2", want: true},
		{name: "prerelease increments", latest: "1.2.3-beta.2", current: "1.2.3-beta.1", want: true},
		{name: "same prerelease", latest: "1.2.3-beta.1", current: "1.2.3-beta.1", want: false},
		{name: "stable newer than prerelease", latest: "1.2.3", current: "1.2.3-beta.2", want: true},
		{name: "same core prerelease older than stable", latest: "1.2.3-beta.2", current: "1.2.3", want: false},
		{name: "invalid latest", latest: "beta", current: "1.2.2", want: true},
		{name: "invalid equal fallback", latest: "beta", current: "beta", want: false},
		{name: "empty latest", latest: "", current: "1.2.2", want: false},
		{name: "empty current", latest: "1.2.2", current: "", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNewerVersion(tt.latest, tt.current); got != tt.want {
				t.Fatalf("isNewerVersion(latest=%q, current=%q) = %v, want %v", tt.latest, tt.current, got, tt.want)
			}
		})
	}
}

func TestIsBetaReleasePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "latest.json", want: false},
		{path: "latest-beta.json", want: true},
		{path: "0.2.3/RPBox_0.2.3_x64-setup.exe", want: false},
		{path: "0.2.4-beta.1/RPBox_0.2.4-beta.1_x64-setup.exe", want: true},
		{path: "0.2.4-rc.1/RPBox_0.2.4-rc.1_x64-setup.exe", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isBetaReleasePath(tt.path); got != tt.want {
				t.Fatalf("isBetaReleasePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDesktopBetaUpdateRequiresEligibleUserAndSwitch(t *testing.T) {
	releaseRoot := t.TempDir()
	t.Chdir(releaseRoot)
	writeDesktopLatest(t, "latest.json", LatestRelease{
		LatestVersion: "0.2.36",
		Notes:         "stable",
		PubDate:       "2026-05-22T10:00:00Z",
		Channel:       updateChannelStable,
	})
	writeDesktopLatest(t, "latest-beta.json", LatestRelease{
		LatestVersion: "0.2.37-beta.1",
		Notes:         "beta",
		PubDate:       "2026-05-22T11:00:00Z",
		Channel:       updateChannelBeta,
	})
	writeReleaseFile(t, "0.2.37-beta.1", "RPBox_0.2.37-beta.1_x64-setup.exe.sig", "beta-signature")

	db := testutil.NewTestDB(t, &model.User{})
	normal := model.User{Username: "normal", Email: "normal@example.com", PassHash: "hash"}
	sponsor := model.User{Username: "sponsor", Email: "sponsor@example.com", PassHash: "hash", IsSponsor: true, SponsorLevel: 1}
	admin := model.User{Username: "admin", Email: "admin@example.com", PassHash: "hash", Role: "admin"}
	if err := db.Create(&normal).Error; err != nil {
		t.Fatalf("create normal user: %v", err)
	}
	if err := db.Create(&sponsor).Error; err != nil {
		t.Fatalf("create sponsor user: %v", err)
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	server := newTestServer(t, db)
	normalToken := newTestToken(t, normal)
	sponsorToken := newTestToken(t, sponsor)
	adminToken := newTestToken(t, admin)

	resp := performUpdaterRequest(server.router, "/api/v1/updater/windows/x86_64/0.2.36", normalToken, true)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("normal user with beta channel: expected 204, got %d body=%s", resp.Code, resp.Body.String())
	}

	resp = performUpdaterRequest(server.router, "/api/v1/updater/windows/x86_64/0.2.36", sponsorToken, false)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("sponsor without beta channel: expected 204, got %d body=%s", resp.Code, resp.Body.String())
	}

	resp = performUpdaterRequest(server.router, "/api/v1/updater/windows/x86_64/0.2.36", sponsorToken, true)
	if resp.Code != http.StatusOK {
		t.Fatalf("sponsor with beta channel: expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var update UpdateResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &update); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if update.Version != "0.2.37-beta.1" {
		t.Fatalf("expected beta version, got %q", update.Version)
	}
	if update.Channel != updateChannelBeta {
		t.Fatalf("expected beta channel, got %q", update.Channel)
	}
	if update.Signature != "beta-signature" {
		t.Fatalf("expected beta signature to be returned, got %q", update.Signature)
	}

	resp = performUpdaterRequest(server.router, "/api/v1/updater/windows/x86_64/0.2.36", adminToken, true)
	if resp.Code != http.StatusOK {
		t.Fatalf("admin with beta channel: expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestDesktopBetaReleaseFilesAreProtected(t *testing.T) {
	releaseRoot := t.TempDir()
	t.Chdir(releaseRoot)
	writeReleaseFile(t, "0.2.36", "RPBox_0.2.36_x64-setup.exe", "stable-installer")
	writeReleaseFile(t, "0.2.37-beta.1", "RPBox_0.2.37-beta.1_x64-setup.exe", "beta-installer")

	db := testutil.NewTestDB(t, &model.User{})
	normal := model.User{Username: "normal", Email: "normal@example.com", PassHash: "hash"}
	sponsor := model.User{Username: "sponsor", Email: "sponsor@example.com", PassHash: "hash", IsSponsor: true, SponsorLevel: 1}
	moderator := model.User{Username: "moderator", Email: "moderator@example.com", PassHash: "hash", Role: "moderator"}
	admin := model.User{Username: "admin", Email: "admin@example.com", PassHash: "hash", Role: "admin"}
	if err := db.Create(&[]model.User{normal, sponsor, moderator, admin}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Where("username = ?", "normal").First(&normal).Error; err != nil {
		t.Fatalf("reload normal user: %v", err)
	}
	if err := db.Where("username = ?", "sponsor").First(&sponsor).Error; err != nil {
		t.Fatalf("reload sponsor user: %v", err)
	}
	if err := db.Where("username = ?", "moderator").First(&moderator).Error; err != nil {
		t.Fatalf("reload moderator user: %v", err)
	}
	if err := db.Where("username = ?", "admin").First(&admin).Error; err != nil {
		t.Fatalf("reload admin user: %v", err)
	}

	server := newTestServer(t, db)

	stablePath := "/releases/0.2.36/RPBox_0.2.36_x64-setup.exe"
	betaPath := "/releases/0.2.37-beta.1/RPBox_0.2.37-beta.1_x64-setup.exe"

	resp := performRequest(server.router, http.MethodGet, stablePath, nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("stable file should be public: expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	resp = performRequest(server.router, http.MethodGet, betaPath, nil, newTestToken(t, normal))
	if resp.Code != http.StatusForbidden {
		t.Fatalf("normal user beta download: expected 403, got %d body=%s", resp.Code, resp.Body.String())
	}

	resp = performRequest(server.router, http.MethodGet, betaPath, nil, newTestToken(t, moderator))
	if resp.Code != http.StatusForbidden {
		t.Fatalf("moderator-only beta download: expected 403, got %d body=%s", resp.Code, resp.Body.String())
	}

	resp = performRequest(server.router, http.MethodGet, betaPath, nil, newTestToken(t, sponsor))
	if resp.Code != http.StatusOK {
		t.Fatalf("sponsor beta download: expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if resp.Body.String() != "beta-installer" {
		t.Fatalf("unexpected beta installer body %q", resp.Body.String())
	}

	resp = performRequest(server.router, http.MethodGet, betaPath, nil, newTestToken(t, admin))
	if resp.Code != http.StatusOK {
		t.Fatalf("admin beta download: expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestDesktopBetaCanUpdateToStableRelease(t *testing.T) {
	releaseRoot := t.TempDir()
	t.Chdir(releaseRoot)
	writeDesktopLatest(t, "latest.json", LatestRelease{
		LatestVersion: "0.2.37",
		Notes:         "stable promotion",
		PubDate:       "2026-05-22T12:00:00Z",
		Channel:       updateChannelStable,
	})
	writeDesktopLatest(t, "latest-beta.json", LatestRelease{
		LatestVersion: "0.2.37-beta.1",
		Notes:         "beta",
		PubDate:       "2026-05-22T11:00:00Z",
		Channel:       updateChannelBeta,
	})
	writeReleaseFile(t, "0.2.37", "RPBox_0.2.37_x64-setup.exe.sig", "stable-signature")

	db := testutil.NewTestDB(t, &model.User{})
	sponsor := model.User{Username: "sponsor", Email: "sponsor@example.com", PassHash: "hash", IsSponsor: true, SponsorLevel: 1}
	if err := db.Create(&sponsor).Error; err != nil {
		t.Fatalf("create sponsor user: %v", err)
	}

	server := newTestServer(t, db)
	resp := performUpdaterRequest(server.router, "/api/v1/updater/windows/x86_64/0.2.37-beta.1", newTestToken(t, sponsor), true)
	if resp.Code != http.StatusOK {
		t.Fatalf("beta to stable update: expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var update UpdateResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &update); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if update.Version != "0.2.37" {
		t.Fatalf("expected stable version, got %q", update.Version)
	}
	if update.Channel != updateChannelStable {
		t.Fatalf("expected stable channel, got %q", update.Channel)
	}
	if update.Signature != "stable-signature" {
		t.Fatalf("expected stable signature to be returned, got %q", update.Signature)
	}
}

func performUpdaterRequest(router http.Handler, path string, token string, beta bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if beta {
		req.Header.Set(desktopUpdateChannelHeader, updateChannelBeta)
	}

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func writeDesktopLatest(t *testing.T, name string, release LatestRelease) {
	t.Helper()

	dir := filepath.Join("releases")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create releases dir: %v", err)
	}
	data, err := json.Marshal(release)
	if err != nil {
		t.Fatalf("marshal latest metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func writeReleaseFile(t *testing.T, version string, name string, content string) {
	t.Helper()

	dir := filepath.Join("releases", version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create release dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write release file %s/%s: %v", version, name, err)
	}
}
