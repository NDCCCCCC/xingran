package vdi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// ClientManager VDI 客户端管理器（单例模式）
// 该管理器通过 serverID 从数据库加载真实 VDI 服务器配置（models.VDIServer）。
// 不再使用占位 URL：当数据库中找不到对应 serverID 的配置或 Endpoint 为空时返回明确错误。
type ClientManager struct {
	clients map[string]*VDIClient
	mu      sync.RWMutex
	db      *gorm.DB
}

var (
	instance *ClientManager
	once     sync.Once
)

// InitClientManager 初始化全局 ClientManager（注入数据库句柄）。
// 必须先调用此方法（或使用 NewClientManager），否则 GetClientManager 会返回一个
// 不含数据库连接的实例，GetClient 会在缺少配置时返回错误。
func InitClientManager(db *gorm.DB) *ClientManager {
	once.Do(func() {
		instance = &ClientManager{
			clients: make(map[string]*VDIClient),
			db:      db,
		}
	})
	return instance
}

// GetClientManager 获取 ClientManager 单例（向后兼容）。
// 注意：若未先调用 InitClientManager 注入 db，则 GetClient 在需要新建客户端时会报错。
func GetClientManager() *ClientManager {
	once.Do(func() {
		instance = &ClientManager{
			clients: make(map[string]*VDIClient),
		}
	})
	return instance
}

// NewClientManager 创建带数据库连接的 ClientManager（推荐用于依赖注入场景）。
func NewClientManager(db *gorm.DB) *ClientManager {
	return &ClientManager{
		clients: make(map[string]*VDIClient),
		db:      db,
	}
}

// ErrVDIServerNotConfigured 表示指定的 VDI 服务器未在数据库中配置或 Endpoint 为空。
var ErrVDIServerNotConfigured = errors.New("VDI server is not configured: missing DB record or empty endpoint")

// loadServerFromDB 根据 serverID 从数据库加载 VDI 服务器配置。
// 找不到记录或 Endpoint 为空时返回 ErrVDIServerNotConfigured。
func (m *ClientManager) loadServerFromDB(serverID string) (*models.VDIServer, error) {
	if m.db == nil {
		return nil, fmt.Errorf("%w (ClientManager has no DB handle)", ErrVDIServerNotConfigured)
	}

	var server models.VDIServer
	if err := m.db.Where("id = ?", serverID).First(&server).Error; err != nil {
		return nil, fmt.Errorf("%w (id=%s): %w", ErrVDIServerNotConfigured, serverID, err)
	}

	if server.Endpoint == "" {
		return nil, fmt.Errorf("%w (id=%s): endpoint is empty", ErrVDIServerNotConfigured, serverID)
	}

	return &server, nil
}

// GetClient 获取或创建 VDI 客户端。
// 行为变更：不再使用占位 URL；必须先在数据库 sys_vdi_server 表中配置好对应 serverID 的记录。
// 若未配置或 Endpoint 为空，返回 ErrVDIServerNotConfigured（包装错误，可用 errors.Is 判断）。
// 客户端按 serverID 缓存，复用现有实例（双重检查锁保持并发安全）。
func (m *ClientManager) GetClient(ctx context.Context, serverID string) (*VDIClient, error) {
	if serverID == "" {
		return nil, fmt.Errorf("%w: serverID is empty", ErrVDIServerNotConfigured)
	}

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

	server, err := m.loadServerFromDB(serverID)
	if err != nil {
		return nil, err
	}

	newClient := &VDIClient{
		ServerURL: server.Endpoint,
		HTTPClient: &http.Client{
			Timeout: 30 * 1000, // 30 秒超时
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// F-06 fix: 不再硬编码,改为从 config.VDI.TLSSkipVerify 读取
					// 默认 true 保持向后兼容(VDI 自签证书),生产应在 yaml 中显式设 false
					InsecureSkipVerify: loadTLSSkipVerify(),
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
