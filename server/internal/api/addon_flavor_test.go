package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/config"
)

func TestAddonLatestKeepsRetailAndForeverChannelsSeparate(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "addons", "RPBox_Addon")
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"RPBox_Addon","latest":"1.0.14","latestByFlavor":{"_forever_":"1.0.15-beta.1"},"versions":[]}`
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	cfg.Storage.Path = root
	server := &Server{cfg: cfg}
	router := gin.New()
	router.GET("/addon/latest", server.getAddonLatest)
	for _, test := range []struct {
		query   string
		status  int
		version string
	}{
		{"", http.StatusOK, "1.0.14"},
		{"?flavor=_retail_", http.StatusOK, "1.0.14"},
		{"?flavor=_forever_", http.StatusOK, "1.0.15-beta.1"},
		{"?flavor=_classic_era_", http.StatusBadRequest, ""},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/addon/latest"+test.query, nil))
		if response.Code != test.status {
			t.Fatalf("%s: status=%d body=%s", test.query, response.Code, response.Body.String())
		}
		if test.version != "" {
			var body struct {
				Version string `json:"version"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Version != test.version {
				t.Fatalf("%s: version=%q err=%v", test.query, body.Version, err)
			}
		}
	}
	if !validGameFlavor("_forever_") {
		t.Fatal("Forever cloud backups must use their own game flavor")
	}
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), []byte(`{"name":"RPBox_Addon","latest":"1.0.14","versions":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/addon/latest?flavor=_forever_", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("retail addon must not be offered to Forever before its beta exists: %d %s", response.Code, response.Body.String())
	}
}
