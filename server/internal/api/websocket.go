package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	ws "github.com/rpbox/server/internal/websocket"
)

func (s *Server) webSocketUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     s.isAllowedWebSocketOrigin,
	}
}

func (s *Server) isAllowedWebSocketOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Native clients are not required to send Origin.
		return true
	}

	// Reject ambiguous duplicate Origin headers even when the first value is
	// trusted. Browser WebSocket handshakes contain at most one Origin value.
	if len(r.Header.Values("Origin")) != 1 || s.cfg == nil {
		return false
	}

	for _, allowedOrigin := range s.cfg.CORS.AllowedOrigins {
		if allowedOrigin != "*" && allowedOrigin == origin {
			return true
		}
	}
	return false
}

// handleWebSocket 处理 WebSocket 连接
func (s *Server) handleWebSocket(c *gin.Context) {
	// 从 JWT 中间件获取用户 ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	// 升级 HTTP 连接到 WebSocket
	upgrader := s.webSocketUpgrader()
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] 升级连接失败: %v", err)
		return
	}

	// 创建客户端
	client := &ws.Client{
		UserID: userID.(uint),
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	// 注册客户端
	s.wsHub.Register(client)

	// 启动读写协程
	go client.WritePump()
	go client.ReadPump(s.wsHub)
}
