package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// NoticeHub WebSocket通知中心
type NoticeHub struct {
	clients    map[string]*Client
	broadcast  chan NoticeMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	stop       chan struct{} // 用于优雅关闭
}

// Client WebSocket客户端
type Client struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
	hub    *NoticeHub
}

// NoticeMessage 通知消息
type NoticeMessage struct {
	Type        string `json:"type"` // new_notice, unread_count
	NoticeID    string `json:"noticeId,omitempty"`
	Title       string `json:"title,omitempty"`
	Content     string `json:"content,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	Timestamp   int64  `json:"timestamp,omitempty"`
	UnreadCount int    `json:"unreadCount,omitempty"`
}

// NewNoticeHub 创建通知中心
func NewNoticeHub() *NoticeHub {
	return &NoticeHub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan NoticeMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan struct{}),
	}
}

// Run 运行通知中心
func (h *NoticeHub) Run() {
	for {
		select {
		case <-h.stop:
			// 收到停止信号，关闭所有连接并退出
			h.mu.Lock()
			for _, client := range h.clients {
				close(client.send)
			}
			h.clients = make(map[string]*Client)
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.userID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// 先收集需要删除的客户端，避免在读锁中进行写操作
			var toDelete []string
			encodedMsg := h.encodeMessage(message)

			h.mu.RLock()
			for userID, client := range h.clients {
				select {
				case client.send <- encodedMsg:
					// 发送成功
				default:
					// 发送失败，标记为待删除
					toDelete = append(toDelete, userID)
				}
			}
			h.mu.RUnlock()

			// 在读锁外执行删除操作
			if len(toDelete) > 0 {
				h.mu.Lock()
				for _, userID := range toDelete {
					if client, ok := h.clients[userID]; ok {
						delete(h.clients, userID)
						close(client.send)
					}
				}
				h.mu.Unlock()
			}
		}
	}
}

// Stop 停止通知中心
func (h *NoticeHub) Stop() {
	close(h.stop)
}

// RegisterClient 注册客户端
func (h *NoticeHub) RegisterClient(userID string, conn *websocket.Conn) *Client {
	client := &Client{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 256),
		hub:    h,
	}
	h.register <- client
	go client.writePump()
	return client
}

// UnregisterClient 注销客户端
func (h *NoticeHub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// BroadcastToUsers 向指定用户广播消息
func (h *NoticeHub) BroadcastToUsers(userIDs []string, message NoticeMessage) {
	// 先收集需要删除的客户端，避免在读锁中进行写操作
	var toDelete []string
	data := h.encodeMessage(message)

	h.mu.RLock()
	for _, userID := range userIDs {
		if client, ok := h.clients[userID]; ok {
			select {
			case client.send <- data:
				// 发送成功
			default:
				// 发送失败，标记为待删除
				toDelete = append(toDelete, userID)
			}
		}
	}
	h.mu.RUnlock()

	// 在读锁外执行删除操作
	if len(toDelete) > 0 {
		h.mu.Lock()
		for _, userID := range toDelete {
			if client, ok := h.clients[userID]; ok {
				delete(h.clients, userID)
				close(client.send)
			}
		}
		h.mu.Unlock()
	}
}

// BroadcastToAll 向所有用户广播消息
func (h *NoticeHub) BroadcastToAll(message NoticeMessage) {
	h.broadcast <- message
}

// encodeMessage 编码消息
func (h *NoticeHub) encodeMessage(msg NoticeMessage) []byte {
	data, _ := json.Marshal(msg)
	return data
}

// writePump 写入循环
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(websocket.TextMessage, message)
		case <-ticker.C:
			c.conn.WriteMessage(websocket.PingMessage, nil)
		}
	}
}
