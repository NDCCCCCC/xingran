package device

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Manager 设备管理器（兼容层）
// 提供旧 API 的兼容方法，内部使用新架构（DeviceExecutor）
type Manager struct {
	db       *gorm.DB
	executor *DeviceExecutor // 新架构执行器
}

// NewManager 创建设备管理器
func NewManager(db *gorm.DB) *Manager {
	return &Manager{
		db: db,
	}
}

// SetExecutor 设置执行器（由 Core 初始化时调用）
func (m *Manager) SetExecutor(executor *DeviceExecutor) {
	m.executor = executor
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

// GetDevice 获取设备信息
func (m *Manager) GetDevice(deviceID string) (*models.NetworkDevice, error) {
	var device models.NetworkDevice
	if err := m.db.Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}
	return &device, nil
}

// GetCredential 获取凭证信息
func (m *Manager) GetCredential(credentialID string) (*models.AuthCredential, error) {
	var credential models.AuthCredential
	if err := m.db.Where("id = ?", credentialID).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("查询凭证失败: %w", err)
	}
	return &credential, nil
}

// GetDefaultCredential 获取默认凭证
func (m *Manager) GetDefaultCredential() (*models.AuthCredential, error) {
	var credential models.AuthCredential
	if err := m.db.Where("is_default = ?", true).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("未找到默认凭证: %w", err)
	}
	return &credential, nil
}

// ========== 以下为兼容层方法，内部使用新架构 ==========

// ConnectToDevice 连接到单个设备（兼容方法）
// 注意：此方法仅保留 API 兼容性，实际连接由连接池管理
// 返回的 ScrapliWrapper 仅供临时使用，不应长期持有
func (m *Manager) ConnectToDevice(ctx context.Context, device *models.NetworkDevice, credential *models.AuthCredential) (*ScrapliWrapper, error) {
	// 此方法不再创建真实连接，返回错误引导使用新架构
	return nil, fmt.Errorf("ConnectToDevice 已废弃，请使用 DeviceExecutor.ExecuteOnDevice")
}

// DisconnectFromDevice 断开设备连接（兼容方法，空操作）
func (m *Manager) DisconnectFromDevice(deviceID string) error {
	// 连接由连接池自动管理，此方法为空操作
	return nil
}

// DisconnectAll 断开所有连接（兼容方法，空操作）
func (m *Manager) DisconnectAll() error {
	// 连接由连接池自动管理，此方法为空操作
	return nil
}

// ExecuteOnDevice 在设备上执行命令（兼容方法）
func (m *Manager) ExecuteOnDevice(ctx context.Context, deviceID string, command string, stripPrompt bool) (string, error) {
	if m.executor == nil {
		return "", fmt.Errorf("DeviceExecutor 未初始化")
	}
	return m.executor.ExecuteOnDevice(ctx, deviceID, command, stripPrompt)
}

// ExecuteOnDevices 在多个设备上执行命令（兼容方法）
func (m *Manager) ExecuteOnDevices(ctx context.Context, deviceIDs []string, command string) map[string]string {
	results := make(map[string]string)
	if m.executor == nil {
		for _, deviceID := range deviceIDs {
			results[deviceID] = "ERROR: DeviceExecutor 未初始化"
		}
		return results
	}

	for _, deviceID := range deviceIDs {
		output, err := m.executor.ExecuteOnDevice(ctx, deviceID, command, true)
		if err != nil {
			results[deviceID] = fmt.Sprintf("ERROR: %v", err)
		} else {
			results[deviceID] = output
		}
	}
	return results
}

// GetConfigFromDevice 获取设备配置（兼容方法）
func (m *Manager) GetConfigFromDevice(ctx context.Context, deviceID string) (string, error) {
	if m.executor == nil {
		return "", fmt.Errorf("DeviceExecutor 未初始化")
	}
	return m.executor.GetConfig(ctx, deviceID)
}

// Close 关闭管理器（兼容方法，空操作）
func (m *Manager) Close() error {
	return nil
}

// GetActiveConnectionCount 获取活动连接数（兼容方法）
func (m *Manager) GetActiveConnectionCount() int {
	// 此方法不再可用，返回 -1 表示已废弃
	return -1
}
