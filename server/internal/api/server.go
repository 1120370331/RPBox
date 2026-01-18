package api

import (
	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/middleware"
	"github.com/rpbox/server/internal/service"
	ws "github.com/rpbox/server/internal/websocket"
)

type Server struct {
	cfg    *config.Config
	router *gin.Engine
	wsHub  *ws.Hub
}

func NewServer(cfg *config.Config) *Server {
	gin.SetMode(cfg.Server.Mode)
	router := gin.Default()
	router.Use(middleware.CORS())
	router.Use(middleware.BodyLimit(50 << 20)) // 限制请求体 50MB（支持大量人物卡和道具数据）

	// 设置 multipart 内存限制为 50MB
	router.MaxMultipartMemory = 50 << 20 // 50 MB

	// 创建 WebSocket Hub
	hub := ws.NewHub()
	go hub.Run()

	s := &Server{
		cfg:    cfg,
		router: router,
		wsHub:  hub,
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
