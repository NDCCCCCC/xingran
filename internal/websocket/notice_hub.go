package websocket

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// RPA 消息类型常量
const (
	MessageTypeRPAProgress  = "rpa_progress"
	MessageTypeRPACompleted = "rpa_completed"
	MessageTypeRPAFailed    = "rpa_failed"
)

// noticeHubBufferSize NoticeHub 内部 channel 缓冲大小
//
// 适用于 broadcast / register / unregister 三个 hub 内部 channel。
// F-13 fix: register/unregister 改无缓冲 → 缓冲 chan,避免高并发连接时
// RegisterClient 同步发送阻塞,客户端在握手阶段可能耗尽 FD 等待。
// 缓冲 256 与 broadcast 保持一致量级,溢出时调用方等待是合理压力反馈。
const noticeHubBufferSize = 256

// NoticeHub WebSocket通知中心
type NoticeHub struct {
	clients    map[string]*Client
	broadcast  chan NoticeMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	stop       chan struct{} // 用于优雅关闭
	stopOnce   sync.Once     // 确保 stop channel 只关闭一次
	// 遵循 Go 最佳实践：使用 sync.Pool 复用高频分配的对象
	toDeletePool sync.Pool // string slice pool for broadcast operations
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

// RPAProgressMessage RPA执行进度消息
type RPAProgressMessage struct {
	Type        string `json:"type"`        // rpa_progress, rpa_completed, rpa_failed
	ExecutionID string `json:"executionId"` // 执行记录ID
	TaskID      string `json:"taskId"`      // 任务ID
	TaskName    string `json:"taskName"`    // 任务名称
	Step        int    `json:"step"`        // 当前步骤
	Total       int    `json:"total"`       // 总步骤数
	Message     string `json:"message"`     // 进度消息
	Status      string `json:"status"`      // 状态
	Timestamp   int64  `json:"timestamp"`   // 时间戳
}

// NewNoticeHub 创建通知中心
func NewNoticeHub() *NoticeHub {
	h := &NoticeHub{
		clients:   make(map[string]*Client),
		broadcast: make(chan NoticeMessage, noticeHubBufferSize),
		register:  make(chan *Client, noticeHubBufferSize),
		unregister: make(chan *Client, noticeHubBufferSize),
		stop:      make(chan struct{}),
	}
	// 初始化对象池，预分配容量减少后续分配
	h.toDeletePool.New = func() interface{} {
		s := make([]string, 0, 16)
		return &s
	}
	return h
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
			// 遵循 Go 最佳实践：使用 sync.Pool 复用切片，减少 GC 压力
			toDelete := h.toDeletePool.Get().(*[]string)
			defer func() {
				*toDelete = (*toDelete)[:0] // 重置长度但保留容量
				h.toDeletePool.Put(toDelete)
			}()

			encodedMsg, err := h.encodeMessage(message)
			if err != nil {
				applogger.Errorf("消息序列化失败: %v", err)
				continue // 跳过此消息
			}

			h.mu.RLock()
			for userID, client := range h.clients {
				select {
				case client.send <- encodedMsg:
					// 发送成功
				default:
					// 发送失败，标记为待删除
					*toDelete = append(*toDelete, userID)
				}
			}
			h.mu.RUnlock()

			// 在读锁外执行删除操作
			if len(*toDelete) > 0 {
				h.mu.Lock()
				for _, userID := range *toDelete {
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
// 遵循 Go 最佳实践：使用 sync.Once 确保只关闭一次
func (h *NoticeHub) Stop() {
	h.stopOnce.Do(func() {
		close(h.stop)
	})
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
	// P1 fix: 启动 readPump 探测僵尸连接 — 否则 writePump 仅在主动写时
	// 才能感知断连,客户端拔网线/睡眠等情况下 conn 残留,FD 与 Client
	// 记录一起泄漏,长期导致 hub 满和 FD 耗尽。
	go client.readPump()
	return client
}

// UnregisterClient 注销客户端
func (h *NoticeHub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// BroadcastToUsers 向指定用户广播消息
func (h *NoticeHub) BroadcastToUsers(userIDs []string, message NoticeMessage) {
	// 先收集需要删除的客户端，避免在读锁中进行写操作
	// 遵循 Go 最佳实践：使用 sync.Pool 复用切片
	toDelete := h.toDeletePool.Get().(*[]string)
	defer func() {
		*toDelete = (*toDelete)[:0]
		h.toDeletePool.Put(toDelete)
	}()

	data, err := h.encodeMessage(message)
	if err != nil {
		applogger.Errorf("消息序列化失败: %v", err)
		return
	}

	h.mu.RLock()
	for _, userID := range userIDs {
		if client, ok := h.clients[userID]; ok {
			select {
			case client.send <- data:
				// 发送成功
			default:
				// 发送失败，标记为待删除
				*toDelete = append(*toDelete, userID)
			}
		}
	}
	h.mu.RUnlock()

	// 在读锁外执行删除操作
	if len(*toDelete) > 0 {
		h.mu.Lock()
		for _, userID := range *toDelete {
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

// BroadcastRPAProgress 广播RPA执行进度消息（全员）
func (h *NoticeHub) BroadcastRPAProgress(message RPAProgressMessage) {
	// 将RPA消息转换为通用消息格式
	data, err := json.Marshal(message)
	if err != nil {
		return
	}
	h.broadcast <- NoticeMessage{
		Type:      message.Type,
		Timestamp: time.Now().Unix(),
		Content:   string(data),
	}
}

// BroadcastRPAProgressToUser 向指定用户推送RPA执行进度消息
func (h *NoticeHub) BroadcastRPAProgressToUser(userID string, message RPAProgressMessage) {
	h.BroadcastRPAProgressToUsers([]string{userID}, message)
}

// BroadcastRPAProgressToUsers 向多个用户推送RPA执行进度消息
func (h *NoticeHub) BroadcastRPAProgressToUsers(userIDs []string, message RPAProgressMessage) {
	// 先收集需要删除的客户端，避免在读锁中进行写操作
	// 遵循 Go 最佳实践：使用 sync.Pool 复用切片
	toDelete := h.toDeletePool.Get().(*[]string)
	defer func() {
		*toDelete = (*toDelete)[:0]
		h.toDeletePool.Put(toDelete)
	}()

	data, err := json.Marshal(message)
	if err != nil {
		applogger.Errorf("RPA消息序列化失败: %v", err)
		return
	}

	h.mu.RLock()
	for _, userID := range userIDs {
		if client, ok := h.clients[userID]; ok {
			select {
			case client.send <- data:
				// 发送成功
			default:
				// 发送失败，标记为待删除
				*toDelete = append(*toDelete, userID)
			}
		}
	}
	h.mu.RUnlock()

	// 在读锁外执行删除操作
	if len(*toDelete) > 0 {
		h.mu.Lock()
		for _, userID := range *toDelete {
			if client, ok := h.clients[userID]; ok {
				delete(h.clients, userID)
				close(client.send)
			}
		}
		h.mu.Unlock()
	}
}

// encodeMessage 编码消息
// 遵循 Go 最佳实践：不忽略错误
func (h *NoticeHub) encodeMessage(msg NoticeMessage) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("消息序列化失败: %w", err)
	}
	return data, nil
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
				// Hub 关闭了通道
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			// 遵循 Go 最佳实践：检查 WriteMessage 的错误
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				applogger.Errorf("WebSocket 写入失败 (用户: %s): %v", c.userID, err)
				return
			}
		case <-ticker.C:
			// 发送心跳 ping
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				applogger.Errorf("WebSocket Ping 发送失败 (用户: %s): %v", c.userID, err)
				return
			}
		}
	}
}

// readPump 读循环,探测僵尸连接 (P1 fix)
//
// 职责:
//   - 设置 ReadDeadline,定时刷新(基于客户端 pong 响应)
//   - 监听 pong 消息确认连接活跃
//   - 监听 close/客户端断连,调用 UnregisterClient 清理 hub
//
// 没有 readPump 时,writePump 只有主动发消息才能检测到断连,
// 客户端拔网线/进程睡眠等情况下 conn 残留,FD + Client 记录泄漏。
func (c *Client) readPump() {
	defer func() {
		// 出循环说明连接异常,主动从 hub 注销
		c.hub.UnregisterClient(c)
		_ = c.conn.Close()
	}()

	// readDeadline = pingInterval (54s) * 2,确保至少 1 个 ping 周期容错
	const readDeadline = 110 * time.Second
	_ = c.conn.SetReadDeadline(time.Now().Add(readDeadline))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	for {
		// 不关心消息内容(本 hub 为单向推送),仅用 ReadMessage
		// 触发 ReadDeadline / Pong / Close 处理。
		if _, _, err := c.conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				applogger.Warnf("WebSocket 异常关闭 (用户: %s): %v", c.userID, err)
			}
			return
		}
	}
}
