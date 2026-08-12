package operations

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
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

	// GetADDevicesByWorkstations 批量查询多工位的 AD 设备 (Phase 35: 导出优化)
	// 返回 map[workstationID][]*WorkstationDevice; 工位无 AD 设备时返回空切片 (不报错)
	// 内部使用 ~3 次批量 SQL 而非 N 次单工位 SQL, 工位数 100+ 时性能差异显著
	GetADDevicesByWorkstations(ctx context.Context, workstationIDs []string) (map[string][]*models.WorkstationDevice, error)

	// GetAssetDevicesByWorkstations 批量查询多工位的资产设备 (Phase 35: 导出优化)
	GetAssetDevicesByWorkstations(ctx context.Context, workstationIDs []string) (map[string][]*models.WorkstationDevice, error)

	// GetPhysicalDevicesByWorkstations 批量查询多工位的物理链路设备 (Phase 35: 导出优化)
	GetPhysicalDevicesByWorkstations(ctx context.Context, workstationIDs []string) (map[string][]*models.WorkstationDevice, error)

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
	//
	// deviceID 入参仅作兼容（前端 set-primary-and-save 路由仍传 ad-{idx}/asset-{idx}），
	// 实际查询键以 req.DeviceSerial 为准。
	SetPrimaryAndSave(ctx context.Context, deviceID string, req *SetPrimaryAndSaveRequest) error

	// SetPrimaryAndSaveBySerial 通过 (workstationID, deviceSerial) 直接以序列号为键
	// 设置主设备并保存。无需前端预先查到 AD/资产设备的临时 id（ad-{idx}/asset-{idx}）。
	// 适用场景：Excel 导入主设备序列号列、批量脚本同步等只有 SN 信息的入口。
	// 合并策略与 SetPrimaryAndSave 一致（共用内部 mergeBySerial helper），
	// req 可为 nil（req 内 DeviceName/Model/... 字段全空时全部依赖 AD/Asset 实时数据）。
	SetPrimaryAndSaveBySerial(ctx context.Context, workstationID string, serial string, req *SetPrimaryAndSaveRequest) error

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
// Phase 45 R5: 按 MAC→port→infoPoint→workstation 反推链路,得到该工位物理接入的设备。
//
// 链路(同 reconciliation_physical_chain 物化视图):
//   asset.mac1/mac2 → sys_device_mac_address → sys_device_port_status
//     → ops_info_points.workstation_id → sys_workstation.id
//
// 注意: 物理链路是客观事实, 与工位是否绑定 user_id 无关。
// (2026-07-21 工位 3f130 排查:B-3f130-2026-07-21 修正了 user_id 早退耦合)
//
// 字段合并策略(用户要求"以资产为准"):
//   - DeviceName / DeviceModel / DeviceType / ResponsibleUser / IP: 优先取 ops_asset
//   - MAC: 优先 ops_asset.mac1,缺失时回退 ad_computer.mac_address
//   - AD 字段(ComputerName / OperatingSystem): 用 ad_computer 补全,但不覆盖资产字段
//
// 退化场景(纯物理事实链,与 user_id 无关):
//   - 工位不存在 → 返回错误
//   - info_point 未绑定到工位 / 软删除 / status ≠ 0 → 无设备返回(空数组)
//   - MAC 未采集 → 无设备返回(空数组)
func (s *workstationDeviceService) GetPhysicalDevices(ctx context.Context, workstationID string) ([]*models.WorkstationDevice, error) {
	if workstationID == "" {
		return nil, apperrors.ParamMissing("工位ID")
	}

	if !s.uuidValidator.MatchString(workstationID) {
		return nil, apperrors.ParamInvalid("工位ID")
	}

	// 1. 工位存在性校验(物理链路查询不依赖 user_id, 只依赖工位本身)
	var workstation models.Workstation
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", workstationID).
		First(&workstation).Error
	if err != nil {
		return nil, fmt.Errorf("工位不存在: %w", err)
	}

	logger.Infof("[GetPhysicalDevices] 工位 %s 物理链路查询开始", workstationID)

	// 2. 走 MAC→port→infoPoint→workstation→user 反推链
	//    反推逻辑:
	//    (1) ops_info_points.workstation_id = 当前工位 → 拿到 port_id
	//    (2) sys_device_port_status.id = port_id → 拿到 device_id (= 网络设备)
	//    (3) sys_device_mac_address.device_id + interface_name → 拿到 mac_address
	//    (4) ops_asset.mac1 = mac_address (或 mac2) → 拿到 asset 详情
	//    (5) sys_ad_computer.mac_address (或 serial_number) → 合并 AD 字段
	//
	//    使用 DISTINCT ON (devicesn) 去重,合并字段"资产优先"
	//
	// 2026-07-02 扩展: 增加 sys_device_mac_history 作为 fallback,解决设备已离线但端口关联信息仍可用
	//   的场景。新增字段:
	//     - HistoryLastSeen *time.Time: 仅历史MAC命中时的最后上线时间
	//     - Confidence      *float64:  1.0=实测, 0.5=仅历史, 0.0=无MAC
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
		HistoryLastSeen *time.Time
		Confidence      *float64
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
           ip.device_id AS effective_device_id,
           ip.name AS info_point_name,
           REGEXP_REPLACE(
               REGEXP_REPLACE(LOWER(port.interface_name), '\s+', '', 'g'),
               '^(gigabitethernet|gigabitether|ge|gi)', 'ge'
           ) AS norm_iface
      FROM ops_info_points ip
      JOIN sys_device_port_status port
        ON port.id::text = ip.port_id
       -- Phase 3 v3 (2026-07-01): MAC JOIN 锚点改用 ip.device_id (不是 port.device_id)
       -- 根因: 用户实测确认 MAC 表里 GE5/44 的 3 条 MAC 都在 aca124c8 (05F #1) 下,
       -- 但 port.device_id 被 collector 错写成 515f4c58 (04F #2)。
       -- 物理真相: MAC 是 collector 按 info_point 写入的(mac_collection_service.go:279),
       -- 所以 MAC 真实位置 = ip.device_id, 不是 port.device_id。
       -- - port.id (UUID PK) 锚定单端口
       -- - ip.device_id 锚定 MAC (collector 写入依据)
       -- - port.device_id 是历史脏数据, 不影响 query
     WHERE ip.workstation_id = ?
       AND ip.deleted_at IS NULL
       AND ip.status = 0
       AND EXISTS (SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)
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
-- 2026-07-02 新增: 历史 MAC CTE,作为设备离线后的 fallback 来源。
-- sys_device_mac_history 在采集器确认设备下线/迁移/消失时写入,同一 (mac,device,iface) 保留最新一行。
latest_mac_history AS (
    SELECT DISTINCT ON (h.mac_address, h.device_id, h.interface_name)
        h.mac_address,
        h.device_id,
        h.interface_name,
        h.last_seen,
        REGEXP_REPLACE(
            REGEXP_REPLACE(LOWER(h.interface_name), '\s+', '', 'g'),
            '^(gigabitethernet|gigabitether|ge|gi)', 'ge'
        ) AS norm_iface,
        LOWER(REGEXP_REPLACE(COALESCE(h.mac_address, ''), '[.:\-]', '', 'g')) AS norm_mac
      FROM sys_device_mac_history h
     ORDER BY h.mac_address, h.device_id, h.interface_name, h.last_seen DESC NULLS LAST
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
SELECT DISTINCT ON (COALESCE(a.devicesn, COALESCE(mac.mac_address, hist.mac_address, '')))
       a.id                                                                AS asset_id,
       a.devicesn                                                          AS device_serial,
       a.device_model_name                                                 AS device_model,
       a.device_type_name                                                  AS device_type,
       -- 优先 ops_asset.mac1;其次实测 sys_device_mac_address;最后历史 sys_device_mac_history(离线兜底)
       COALESCE(a.mac1, mac.mac_address, hist.mac_address)                 AS mac_address,
       a.machine_ip                                                        AS ip_address,
       a.nowuser_name                                                      AS responsible_user,
       wp.interface_name                                                   AS port_name,
       wp.info_point_name                                                  AS info_point_name,
       hist.last_seen                                                      AS history_last_seen,
       -- 置信度分级:实测=1.0,仅历史=0.5,无MAC=0.0
       CASE
         WHEN mac.mac_address IS NOT NULL THEN 1.0
         WHEN hist.mac_address IS NOT NULL THEN 0.5
         ELSE 0.0
       END                                                                 AS confidence,
       ad_devices.id                                                       AS ad_computer_id,
       ad_devices.computer_name                                            AS ad_device_name,
       ad_devices.operating_system                                         AS ad_operating_system
  FROM workstation_ports wp
  LEFT JOIN latest_mac mac
       ON mac.norm_iface        = wp.norm_iface
      AND mac.device_id::text    = wp.effective_device_id::text
  LEFT JOIN latest_mac_history hist
       ON hist.norm_iface        = wp.norm_iface
      AND hist.device_id::text    = wp.effective_device_id::text
  LEFT JOIN ops_asset a
       ON a.deleted_at IS NULL
      AND (LOWER(REGEXP_REPLACE(COALESCE(a.mac1, ''), '[.:\-]', '', 'g')) = COALESCE(mac.norm_mac, hist.norm_mac)
        OR LOWER(REGEXP_REPLACE(COALESCE(a.mac2, ''), '[.:\-]', '', 'g')) = COALESCE(mac.norm_mac, hist.norm_mac))
  LEFT JOIN ad_devices
       ON ad_devices.norm_mac = COALESCE(mac.norm_mac, hist.norm_mac)
       OR (ad_devices.serial_number IS NOT NULL AND ad_devices.serial_number = a.devicesn)
 ORDER BY COALESCE(a.devicesn, COALESCE(mac.mac_address, hist.mac_address, '')),
          confidence DESC NULLS LAST,
          wp.interface_name NULLS LAST;
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
		// Phase 3 v3 兜底 (2026-07-01): 当 AD + 资产都没有名字时, 用 MAC + 端口名兜底
		// 避免前端因 deviceName 为 nil 过滤掉设备 (5F003 等工位实际有设备但元数据缺失)
		if deviceName == nil && row.MACAddress != nil && *row.MACAddress != "" {
			mn := fmt.Sprintf("设备 (%s)", *row.MACAddress)
			deviceName = &mn
		}
		if deviceName == nil && row.PortName != nil && *row.PortName != "" {
			pn := fmt.Sprintf("端口 %s", *row.PortName)
			deviceName = &pn
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

		// 历史关联降级提示: 当且仅当设备在 sys_device_mac_address(实测) 中未命中,
		// 但 sys_device_mac_history(历史) 命中时,在 description 末尾追加 "历史关联 (最后上线时间: ...)"
		// 置信度由 SQL CASE 计算,本函数原样回填;置信度低于 1.0 的行默认排在实测之后(DESC 排序)
		if row.Confidence != nil && *row.Confidence < 1.0 && row.HistoryLastSeen != nil && !row.HistoryLastSeen.IsZero() {
			hint := fmt.Sprintf("\n历史关联 (最后上线时间: %s)", row.HistoryLastSeen.Format("2006-01-02 15:04:05"))
			if portDesc == nil {
				pd := hint
				portDesc = &pd
			} else {
				pdv := *portDesc + hint
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
			// 2026-07-21 B-3f130: 不再硬编码 &userID; 工位未绑定用户时为 nil。
			// 物理链路与 user_id 解耦, 但有 user_id 时仍继承工位绑定的责任人语义。
			ResponsibleUserID: workstation.UserID,
			Status:            0,
			IsPrimary:         false,
			Priority:          0,
			Description:       portDesc,
			Confidence:        row.Confidence,
			HistoryLastSeen:   row.HistoryLastSeen,
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

	// 查询策略：取并集
	//   1) 管理者(managed_by = UserDN): AD 属性的 ManagedBy 字段
	//   2) 最后登录者(original_description LIKE '%|username|%'): description 字段
	//      按 "|" 切分, parts[1] = 最后登录用户名(见 addomain.parseComputerDescriptionForUser)
	//      该字段在工位实际关联场景比 managed_by 更可靠(参考 memory: workstation-ad-device-managedby-vs-description)
	var adComputers []models.ADComputer
	err = s.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where("managed_by = ? OR original_description LIKE ?",
			adUser.UserDN, "%|"+sysUser.Username+"|%").
		Find(&adComputers).Error

	if err != nil {
		return nil, fmt.Errorf("查询AD设备失败: %w", err)
	}

	logger.Infof("[GetADDevicesByUser] 找到 %d 个AD设备(managed_by ∪ last_logged_user)", len(adComputers))

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
		logger.Infof("[GetADDevicesByUser] 设备详情 - Name: %s, SN: %s, MAC: %s, IP: %s, ManagedBy: %s",
			computer.ComputerName, computer.SerialNumber, computer.MacAddress, computer.IPAddress, computer.ManagedBy)
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
//
// deviceID 入参仅作兼容（前端 set-primary-and-save 路由仍传 ad-{idx}/asset-{idx}），
// 实际查询键以 req.DeviceSerial 为准；deviceID 非空时记录 warn 提醒上游。
func (s *workstationDeviceService) SetPrimaryAndSave(ctx context.Context, deviceID string, req *SetPrimaryAndSaveRequest) error {
	if deviceID != "" {
		// deviceID 在新代码中已被 req.DeviceSerial 取代；保留入参仅作兼容。
		// 触发场景：前端 work-station-device/{id}/set-primary-and-save 路由仍带临时 id。
		logger.Debugf("[SetPrimaryAndSave] deviceID 入参(%s)已被 req.DeviceSerial 取代,忽略", deviceID)
	}

	// 验证请求参数
	if req == nil {
		return apperrors.ParamMissing("请求参数")
	}
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

	merged := s.mergeBySerial(ctx, req.WorkstationID, req.DeviceSerial, req)

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
			WorkstationID:   req.WorkstationID,
			DeviceSource:    models.DeviceSourceManual,
			AssetID:         merged.assetID,
			ADComputerID:    merged.adComputerID,
			DeviceSerial:    &req.DeviceSerial,
			DeviceName:      merged.finalDeviceName,
			DeviceModel:     merged.finalDeviceModel,
			DeviceType:      merged.finalDeviceType,
			MACAddress:      merged.finalMACAddress,
			IPAddress:       merged.finalIPAddress,
			ResponsibleUser: merged.finalResponsible,
			Status:          0,
			IsPrimary:       true,
			Priority:        0,
		}

		if err := tx.Create(newDevice).Error; err != nil {
			return fmt.Errorf("保存设备失败: %w", err)
		}

		logger.Infof("[SetPrimaryAndSave] 成功保存主设备 - WorkstationID: %s, DeviceSerial: %s, AD命中=%v, Asset命中=%v",
			req.WorkstationID, req.DeviceSerial, merged.adHit, merged.assetHit)

		return nil
	})
}

// SetPrimaryAndSaveBySerial 通过 (workstationID, serial) 直接以序列号为键
// 设置主设备并保存。无需前端预先查到 AD/资产设备的临时 id（ad-{idx}/asset-{idx}）。
// 适用场景：Excel 导入主设备序列号列、批量脚本同步等只有 SN 信息的入口。
//
// 合并策略与 SetPrimaryAndSave 一致（共用 mergeBySerial helper）。
// req 可为 nil；为 nil 时所有字段依赖 AD/Asset 实时数据，req.DeviceName/... 全为空。
func (s *workstationDeviceService) SetPrimaryAndSaveBySerial(ctx context.Context, workstationID string, serial string, req *SetPrimaryAndSaveRequest) error {
	if workstationID == "" {
		return apperrors.ParamMissing("工位ID")
	}
	if serial == "" {
		return apperrors.ParamMissing("设备序列号")
	}

	// 验证工位存在
	var workstation models.Workstation
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", workstationID).
		First(&workstation).Error
	if err != nil {
		return fmt.Errorf("工位不存在: %w", err)
	}

	// 归一化 req：nil → 新建；非 nil 但字段空 → 填充
	if req == nil {
		req = &SetPrimaryAndSaveRequest{}
	}
	if req.WorkstationID == "" {
		req.WorkstationID = workstationID
	}
	if req.DeviceSerial == "" {
		req.DeviceSerial = serial
	}

	merged := s.mergeBySerial(ctx, workstationID, serial, req)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 清理该工位下旧的 AD/资产来源设备（保留 manual 记录）
		if err := tx.
			Where("workstation_id = ? AND device_source IN ? AND deleted_at IS NULL",
				workstationID, []string{string(models.DeviceSourceAD), string(models.DeviceSourceAsset)}).
			Delete(&models.WorkstationDevice{}).Error; err != nil {
			return fmt.Errorf("清理旧AD/资产设备失败: %w", err)
		}

		// 取消该工位下的所有主设备
		if err := tx.
			Model(&models.WorkstationDevice{}).
			Where("workstation_id = ? AND deleted_at IS NULL", workstationID).
			Update("is_primary", false).Error; err != nil {
			return fmt.Errorf("取消主设备失败: %w", err)
		}

		// 创建新的手动设备记录（合并字段）
		newDevice := &models.WorkstationDevice{
			WorkstationID:   workstationID,
			DeviceSource:    models.DeviceSourceManual,
			AssetID:         merged.assetID,
			ADComputerID:    merged.adComputerID,
			DeviceSerial:    &serial,
			DeviceName:      merged.finalDeviceName,
			DeviceModel:     merged.finalDeviceModel,
			DeviceType:      merged.finalDeviceType,
			MACAddress:      merged.finalMACAddress,
			IPAddress:       merged.finalIPAddress,
			ResponsibleUser: merged.finalResponsible,
			Status:          0,
			IsPrimary:       true,
			Priority:        0,
		}

		if err := tx.Create(newDevice).Error; err != nil {
			return fmt.Errorf("保存设备失败: %w", err)
		}

		logger.Infof("[SetPrimaryAndSaveBySerial] 成功保存主设备 - WorkstationID: %s, DeviceSerial: %s, AD命中=%v, Asset命中=%v",
			workstationID, serial, merged.adHit, merged.assetHit)

		return nil
	})
}

// adAssetMergeResult mergeBySerial 内部返回结构,集中承载合并后的字段值。
type adAssetMergeResult struct {
	finalDeviceName  *string
	finalDeviceModel *string
	finalDeviceType  *string
	finalMACAddress  *string
	finalIPAddress   *string
	finalResponsible *string
	assetID          *string
	adComputerID     *string
	adHit            bool
	assetHit         bool
}

// mergeBySerial 以 device_serial 为键,实时拉取 AD 与资产两侧设备列表,
// 按"AD优先 name/MAC/IP, Asset优先 model/type/responsibleUser"策略合并字段。
//
// 任一来源查询失败时仅记录 warn 并降级继续,不阻塞主流程。
// 命中标志(adHit/assetHit)仅用于日志,不影响合并结果。
//
// 这是 SetPrimaryAndSave / SetPrimaryAndSaveBySerial 共用的唯一合并实现,
// 不复制粘贴。
func (s *workstationDeviceService) mergeBySerial(ctx context.Context, workstationID string, serial string, req *SetPrimaryAndSaveRequest) adAssetMergeResult {
	// 实时拉取 AD/资产两侧设备列表,按 device_serial 做合并。
	// 拉取失败仅记录 warn 并降级继续,不阻塞保存流程。
	adDevices, adErr := s.GetADDevices(ctx, workstationID)
	if adErr != nil {
		logger.Warnf("[mergeBySerial] 拉取AD设备失败,降级继续 - WorkstationID: %s, Error: %v",
			workstationID, adErr)
		adDevices = nil
	}

	assetDevices, assetErr := s.GetAssetDevices(ctx, workstationID)
	if assetErr != nil {
		logger.Warnf("[mergeBySerial] 拉取资产设备失败,降级继续 - WorkstationID: %s, Error: %v",
			workstationID, assetErr)
		assetDevices = nil
	}

	// 构造以 DeviceSerial 为键的两侧 map(已存在的实时设备不持久化,仅用于查找)。
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

	_, adHit := adBySN[serial]
	_, assetHit := assetBySN[serial]
	logger.Infof("[mergeBySerial] 合并 - WorkstationID: %s, SN: %s, AD命中: %v, Asset命中: %v",
		workstationID, serial, adHit, assetHit)

	var r adAssetMergeResult
	r.adHit = adHit
	r.assetHit = assetHit

	// deviceName: AD 优先,其次 req
	if ad, ok := adBySN[serial]; ok && ad != nil && ad.DeviceName != nil && *ad.DeviceName != "" {
		name := *ad.DeviceName
		r.finalDeviceName = &name
	} else if req != nil && req.DeviceName != "" {
		name := req.DeviceName
		r.finalDeviceName = &name
	}

	// deviceModel: asset 优先,其次 req
	if asset, ok := assetBySN[serial]; ok && asset != nil && asset.DeviceModel != nil && *asset.DeviceModel != "" {
		r.finalDeviceModel = asset.DeviceModel
	} else if req != nil {
		r.finalDeviceModel = req.DeviceModel
	}

	// deviceType: asset 优先,其次 req
	if asset, ok := assetBySN[serial]; ok && asset != nil && asset.DeviceType != nil && *asset.DeviceType != "" {
		r.finalDeviceType = asset.DeviceType
	} else if req != nil {
		r.finalDeviceType = req.DeviceType
	}

	// macAddress: AD > asset > req
	if ad, ok := adBySN[serial]; ok && ad != nil && ad.MACAddress != nil && *ad.MACAddress != "" {
		r.finalMACAddress = ad.MACAddress
	} else if asset, ok := assetBySN[serial]; ok && asset != nil && asset.MACAddress != nil && *asset.MACAddress != "" {
		r.finalMACAddress = asset.MACAddress
	} else if req != nil {
		r.finalMACAddress = req.MACAddress
	}

	// ipAddress: AD > req
	if ad, ok := adBySN[serial]; ok && ad != nil && ad.IPAddress != nil && *ad.IPAddress != "" {
		r.finalIPAddress = ad.IPAddress
	} else if req != nil {
		r.finalIPAddress = req.IPAddress
	}

	// responsibleUser: asset 优先,其次 req
	if asset, ok := assetBySN[serial]; ok && asset != nil && asset.ResponsibleUser != nil && *asset.ResponsibleUser != "" {
		r.finalResponsible = asset.ResponsibleUser
	} else if req != nil {
		r.finalResponsible = req.ResponsibleUser
	}

	// assetID / adComputerID: 命中时填充
	if asset, ok := assetBySN[serial]; ok && asset != nil && asset.AssetID != nil {
		r.assetID = asset.AssetID
	}
	if ad, ok := adBySN[serial]; ok && ad != nil && ad.ADComputerID != nil {
		r.adComputerID = ad.ADComputerID
	}

	return r
}

// macIDFragment 从可选 MAC 地址生成稳定的 ID 片段 (避免 *string 在 fmt 中类型不匹配)
func macIDFragment(mac *string) string {
	if mac == nil {
		return "nil"
	}
	return *mac
}

// safePrefix 安全截取前 N 字符, 不足时返回整个字符串 (避免 panic)
func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// =========================================================================
// Phase 35: 批量查询 (供 Excel 导出等 N+1 场景使用)
// =========================================================================
//
// 设计要点:
//   - 替代循环调用 GetADDevices/GetAssetDevices/GetPhysicalDevices, 减少 717 次 → 3 次 SQL
//   - 工位无 user_id 时返回空切片 (不报错), 与单工位方法行为一致
//   - AD LIKE 多 username 拼 OR 由代码层生成, 避免 PG 参数数量上限问题
//   - 返回 map[wsID][]*WorkstationDevice, 调用方按需取

// GetADDevicesByWorkstations 批量查询多工位的 AD 设备
//
// 实现路径 (5 次批量 SQL, 替代 239 单工位 = ~717 次 SQL):
//   1. sys_workstation 批量查 wsID → userID
//   2. sys_user 批量查 userID → username
//   3. sys_ad_user 批量查 username → user_dn
//   4. sys_ad_computer 批量查 (managed_by IN user_dns OR description LIKE 任意 username)
//   5. 代码层映射: computer → user_dn/username → userID → wsID
//
// 实现要点: 所有 Scan 用 map[string]interface{} 而非 struct, 规避 GORM struct 扫描丢失中文字段的问题
func (s *workstationDeviceService) GetADDevicesByWorkstations(
	ctx context.Context,
	workstationIDs []string,
) (map[string][]*models.WorkstationDevice, error) {
	result := make(map[string][]*models.WorkstationDevice, len(workstationIDs))
	for _, id := range workstationIDs {
		result[id] = []*models.WorkstationDevice{}
	}
	if len(workstationIDs) == 0 {
		return result, nil
	}

	// 1. 批量查工位 → wsID → userID (用 map 避免 struct 扫描丢失字段)
	var wsRows []map[string]interface{}
	if err := s.db.WithContext(ctx).
		Table("sys_workstation").
		Select("id, user_id").
		Where("id IN ? AND deleted_at IS NULL", workstationIDs).
		Scan(&wsRows).Error; err != nil {
		return result, fmt.Errorf("批量查询 sys_workstation 失败: %w", err)
	}

	userIDSet := make(map[string]bool)
	wsToUser := make(map[string]string)
	for _, w := range wsRows {
		uid := stringFromMap(w, "user_id")
		wsid := stringFromMap(w, "id")
		if uid == "" || wsid == "" {
			continue
		}
		userIDSet[uid] = true
		wsToUser[wsid] = uid
	}
	if len(userIDSet) == 0 {
		return result, nil
	}
	userIDs := make([]string, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	// 2. 批量查 sys_user → username
	var userRows []map[string]interface{}
	if err := s.db.WithContext(ctx).
		Table("sys_user").
		Select("id, username").
		Where("id IN ? AND deleted_at IS NULL", userIDs).
		Scan(&userRows).Error; err != nil {
		return result, fmt.Errorf("批量查询 sys_user 失败: %w", err)
	}
	userIDToName := make(map[string]string, len(userRows))
	nameToUserID := make(map[string]string, len(userRows))
	for _, u := range userRows {
		uid := stringFromMap(u, "id")
		uname := stringFromMap(u, "username")
		if uid == "" {
			continue
		}
		userIDToName[uid] = uname
		nameToUserID[uname] = uid
	}

	// 3. 批量查 sys_ad_user → user_dn
	usernames := make([]string, 0)
	for _, uname := range userIDToName {
		if uname != "" {
			usernames = append(usernames, uname)
		}
	}
	var adUserRows []map[string]interface{}
	if err := s.db.WithContext(ctx).
		Table("sys_ad_user").
		Select("username, user_dn").
		Where("username IN ? AND deleted_at IS NULL", usernames).
		Scan(&adUserRows).Error; err != nil {
		return result, fmt.Errorf("批量查询 sys_ad_user 失败: %w", err)
	}
	nameToDN := make(map[string]string, len(adUserRows))
	dnToName := make(map[string]string, len(adUserRows))
	for _, au := range adUserRows {
		uname := stringFromMap(au, "username")
		udn := stringFromMap(au, "user_dn")
		if uname == "" {
			continue
		}
		nameToDN[uname] = udn
		dnToName[udn] = uname
	}
	if len(nameToDN) == 0 {
		return result, nil
	}

	// 4. 批量查 sys_ad_computer
	userDNs := make([]string, 0, len(nameToDN))
	for _, dn := range nameToDN {
		userDNs = append(userDNs, dn)
	}
	tx := s.db.WithContext(ctx).
		Table("sys_ad_computer").
		Select("id, serial_number, computer_name, mac_address, ip_address, operating_system, managed_by, original_description").
		Where("deleted_at IS NULL").
		Where("managed_by IN ?", userDNs)
	for _, uname := range usernames {
		tx = tx.Or("original_description LIKE ?", "%|"+uname+"|%")
	}
	var adComputerRows []map[string]interface{}
	if err := tx.Scan(&adComputerRows).Error; err != nil {
		return result, fmt.Errorf("批量查询 sys_ad_computer 失败: %w", err)
	}

	// 5. 代码层映射
	for _, comp := range adComputerRows {
		managedBy := stringFromMap(comp, "managed_by")
		desc := stringFromMap(comp, "original_description")
		var ownerUsername string
		if dn, ok := dnToName[managedBy]; ok {
			ownerUsername = dn
		} else if desc != "" {
			for _, uname := range usernames {
				if strings.Contains(desc, "|"+uname+"|") {
					ownerUsername = uname
					break
				}
			}
		}
		if ownerUsername == "" {
			continue
		}
		ownerUserID := nameToUserID[ownerUsername]
		if ownerUserID == "" {
			continue
		}

		adID := stringFromMap(comp, "id")
		sn := stringFromMap(comp, "serial_number")
		cn := stringFromMap(comp, "computer_name")
		mac := stringFromMap(comp, "mac_address")
		ip := stringFromMap(comp, "ip_address")
		_ = stringFromMap(comp, "operating_system") // OS 字段未放入 WorkstationDevice

		for _, wsID := range workstationIDs {
			if wsToUser[wsID] != ownerUserID {
				continue
			}
			result[wsID] = append(result[wsID], &models.WorkstationDevice{
				BaseModel:     models.BaseModel{ID: fmt.Sprintf("ad-%s-%s", safePrefix(wsID, 8), safePrefix(adID, 8))},
				WorkstationID: wsID,
				DeviceSource:  models.DeviceSourceAD,
				ADComputerID:  &adID,
				DeviceSerial:  &sn,
				DeviceName:    &cn,
				MACAddress:    &mac,
				IPAddress:     &ip,
				Status:        0,
				IsPrimary:     false,
				Priority:      0,
			})
		}
	}

	return result, nil
}

func (s *workstationDeviceService) GetAssetDevicesByWorkstations(
	ctx context.Context,
	workstationIDs []string,
) (map[string][]*models.WorkstationDevice, error) {
	result := make(map[string][]*models.WorkstationDevice, len(workstationIDs))
	for _, id := range workstationIDs {
		result[id] = []*models.WorkstationDevice{}
	}
	if len(workstationIDs) == 0 {
		return result, nil
	}

	// 1. 批量查工位 → wsID → userID
	type wsRow struct {
		ID     string
		UserID *string
	}
	var wsRows []wsRow
	if err := s.db.WithContext(ctx).
		Table("sys_workstation").
		Select("id, user_id").
		Where("id IN ? AND deleted_at IS NULL", workstationIDs).
		Scan(&wsRows).Error; err != nil {
		return result, fmt.Errorf("批量查询 sys_workstation 失败: %w", err)
	}

	userIDSet := make(map[string]bool)
	wsToUser := make(map[string]string)
	for _, w := range wsRows {
		if w.UserID != nil && *w.UserID != "" {
			userIDSet[*w.UserID] = true
			wsToUser[w.ID] = *w.UserID
		}
	}
	if len(userIDSet) == 0 {
		return result, nil
	}
	userIDs := make([]string, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	// 2. 批量查 sys_user → (id, nickname, dept_id) (用 map 避免 GORM struct 扫描丢失字段)
	var userRows []map[string]interface{}
	if err := s.db.WithContext(ctx).
		Table("sys_user").
		Select("id, nickname, dept_id").
		Where("id IN ? AND deleted_at IS NULL", userIDs).
		Scan(&userRows).Error; err != nil {
		return result, fmt.Errorf("批量查询 sys_user 失败: %w", err)
	}

	// userID → nickname, nickname → userID
	userIDToNick := make(map[string]string, len(userRows))
	nickToUserID := make(map[string]string, len(userRows))
	nicknames := make([]string, 0, len(userRows))
	for _, u := range userRows {
		uid := stringFromMap(u, "id")
		nick := stringFromMap(u, "nickname")
		if uid == "" || nick == "" {
			continue
		}
		userIDToNick[uid] = nick
		nickToUserID[nick] = uid
		nicknames = append(nicknames, nick)
	}
	if len(nicknames) == 0 {
		return result, nil
	}

	// 3. 批量查 ops_asset → nowuser_name IN nicknames AND deleted_at IS NULL
	var assetRows []map[string]interface{}
	if err := s.db.WithContext(ctx).
		Table("ops_asset").
		Select("id, devicesn, device_model_name, device_type_name, mac1, nowuser_name").
		Where("nowuser_name IN ? AND deleted_at IS NULL", nicknames).
		Scan(&assetRows).Error; err != nil {
		return result, fmt.Errorf("批量查询 ops_asset 失败: %w", err)
	}

	// 4. 代码层映射: asset → nickname → userID → wsID
	for _, a := range assetRows {
		nowUser := stringFromMap(a, "nowuser_name")
		if nowUser == "" {
			continue
		}
		ownerUserID := nickToUserID[nowUser]
		if ownerUserID == "" {
			continue
		}
		for _, wsID := range workstationIDs {
			if wsToUser[wsID] != ownerUserID {
				continue
			}
			id := stringFromMap(a, "id")
			sn := stringFromMap(a, "devicesn")
			modelName := stringFromMap(a, "device_model_name")
			typeName := stringFromMap(a, "device_type_name")
			mac := stringFromMap(a, "mac1")
			result[wsID] = append(result[wsID], &models.WorkstationDevice{
				BaseModel:     models.BaseModel{ID: fmt.Sprintf("asset-%s-%s", safePrefix(wsID, 8), safePrefix(id, 8))},
				WorkstationID: wsID,
				DeviceSource:  models.DeviceSourceAsset,
				AssetID:       &id,
				DeviceSerial:  &sn,
				DeviceModel:   strPtr(modelName),
				DeviceType:    strPtr(typeName),
				MACAddress:    strPtr(mac),
				ResponsibleUser: strPtr(nowUser),
				Status:    0,
				IsPrimary: false,
				Priority:  0,
			})
		}
	}

	return result, nil
}

// stringFromMap 安全地从 map[string]interface{} 取字符串值 (nil/非字符串返回 "")
func stringFromMap(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// strPtr 安全地把字符串转为 *string (空字符串返回 nil)
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// GetPhysicalDevicesByWorkstations 批量查询多工位的物理链路设备
//
// 实现路径:
//   - 在原 GetPhysicalDevices 的 SQL 中, 把 workstation_ports CTE 的
//     `WHERE ip.workstation_id = ?` 改为 `WHERE ip.workstation_id IN (?)`,
//     并在主 SELECT 中加入 `wp.workstation_id` 用于结果按工位分组
//   - 1 次 SQL, 替代 239 次单工位查询
//   - (2026-07-22 B-3f130-fix) 误用 ANY(?) 触发 PG syntax error (ANY(?) 期望数组),
//     改为 IN (?) 兼容 GORM 自动展开 []string → ?,?,?
//
// 与单工位方法 GetPhysicalDevices 行为差异:
//   - 单工位返回 1 个 wsID 的设备, 本方法返回多 wsID 的 map[wsID][]devices
//   - 单工位需要先 SELECT sys_workstation 取 userID (因 ResponsibleUserID 字段),
//     批量版本为节省 SQL 改为按需查 (如有 wsID 仍需 userID, 走 batch enrichment)
func (s *workstationDeviceService) GetPhysicalDevicesByWorkstations(
	ctx context.Context,
	workstationIDs []string,
) (map[string][]*models.WorkstationDevice, error) {
	result := make(map[string][]*models.WorkstationDevice, len(workstationIDs))
	for _, id := range workstationIDs {
		result[id] = []*models.WorkstationDevice{}
	}
	if len(workstationIDs) == 0 {
		return result, nil
	}

	// 批量查 sys_workstation 取 user_id (用于 ResponsibleUserID 字段, 与单工位方法语义一致)
	type wsRow struct {
		ID     string
		UserID *string
	}
	var wsRows []wsRow
	if err := s.db.WithContext(ctx).
		Table("sys_workstation").
		Select("id, user_id").
		Where("id IN ? AND deleted_at IS NULL", workstationIDs).
		Scan(&wsRows).Error; err != nil {
		return result, fmt.Errorf("批量查询 sys_workstation 失败: %w", err)
	}
	wsUserMap := make(map[string]*string, len(wsRows))
	for _, w := range wsRows {
		wsUserMap[w.ID] = w.UserID
	}

	// 修改后的 SQL: workstation_id IN (?), 并加入 wp.workstation_id
	// (注意: ANY(?) 在 PG 中需要数组形式, GORM 展开 []string 为 ?,?,? 会报 syntax error;
	//        改用 IN (?) 兼容性更好)
	rawSQL := `
WITH workstation_ports AS (
    SELECT DISTINCT port.id AS port_id,
           port.interface_name,
           ip.device_id AS effective_device_id,
           ip.name AS info_point_name,
           ip.workstation_id,
           REGEXP_REPLACE(
               REGEXP_REPLACE(LOWER(port.interface_name), '\s+', '', 'g'),
               '^(gigabitethernet|gigabitether|ge|gi)', 'ge'
           ) AS norm_iface
      FROM ops_info_points ip
      JOIN sys_device_port_status port
        ON port.id::text = ip.port_id
     WHERE ip.workstation_id IN (?)
       AND ip.deleted_at IS NULL
       AND ip.status = 0
       AND EXISTS (SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)
),
latest_mac AS (
    SELECT DISTINCT ON (m.mac_address, m.device_id, m.interface_name)
        m.mac_address, m.device_id, m.interface_name,
        REGEXP_REPLACE(
            REGEXP_REPLACE(LOWER(m.interface_name), '\s+', '', 'g'),
            '^(gigabitethernet|gigabitether|ge|gi)', 'ge'
        ) AS norm_iface,
        LOWER(REGEXP_REPLACE(COALESCE(m.mac_address, ''), '[.:\-]', '', 'g')) AS norm_mac
      FROM sys_device_mac_address m
     ORDER BY m.mac_address, m.device_id, m.interface_name, m.collected_at DESC NULLS LAST
),
latest_mac_history AS (
    SELECT DISTINCT ON (h.mac_address, h.device_id, h.interface_name)
        h.mac_address, h.device_id, h.interface_name, h.last_seen,
        REGEXP_REPLACE(
            REGEXP_REPLACE(LOWER(h.interface_name), '\s+', '', 'g'),
            '^(gigabitethernet|gigabitether|ge|gi)', 'ge'
        ) AS norm_iface,
        LOWER(REGEXP_REPLACE(COALESCE(h.mac_address, ''), '[.:\-]', '', 'g')) AS norm_mac
      FROM sys_device_mac_history h
     ORDER BY h.mac_address, h.device_id, h.interface_name, h.last_seen DESC NULLS LAST
),
ad_devices AS (
    SELECT DISTINCT ON (LOWER(REGEXP_REPLACE(COALESCE(ad.mac_address, ''), '[.:\-]', '', 'g')))
        ad.id, ad.mac_address, ad.serial_number, ad.computer_name, ad.ip_address, ad.operating_system,
        LOWER(REGEXP_REPLACE(COALESCE(ad.mac_address, ''), '[.:\-]', '', 'g')) AS norm_mac
      FROM sys_ad_computer ad
     WHERE ad.deleted_at IS NULL
     ORDER BY LOWER(REGEXP_REPLACE(COALESCE(ad.mac_address, ''), '[.:\-]', '', 'g')),
              ad.updated_at DESC NULLS LAST
)
SELECT wp.workstation_id                                                   AS ws_id,
       a.id                                                                AS asset_id,
       a.devicesn                                                          AS device_serial,
       a.device_model_name                                                 AS device_model,
       a.device_type_name                                                  AS device_type,
       COALESCE(a.mac1, mac.mac_address, hist.mac_address)                 AS mac_address,
       a.machine_ip                                                        AS ip_address,
       a.nowuser_name                                                      AS responsible_user,
       wp.interface_name                                                   AS port_name,
       wp.info_point_name                                                  AS info_point_name,
       hist.last_seen                                                      AS history_last_seen,
       CASE WHEN mac.mac_address IS NOT NULL THEN 1.0
            WHEN hist.mac_address IS NOT NULL THEN 0.5
            ELSE 0.0 END                                                    AS confidence,
       ad_devices.id                                                       AS ad_computer_id,
       ad_devices.computer_name                                            AS ad_device_name,
       ad_devices.operating_system                                         AS ad_operating_system
  FROM workstation_ports wp
  LEFT JOIN latest_mac mac
       ON mac.norm_iface = wp.norm_iface AND mac.device_id::text = wp.effective_device_id::text
  LEFT JOIN latest_mac_history hist
       ON hist.norm_iface = wp.norm_iface AND hist.device_id::text = wp.effective_device_id::text
  LEFT JOIN ops_asset a
       ON a.deleted_at IS NULL
      AND (LOWER(REGEXP_REPLACE(COALESCE(a.mac1, ''), '[.:\-]', '', 'g')) = COALESCE(mac.norm_mac, hist.norm_mac)
        OR LOWER(REGEXP_REPLACE(COALESCE(a.mac2, ''), '[.:\-]', '', 'g')) = COALESCE(mac.norm_mac, hist.norm_mac))
  LEFT JOIN ad_devices
       ON ad_devices.norm_mac = COALESCE(mac.norm_mac, hist.norm_mac)
       OR (ad_devices.serial_number IS NOT NULL AND ad_devices.serial_number = a.devicesn)
 ORDER BY wp.workstation_id,
          COALESCE(a.devicesn, COALESCE(mac.mac_address, hist.mac_address, '')),
          confidence DESC NULLS LAST,
          wp.interface_name NULLS LAST;
`

	type physicalRow struct {
		WSID              string  `gorm:"column:ws_id"`
		AssetID           *string `gorm:"column:asset_id"`
		DeviceSerial      *string `gorm:"column:device_serial"`
		DeviceModel       *string `gorm:"column:device_model"`
		DeviceType        *string `gorm:"column:device_type"`
		MACAddress        *string `gorm:"column:mac_address"`
		IPAddress         *string `gorm:"column:ip_address"`
		ResponsibleUser   *string `gorm:"column:responsible_user"`
		PortName          *string `gorm:"column:port_name"`
		InfoPointName     *string `gorm:"column:info_point_name"`
		HistoryLastSeen   *time.Time `gorm:"column:history_last_seen"`
		Confidence        *float64 `gorm:"column:confidence"`
		ADComputerID      *string `gorm:"column:ad_computer_id"`
		ADDeviceName      *string `gorm:"column:ad_device_name"`
		ADOperatingSystem *string `gorm:"column:ad_operating_system"`
	}
	var rows []physicalRow
	if err := s.db.WithContext(ctx).Raw(rawSQL, workstationIDs).Scan(&rows).Error; err != nil {
		return result, fmt.Errorf("批量查询物理链路设备失败: %w", err)
	}

	// 按 wsID 分组并构造 WorkstationDevice
	for _, row := range rows {
		wsID := row.WSID

		var deviceName *string
		if row.ADDeviceName != nil && *row.ADDeviceName != "" {
			deviceName = row.ADDeviceName
		} else if row.DeviceModel != nil && *row.DeviceModel != "" {
			dm := *row.DeviceModel
			deviceName = &dm
		}
		if deviceName == nil && row.MACAddress != nil && *row.MACAddress != "" {
			mn := fmt.Sprintf("设备 (%s)", *row.MACAddress)
			deviceName = &mn
		}
		if deviceName == nil && row.PortName != nil && *row.PortName != "" {
			pn := fmt.Sprintf("端口 %s", *row.PortName)
			deviceName = &pn
		}

		var portDesc *string
		if row.PortName != nil {
			pd := fmt.Sprintf("端口 %s", *row.PortName)
			portDesc = &pd
			if row.InfoPointName != nil && *row.InfoPointName != "" {
				pdv := fmt.Sprintf("端口 %s (信息点 %s)", *row.PortName, *row.InfoPointName)
				portDesc = &pdv
			}
		}
		if row.Confidence != nil && *row.Confidence < 1.0 && row.HistoryLastSeen != nil && !row.HistoryLastSeen.IsZero() {
			hint := fmt.Sprintf("\n历史关联 (最后上线时间: %s)", row.HistoryLastSeen.Format("2006-01-02 15:04:05"))
			if portDesc == nil {
				pd := hint
				portDesc = &pd
			} else {
				pdv := *portDesc + hint
				portDesc = &pdv
			}
		}

		result[wsID] = append(result[wsID], &models.WorkstationDevice{
			BaseModel:        models.BaseModel{ID: fmt.Sprintf("physical-%s-%s", safePrefix(wsID, 8), macIDFragment(row.MACAddress))},
			WorkstationID:    wsID,
			DeviceSource:     models.DeviceSourcePhysical,
			AssetID:          row.AssetID,
			ADComputerID:     row.ADComputerID,
			DeviceSerial:     row.DeviceSerial,
			DeviceName:       deviceName,
			DeviceModel:      row.DeviceModel,
			DeviceType:       row.DeviceType,
			MACAddress:       row.MACAddress,
			IPAddress:        row.IPAddress,
			ResponsibleUser:  row.ResponsibleUser,
			ResponsibleUserID: wsUserMap[wsID],
			Status:            0,
			IsPrimary:         false,
			Priority:          0,
			Description:       portDesc,
			Confidence:        row.Confidence,
			HistoryLastSeen:   row.HistoryLastSeen,
		})
	}

	logger.Infof("[GetPhysicalDevicesByWorkstations] 批量查询命中 %d 行, 覆盖 %d 工位",
		len(rows), len(workstationIDs))
	return result, nil
}
