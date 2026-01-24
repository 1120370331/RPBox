package api

import (
	"sync"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/middleware"
	"github.com/rpbox/server/internal/service"
	ws "github.com/rpbox/server/internal/websocket"
	"github.com/rpbox/server/pkg/email"
)

type Server struct {
	cfg                 *config.Config
	router              *gin.Engine
	wsHub               *ws.Hub
	emailClient         *email.SMTPClient
	verificationService *service.VerificationService
	ossBucket           *oss.Bucket
	ossInitOnce         sync.Once
	ossInitErr          error
}

func NewServer(cfg *config.Config) *Server {
	gin.SetMode(cfg.Server.Mode)
	router := gin.Default()
	if cfg.Server.Mode == gin.ReleaseMode {
		router.Use(middleware.HTTPSRedirect())
	}
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS(cfg))
	router.Use(middleware.RateLimit(cfg.RateLimit.Global.RPS, cfg.RateLimit.Global.Burst))
	maxBodySizeMB := cfg.Server.MaxBodySizeMB
	if maxBodySizeMB <= 0 {
		maxBodySizeMB = 200
	}
	maxBodySizeBytes := int64(maxBodySizeMB) << 20
	router.Use(middleware.BodyLimit(maxBodySizeBytes))

	// 设置 multipart 内存限制
	router.MaxMultipartMemory = maxBodySizeBytes

	// 创建 WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	// 初始化 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

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
	}

	// 设置通知服务的 Hub 引用
	service.SetNotificationHub(hub)

	s.setupRoutes()
	return s
}

func (s *Server) Run() error {
	return s.router.Run(":" + s.cfg.Server.Port)
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
