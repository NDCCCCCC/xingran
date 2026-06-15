package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ConnectionState 连接状态
type ConnectionState int

const (
	Disconnected ConnectionState = iota
	Connecting
	Connected
	Reconnecting
)

func (s ConnectionState) String() string {
	switch s {
	case Disconnected:
		return "disconnected"
	case Connecting:
		return "connecting"
	case Connected:
		return "connected"
	case Reconnecting:
		return "reconnecting"
	default:
		return "unknown"
	}
}

// agentMaxReconnectDelay Agent 重连延迟上限
const agentMaxReconnectDelay = 5 * time.Minute

// ConnectionManager 连接管理器
type ConnectionManager struct {
	mu              sync.RWMutex
	state           ConnectionState
	lastConnected   time.Time
	lastDisconnect  time.Time
	reconnectCount  int
	maxReconnects   int
	reconnectDelay  time.Duration
	authenticator   *JWTAuthenticator
	onStateChange   func(ConnectionState)
	stopCh          chan struct{}
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager(auth *JWTAuthenticator) *ConnectionManager {
	return &ConnectionManager{
		state:          Disconnected,
		maxReconnects:  10,
		reconnectDelay: 5 * time.Second,
		authenticator:  auth,
		stopCh:         make(chan struct{}),
	}
}

// SetStateChangeCallback 设置状态变更回调
func (cm *ConnectionManager) SetStateChangeCallback(fn func(ConnectionState)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.onStateChange = fn
}

// GetState 获取当前连接状态
func (cm *ConnectionManager) GetState() ConnectionState {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.state
}

// IsConnected 检查是否已连接
func (cm *ConnectionManager) IsConnected() bool {
	return cm.GetState() == Connected
}

// Connect 连接到后端
func (cm *ConnectionManager) Connect(ctx context.Context) error {
	cm.mu.Lock()
	cm.state = Connecting
	cm.mu.Unlock()

	cm.notifyStateChange(Connecting)

	// 尝试注册
	if err := cm.authenticator.Register(ctx); err != nil {
		cm.mu.Lock()
		cm.state = Disconnected
		cm.lastDisconnect = time.Now()
		cm.mu.Unlock()

		cm.notifyStateChange(Disconnected)
		return fmt.Errorf("registration failed: %w", err)
	}

	// 发送初始心跳
	if err := cm.authenticator.SendHeartbeat(ctx); err != nil {
		cm.mu.Lock()
		cm.state = Disconnected
		cm.lastDisconnect = time.Now()
		cm.mu.Unlock()

		cm.notifyStateChange(Disconnected)
		return fmt.Errorf("heartbeat failed: %w", err)
	}

	// 连接成功
	cm.mu.Lock()
	cm.state = Connected
	cm.lastConnected = time.Now()
	cm.reconnectCount = 0
	cm.mu.Unlock()

	cm.notifyStateChange(Connected)
	return nil
}

// Disconnect 断开连接
func (cm *ConnectionManager) Disconnect() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.state == Connected || cm.state == Reconnecting {
		cm.state = Disconnected
		cm.lastDisconnect = time.Now()
		cm.notifyStateChange(Disconnected)
	}

	// 停止重连协程
	select {
	case cm.stopCh <- struct{}{}:
	default:
	}
}

// Reconnect 重连
func (cm *ConnectionManager) Reconnect(ctx context.Context) error {
	cm.mu.Lock()
	if cm.reconnectCount >= cm.maxReconnects {
		cm.mu.Unlock()
		return fmt.Errorf("max reconnects (%d) exceeded", cm.maxReconnects)
	}
	cm.reconnectCount++
	currentDelay := cm.reconnectDelay * time.Duration(cm.reconnectCount)
	if currentDelay > agentMaxReconnectDelay {
		currentDelay = agentMaxReconnectDelay
	}
	cm.state = Reconnecting
	cm.mu.Unlock()

	cm.notifyStateChange(Reconnecting)

	// 等待延迟
	select {
	case <-time.After(currentDelay):
	case <-ctx.Done():
		return ctx.Err()
	case <-cm.stopCh:
		return fmt.Errorf("reconnect canceled")
	}

	// 尝试重新连接
	return cm.Connect(ctx)
}

// StartHealthMonitor 启动健康监控
func (cm *ConnectionManager) StartHealthMonitor(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if cm.IsConnected() {
				if err := cm.authenticator.SendHeartbeat(ctx); err != nil {
					WithFields(logrus.Fields{
						"error": err.Error(),
					}).Warn("Health check failed")
					go func() {
						if err := cm.Reconnect(context.Background()); err != nil {
							WithFields(logrus.Fields{
								"error":           err.Error(),
								"reconnect_count": cm.reconnectCount,
							}).Warn("Reconnect failed")
						}
					}()
				}
			} else {
				go func() {
					if err := cm.Reconnect(context.Background()); err != nil {
						WithFields(logrus.Fields{
							"error":           err.Error(),
							"reconnect_count": cm.reconnectCount,
						}).Warn("Reconnect failed")
					}
				}()
			}
		case <-ctx.Done():
			return
		case <-cm.stopCh:
			return
		}
	}
}

// notifyStateChange 通知状态变更
func (cm *ConnectionManager) notifyStateChange(state ConnectionState) {
	if cm.onStateChange != nil {
		cm.onStateChange(state)
	}
}

// GetStats 获取连接统计信息
func (cm *ConnectionManager) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return map[string]interface{}{
		"state":           cm.state.String(),
		"last_connected":  cm.lastConnected,
		"last_disconnect": cm.lastDisconnect,
		"reconnect_count": cm.reconnectCount,
	}
}
