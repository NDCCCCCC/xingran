package vdi

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"
)

// ClientManager VDI 客户端管理器（单例模式）
type ClientManager struct {
	clients map[string]*VDIClient
	mu      sync.RWMutex
}

var (
	instance *ClientManager
	once     sync.Once
)

// GetClientManager 获取 ClientManager 单例
func GetClientManager() *ClientManager {
	once.Do(func() {
		instance = &ClientManager{
			clients: make(map[string]*VDIClient),
		}
	})
	return instance
}

// GetClient 获取或创建 VDI 客户端
func (m *ClientManager) GetClient(ctx context.Context, serverID string) (*VDIClient, error) {
	m.mu.RLock()
	client, exists := m.clients[serverID]
	m.mu.RUnlock()

	if exists {
		return client, nil
	}

	// 创建新客户端
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if client, exists := m.clients[serverID]; exists {
		return client, nil
	}

	// TODO: 从数据库加载 VDI 服务器配置
	// 这里暂时创建一个默认客户端
	newClient := &VDIClient{
		ServerURL: "https://mock-vdi-server.example.com",
		HTTPClient: &http.Client{
			Timeout: 30 * 1000, // 30 秒超时
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // 仅用于测试，生产环境应验证证书
				},
			},
		},
	}

	m.clients[serverID] = newClient
	return newClient, nil
}

// RemoveClient 移除客户端
func (m *ClientManager) RemoveClient(serverID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, serverID)
}

// ClearAll 清空所有客户端
func (m *ClientManager) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients = make(map[string]*VDIClient)
}
