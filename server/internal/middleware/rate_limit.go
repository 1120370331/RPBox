package middleware

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu    sync.Mutex
	ips   map[string]*ipLimiter
	limit rate.Limit
	burst int
	ttl   time.Duration
}

func newIPRateLimiter(rps float64, burst int, ttl time.Duration) *ipRateLimiter {
	limiter := &ipRateLimiter{
		ips:   make(map[string]*ipLimiter),
		limit: rate.Limit(rps),
		burst: burst,
		ttl:   ttl,
	}
	go limiter.cleanupLoop()
	return limiter
}

func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.ips[ip]
	if !exists {
		entry = &ipLimiter{
			limiter:  rate.NewLimiter(l.limit, l.burst),
			lastSeen: time.Now(),
		}
		l.ips[ip] = entry
		return entry.limiter
	}

	entry.lastSeen = time.Now()
	return entry.limiter
}

func (l *ipRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.ttl)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-l.ttl)
		l.mu.Lock()
		for ip, entry := range l.ips {
			if entry.lastSeen.Before(cutoff) {
				delete(l.ips, ip)
			}
		}
		l.mu.Unlock()
	}
}

// RateLimit applies an IP-based token bucket limiter.
func RateLimit(rps float64, burst int) gin.HandlerFunc {
	return rateLimitMiddleware(rps, burst, false)
}

// StrictRateLimit applies an IP-based limiter and logs on violations.
func StrictRateLimit(rps float64, burst int) gin.HandlerFunc {
	return rateLimitMiddleware(rps, burst, true)
}

func rateLimitMiddleware(rps float64, burst int, logOnLimit bool) gin.HandlerFunc {
	if rps <= 0 || burst <= 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	limiter := newIPRateLimiter(rps, burst, 10*time.Minute)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.getLimiter(ip).Allow() {
			if logOnLimit {
				log.Printf("[RateLimit] ip=%s method=%s path=%s", ip, c.Request.Method, c.Request.URL.Path)
			}
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests",
				"code":  http.StatusTooManyRequests,
			})
			return
		}

		c.Next()
	}
}
