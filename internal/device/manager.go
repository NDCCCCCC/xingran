package device

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// Manager 设备连接管理器
type Manager struct {
	db              *gorm.DB
	sm4Cipher       *security.SM4Cipher // SM4 加密器，用于解密凭证密码
	connections     map[string]*ScrapliWrapper
	connectionsLock sync.RWMutex
	semaphore       *semaphore.Weighted
	maxConcurrent   int
	timeout         time.Duration
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	MaxConcurrent int           // 最大并发连接数
	Timeout       time.Duration // 连接超时时间
}

// DefaultManagerConfig 默认管理器配置
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		MaxConcurrent: 10,
		Timeout:       30 * time.Second,
	}
}

// NewManager 创建设备连接管理器
func NewManager(db *gorm.DB, sm4Cipher *security.SM4Cipher, config *ManagerConfig) *Manager {
	if config == nil {
		config = DefaultManagerConfig()
	}

	return &Manager{
		db:            db,
		sm4Cipher:     sm4Cipher,
		connections:   make(map[string]*ScrapliWrapper),
		semaphore:     semaphore.NewWeighted(int64(config.MaxConcurrent)),
		maxConcurrent: config.MaxConcurrent,
		timeout:       config.Timeout,
	}
}

// ConnectToDevice 连接到单个设备
func (m *Manager) ConnectToDevice(ctx context.Context, device *models.NetworkDevice, credential *models.AuthCredential) (*ScrapliWrapper, error) {
	if device == nil {
		return nil, fmt.Errorf("设备信息不能为空")
	}
	if credential == nil {
		return nil, fmt.Errorf("授权凭证不能为空")
	}

	// 解密密码（严格模式）
	password := credential.Password
	if m.sm4Cipher != nil && password != "" {
		decrypted, err := m.sm4Cipher.Decrypt(password)
		if err != nil {
			return nil, fmt.Errorf("密码解密失败，请重新设置凭证密码: %v", err)
		}
		password = decrypted
	} else if m.sm4Cipher == nil && password != "" {
		// SM4 加密器未初始化
		return nil, fmt.Errorf("SM4加密器未初始化，无法解密密码")
	}

	// 检查是否已有连接
	m.connectionsLock.RLock()
	if wrapper, exists := m.connections[device.ID]; exists && wrapper.IsConnected() {
		m.connectionsLock.RUnlock()
		return wrapper, nil
	}
	m.connectionsLock.RUnlock()

	// 获取信号量（控制并发）
	if err := m.semaphore.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("获取连接信号量失败: %w", err)
	}
	defer m.semaphore.Release(1)

	// 创建连接（使用凭证中的协议类型）
	var wrapper *ScrapliWrapper
	var err error

	if device.Port > 0 && device.Port != 22 {
		wrapper, err = NewScrapliWrapperWithPort(device, credential.Username, password, device.Port, credential.ProtocolType)
	} else {
		wrapper, err = NewScrapliWrapper(device, credential.Username, password, credential.ProtocolType)
	}

	if err != nil {
		return nil, fmt.Errorf("创建连接失败: %w", err)
	}

	// 设置超时上下文（用于连接超时控制）
	_, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// 连接设备
	if err := wrapper.Open(); err != nil {
		return nil, fmt.Errorf("连接设备失败 [%s]: %w", device.IPAddress, err)
	}

	// 保存连接
	m.connectionsLock.Lock()
	m.connections[device.ID] = wrapper
	m.connectionsLock.Unlock()

	return wrapper, nil
}

// DisconnectFromDevice 断开设备连接
func (m *Manager) DisconnectFromDevice(deviceID string) error {
	m.connectionsLock.Lock()
	defer m.connectionsLock.Unlock()

	if wrapper, exists := m.connections[deviceID]; exists {
		if err := wrapper.Close(); err != nil {
			return fmt.Errorf("关闭连接失败: %w", err)
		}
		delete(m.connections, deviceID)
	}

	return nil
}

// DisconnectAll 断开所有连接
func (m *Manager) DisconnectAll() error {
	m.connectionsLock.Lock()
	defer m.connectionsLock.Unlock()

	var errs []error
	for deviceID, wrapper := range m.connections {
		if err := wrapper.Close(); err != nil {
			errs = append(errs, fmt.Errorf("断开设备 %s 失败: %w", deviceID, err))
		}
	}

	m.connections = make(map[string]*ScrapliWrapper)

	if len(errs) > 0 {
		return fmt.Errorf("断开连接时发生错误: %v", errs)
	}

	return nil
}

// GetConnection 获取已有连接
func (m *Manager) GetConnection(deviceID string) (*ScrapliWrapper, bool) {
	m.connectionsLock.RLock()
	defer m.connectionsLock.RUnlock()

	wrapper, exists := m.connections[deviceID]
	if exists && wrapper.IsConnected() {
		return wrapper, true
	}
	return nil, false
}

// ConnectWithCredential 通过设备ID和凭证ID连接
func (m *Manager) ConnectWithCredential(ctx context.Context, deviceID, credentialID string) (*ScrapliWrapper, error) {
	// 查询设备信息
	var device models.NetworkDevice
	if err := m.db.Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	// 查询凭证信息
	var credential models.AuthCredential
	if err := m.db.Where("id = ?", credentialID).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("查询凭证失败: %w", err)
	}

	return m.ConnectToDevice(ctx, &device, &credential)
}

// GetDeviceWithCredentials 获取设备及关联的凭证
func (m *Manager) GetDeviceWithCredentials(ctx context.Context, deviceID string) (*models.NetworkDevice, *models.AuthCredential, error) {
	// 查询设备信息
	var device models.NetworkDevice
	if err := m.db.Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, nil, fmt.Errorf("查询设备失败: %w", err)
	}

	// 如果设备关联了凭证，使用关联凭证
	if device.CredentialID != nil && *device.CredentialID != "" {
		var credential models.AuthCredential
		if err := m.db.Where("id = ?", *device.CredentialID).First(&credential).Error; err != nil {
			return nil, nil, fmt.Errorf("查询凭证失败: %w", err)
		}
		return &device, &credential, nil
	}

	// 否则查找默认凭证
	var credential models.AuthCredential
	if err := m.db.Where("is_default = ?", true).First(&credential).Error; err != nil {
		return nil, nil, fmt.Errorf("未找到默认凭证: %w", err)
	}

	return &device, &credential, nil
}

// ExecuteOnDevice 在设备上执行命令
func (m *Manager) ExecuteOnDevice(ctx context.Context, deviceID string, command string, stripPrompt bool) (string, error) {
	// 获取设备信息和凭证
	device, credential, err := m.GetDeviceWithCredentials(ctx, deviceID)
	if err != nil {
		return "", err
	}

	// 连接设备
	wrapper, err := m.ConnectToDevice(ctx, device, credential)
	if err != nil {
		return "", err
	}

	// 执行命令
	response, err := wrapper.SendCommand(command, stripPrompt)
	if err != nil {
		return "", err
	}

	return response.Result, nil
}

// ExecuteOnDevices 在多个设备上执行命令（并发）
func (m *Manager) ExecuteOnDevices(ctx context.Context, deviceIDs []string, command string) map[string]string {
	results := make(map[string]string)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, deviceID := range deviceIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			output, err := m.ExecuteOnDevice(ctx, id, command, true)
			mu.Lock()
			if err != nil {
				results[id] = fmt.Sprintf("ERROR: %v", err)
			} else {
				results[id] = output
			}
			mu.Unlock()
		}(deviceID)
	}

	wg.Wait()
	return results
}

// GetConfigFromDevice 获取设备配置
func (m *Manager) GetConfigFromDevice(ctx context.Context, deviceID string) (string, error) {
	// 获取设备信息和凭证
	device, credential, err := m.GetDeviceWithCredentials(ctx, deviceID)
	if err != nil {
		return "", err
	}

	// 连接设备
	wrapper, err := m.ConnectToDevice(ctx, device, credential)
	if err != nil {
		return "", err
	}

	// 获取配置
	return wrapper.GetConfig()
}

// Close 关闭管理器
func (m *Manager) Close() error {
	return m.DisconnectAll()
}

// GetActiveConnectionCount 获取活动连接数
func (m *Manager) GetActiveConnectionCount() int {
	m.connectionsLock.RLock()
	defer m.connectionsLock.RUnlock()

	count := 0
	for _, wrapper := range m.connections {
		if wrapper.IsConnected() {
			count++
		}
	}
	return count
}
