package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/testutil"
)

func TestNewServerMultipartMemoryLimit(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	// Fail the optional Redis probe immediately instead of waiting on a real
	// service; request-limit setup is independent from cache availability.
	cfg.Redis.Port = "1"

	server := NewServer(cfg)
	if server.router.MaxMultipartMemory != multipartMemoryThresholdBytes {
		t.Fatalf("MaxMultipartMemory = %d, want %d", server.router.MaxMultipartMemory, multipartMemoryThresholdBytes)
	}

	if got := multipartMemoryLimit(4 << 20); got != 4<<20 {
		t.Fatalf("multipartMemoryLimit(4 MiB) = %d, want %d", got, int64(4<<20))
	}
}

func TestAuthPublicBodyLimitRejectsDeclaredOversize(t *testing.T) {
	server := newAuthBodyLimitRouteTestServer(t)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(strings.Repeat("x", int(authPublicBodyLimitBytes+1))),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusRequestEntityTooLarge, response.Body.String())
	}
}

func TestAuthPublicBodyLimitTruncatesChunkedOversize(t *testing.T) {
	server := newAuthBodyLimitRouteTestServer(t)
	payload := `{"username":"` + strings.Repeat("x", int(authPublicBodyLimitBytes)) + `","password":"secret"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(payload))
	request.ContentLength = -1
	request.TransferEncoding = []string{"chunked"}
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)

	if response.Code >= http.StatusOK && response.Code < http.StatusMultipleChoices {
		t.Fatalf("unknown-length oversized auth body unexpectedly succeeded with %d; body=%s", response.Code, response.Body.String())
	}
}

func TestAuthPublicBodyLimitAllowsSmallRequest(t *testing.T) {
	server := newAuthBodyLimitRouteTestServer(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)

	// The existing login handler rejects missing credentials with 400. This
	// proves the route-specific limiter forwarded a legitimate small request.
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want existing handler status %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func TestAuthPublicBodyLimitLeavesOptionsAndGetUnchanged(t *testing.T) {
	server := newAuthBodyLimitRouteTestServer(t)

	optionsRequest := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	optionsResponse := httptest.NewRecorder()
	server.router.ServeHTTP(optionsResponse, optionsRequest)
	if optionsResponse.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want %d", optionsResponse.Code, http.StatusNoContent)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	getResponse := httptest.NewRecorder()
	server.router.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusNotFound {
		t.Fatalf("GET status = %d, want existing route status %d", getResponse.Code, http.StatusNotFound)
	}
}

func TestAuthPublicRateLimitRunsBeforeBodyLimit(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	cfg.Server.Mode = gin.TestMode
	cfg.RateLimit.Auth.RPS = 0.000001
	cfg.RateLimit.Auth.Burst = 1
	server := &Server{
		cfg:    cfg,
		router: gin.New(),
	}
	server.setupRoutes()

	oversizedBody := strings.Repeat("x", int(authPublicBodyLimitBytes+1))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(oversizedBody))
	request.Header.Set("Content-Type", "application/json")
	firstResponse := httptest.NewRecorder()
	server.router.ServeHTTP(firstResponse, request)
	if firstResponse.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("first oversized request status = %d, want %d", firstResponse.Code, http.StatusRequestEntityTooLarge)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(oversizedBody))
	request.Header.Set("Content-Type", "application/json")
	secondResponse := httptest.NewRecorder()
	server.router.ServeHTTP(secondResponse, request)
	if secondResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("second oversized request status = %d, want %d", secondResponse.Code, http.StatusTooManyRequests)
	}
}

func newAuthBodyLimitRouteTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := testutil.NewTestConfig(t)
	cfg.Server.Mode = gin.TestMode
	server := &Server{
		cfg:    cfg,
		router: gin.New(),
	}
	server.setupRoutes()
	return server
}
