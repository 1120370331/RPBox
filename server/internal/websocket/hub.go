package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// MessageType 消息类型
type MessageType string

const (
	// MessageTypeNewNotification 新通知
	MessageTypeNewNotification MessageType = "new_notification"
	// MessageTypeUnreadCountUpdate 未读数量更新
	MessageTypeUnreadCountUpdate MessageType = "unread_count_update"
	// MessageTypePing 心跳
	MessageTypePing MessageType = "ping"
	// MessageTypePong 心跳响应
	MessageTypePong MessageType = "pong"
)

// Message WebSocket 消息结构
type Message struct {
	Type MessageType `json:"type"`
	Data interface{} `json:"data"`
}

// Client WebSocket 客户端
type Client struct {
	UserID uint
	Conn   *websocket.Conn
	Send   chan []byte
}

// Hub WebSocket 连接管理器
type Hub struct {
	// 用户ID -> 客户端连接映射
	clients map[uint]*Client
	// 注册新客户端
	register chan *Client
	// 注销客户端
	unregister chan *Client
	// 互斥锁
	mu sync.RWMutex
}

// NewHub 创建新的 Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run 运行 Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// 如果用户已有连接，先关闭旧连接
			if oldClient, exists := h.clients[client.UserID]; exists {
				close(oldClient.Send)
				oldClient.Conn.Close()
			}
			h.clients[client.UserID] = client
			h.mu.Unlock()
			log.Printf("[WebSocket] 用户 %d 已连接", client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, exists := h.clients[client.UserID]; exists {
				delete(h.clients, client.UserID)
				close(client.Send)
				client.Conn.Close()
			}
			h.mu.Unlock()
			log.Printf("[WebSocket] 用户 %d 已断开", client.UserID)
		}
	}
}

// Register 注册客户端
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 注销客户端
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// SendToUser 发送消息给指定用户
func (h *Hub) SendToUser(userID uint, msgType MessageType, data interface{}) error {
	h.mu.RLock()
	client, exists := h.clients[userID]
	h.mu.RUnlock()

	if !exists {
		// 用户不在线，不是错误
		return nil
	}

	msg := Message{
		Type: msgType,
		Data: data,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case client.Send <- msgBytes:
		return nil
	default:
		// 发送缓冲区满，关闭连接
		h.Unregister(client)
		return nil
	}
}

// IsUserOnline 检查用户是否在线
func (h *Hub) IsUserOnline(userID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.clients[userID]
	return exists
}

// GetOnlineCount 获取在线用户数
func (h *Hub) GetOnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
