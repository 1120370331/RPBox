package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/config"
)

func TestTRP3LatestSelectsPackagesForEachClient(t *testing.T) {
	cfg := &config.Config{}
	cfg.Storage.Path = t.TempDir()
	server := &Server{cfg: cfg}
	router := gin.New()
	router.GET("/addon/trp3/latest", server.getTRP3Latest)
	router.GET("/addon/trp3/download/:id", server.downloadTRP3Addon)
	get := func(path string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		return response
	}
	read := func(flavor string) map[string]TRP3AddonLatestInfo {
		t.Helper()
		response := get("/addon/trp3/latest?flavor=" + flavor)
		if response.Code != http.StatusOK {
			t.Fatalf("%s: status=%d body=%s", flavor, response.Code, response.Body.String())
		}
		var payload TRP3LatestResponse
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		result := make(map[string]TRP3AddonLatestInfo)
		for _, addon := range payload.Addons {
			result[addon.ID] = addon
		}
		return result
	}

	retail := read("_retail_")
	forever := read("_forever_")
	classic := read("_classic_era_")
	if retail["total-rp-3"].LatestVersion != "3.5.4" || forever["total-rp-3"].LatestVersion != "3.5.4" {
		t.Fatal("modern clients did not receive the modern TRP3 package")
	}
	if classic["total-rp-3"].LatestVersion != "3.4.1" || !strings.Contains(classic["total-rp-3"].DownloadURL, "flavor=_classic_era_") {
		t.Fatalf("classic client received the wrong TRP3 package: %+v", classic["total-rp-3"])
	}
	if classic["musician"].LatestVersion != "1.9.18.0" || classic["musician"].DownloadURL == "" {
		t.Fatal("classic client lost the compatible Musician package")
	}
	if classic["total-rp-3-extended"].DownloadURL != "" || classic["total-rp-3-extended"].CompatibilityNote == "" {
		t.Fatal("unsupported Classic Extended package was offered")
	}
	for _, addon := range read("_classic_ptr_") {
		if addon.DownloadURL != "" || addon.CompatibilityNote == "" {
			t.Fatalf("test client was offered an unverified package: %+v", addon)
		}
	}
	if response := get("/addon/trp3/download/total-rp-3-extended?flavor=_classic_era_"); response.Code != http.StatusNotFound {
		t.Fatalf("unsupported package download status=%d", response.Code)
	}
	if response := get("/addon/trp3/latest?flavor=unknown"); response.Code != http.StatusBadRequest {
		t.Fatalf("unknown client flavor status=%d", response.Code)
	}
}
