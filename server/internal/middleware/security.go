package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/config"
)

// SecurityHeaders adds basic security headers to every response.
func SecurityHeaders(cfg *config.Config) gin.HandlerFunc {
	trustedProxies := newTrustedProxySet(cfg)
	return func(c *gin.Context) {
		if isHTTPS(c, trustedProxies) {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		switch c.Request.URL.Path {
		case "/api/v1/auth/login", "/api/v1/user/info":
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
			c.Header("Pragma", "no-cache")
		}

		c.Next()
	}
}

// HTTPSRedirect redirects HTTP traffic to HTTPS.
func HTTPSRedirect(cfg *config.Config) gin.HandlerFunc {
	trustedProxies := newTrustedProxySet(cfg)
	return func(c *gin.Context) {
		if isHTTPS(c, trustedProxies) {
			c.Next()
			return
		}

		authority, ok := redirectAuthority(cfg, c.Request.Host)
		if !ok {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		httpsURL := "https://" + authority + c.Request.URL.RequestURI()
		c.Redirect(http.StatusMovedPermanently, httpsURL)
		c.Abort()
	}
}

func isHTTPS(c *gin.Context, trustedProxies trustedProxySet) bool {
	if c.Request.TLS != nil {
		return true
	}
	return trustedProxies.containsRemoteAddr(c.Request.RemoteAddr) &&
		strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https")
}

type trustedProxySet struct {
	networks []*net.IPNet
}

func newTrustedProxySet(cfg *config.Config) trustedProxySet {
	var result trustedProxySet
	if cfg == nil {
		return result
	}
	for _, raw := range cfg.Server.TrustedProxies {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			return trustedProxySet{}
		}
		if _, network, err := net.ParseCIDR(entry); err == nil {
			result.networks = append(result.networks, network)
			continue
		}
		ip := net.ParseIP(entry)
		if ip == nil {
			return trustedProxySet{}
		}
		bits := 128
		if ip.To4() != nil {
			ip = ip.To4()
			bits = 32
		}
		result.networks = append(result.networks, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
	}
	return result
}

func (s trustedProxySet) containsRemoteAddr(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = strings.Trim(remoteAddr, "[]")
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, network := range s.networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func redirectAuthority(cfg *config.Config, requestHost string) (string, bool) {
	if cfg != nil {
		if authority, ok := configuredAuthority(cfg.Server.ApiHost); ok {
			return authority, true
		}
		// Request.Host is only a development fallback. Release never reflects an
		// attacker-controlled Host into Location.
		if cfg.Server.Mode == gin.ReleaseMode {
			return "", false
		}
	}
	return validRequestAuthority(requestHost)
}

func configuredAuthority(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || !isLegalAuthority(parsed) || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	if _, err := parsed.MarshalBinary(); err != nil {
		return "", false
	}
	return parsed.Host, true
}

func validRequestAuthority(host string) (string, bool) {
	if strings.TrimSpace(host) != host || host == "" || strings.ContainsAny(host, "/\\?#@") {
		return "", false
	}
	parsed, err := url.Parse("https://" + host)
	if err != nil || parsed.Host != host || !isLegalAuthority(parsed) || parsed.User != nil {
		return "", false
	}
	return host, true
}

func isLegalAuthority(parsed *url.URL) bool {
	hostname := parsed.Hostname()
	if hostname == "" {
		return false
	}
	if port := parsed.Port(); port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return false
		}
	}
	if net.ParseIP(hostname) != nil {
		return true
	}

	hostname = strings.TrimSuffix(hostname, ".")
	if hostname == "" || len(hostname) > 253 {
		return false
	}
	for _, label := range strings.Split(hostname, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			if (character < 'a' || character > 'z') &&
				(character < 'A' || character > 'Z') &&
				(character < '0' || character > '9') && character != '-' {
				return false
			}
		}
	}
	return true
}
