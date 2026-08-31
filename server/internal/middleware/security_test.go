package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/config"
)

func TestSecurityHeadersTrustsHTTPSOnlyAtConfiguredBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		trustedProxies []string
		remoteAddr     string
		headers        map[string]string
		tls            bool
		wantHSTS       bool
	}{
		{
			name:       "direct client cannot spoof forwarded proto",
			remoteAddr: "203.0.113.9:54321",
			headers:    map[string]string{"X-Forwarded-Proto": "https"},
		},
		{
			name:           "trusted proxy may assert forwarded proto",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.1.2.3:54321",
			headers:        map[string]string{"X-Forwarded-Proto": "https"},
			wantHSTS:       true,
		},
		{
			name:           "invalid proxy config disables all trust",
			trustedProxies: []string{"10.0.0.0/8", "not-a-cidr"},
			remoteAddr:     "10.1.2.3:54321",
			headers:        map[string]string{"X-Forwarded-Proto": "https"},
		},
		{
			name:           "nonstandard forwarded ssl header is ignored",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.1.2.3:54321",
			headers:        map[string]string{"X-Forwarded-Ssl": "on"},
		},
		{
			name:           "ambiguous forwarded proto chain is rejected",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.1.2.3:54321",
			headers:        map[string]string{"X-Forwarded-Proto": "https, http"},
		},
		{
			name:       "direct TLS is authoritative",
			remoteAddr: "203.0.113.9:54321",
			tls:        true,
			wantHSTS:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Server: config.ServerConfig{TrustedProxies: tt.trustedProxies}}
			router := gin.New()
			router.Use(SecurityHeaders(cfg))
			router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.RemoteAddr = tt.remoteAddr
			if tt.tls {
				request.TLS = &tls.ConnectionState{}
			}
			for key, value := range tt.headers {
				request.Header.Set(key, value)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			hasHSTS := response.Header().Get("Strict-Transport-Security") != ""
			if hasHSTS != tt.wantHSTS {
				t.Fatalf("HSTS presence = %v, want %v", hasHSTS, tt.wantHSTS)
			}
		})
	}
}

func TestHTTPSRedirectUsesConfiguredAuthorityInsteadOfRequestHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Server: config.ServerConfig{
		Mode:    gin.ReleaseMode,
		ApiHost: "https://api.example.com",
	}}
	router := gin.New()
	router.Use(HTTPSRedirect(cfg))
	router.GET("/api/v1/health", func(c *gin.Context) { c.Status(http.StatusOK) })

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health?ready=1", nil)
	request.Host = "attacker.example"
	request.RemoteAddr = "203.0.113.9:54321"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", response.Code)
	}
	if got := response.Header().Get("Location"); got != "https://api.example.com/api/v1/health?ready=1" {
		t.Fatalf("unexpected redirect location %q", got)
	}
}

func TestHTTPSRedirectReleaseFailsClosedWithoutValidAPIHost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, apiHost := range []string{
		"",
		"http://api.example.com",
		"https://api.example.com/base",
		"https://invalid,host.example",
		"https://api.example.com:99999",
	} {
		t.Run(apiHost, func(t *testing.T) {
			cfg := &config.Config{Server: config.ServerConfig{Mode: gin.ReleaseMode, ApiHost: apiHost}}
			router := gin.New()
			router.Use(HTTPSRedirect(cfg))
			router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Host = "attacker.example"
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", response.Code)
			}
			if location := response.Header().Get("Location"); location != "" {
				t.Fatalf("expected no reflected Location, got %q", location)
			}
		})
	}
}

func TestHTTPSRedirectDevelopmentFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Server: config.ServerConfig{Mode: gin.DebugMode}}
	router := gin.New()
	router.Use(HTTPSRedirect(cfg))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Host = "localhost:8080"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if got := response.Header().Get("Location"); got != "https://localhost:8080/" {
		t.Fatalf("unexpected development redirect %q", got)
	}
}

func TestHTTPSRedirectHonorsForwardedProtoOnlyFromTrustedProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Server: config.ServerConfig{
		Mode:           gin.ReleaseMode,
		ApiHost:        "https://api.example.com",
		TrustedProxies: []string{"10.0.0.0/8"},
	}}

	for _, tt := range []struct {
		name       string
		remoteAddr string
		wantCode   int
	}{
		{name: "trusted", remoteAddr: "10.1.2.3:54321", wantCode: http.StatusOK},
		{name: "untrusted", remoteAddr: "203.0.113.9:54321", wantCode: http.StatusMovedPermanently},
	} {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(HTTPSRedirect(cfg))
			router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.RemoteAddr = tt.remoteAddr
			request.Header.Set("X-Forwarded-Proto", "https")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantCode)
			}
		})
	}
}
