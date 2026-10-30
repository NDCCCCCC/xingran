package services

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// DeviceCredentialHelper 设备凭证辅助类
// 统一管理设备凭证的获取和缓存逻辑
type DeviceCredentialHelper struct {
	db *gorm.DB
}

// NewDeviceCredentialHelper 创建设备凭证辅助类
func NewDeviceCredentialHelper(db *gorm.DB) *DeviceCredentialHelper {
	return &DeviceCredentialHelper{db: db}
}

// GetDeviceCredential 获取设备凭证
// 优先使用设备关联的凭证，如果设备没有关联凭证则使用默认凭证
func (h *DeviceCredentialHelper) GetDeviceCredential(ctx context.Context, device *models.NetworkDevice) (*models.AuthCredential, error) {
	// 如果设备有关联的凭证ID，优先使用
	if device.CredentialID != nil && *device.CredentialID != "" {
		var credential models.AuthCredential
		if err := h.db.WithContext(ctx).Where("id = ?", *device.CredentialID).First(&credential).Error; err != nil {
			return nil, fmt.Errorf("查询设备关联凭证失败: %w", err)
		}
		return &credential, nil
	}

	// 否则使用默认凭证
	var credential models.AuthCredential
	if err := h.db.WithContext(ctx).Where("is_default = ?", true).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("未找到可用凭证")
	}
	return &credential, nil
}

// GetCredentialByID 根据ID获取凭证
func (h *DeviceCredentialHelper) GetCredentialByID(ctx context.Context, credentialID string) (*models.AuthCredential, error) {
	var credential models.AuthCredential
	if err := h.db.WithContext(ctx).Where("id = ?", credentialID).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("查询凭证失败: %w", err)
	}
	return &credential, nil
}

// GetDefaultCredential 获取默认凭证
func (h *DeviceCredentialHelper) GetDefaultCredential(ctx context.Context) (*models.AuthCredential, error) {
	var credential models.AuthCredential
	if err := h.db.WithContext(ctx).Where("is_default = ?", true).First(&credential).Error; err != nil {
		return nil, fmt.Errorf("未找到默认凭证")
	}
	return &credential, nil
}

// GetCredentialsForDevices 批量获取多个设备的凭证
// 返回一个map，key为设备ID，value为对应的凭证
// 这个方法可以避免N+1查询问题
func (h *DeviceCredentialHelper) GetCredentialsForDevices(ctx context.Context, devices []models.NetworkDevice) (map[string]*models.AuthCredential, error) {
	// 分离有关联凭证的设备和没有关联凭证的设备
	var deviceIDsNeedingDefault []string
	credentialIDsMap := make(map[string][]string) // credentialID -> deviceIDs

	for _, device := range devices {
		if device.CredentialID != nil && *device.CredentialID != "" {
			credentialIDsMap[*device.CredentialID] = append(credentialIDsMap[*device.CredentialID], device.ID)
		} else {
			deviceIDsNeedingDefault = append(deviceIDsNeedingDefault, device.ID)
		}
	}

	result := make(map[string]*models.AuthCredential)

	// 批量查询所有需要的凭证
	var credentialIDs []string
	for credID := range credentialIDsMap {
		credentialIDs = append(credentialIDs, credID)
	}

	var credentials []models.AuthCredential
	if len(credentialIDs) > 0 {
		if err := h.db.WithContext(ctx).Where("id IN ?", credentialIDs).Find(&credentials).Error; err != nil {
			return nil, fmt.Errorf("批量查询凭证失败: %w", err)
		}

		// 构建凭证ID到凭证的映射
		credMap := make(map[string]*models.AuthCredential)
		for i := range credentials {
			credMap[credentials[i].ID] = &credentials[i]
		}

		// 为每个设备分配凭证
		for credID, deviceIDs := range credentialIDsMap {
			cred := credMap[credID]
			if cred != nil {
				for _, deviceID := range deviceIDs {
					result[deviceID] = cred
				}
			}
		}
	}

	// 获取默认凭证（如果有设备需要）
	if len(deviceIDsNeedingDefault) > 0 {
		defaultCred, err := h.GetDefaultCredential(ctx)
		if err != nil {
			return nil, fmt.Errorf("获取默认凭证失败: %w", err)
		}
		for _, deviceID := range deviceIDsNeedingDefault {
			result[deviceID] = defaultCred
		}
	}

	return result, nil
}
