package operations

import (
	"context"
	"fmt"
	"regexp"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/constants"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// WorkstationDeviceService 工位设备关联服务接口
type WorkstationDeviceService interface {
	// GetDevicesByWorkstation 获取工位关联设备列表（可选来源过滤）
	GetDevicesByWorkstation(ctx context.Context, workstationID string, source ...string) ([]*models.WorkstationDevice, error)

	// GetADDevices 实时查询AD设备（不保存到数据库）
	GetADDevices(ctx context.Context, workstationID string) ([]*models.WorkstationDevice, error)

	// GetAssetDevices 实时查询资产设备（不保存到数据库）
	GetAssetDevices(ctx context.Context, workstationID string) ([]*models.WorkstationDevice, error)

	// GetPhysicalDevices 实时查询物理链路设备（不保存到数据库）
	// Phase 45 R5: MAC→port→infoPoint→workstation 反推链汇总
	GetPhysicalDevices(ctx context.Context, workstationID string) ([]*models.WorkstationDevice, error)

	// GetADDevicesByUser 获取用户的域控设备
	GetADDevicesByUser(ctx context.Context, userID string) ([]*ADDeviceMatch, error)

	// GetAssetDevicesByUser 获取用户的资产设备
	GetAssetDevicesByUser(ctx context.Context, userID, username, nickname string) ([]*AssetDeviceMatch, error)

	// AddDeviceManual 手动添加设备（序列号匹配）
	AddDeviceManual(ctx context.Context, req *AddDeviceRequest) (*models.WorkstationDevice, error)

	// SetPrimaryAndSave 设置主设备并保存到数据库（用于AD/资产设备转为手动设备）。
	// 在保存为 manual 设备前，会以 device_serial 为键实时拉取 AD/资产两侧数据并合并：
	//   - deviceName 优先取 AD 的 DeviceName，再 fallback 到 req
	//   - deviceModel/deviceType 优先取资产
	//   - macAddress 优先取 AD MAC，再 fallback 资产 MAC1，再 req
	//   - ipAddress 优先取 AD 的 IPAddress，再 fallback req
	//   - responsibleUser 优先取资产的 NowUserName，再 fallback req
	//   - assetID/adComputerID 在两侧分别命中时填充
	// 任一来源查询失败时仅记录 warn 并降级继续，不阻塞保存。
	SetPrimaryAndSave(ctx context.Context, deviceID string, req *SetPrimaryAndSaveRequest) error

	// UpdateDevice 更新设备信息
	UpdateDevice(ctx context.Context, id string, req *UpdateDeviceRequest) error

	// DeleteDevice 删除设备关联
	DeleteDevice(ctx context.Context, id string) error

	// SetPrimaryDevice 设置主设备
	SetPrimaryDevice(ctx context.Context, id string) error

	// SyncFromAD 同步域控设备到实际结果
	SyncFromAD(ctx context.Context, workstationID string) error

	// SyncFromAsset 同步资产设备到实际结果
	SyncFromAsset(ctx context.Context, workstationID string) error
}

// SetPrimaryAndSaveRequest 设置主设备并保存请求
type SetPrimaryAndSaveRequest struct {
	WorkstationID  string  `json:"workstationId"`
	DeviceSerial   string  `json:"deviceSerial"`
	DeviceName     string  `json:"deviceName"`
	DeviceModel    *string `json:"deviceModel,omitempty"`
	DeviceType     *string `json:"deviceType,omitempty"`
	MACAddress     *string `json:"macAddress,omitempty"`
	IPAddress      *string `json:"ipAddress,omitempty"`
	ResponsibleUser *string `json:"responsibleUser,omitempty"`
}

// AddDeviceRequest 手动添加设备请求
type AddDeviceRequest struct {
	WorkstationID  string  `json:"workstationId"`
	DeviceSerial   string  `json:"deviceSerial"`
	DeviceName     *string `json:"deviceName,omitempty"`
	DeviceModel    *string `json:"deviceModel,omitempty"`
	DeviceType     *string `json:"deviceType,omitempty"`
	MACAddress     *string `json:"macAddress,omitempty"`
	IPAddress      *string `json:"ipAddress,omitempty"`
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
	IPAddress      *string `json:"ipAddress,omitempty"`
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
		uuidValidator: constants.UuidPattern,
	}
}

// GetDevicesByWorkstation 获取工位关联设备列表（支持来源过滤）
func (s *workstationDeviceService) GetDevicesByWorkstation(ctx context.Context, workstationID string, source ...string) ([]*models.WorkstationDevice, error) {
	// 验证工位ID格式
	if workstationID == "" {
		return nil, apperrors.ParamMissing("工位ID")
	}
	if !s.uuidValidator.MatchString(workstationID) {
		return nil, apperrors.ParamInvalid("工位ID")
	}

	var devices []*models.WorkstationDevice
	query := s.db.WithContext(ctx).
		Where("workstation_id = ? AND deleted_at IS NULL", workstationID)

	// 如果指定了来源，添加来源过滤
	if len(source) > 0 && source[0] != "" {
		query = query.Where("device_source = ?", source[0])
	}

	err := query.
		Order("priority DESC, created_at ASC").
		Find(&devices).Error

	if err != nil {
		return nil, fmt.Errorf("查询工位设备失败: %w", err)
	}

	return devices, nil
}

// GetADDevices 实时查询AD设备（不保存到数据库）
func (s *workstationDeviceService) GetADDevices(ctx context.Context, workstationID string) ([]*models.WorkstationDevice, error) {
	if workstationID == "" {
		return nil, apperrors.ParamMissing("工位ID")
	}

	// 获取工位信息以查找绑定的用户
	var workstation models.Workstation
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", workstationID).
		First(&workstation).Error

	if err != nil {
		return nil, fmt.Errorf("工位不存在: %w", err)
	}

	// 如果工位没有绑定用户ID，返回空
	var userID string
	if workstation.UserID != nil && *workstation.UserID != "" {
		userID = *workstation.UserID
	} else {
		return []*models.WorkstationDevice{}, nil
	}

	// 获取用户的域控设备
	adDevices, err := s.GetADDevicesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取域控设备失败: %w", err)
	}

	// 转换为 WorkstationDevice 格式（不保存）
	devices := make([]*models.WorkstationDevice, 0, len(adDevices))
	for i, adDevice := range adDevices {
		tempID := fmt.Sprintf("ad-%d", i)
		adDeviceCopy := adDevice
		devices = append(devices, &models.WorkstationDevice{
			BaseModel:        models.BaseModel{ID: tempID},
			WorkstationID:    workstationID,
			DeviceSource:     models.DeviceSourceAD,
			ADComputerID:     &adDeviceCopy.ADComputerID,
			DeviceSerial:     &adDeviceCopy.DeviceSerial,
			DeviceName:       &adDeviceCopy.DeviceName,
			MACAddress:       &adDeviceCopy.MACAddress,
			Status:           0,
			IsPrimary:        false,
			Priority:         0,
		})
	}

	return devices, nil
}

// GetAssetDevices 实时查询资产设备（不保存到数据库）
func (s *workstationDeviceService) GetAssetDevices(ctx context.Context, workstationID string) ([]*models.WorkstationDevice, error) {
	if workstationID == "" {
		return nil, apperrors.ParamMissing("工位ID")
	}

	// 获取工位信息以查找绑定的用户
	var workstation models.Workstation
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", workstationID).
		First(&workstation).Error

	if err != nil {
		return nil, fmt.Errorf("工位不存在: %w", err)
	}

	// 获取工位绑定的用户信息
	var userInfo struct {
		ID       string
		Username string
		Nickname *string
	}

	if workstation.UserID != nil && *workstation.UserID != "" {
		err := s.db.WithContext(ctx).
			Table("sys_user").
			Select("id, username, nickname").
			Where("id = ? AND deleted_at IS NULL", *workstation.UserID).
			First(&userInfo).Error
		if err != nil {
			return nil, fmt.Errorf("查询用户信息失败: %w", err)
		}
	} else {
		return []*models.WorkstationDevice{}, nil
	}

	// 获取用户的资产设备
	assetDevices, err := s.GetAssetDevicesByUser(ctx, userInfo.ID, userInfo.Username,
		func() string {
			if userInfo.Nickname != nil {
				return *userInfo.Nickname
			}
			return ""
		}())
	if err != nil {
		return nil, fmt.Errorf("获取资产设备失败: %w", err)
	}

	// 转换为 WorkstationDevice 格式（不保存）
	devices := make([]*models.WorkstationDevice, 0, len(assetDevices))
	for i, assetDevice := range assetDevices {
		tempID := fmt.Sprintf("asset-%d", i)
		assetDeviceCopy := assetDevice
		devices = append(devices, &models.WorkstationDevice{
			BaseModel:        models.BaseModel{ID: tempID},
			WorkstationID:    workstationID,
			DeviceSource:     models.DeviceSourceAsset,
			AssetID:          &assetDeviceCopy.AssetID,
			DeviceSerial:     &assetDeviceCopy.DeviceSN,
			DeviceModel:      assetDeviceCopy.DeviceModel,
			DeviceType:       assetDeviceCopy.DeviceType,
			MACAddress:       assetDeviceCopy.MACAddress,
			ResponsibleUser:  assetDeviceCopy.ResponsibleUser,
			Status:           0,
			IsPrimary:        false,
			Priority:         0,
		})
	}

	return devices, nil
}

// GetPhysicalDevices 实时查询物理链路设备（不保存到数据库）
//
// Phase 45 R5: 按 MAC→port→infoPoint→workstation 反推链路,找到该工位用户实际接入的设备
//
// 链路(同 reconciliation_physical_chain 物化视图):
//   asset.mac1/mac2 → sys_device_mac_address → sys_device_port_status
//     → ops_info_points.workstation_id → sys_workstation.user_id (= 工位绑定用户)
//
// 字段合并策略(用户要求"以资产为准"):
//   - DeviceName / DeviceModel / DeviceType / ResponsibleUser / IP: 优先取 ops_asset
//   - MAC: 优先 ops_asset.mac1,缺失时回退 ad_computer.mac_address
//   - AD 字段(ComputerName / OperatingSystem): 用 ad_computer 补全,但不覆盖资产字段
//
// 退化场景(与 GetAssetDevicesByUser 双策略同款):
//   - 工位未绑定 user_id → 返回空(不擅自用工作站 IP 反查)
//   - MAC 未采集 → 无设备返回
func (s *workstationDeviceService) GetPhysicalDevices(ctx context.Context, workstationID string) ([]*models.WorkstationDevice, error) {
	if workstationID == "" {
		return nil, apperrors.ParamMissing("工位ID")
	}

	if !s.uuidValidator.MatchString(workstationID) {
		return nil, apperrors.ParamInvalid("工位ID")
	}

	// 1. 获取工位信息以查找绑定的用户
	var workstation models.Workstation
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", workstationID).
		First(&workstation).Error
	if err != nil {
		return nil, fmt.Errorf("工位不存在: %w", err)
	}

	// 工位未绑定用户时返回空数组(对齐 AD/资产设备行为)
	if workstation.UserID == nil || *workstation.UserID == "" {
		return []*models.WorkstationDevice{}, nil
	}
	userID := *workstation.UserID

	logger.Infof("[GetPhysicalDevices] 工位 %s 绑定用户: %s", workstationID, userID)

	// 2. 走 MAC→port→infoPoint→workstation→user 反推链
	//    反推逻辑:
	//    (1) ops_info_points.workstation_id = 当前工位 → 拿到 port_id
	//    (2) sys_device_port_status.id = port_id → 拿到 device_id (= 网络设备)
	//    (3) sys_device_mac_address.device_id + interface_name → 拿到 mac_address
	//    (4) ops_asset.mac1 = mac_address (或 mac2) → 拿到 asset 详情
	//    (5) sys_ad_computer.mac_address (或 serial_number) → 合并 AD 字段
	//
	//    使用 DISTINCT ON (devicesn) 去重,合并字段"资产优先"
	type physicalRow struct {
		AssetID         *string
		DeviceSerial    *string
		DeviceModel     *string
		DeviceType      *string
		MACAddress      *string
		IPAddress       *string
		ResponsibleUser *string
		PortName        *string
		InfoPointName   *string
		// AD 侧只补字段,不覆盖
		ADComputerID *string
		ADDeviceName *string
		ADOperatingSystem *string
	}

	var rows []physicalRow

	rawSQL := `
WITH workstation_ports AS (
    SELECT DISTINCT port.id AS port_id,
           port.interface_name,
           port.device_id,
           ip.name AS info_point_name,
           REGEXP_REPLACE(
               REGEXP_REPLACE(LOWER(port.interface_name), '\s+', '', 'g'),
               '^(gigabitethernet|gigabitether|ge|gi)', 'ge'
           ) AS norm_iface
      FROM ops_info_points ip
      JOIN sys_device_port_status port
        ON port.id::text = ip.port_id
       -- Phase 3 (2026-06-30 数据治理): 加严格 device_id 防御性 JOIN
       -- migration_182 已对齐 1247 行 port_status.device_id ← ops_info_points.device_id
       -- migration_183 已加 FK 约束,此 JOIN 安全(同步路径: 设备名/IP + 端口名)
       -- port_id (UUID PK) 锚定单端口,device_id 一致性保证 MAC 落表位置正确
       AND port.device_id::text = ip.device_id::text
     WHERE ip.workstation_id = ?
       AND ip.deleted_at IS NULL
       AND ip.status = 0
),
latest_mac AS (
    SELECT DISTINCT ON (m.mac_address, m.device_id, m.interface_name)
        m.mac_address,
        m.device_id,
        m.interface_name,
        REGEXP_REPLACE(
            REGEXP_REPLACE(LOWER(m.interface_name), '\s+', '', 'g'),
            '^(gigabitethernet|gigabitether|ge|gi)', 'ge'
        ) AS norm_iface,
        -- MAC 归一化:小写 + 去 . : - 三种分隔符,统一 12 位 hex
        -- 解决 sys_device_mac_address('b022.7a2e.4a4f') 与 ops_asset.mac1('B0:22:7A:2E:4A:4F') 格式不一致
        LOWER(REGEXP_REPLACE(COALESCE(m.mac_address, ''), '[.:\-]', '', 'g')) AS norm_mac
      FROM sys_device_mac_address m
     ORDER BY m.mac_address, m.device_id, m.interface_name, m.collected_at DESC NULLS LAST
),
ad_devices AS (
    SELECT DISTINCT ON (LOWER(REGEXP_REPLACE(COALESCE(ad.mac_address, ''), '[.:\-]', '', 'g')))
        ad.id,
        ad.mac_address,
        ad.serial_number,
        ad.computer_name,
        ad.ip_address,
        ad.operating_system,
        LOWER(REGEXP_REPLACE(COALESCE(ad.mac_address, ''), '[.:\-]', '', 'g')) AS norm_mac
      FROM sys_ad_computer ad
     WHERE ad.deleted_at IS NULL
     ORDER BY LOWER(REGEXP_REPLACE(COALESCE(ad.mac_address, ''), '[.:\-]', '', 'g')),
              ad.updated_at DESC NULLS LAST
)
SELECT DISTINCT ON (COALESCE(a.devicesn,''))
       a.id                                                                AS asset_id,
       a.devicesn                                                          AS device_serial,
       a.device_model_name                                                 AS device_model,
       a.device_type_name                                                  AS device_type,
       -- 优先用 ops_asset 的 mac1(原格式展示给前端),若未命中再用 mac_address 表的
       COALESCE(a.mac1, mac.mac_address)                                   AS mac_address,
       a.machine_ip                                                        AS ip_address,
       a.nowuser_name                                                      AS responsible_user,
       wp.interface_name                                                   AS port_name,
       wp.info_point_name                                                  AS info_point_name,
       ad_devices.id                                                       AS ad_computer_id,
       ad_devices.computer_name                                            AS ad_device_name,
       ad_devices.operating_system                                         AS ad_operating_system
  FROM workstation_ports wp
  JOIN latest_mac mac
       ON mac.norm_iface        = wp.norm_iface
      AND mac.device_id::text    = wp.device_id::text
  LEFT JOIN ops_asset a
       ON a.deleted_at IS NULL
      AND (LOWER(REGEXP_REPLACE(COALESCE(a.mac1, ''), '[.:\-]', '', 'g')) = mac.norm_mac
        OR LOWER(REGEXP_REPLACE(COALESCE(a.mac2, ''), '[.:\-]', '', 'g')) = mac.norm_mac)
  LEFT JOIN ad_devices
       ON ad_devices.norm_mac = mac.norm_mac
       OR (ad_devices.serial_number IS NOT NULL AND ad_devices.serial_number = a.devicesn)
 ORDER BY COALESCE(a.devicesn,''), wp.interface_name NULLS LAST;
`

	if err := s.db.WithContext(ctx).Raw(rawSQL, workstationID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询物理链路设备失败: %w", err)
	}

	logger.Infof("[GetPhysicalDevices] MAC→port→infoPoint→workstation 反查命中 %d 行", len(rows))

	// 3. 转换为 WorkstationDevice(不写入数据库)
	devices := make([]*models.WorkstationDevice, 0, len(rows))
	for i, row := range rows {
		tempID := fmt.Sprintf("physical-%d", i)

		// 字段冲突策略：资产优先
		var deviceName *string
		if row.ADDeviceName != nil && *row.ADDeviceName != "" {
			deviceName = row.ADDeviceName
		} else if row.DeviceModel != nil && *row.DeviceModel != "" {
			// 资产侧没有 device_name 字段,用 model 兜底
			dm := *row.DeviceModel
			deviceName = &dm
		}

		var os *string
		if row.ADOperatingSystem != nil {
			os = row.ADOperatingSystem
		}
		_ = os // 暂不放入 WorkstationDevice(没有 OS 字段);保留备查

		var portDesc *string
		if row.PortName != nil {
			pd := fmt.Sprintf("端口 %s", *row.PortName)
			portDesc = &pd
			if row.InfoPointName != nil && *row.InfoPointName != "" {
				pdv := fmt.Sprintf("端口 %s (信息点 %s)", *row.PortName, *row.InfoPointName)
				portDesc = &pdv
			}
		}

		devices = append(devices, &models.WorkstationDevice{
			BaseModel:         models.BaseModel{ID: tempID},
			WorkstationID:     workstationID,
			DeviceSource:      models.DeviceSourcePhysical,
			AssetID:           row.AssetID,
			ADComputerID:      row.ADComputerID,
			DeviceSerial:      row.DeviceSerial,
			DeviceName:        deviceName,
			DeviceModel:       row.DeviceModel,
			DeviceType:        row.DeviceType,
			MACAddress:        row.MACAddress,
			IPAddress:         row.IPAddress,
			ResponsibleUser:   row.ResponsibleUser,
			ResponsibleUserID: &userID,
			Status:            0,
			IsPrimary:         false,
			Priority:          0,
			Description:       portDesc,
		})
	}

	return devices, nil
}

// GetADDevicesByUser 获取用户的域控设备（通过用户DN关联）
func (s *workstationDeviceService) GetADDevicesByUser(ctx context.Context, userID string) ([]*ADDeviceMatch, error) {
	if userID == "" {
		return nil, apperrors.ParamMissing("用户ID")
	}

	logger.Infof("[GetADDevicesByUser] 开始查询AD设备 - UserID: %s", userID)

	// 获取系统用户的username
	var sysUser models.User
	err := s.db.WithContext(ctx).
		Select("username").
		Where("id = ? AND deleted_at IS NULL", userID).
		First(&sysUser).Error

	if err != nil {
		logger.Warnf("[GetADDevicesByUser] 系统用户不存在: %v", err)
		return []*ADDeviceMatch{}, nil
	}

	logger.Infof("[GetADDevicesByUser] 系统用户名: %s", sysUser.Username)

	// 通过username查询AD用户获取DN
	var adUser models.ADUser
	err = s.db.WithContext(ctx).
		Where("username = ? AND deleted_at IS NULL", sysUser.Username).
		First(&adUser).Error

	if err != nil {
		logger.Warnf("[GetADDevicesByUser] 用户无AD记录: %v", err)
		return []*ADDeviceMatch{}, nil
	}

	logger.Infof("[GetADDevicesByUser] 用户AD DN: %s", adUser.UserDN)

	// 查询该用户管理的AD设备（通过 ManagedBy 字段）
	var adComputers []models.ADComputer
	err = s.db.WithContext(ctx).
		Where("managed_by = ? AND deleted_at IS NULL", adUser.UserDN).
		Find(&adComputers).Error

	if err != nil {
		return nil, fmt.Errorf("查询AD设备失败: %w", err)
	}

	logger.Infof("[GetADDevicesByUser] 找到 %d 个AD设备", len(adComputers))

	matches := make([]*ADDeviceMatch, 0, len(adComputers))
	for _, computer := range adComputers {
		matches = append(matches, &ADDeviceMatch{
			ADComputerID:    computer.ID,
			DeviceSerial:    computer.SerialNumber,
			DeviceName:      computer.ComputerName,
			MACAddress:      computer.MacAddress,
			IPAddress:       computer.IPAddress,
			OperatingSystem: computer.OperatingSystem,
		})
		logger.Infof("[GetADDevicesByUser] 设备详情 - Name: %s, SN: %s, MAC: %s, IP: %s",
			computer.ComputerName, computer.SerialNumber, computer.MacAddress, computer.IPAddress)
	}

	return matches, nil
}

// GetAssetDevicesByUser 获取用户的资产设备（使用姓名匹配，同名时加部门名称匹配）
func (s *workstationDeviceService) GetAssetDevicesByUser(ctx context.Context, userID, username, nickname string) ([]*AssetDeviceMatch, error) {
	if nickname == "" {
		logger.Warnf("[GetAssetDevicesByUser] 缺少用户姓名，无法匹配资产设备")
		return []*AssetDeviceMatch{}, nil
	}

	logger.Infof("[GetAssetDevicesByUser] 开始查询资产设备 - UserID: %s, Username: %s, Nickname: %s",
		userID, username, nickname)

	// 获取用户的部门信息（用于同名匹配）
	var user struct {
		ID       string
		Nickname string
		DeptID   *string
		DeptName *string
	}

	err := s.db.WithContext(ctx).
		Table("sys_user").
		Select("sys_user.id, sys_user.nickname, sys_user.dept_id, dept.dept_name").
		Joins("LEFT JOIN sys_dept dept ON sys_user.dept_id = dept.id").
		Where("sys_user.id = ? AND sys_user.deleted_at IS NULL", userID).
		First(&user).Error

	if err != nil {
		return nil, fmt.Errorf("查询用户部门信息失败: %w", err)
	}

	deptName := ""
	if user.DeptName != nil {
		deptName = *user.DeptName
	}

	logger.Infof("[GetAssetDevicesByUser] 用户部门信息 - DeptID: %v, DeptName: %s",
		user.DeptID, deptName)

	var assets []*models.Asset

	// 策略1: 使用姓名匹配
	logger.Infof("[GetAssetDevicesByUser] 策略1: 使用 nowuser_name='%s' 查询", nickname)
	err = s.db.WithContext(ctx).
		Where("nowuser_name = ? AND deleted_at IS NULL", nickname).
		Find(&assets).Error

	if err != nil {
		return nil, fmt.Errorf("查询用户资产设备失败: %w", err)
	}
	logger.Infof("[GetAssetDevicesByUser] 策略1结果: 找到 %d 个设备", len(assets))

	// 策略2: 如果有多个同名结果，使用姓名+部门名称精确匹配
	if len(assets) > 1 && deptName != "" {
		logger.Infof("[GetAssetDevicesByUser] 检测到 %d 个同名设备，使用部门名称进行精确匹配", len(assets))
		logger.Infof("[GetAssetDevicesByUser] 策略2: 使用 nowuser_name='%s' AND deptname='%s' 查询", nickname, deptName)

		var filteredAssets []*models.Asset
		err = s.db.WithContext(ctx).
			Where("nowuser_name = ? AND deptname = ? AND deleted_at IS NULL", nickname, deptName).
			Find(&filteredAssets).Error

		if err != nil {
			return nil, fmt.Errorf("精确查询用户资产设备失败: %w", err)
		}

		// 如果精确匹配找到结果，使用精确匹配结果
		if len(filteredAssets) > 0 {
			assets = filteredAssets
			logger.Infof("[GetAssetDevicesByUser] 策略2结果: 精确匹配找到 %d 个设备", len(assets))
		} else {
			logger.Warnf("[GetAssetDevicesByUser] 策略2结果: 部门精确匹配无结果，使用策略1的所有结果")
		}
	}

	logger.Infof("[GetAssetDevicesByUser] 最终结果: 找到 %d 个设备", len(assets))

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
		logger.Infof("[GetAssetDevicesByUser] 设备详情 - SN: %s, Model: %v, Type: %v, MAC: %v, NowUserName: %v, DeptName: %v",
			asset.DeviceSN,
			asset.DeviceModelName,
			asset.DeviceTypeName,
			asset.MAC1,
			asset.NowUserName,
			asset.DeptName)
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
		deviceModel = asset.DeviceModelName
		deviceType = asset.DeviceTypeName
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
		IPAddress:        req.IPAddress,
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

	// 如果工位没有绑定用户ID,无法同步
	var userID string
	if workstation.UserID != nil && *workstation.UserID != "" {
		userID = *workstation.UserID
	} else {
		return fmt.Errorf("工位未绑定用户ID,无法同步域控设备")
	}

	logger.Infof("[SyncFromAD] 工位ID: %s, 用户ID: %s", workstationID, userID)

	// 获取用户的域控设备
	adDevices, err := s.GetADDevicesByUser(ctx, userID)
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

	// 获取工位绑定的用户信息（包括账号和姓名）
	var userInfo struct {
		ID       string
		Username string
		Nickname *string
	}

	if workstation.UserID != nil && *workstation.UserID != "" {
		err := s.db.WithContext(ctx).
			Table("sys_user").
			Select("id, username, nickname").
			Where("id = ? AND deleted_at IS NULL", *workstation.UserID).
			First(&userInfo).Error
		if err != nil {
			return fmt.Errorf("查询用户信息失败: %w", err)
		}
	} else {
		return fmt.Errorf("工位未绑定用户ID,无法同步资产设备")
	}

	// 添加调试日志
	logger.Infof("[SyncFromAsset] 工位ID: %s, 用户ID: %s, 账号: %s, 姓名: %s",
		workstationID, userInfo.ID, userInfo.Username,
		func() string {
			if userInfo.Nickname != nil {
				return *userInfo.Nickname
			}
			return "(空)"
		}())

	// 获取用户的资产设备
	assetDevices, err := s.GetAssetDevicesByUser(ctx, userInfo.ID, userInfo.Username,
		func() string {
			if userInfo.Nickname != nil {
				return *userInfo.Nickname
			}
			return ""
		}())
	if err != nil {
		return fmt.Errorf("获取资产设备失败: %w", err)
	}

	logger.Infof("[SyncFromAsset] 查询到 %d 个资产设备", len(assetDevices))

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

		logger.Infof("[SyncFromAsset] 添加设备: SN=%s, Model=%v, Type=%v",
			assetDevice.DeviceSN, assetDevice.DeviceModel, assetDevice.DeviceType)
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
	updates := make(map[string]any)

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
	if req.IPAddress != nil {
		updates["ip_address"] = *req.IPAddress
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

// SetPrimaryAndSave 设置主设备并保存到数据库（用于AD/资产设备转为手动设备）
//
// 合并策略：以传入的 device_serial 为键，实时拉取 AD 与资产两侧设备列表，
// 按以下优先级填充合并字段。任一来源查询失败时仅记录 warn 并降级继续。
//   - deviceName:    AD.DeviceName > req.DeviceName
//   - deviceModel:   asset.DeviceModelName > req.DeviceModel
//   - deviceType:    asset.DeviceTypeName  > req.DeviceType
//   - macAddress:    AD.MACAddress > asset.MAC1 > req.MACAddress
//   - ipAddress:     AD.IPAddress  > req.IPAddress
//   - responsibleUser: asset.NowUserName > req.ResponsibleUser
//   - assetID:       asset 命中时填 asset.ID
//   - adComputerID:  AD 命中时填 ADComputerID
//
// 事务内：删除工位下 device_source IN ('ad','asset') 的旧记录 -> 取消旧主设备
// -> 写入合并后的 manual 记录（IsPrimary=true）。
func (s *workstationDeviceService) SetPrimaryAndSave(ctx context.Context, deviceID string, req *SetPrimaryAndSaveRequest) error {
	if deviceID == "" {
		return apperrors.ParamMissing("设备ID")
	}

	// 验证请求参数
	if req.WorkstationID == "" {
		return apperrors.ParamMissing("工位ID")
	}
	if req.DeviceSerial == "" {
		return apperrors.ParamMissing("设备序列号")
	}

	// 验证工位存在
	var workstation models.Workstation
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", req.WorkstationID).
		First(&workstation).Error
	if err != nil {
		return fmt.Errorf("工位不存在: %w", err)
	}

	// 实时拉取 AD/资产两侧设备列表，按 device_serial 做合并。
	// 拉取失败仅记录 warn 并降级继续，不阻塞保存流程。
	adDevices, adErr := s.GetADDevices(ctx, req.WorkstationID)
	if adErr != nil {
		logger.Warnf("[SetPrimaryAndSave] 拉取AD设备失败,降级继续 - WorkstationID: %s, Error: %v",
			req.WorkstationID, adErr)
		adDevices = nil
	}

	assetDevices, assetErr := s.GetAssetDevices(ctx, req.WorkstationID)
	if assetErr != nil {
		logger.Warnf("[SetPrimaryAndSave] 拉取资产设备失败,降级继续 - WorkstationID: %s, Error: %v",
			req.WorkstationID, assetErr)
		assetDevices = nil
	}

	// 构造以 DeviceSerial 为键的两侧 map（已存在的实时设备不持久化，仅用于查找）。
	adBySN := make(map[string]*models.WorkstationDevice, len(adDevices))
	for _, d := range adDevices {
		if d == nil || d.DeviceSerial == nil {
			continue
		}
		adBySN[*d.DeviceSerial] = d
	}
	assetBySN := make(map[string]*models.WorkstationDevice, len(assetDevices))
	for _, d := range assetDevices {
		if d == nil || d.DeviceSerial == nil {
			continue
		}
		assetBySN[*d.DeviceSerial] = d
	}

	// 命中两侧（仅做日志，不影响合并结果）
	_, adHit := adBySN[req.DeviceSerial]
	_, assetHit := assetBySN[req.DeviceSerial]
	logger.Infof("[SetPrimaryAndSave] 合并 - WorkstationID: %s, SN: %s, AD命中: %v, Asset命中: %v",
		req.WorkstationID, req.DeviceSerial, adHit, assetHit)

	// 合并字段：按"AD优先name/MAC/IP, Asset优先model/type/responsibleUser"策略
	var (
		finalDeviceName    *string
		finalDeviceModel   *string
		finalDeviceType    *string
		finalMACAddress    *string
		finalIPAddress     *string
		finalResponsible   *string
		assetID            *string
		adComputerID       *string
	)

	// deviceName: AD 优先，其次 req
	if ad, ok := adBySN[req.DeviceSerial]; ok && ad != nil && ad.DeviceName != nil && *ad.DeviceName != "" {
		name := *ad.DeviceName
		finalDeviceName = &name
	} else if req.DeviceName != "" {
		name := req.DeviceName
		finalDeviceName = &name
	}

	// deviceModel: asset 优先，其次 req
	if asset, ok := assetBySN[req.DeviceSerial]; ok && asset != nil && asset.DeviceModel != nil && *asset.DeviceModel != "" {
		finalDeviceModel = asset.DeviceModel
	} else {
		finalDeviceModel = req.DeviceModel
	}

	// deviceType: asset 优先，其次 req
	if asset, ok := assetBySN[req.DeviceSerial]; ok && asset != nil && asset.DeviceType != nil && *asset.DeviceType != "" {
		finalDeviceType = asset.DeviceType
	} else {
		finalDeviceType = req.DeviceType
	}

	// macAddress: AD > asset > req
	if ad, ok := adBySN[req.DeviceSerial]; ok && ad != nil && ad.MACAddress != nil && *ad.MACAddress != "" {
		finalMACAddress = ad.MACAddress
	} else if asset, ok := assetBySN[req.DeviceSerial]; ok && asset != nil && asset.MACAddress != nil && *asset.MACAddress != "" {
		finalMACAddress = asset.MACAddress
	} else {
		finalMACAddress = req.MACAddress
	}

	// ipAddress: AD > req
	if ad, ok := adBySN[req.DeviceSerial]; ok && ad != nil && ad.IPAddress != nil && *ad.IPAddress != "" {
		finalIPAddress = ad.IPAddress
	} else {
		finalIPAddress = req.IPAddress
	}

	// responsibleUser: asset 优先，其次 req
	if asset, ok := assetBySN[req.DeviceSerial]; ok && asset != nil && asset.ResponsibleUser != nil && *asset.ResponsibleUser != "" {
		finalResponsible = asset.ResponsibleUser
	} else {
		finalResponsible = req.ResponsibleUser
	}

	// assetID / adComputerID: 命中时填充
	if asset, ok := assetBySN[req.DeviceSerial]; ok && asset != nil && asset.AssetID != nil {
		assetID = asset.AssetID
	}
	if ad, ok := adBySN[req.DeviceSerial]; ok && ad != nil && ad.ADComputerID != nil {
		adComputerID = ad.ADComputerID
	}

	// 使用事务确保原子性
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 清理该工位下旧的 AD/资产来源设备（保留 manual 记录）
		if err := tx.
			Where("workstation_id = ? AND device_source IN ? AND deleted_at IS NULL",
				req.WorkstationID, []string{string(models.DeviceSourceAD), string(models.DeviceSourceAsset)}).
			Delete(&models.WorkstationDevice{}).Error; err != nil {
			return fmt.Errorf("清理旧AD/资产设备失败: %w", err)
		}

		// 取消该工位下的所有主设备
		if err := tx.
			Model(&models.WorkstationDevice{}).
			Where("workstation_id = ? AND deleted_at IS NULL", req.WorkstationID).
			Update("is_primary", false).Error; err != nil {
			return fmt.Errorf("取消主设备失败: %w", err)
		}

		// 创建新的手动设备记录（合并字段）
		newDevice := &models.WorkstationDevice{
			WorkstationID:    req.WorkstationID,
			DeviceSource:     models.DeviceSourceManual,
			AssetID:          assetID,
			ADComputerID:     adComputerID,
			DeviceSerial:     &req.DeviceSerial,
			DeviceName:       finalDeviceName,
			DeviceModel:      finalDeviceModel,
			DeviceType:       finalDeviceType,
			MACAddress:       finalMACAddress,
			IPAddress:        finalIPAddress,
			ResponsibleUser:  finalResponsible,
			Status:           0,
			IsPrimary:        true,
			Priority:         0,
		}

		if err := tx.Create(newDevice).Error; err != nil {
			return fmt.Errorf("保存设备失败: %w", err)
		}

		logger.Infof("[SetPrimaryAndSave] 成功保存主设备 - WorkstationID: %s, DeviceSerial: %s, AD命中=%v, Asset命中=%v",
			req.WorkstationID, req.DeviceSerial, adHit, assetHit)

		return nil
	})
}
