package api

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rpbox/server/internal/cache"
	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/middleware"
	"github.com/rpbox/server/internal/service"
	ws "github.com/rpbox/server/internal/websocket"
	"github.com/rpbox/server/pkg/email"
)

type Server struct {
	cfg                  *config.Config
	router               *gin.Engine
	wsHub                *ws.Hub
	emailClient          *email.SMTPClient
	verificationService  *service.VerificationService
	cache                cache.Cache
	ossBucket            *oss.Bucket
	ossInitOnce          sync.Once
	ossInitErr           error
	trp3LatestMu         sync.Mutex
	trp3LatestCache      *TRP3LatestResponse
	trp3LatestCacheUntil time.Time
	trp3CacheLocksMu     sync.Mutex
	trp3CacheLocks       map[string]*sync.Mutex
	trp3MirrorManifestMu sync.Mutex
}

const (
	// multipartMemoryThresholdBytes is only the in-memory portion of a
	// multipart form. net/http transparently spills larger file parts to the
	// process temporary directory, while BodyLimit remains the hard request
	// limit.
	multipartMemoryThresholdBytes int64 = 8 << 20
	authPublicBodyLimitBytes      int64 = 1 << 20
)

func NewServer(cfg *config.Config) *Server {
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(gin.LoggerWithFormatter(safeGinLogFormatter), gin.Recovery())
	configureTrustedProxies(router, cfg.Server.TrustedProxies)
	if cfg.Server.Mode == gin.ReleaseMode {
		router.Use(middleware.HTTPSRedirect(cfg))
	}
	router.Use(middleware.SecurityHeaders(cfg))
	router.Use(middleware.CORS(cfg))
	router.Use(middleware.RateLimit(cfg.RateLimit.Global.RPS, cfg.RateLimit.Global.Burst))
	maxBodySizeMB := cfg.Server.MaxBodySizeMB
	if maxBodySizeMB <= 0 {
		maxBodySizeMB = 200
	}
	maxBodySizeBytes := int64(maxBodySizeMB) << 20
	router.Use(middleware.BodyLimit(maxBodySizeBytes))

	// Keep multipart parsing memory bounded independently from the configurable
	// hard request limit. This still permits legitimate larger uploads because
	// net/http stores the excess file data in temporary files.
	router.MaxMultipartMemory = multipartMemoryLimit(maxBodySizeBytes)

	// 创建 WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// 初始化 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	var cacheClient cache.Cache

	// 验证 Redis 连接
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("[Cache] Redis connection failed: %v; fallback to no-cache mode", err)
		cacheClient = nil
	} else {
		log.Printf("[Cache] Redis connected to %s:%s", cfg.Redis.Host, cfg.Redis.Port)
		cacheClient = cache.NewRedisCache(redisClient, cache.Options{Jitter: 5 * time.Second})
	}

	// 初始化邮件客户端
	emailClient := email.NewSMTPClient(&email.SMTPConfig{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	})

	// 初始化验证码服务
	verificationService := service.NewVerificationService(redisClient)

	s := &Server{
		cfg:                 cfg,
		router:              router,
		wsHub:               hub,
		emailClient:         emailClient,
		verificationService: verificationService,
		cache:               cacheClient,
	}

	// 设置通知服务的 Hub 引用
	service.SetNotificationHub(hub)

	s.setupRoutes()
	return s
}

func multipartMemoryLimit(maxBodySizeBytes int64) int64 {
	if maxBodySizeBytes < multipartMemoryThresholdBytes {
		return maxBodySizeBytes
	}
	return multipartMemoryThresholdBytes
}

func safeGinLogFormatter(param gin.LogFormatterParams) string {
	var statusColor, methodColor, resetColor string
	if param.IsOutputColor() {
		statusColor = param.StatusCodeColor()
		methodColor = param.MethodColor()
		resetColor = param.ResetColor()
	}

	if param.Latency > time.Minute {
		param.Latency = param.Latency.Truncate(time.Second)
	}
	return fmt.Sprintf("[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n%s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		redactTokenQuery(param.Path),
		param.ErrorMessage,
	)
}

// redactTokenQuery preserves the request path and all non-secret query data
// while removing every token parameter value from access logs. It deliberately
// decodes query keys so encoded spellings such as %74oken cannot bypass it.
func redactTokenQuery(path string) string {
	queryStart := strings.IndexByte(path, '?')
	if queryStart < 0 || queryStart == len(path)-1 {
		return path
	}

	parts := strings.Split(path[queryStart+1:], "&")
	for index, part := range parts {
		rawKey, _, _ := strings.Cut(part, "=")
		key, err := url.QueryUnescape(rawKey)
		if err == nil && key == "token" {
			parts[index] = rawKey + "=[REDACTED]"
		}
	}
	return path[:queryStart+1] + strings.Join(parts, "&")
}

// configureTrustedProxies applies an explicit proxy boundary. Gin otherwise
// trusts every proxy by default, which lets direct clients spoof ClientIP via
// X-Forwarded-For. Invalid configuration fails closed and leaves proxy-header
// processing disabled.
func configureTrustedProxies(router *gin.Engine, trustedProxies []string) {
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Printf("[Security] failed to disable trusted proxies: %v", err)
		return
	}
	if len(trustedProxies) == 0 {
		return
	}
	if err := router.SetTrustedProxies(trustedProxies); err != nil {
		log.Printf("[Security] invalid server.trusted_proxies configuration; proxy headers disabled: %v", err)
		if disableErr := router.SetTrustedProxies(nil); disableErr != nil {
			log.Printf("[Security] failed to disable trusted proxies after invalid configuration: %v", disableErr)
		}
	}
}

func (s *Server) Run() error {
	return s.router.Run(":" + s.cfg.Server.Port)
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
