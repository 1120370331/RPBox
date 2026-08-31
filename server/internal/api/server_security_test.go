package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRedactTokenQuery(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "keeps other query parameters",
			path: "/api/v1/ws/notifications?channel=alerts&token=header.payload.signature&since=42",
			want: "/api/v1/ws/notifications?channel=alerts&token=[REDACTED]&since=42",
		},
		{
			name: "redacts URL encoded token key and value",
			path: "/api/v1/ws/notifications?%74%6f%6b%65%6e=secret%2Bvalue%2Fmore&other=a%20b",
			want: "/api/v1/ws/notifications?%74%6f%6b%65%6e=[REDACTED]&other=a%20b",
		},
		{
			name: "redacts every duplicate token",
			path: "/api/v1/ws/notifications?token=first&x=1&token=second",
			want: "/api/v1/ws/notifications?token=[REDACTED]&x=1&token=[REDACTED]",
		},
		{
			name: "leaves unrelated URL unchanged",
			path: "/api/v1/posts?query=token%3Dpublic&limit=20",
			want: "/api/v1/posts?query=token%3Dpublic&limit=20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactTokenQuery(tt.path); got != tt.want {
				t.Fatalf("redactTokenQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSafeGinLoggerDoesNotLeakEncodedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: safeGinLogFormatter,
		Output:    &output,
	}))
	router.GET("/api/v1/ws/notifications", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/ws/notifications?room=main&%74oken=header%2Epayload%2Esignature&cursor=9",
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	logged := output.String()
	for _, secret := range []string{"header.payload.signature", "header%2Epayload%2Esignature"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("access log leaked token %q: %s", secret, logged)
		}
	}
	for _, expected := range []string{
		"%74oken=[REDACTED]",
		"room=main",
		"cursor=9",
		" 204 ",
		"GET",
	} {
		if !strings.Contains(logged, expected) {
			t.Fatalf("access log missing %q: %s", expected, logged)
		}
	}
}

func TestConfigureTrustedProxiesControlsClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		trustedProxies []string
		remoteAddr     string
		forwardedFor   string
		wantClientIP   string
	}{
		{
			name:         "external forwarded for spoof is ignored by default",
			remoteAddr:   "203.0.113.9:54321",
			forwardedFor: "198.51.100.77",
			wantClientIP: "203.0.113.9",
		},
		{
			name:           "trusted proxy restores original client IP",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.1.2.3:54321",
			forwardedFor:   "198.51.100.77",
			wantClientIP:   "198.51.100.77",
		},
		{
			name:           "untrusted peer cannot use otherwise valid configuration",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "203.0.113.9:54321",
			forwardedFor:   "198.51.100.77",
			wantClientIP:   "203.0.113.9",
		},
		{
			name:           "one invalid entry disables proxy trust",
			trustedProxies: []string{"10.0.0.0/8", "invalid"},
			remoteAddr:     "10.1.2.3:54321",
			forwardedFor:   "198.51.100.77",
			wantClientIP:   "10.1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			configureTrustedProxies(router, tt.trustedProxies)
			router.GET("/client-ip", func(c *gin.Context) {
				c.String(http.StatusOK, c.ClientIP())
			})

			request := httptest.NewRequest(http.MethodGet, "/client-ip", nil)
			request.RemoteAddr = tt.remoteAddr
			request.Header.Set("X-Forwarded-For", tt.forwardedFor)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if got := response.Body.String(); got != tt.wantClientIP {
				t.Fatalf("ClientIP = %q, want %q", got, tt.wantClientIP)
			}
		})
	}
}
