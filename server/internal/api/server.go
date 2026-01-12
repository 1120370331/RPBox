package api

import (
	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/config"
)

type Server struct {
	cfg    *config.Config
	router *gin.Engine
}

func NewServer(cfg *config.Config) *Server {
	gin.SetMode(cfg.Server.Mode)
	router := gin.Default()

	s := &Server{
		cfg:    cfg,
		router: router,
	}
	s.setupRoutes()
	return s
}

func (s *Server) Run() error {
	return s.router.Run(":" + s.cfg.Server.Port)
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
