package operations

import (
	"context"
	"fmt"
	"regexp"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// WorkstationDeviceService 工位设备关联服务接口
type WorkstationDeviceService interface {
	// GetDevicesByWorkstation 获取工位关联设备列表
	GetDevicesByWorkstation(ctx context.Context, workstationID string) ([]*models.WorkstationDevice, error)

	// GetADDevicesByUser 获取用户的域控设备
	GetADDevicesByUser(ctx context.Context, username string) ([]*ADDeviceMatch, error)

	// GetAssetDevicesByUser 获取用户的资产设备
	GetAssetDevicesByUser(ctx context.Context, username string) ([]*AssetDeviceMatch, error)

	// AddDeviceManual 手动添加设备（序列号匹配）
	AddDeviceManual(ctx context.Context, req *AddDeviceRequest) (*models.WorkstationDevice, error)

	// SyncFromAD 同步域控设备到实际结果
	SyncFromAD(ctx context.Context, workstationID string) error

	// SyncFromAsset 同步资产设备到实际结果
	SyncFromAsset(ctx context.Context, workstationID string) error

	// UpdateDevice 更新设备信息
	UpdateDevice(ctx context.Context, id string, req *UpdateDeviceRequest) error

	// DeleteDevice 删除设备关联
	DeleteDevice(ctx context.Context, id string) error

	// SetPrimaryDevice 设置主设备
	SetPrimaryDevice(ctx context.Context, id string) error
}

// AddDeviceRequest 手动添加设备请求
type AddDeviceRequest struct {
	WorkstationID  string  `json:"workstationId"`
	DeviceSerial   string  `json:"deviceSerial"`
	DeviceName     *string `json:"deviceName,omitempty"`
	DeviceModel    *string `json:"deviceModel,omitempty"`
	DeviceType     *string `json:"deviceType,omitempty"`
	MACAddress     *string `json:"macAddress,omitempty"`
	ResponsibleUser *string `json:"responsibleUser,omitempty"`
	Description    *string `json:"description,omitempty"`
}

// UpdateDeviceRequest 更新设备请求
type UpdateDeviceRequest struct {
	DeviceSerial   *string `json:"deviceSerial,omitempty"`
	DeviceName     *string `json:"deviceName,omitempty"`
	DeviceModel    *string `json:"deviceModel,omitempty"`
	DeviceType     *string `json:"deviceType,omitempty"`
	MACAddress     *string `json:"macAddress,omitempty"`
	ResponsibleUser *string `json:"responsibleUser,omitempty"`
	Status         *int    `json:"status,omitempty"`
	Priority       *int    `json:"priority,omitempty"`
	IsPrimary      *bool   `json:"isPrimary,omitempty"`
	Description    *string `json:"description,omitempty"`
}

// ADDeviceMatch 域控设备匹配结果
type ADDeviceMatch struct {
	ADComputerID    string  `json:"adComputerId"`
	DeviceSerial    string  `json:"deviceSerial"`
	DeviceName      string  `json:"deviceName"`
	MACAddress      string  `json:"macAddress"`
	IPAddress       string  `json:"ipAddress"`
	OperatingSystem string  `json:"operatingSystem"`
}

// AssetDeviceMatch 资产设备匹配结果
type AssetDeviceMatch struct {
	AssetID          string  `json:"assetId"`
	DeviceSN         string  `json:"deviceSerial"`
	DeviceModel      *string `json:"deviceModel,omitempty"`
	DeviceType       *string `json:"deviceType,omitempty"`
	MACAddress       *string `json:"macAddress,omitempty"`
	ResponsibleUser  *string `json:"responsibleUser,omitempty"`
}

type workstationDeviceService struct {
	db            *gorm.DB
	uuidValidator *regexp.Regexp
}

// NewWorkstationDeviceService 创建工位设备服务实例
func NewWorkstationDeviceService(db *gorm.DB) WorkstationDeviceService {
	return &workstationDeviceService{
		db:            db,
		uuidValidator: regexp.MustCompile(uuidPattern),
	}
}

// GetDevicesByWorkstation 获取工位关联设备列表
func (s *workstationDeviceService) GetDevicesByWorkstation(ctx context.Context, workstationID string) ([]*models.WorkstationDevice, error) {
	// 验证工位ID格式
	if workstationID == "" {
		return nil, apperrors.ParamMissing("工位ID")
	}
	if !s.uuidValidator.MatchString(workstationID) {
		return nil, apperrors.ParamInvalid("工位ID")
	}

	var devices []*models.WorkstationDevice
	err := s.db.WithContext(ctx).
		Where("workstation_id = ? AND deleted_at IS NULL", workstationID).
		Order("priority DESC, created_at ASC").
		Find(&devices).Error

	if err != nil {
		return nil, fmt.Errorf("查询工位设备失败: %w", err)
	}

	return devices, nil
}

// GetADDevicesByUser 获取用户的域控设备
func (s *workstationDeviceService) GetADDevicesByUser(ctx context.Context, username string) ([]*ADDeviceMatch, error) {
	if username == "" {
		return nil, apperrors.ParamMissing("用户名")
	}

	// 查询该用户的AD设备记录
	// TODO: 实现AD用户与设备的关联逻辑
	// 当前需要扩展ADUser模型或创建ADUserComputer关联表

	return []*ADDeviceMatch{}, nil
}

// GetAssetDevicesByUser 获取用户的资产设备
func (s *workstationDeviceService) GetAssetDevicesByUser(ctx context.Context, username string) ([]*AssetDeviceMatch, error) {
	if username == "" {
		return nil, apperrors.ParamMissing("用户名")
	}

	var assets []*models.Asset
	err := s.db.WithContext(ctx).
		Where("nowuser_name = ? AND deleted_at IS NULL", username).
		Find(&assets).Error

	if err != nil {
		return nil, fmt.Errorf("查询用户资产设备失败: %w", err)
	}

	matches := make([]*AssetDeviceMatch, 0, len(assets))
	for _, asset := range assets {
		matches = append(matches, &AssetDeviceMatch{
			AssetID:         asset.ID,
			DeviceSN:        asset.DeviceSN,
			DeviceModel:     asset.DeviceModelName,
			DeviceType:      asset.DeviceTypeName,
			MACAddress:      asset.MAC1,
			ResponsibleUser: asset.NowUserName,
		})
	}

	return matches, nil
}

// AddDeviceManual 手动添加设备
func (s *workstationDeviceService) AddDeviceManual(ctx context.Context, req *AddDeviceRequest) (*models.WorkstationDevice, error) {
	// 验证工位ID
	if req.WorkstationID == "" {
		return nil, apperrors.ParamMissing("工位ID")
	}
	if !s.uuidValidator.MatchString(req.WorkstationID) {
		return nil, apperrors.ParamInvalid("工位ID")
	}

	// 验证序列号
	if req.DeviceSerial == "" {
		return nil, apperrors.ParamMissing("设备序列号")
	}

	// 尝试通过序列号匹配资产系统
	var assetID *string
	var deviceModel, deviceType *string

	var asset models.Asset
	err := s.db.WithContext(ctx).
		Where("devicesn = ? AND deleted_at IS NULL", req.DeviceSerial).
		First(&asset).Error

	if err == nil {
		// 找到匹配的资产
		assetID = &asset.ID
		if deviceModel == nil {
			deviceModel = asset.DeviceModelName
		}
		if deviceType == nil {
			deviceType = asset.DeviceTypeName
		}
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询资产失败: %w", err)
	}

	device := &models.WorkstationDevice{
		WorkstationID:    req.WorkstationID,
		DeviceSource:     models.DeviceSourceManual,
		DeviceSerial:     &req.DeviceSerial,
		AssetID:          assetID,
		DeviceName:       req.DeviceName,
		DeviceModel:      deviceModel,
		DeviceType:       deviceType,
		MACAddress:       req.MACAddress,
		ResponsibleUser:  req.ResponsibleUser,
		Status:           0, // 默认正常
		IsPrimary:        false,
		Priority:         0,
		Description:      req.Description,
	}

	if err := s.db.WithContext(ctx).Create(device).Error; err != nil {
		return nil, fmt.Errorf("添加设备失败: %w", err)
	}

	return device, nil
}

// SyncFromAD 同步域控设备到实际结果
func (s *workstationDeviceService) SyncFromAD(ctx context.Context, workstationID string) error {
	if workstationID == "" {
		return apperrors.ParamMissing("工位ID")
	}

	// 获取工位信息以查找绑定的用户
	var workstation models.Workstation
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", workstationID).
		First(&workstation).Error

	if err != nil {
		return fmt.Errorf("工位不存在: %w", err)
	}

	// 如果工位没有绑定用户,无法同步
	if workstation.UserName == nil || *workstation.UserName == "" {
		return fmt.Errorf("工位未绑定用户,无法同步域控设备")
	}

	// 获取用户的域控设备
	adDevices, err := s.GetADDevicesByUser(ctx, *workstation.UserName)
	if err != nil {
		return fmt.Errorf("获取域控设备失败: %w", err)
	}

	// 删除现有的域控来源设备
	if err := s.db.WithContext(ctx).
		Where("workstation_id = ? AND device_source = ? AND deleted_at IS NULL", workstationID, models.DeviceSourceAD).
		Delete(&models.WorkstationDevice{}).Error; err != nil {
		return fmt.Errorf("删除现有域控设备失败: %w", err)
	}

	// 添加新的域控设备
	for _, adDevice := range adDevices {
		device := &models.WorkstationDevice{
			WorkstationID:    workstationID,
			DeviceSource:     models.DeviceSourceAD,
			ADComputerID:     &adDevice.ADComputerID,
			DeviceSerial:     &adDevice.DeviceSerial,
			DeviceName:       &adDevice.DeviceName,
			MACAddress:       &adDevice.MACAddress,
			Status:           0,
			IsPrimary:        false,
			Priority:         0,
		}

		if err := s.db.WithContext(ctx).Create(device).Error; err != nil {
			return fmt.Errorf("添加域控设备失败: %w", err)
		}
	}

	return nil
}

// SyncFromAsset 同步资产设备到实际结果
func (s *workstationDeviceService) SyncFromAsset(ctx context.Context, workstationID string) error {
	if workstationID == "" {
		return apperrors.ParamMissing("工位ID")
	}

	// 获取工位信息以查找绑定的用户
	var workstation models.Workstation
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", workstationID).
		First(&workstation).Error

	if err != nil {
		return fmt.Errorf("工位不存在: %w", err)
	}

	// 如果工位没有绑定用户,无法同步
	if workstation.UserName == nil || *workstation.UserName == "" {
		return fmt.Errorf("工位未绑定用户,无法同步资产设备")
	}

	// 获取用户的资产设备
	assetDevices, err := s.GetAssetDevicesByUser(ctx, *workstation.UserName)
	if err != nil {
		return fmt.Errorf("获取资产设备失败: %w", err)
	}

	// 删除现有的资产来源设备
	if err := s.db.WithContext(ctx).
		Where("workstation_id = ? AND device_source = ? AND deleted_at IS NULL", workstationID, models.DeviceSourceAsset).
		Delete(&models.WorkstationDevice{}).Error; err != nil {
		return fmt.Errorf("删除现有资产设备失败: %w", err)
	}

	// 添加新的资产设备
	for _, assetDevice := range assetDevices {
		device := &models.WorkstationDevice{
			WorkstationID:     workstationID,
			DeviceSource:      models.DeviceSourceAsset,
			AssetID:           &assetDevice.AssetID,
			DeviceSerial:      &assetDevice.DeviceSN,
			DeviceModel:       assetDevice.DeviceModel,
			DeviceType:        assetDevice.DeviceType,
			MACAddress:        assetDevice.MACAddress,
			ResponsibleUser:   assetDevice.ResponsibleUser,
			Status:            0,
			IsPrimary:         false,
			Priority:          0,
		}

		if err := s.db.WithContext(ctx).Create(device).Error; err != nil {
			return fmt.Errorf("添加资产设备失败: %w", err)
		}
	}

	return nil
}

// UpdateDevice 更新设备信息
func (s *workstationDeviceService) UpdateDevice(ctx context.Context, id string, req *UpdateDeviceRequest) error {
	if id == "" {
		return apperrors.ParamMissing("设备ID")
	}

	// 验证设备存在
	var device models.WorkstationDevice
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&device).Error

	if err != nil {
		return fmt.Errorf("设备不存在: %w", err)
	}

	// 更新字段
	updates := make(map[string]interface{})

	if req.DeviceSerial != nil {
		updates["device_serial"] = *req.DeviceSerial
	}
	if req.DeviceName != nil {
		updates["device_name"] = *req.DeviceName
	}
	if req.DeviceModel != nil {
		updates["device_model"] = *req.DeviceModel
	}
	if req.DeviceType != nil {
		updates["device_type"] = *req.DeviceType
	}
	if req.MACAddress != nil {
		updates["mac_address"] = *req.MACAddress
	}
	if req.ResponsibleUser != nil {
		updates["responsible_user"] = *req.ResponsibleUser
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.IsPrimary != nil {
		updates["is_primary"] = *req.IsPrimary
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if err := s.db.WithContext(ctx).
		Model(&device).
		Updates(updates).
		Error; err != nil {
		return fmt.Errorf("更新设备失败: %w", err)
	}

	return nil
}

// DeleteDevice 删除设备关联
func (s *workstationDeviceService) DeleteDevice(ctx context.Context, id string) error {
	if id == "" {
		return apperrors.ParamMissing("设备ID")
	}

	err := s.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.WorkstationDevice{}).Error

	if err != nil {
		return fmt.Errorf("删除设备失败: %w", err)
	}

	return nil
}

// SetPrimaryDevice 设置主设备
func (s *workstationDeviceService) SetPrimaryDevice(ctx context.Context, id string) error {
	if id == "" {
		return apperrors.ParamMissing("设备ID")
	}

	// 验证设备存在并获取工位ID
	var device models.WorkstationDevice
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&device).Error

	if err != nil {
		return fmt.Errorf("设备不存在: %w", err)
	}

	// 使用事务确保原子性
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 取消该工位下的所有主设备
		if err := tx.
			Model(&models.WorkstationDevice{}).
			Where("workstation_id = ? AND deleted_at IS NULL", device.WorkstationID).
			Update("is_primary", false).Error; err != nil {
			return fmt.Errorf("取消主设备失败: %w", err)
		}

		// 设置新的主设备
		if err := tx.
			Model(&models.WorkstationDevice{}).
			Where("id = ?", id).
			Update("is_primary", true).Error; err != nil {
			return fmt.Errorf("设置主设备失败: %w", err)
		}

		return nil
	})
}
