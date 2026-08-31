package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/config"
)

func TestCORSReleaseWithoutOriginsFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Server: config.ServerConfig{Mode: gin.ReleaseMode}}

	response := performCORSRequest(cfg, http.MethodOptions, "https://evil.example")
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected OPTIONS 204, got %d", response.Code)
	}
	for _, header := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
	} {
		if value := response.Header().Get(header); value != "" {
			t.Fatalf("expected %s to be absent, got %q", header, value)
		}
	}
}

func TestCORSDebugWithoutOriginsKeepsDevelopmentFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Server: config.ServerConfig{Mode: gin.DebugMode}}

	response := performCORSRequest(cfg, http.MethodGet, "http://localhost:5173")
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected debug origin reflection, got %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected debug credentials header, got %q", got)
	}
}

func TestCORSUsesExactConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Server: config.ServerConfig{Mode: gin.ReleaseMode},
		CORS: config.CORSConfig{AllowedOrigins: []string{
			"https://app.example.com",
		}},
	}

	allowed := performCORSRequest(cfg, http.MethodGet, "https://app.example.com")
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("expected configured origin, got %q", got)
	}

	for _, origin := range []string{
		"https://app.example.com.evil.test",
		"https://app.example.com/",
		"HTTPS://app.example.com",
	} {
		response := performCORSRequest(cfg, http.MethodGet, origin)
		if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("expected origin %q to be denied, got ACAO %q", origin, got)
		}
		if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "" {
			t.Fatalf("expected origin %q to have no credentials header, got %q", origin, got)
		}
	}
}

func performCORSRequest(cfg *config.Config, method, origin string) *httptest.ResponseRecorder {
	router := gin.New()
	router.Use(CORS(cfg))
	router.Any("/resource", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(method, "/resource", nil)
	request.Header.Set("Origin", origin)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
