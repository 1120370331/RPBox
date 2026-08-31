package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/config"
)

// CORS 允许前端（开发时可能是不同端口）访问 API，并处理预检请求。
func CORS(cfg *config.Config) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{})
	for _, origin := range cfg.CORS.AllowedOrigins {
		if origin != "" {
			allowedOrigins[origin] = struct{}{}
		}
	}
	if cfg.Server.Mode == "debug" {
		for _, origin := range cfg.CORS.DevOrigins {
			if origin != "" {
				allowedOrigins[origin] = struct{}{}
			}
		}
	}
	// An unconfigured release server fails closed. The permissive fallback is
	// kept only for explicitly configured debug mode to preserve local setup.
	allowAll := cfg.Server.Mode == gin.DebugMode && len(allowedOrigins) == 0

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := false
		if origin != "" {
			c.Header("Vary", "Origin")
			if allowAll {
				allowed = true
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
			} else if _, ok := allowedOrigins[origin]; ok {
				allowed = true
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}
		if allowed {
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Max-Age", "86400")
		}

		// 预检请求直接返回
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
