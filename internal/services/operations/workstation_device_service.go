package operations

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
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

	// GetPhysicalDevicesWithDiagnosis 实时查询物理链路设备 + **空因诊断**（DBG-07 / #7）。
	//
	// 与 GetPhysicalDevices 的关系：数据完全一致（Devices 字段与 GetPhysicalDevices 返回值相同），
	// 额外返回 EmptyReason / EmptyMessage / Diagnosis 三块可诊断信息。
	// GetPhysicalDevices 的签名与语义保持不变（向后兼容，旧调用方/旧前端零改动）。
	GetPhysicalDevicesWithDiagnosis(ctx context.Context, workstationID string) (*PhysicalDevicesResult, error)

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
	WorkstationID   string  `json:"workstationId"`
	DeviceSerial    string  `json:"deviceSerial"`
	DeviceName      string  `json:"deviceName"`
	DeviceModel     *string `json:"deviceModel,omitempty"`
	DeviceType      *string `json:"deviceType,omitempty"`
	MACAddress      *string `json:"macAddress,omitempty"`
	IPAddress       *string `json:"ipAddress,omitempty"`
	ResponsibleUser *string `json:"responsibleUser,omitempty"`
}

// AddDeviceRequest 手动添加设备请求
type AddDeviceRequest struct {
	WorkstationID   string  `json:"workstationId"`
	DeviceSerial    string  `json:"deviceSerial"`
	DeviceName      *string `json:"deviceName,omitempty"`
	DeviceModel     *string `json:"deviceModel,omitempty"`
	DeviceType      *string `json:"deviceType,omitempty"`
	MACAddress      *string `json:"macAddress,omitempty"`
	IPAddress       *string `json:"ipAddress,omitempty"`
	ResponsibleUser *string `json:"responsibleUser,omitempty"`
	Description     *string `json:"description,omitempty"`
}

// UpdateDeviceRequest 更新设备请求
type UpdateDeviceRequest struct {
	DeviceSerial    *string `json:"deviceSerial,omitempty"`
	DeviceName      *string `json:"deviceName,omitempty"`
	DeviceModel     *string `json:"deviceModel,omitempty"`
	DeviceType      *string `json:"deviceType,omitempty"`
	MACAddress      *string `json:"macAddress,omitempty"`
	IPAddress       *string `json:"ipAddress,omitempty"`
	ResponsibleUser *string `json:"responsibleUser,omitempty"`
	Status          *int    `json:"status,omitempty"`
	Priority        *int    `json:"priority,omitempty"`
	IsPrimary       *bool   `json:"isPrimary,omitempty"`
	Description     *string `json:"description,omitempty"`
}

// ADDeviceMatch 域控设备匹配结果
type ADDeviceMatch struct {
	ADComputerID    string `json:"adComputerId"`
	DeviceSerial    string `json:"deviceSerial"`
	DeviceName      string `json:"deviceName"`
	MACAddress      string `json:"macAddress"`
	IPAddress       string `json:"ipAddress"`
	OperatingSystem string `json:"operatingSystem"`
}

// AssetDeviceMatch 资产设备匹配结果
type AssetDeviceMatch struct {
	AssetID         string  `json:"assetId"`
	DeviceSN        string  `json:"deviceSerial"`
	DeviceModel     *string `json:"deviceModel,omitempty"`
	DeviceType      *string `json:"deviceType,omitempty"`
	MACAddress      *string `json:"macAddress,omitempty"`
	ResponsibleUser *string `json:"responsibleUser,omitempty"`
}

type workstationDeviceService struct {
	db            *gorm.DB
	uuidValidator *regexp.Regexp

	// physicalDevicesFetcher **仅测试注入用**：非 nil 时
	// GetPhysicalDevicesWithDiagnosis 改用它取设备列表。
	//
	// Phase 148：历史 CTE 已移除 —— 单工位与批量召回现共用同一份 GORM 链式 +
	// Go 层合并实现（recallPhysicalRows），双方言均可执行。此钩子保留用于让回归测试
	// 注入替代召回（例如隔离诊断组装逻辑），生产路径恒为 nil。
	// 「不存在第二套 recall 逻辑、口径不分叉」由共享入口机制保证。
	physicalDevicesFetcher func(ctx context.Context, workstationID string) ([]*models.WorkstationDevice, error)
}

// NewWorkstationDeviceService 创建工位设备服务实例
func NewWorkstationDeviceService(db *gorm.DB) WorkstationDeviceService {
	return &workstationDeviceService{
		db:            db,
		uuidValidator: constants.UUIDPattern,
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
	// Phase 133 / DS-05 裁定 2：工位不在可见集 → 返回空集（读端点不报错、不越权）。
	if !workstationDeviceReadAllowed(ctx, s.db, workstationID) {
		return []*models.WorkstationDevice{}, nil
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
	// Phase 133 / DS-05 裁定 2：工位不在可见集 → 返回空集。
	if !workstationDeviceReadAllowed(ctx, s.db, workstationID) {
		return []*models.WorkstationDevice{}, nil
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
			BaseModel:     models.BaseModel{ID: tempID},
			WorkstationID: workstationID,
			DeviceSource:  models.DeviceSourceAD,
			ADComputerID:  &adDeviceCopy.ADComputerID,
			DeviceSerial:  &adDeviceCopy.DeviceSerial,
			DeviceName:    &adDeviceCopy.DeviceName,
			MACAddress:    &adDeviceCopy.MACAddress,
			IPAddress:     &adDeviceCopy.IPAddress,
			Status:        models.WorkstationDeviceStatusNormal,
			IsPrimary:     false,
			Priority:      0,
		})
	}

	return devices, nil
}

// GetAssetDevices 实时查询资产设备（不保存到数据库）
func (s *workstationDeviceService) GetAssetDevices(ctx context.Context, workstationID string) ([]*models.WorkstationDevice, error) {
	if workstationID == "" {
		return nil, apperrors.ParamMissing("工位ID")
	}
	// Phase 133 / DS-05 裁定 2：工位不在可见集 → 返回空集。
	if !workstationDeviceReadAllowed(ctx, s.db, workstationID) {
		return []*models.WorkstationDevice{}, nil
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
			BaseModel:       models.BaseModel{ID: tempID},
			WorkstationID:   workstationID,
			DeviceSource:    models.DeviceSourceAsset,
			AssetID:         &assetDeviceCopy.AssetID,
			DeviceSerial:    &assetDeviceCopy.DeviceSN,
			DeviceModel:     assetDeviceCopy.DeviceModel,
			DeviceType:      assetDeviceCopy.DeviceType,
			MACAddress:      assetDeviceCopy.MACAddress,
			ResponsibleUser: assetDeviceCopy.ResponsibleUser,
			Status:          models.WorkstationDeviceStatusNormal,
			IsPrimary:       false,
			Priority:        0,
		})
	}

	return devices, nil
}

// GetPhysicalDevices 实时查询物理链路设备（不保存到数据库）
//
// Phase 45 R5: 按 MAC→port→infoPoint→workstation 反推链路,得到该工位物理接入的设备。
//
// 链路(同 reconciliation_physical_chain 物化视图):
//
//	asset.mac1/mac2 → sys_device_mac_address → sys_device_port_status
//	  → ops_info_points.workstation_id → sys_workstation.id
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
	// Phase 133 / DS-05 裁定 2：工位不在可见集 → 返回空集（不触发任何方言查询）。
	if !workstationDeviceReadAllowed(ctx, s.db, workstationID) {
		return []*models.WorkstationDevice{}, nil
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

	// 2. Phase 148 / DIALECT-02：走唯一一份共享召回 recallPhysicalRows
	//    （GORM 链式查询 + Go 层归一化/去重/排序/合并，替代原 PG 专属 raw CTE）。
	//    单工位与批量 GetPhysicalDevicesByWorkstations 调用同一入口，口径不分叉。
	recall, err := recallPhysicalRows(ctx, s.db, []string{workstationID}, int(operations.InfoPointStatusNormal))
	if err != nil {
		return nil, fmt.Errorf("查询物理链路设备失败: %w", err)
	}
	rows := recall[workstationID]

	logger.Infof("[GetPhysicalDevices] MAC→port→infoPoint→workstation 反查命中 %d 行", len(rows))

	// 3. 转换为 WorkstationDevice(不写入数据库) —— 字段组装与批量共用 buildPhysicalDevice
	devices := make([]*models.WorkstationDevice, 0, len(rows))
	for i, row := range rows {
		tempID := fmt.Sprintf("physical-%d", i)
		devices = append(devices, buildPhysicalDevice(row, workstationID, workstation.UserID, tempID))
	}

	return devices, nil
}
// GetPhysicalDevicesWithDiagnosis 实时查询物理链路设备 + 空因诊断（DBG-07 / #7）。
//
// 契约：
//   - `Devices` 与 `GetPhysicalDevices` 的返回值**逐条一致**（同一实现、同一次查询）；
//   - 旧方法 `GetPhysicalDevices` 签名/语义零改动 → 旧调用方与旧前端不受影响；
//   - 空因四分类互斥判定，见 PhysicalEmptyReason* 常量。
//
// 为什么需要它：3f130 的实测读数是「信息点/端口/MAC 全通、仅 user_id=NULL」，
// 旧实现无论哪种断点都只回裸 `[]`，前端只能显示空白，无法区分是没绑用户、
// 信息点绑错、还是 MAC 没采到。诊断把「空白」变成可读结论。
func (s *workstationDeviceService) GetPhysicalDevicesWithDiagnosis(ctx context.Context, workstationID string) (*PhysicalDevicesResult, error) {
	// 0. 参数校验与旧方法保持一致（注入路径下也必须有，不能因测试钩子放宽入口约束）
	if workstationID == "" {
		return nil, apperrors.ParamMissing("工位ID")
	}
	if !s.uuidValidator.MatchString(workstationID) {
		return nil, apperrors.ParamInvalid("工位ID")
	}
	// Phase 133 / DS-05 裁定 2：工位不在可见集 → 返回空结果（不泄露越权工位诊断）。
	if !workstationDeviceReadAllowed(ctx, s.db, workstationID) {
		return &PhysicalDevicesResult{
			Devices:      []*models.WorkstationDevice{},
			Total:        0,
			EmptyMessage: "无权访问该工位的数据",
			Warnings:     nil,
			Diagnosis:    PhysicalChainDiagnosis{},
		}, nil
	}

	// 1. 设备列表：复用原实现，保证两个入口的数据绝不分叉
	//
	// （physicalDevicesFetcher 非 nil = 测试注入，见字段注释；生产恒为 nil 走原路径）
	devices, err := func() ([]*models.WorkstationDevice, error) {
		if s.physicalDevicesFetcher != nil {
			return s.physicalDevicesFetcher(ctx, workstationID)
		}
		return s.GetPhysicalDevices(ctx, workstationID)
	}()
	if err != nil {
		return nil, err
	}
	if devices == nil {
		devices = make([]*models.WorkstationDevice, 0)
	}

	// 2. 取工位绑定信息（GetPhysicalDevices 已校验过存在性与 UUID 格式，此处必成功）
	var workstation models.Workstation
	if err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", workstationID).
		First(&workstation).Error; err != nil {
		return nil, fmt.Errorf("工位不存在: %w", err)
	}

	// 3. 逐段链路计数
	diag, err := s.diagnosePhysicalChain(ctx, workstationID, &workstation, devices)
	if err != nil {
		return nil, err
	}

	result := &PhysicalDevicesResult{
		Devices:   devices,
		Total:     len(devices),
		Diagnosis: diag,
	}

	// 4. 空因判定（先命中先返回，顺序即优先级）+ 非致命告警
	result.EmptyReason, result.EmptyMessage = classifyPhysicalEmptyReason(diag, len(devices))
	result.Warnings = collectPhysicalWarnings(diag, len(devices))

	logger.Infof("[GetPhysicalDevicesWithDiagnosis] 工位 %s: total=%d emptyReason=%q 链路计数=%+v",
		workstationID, result.Total, result.EmptyReason, diag)

	return result, nil
}

// diagnosePhysicalChain 逐段计数物理链路（与 GetPhysicalDevices 的 CTE 口径对齐）。
//
// 实现刻意用 GORM chainable + Go 侧集合求交，与 Phase 148 的共享召回同款范式：
//   - 不含任何 PG 专属语法，使诊断可在 SQLite 测试基建上真实执行
//     （回归测试不依赖真实 PG，见任务边界）；
//   - 历史 CTE 已移除，诊断仍是纯读、纯旁路，不改变召回口径。
func (s *workstationDeviceService) diagnosePhysicalChain(
	ctx context.Context,
	workstationID string,
	workstation *models.Workstation,
	devices []*models.WorkstationDevice,
) (PhysicalChainDiagnosis, error) {
	diag := PhysicalChainDiagnosis{WorkstationExists: true}

	// 4.1 用户绑定（3f130 的断点）
	diag.UserBound = workstation.UserID != nil && *workstation.UserID != ""

	// 4.2 信息点总数（含软删/停用）
	var total int64
	if err := s.db.WithContext(ctx).
		Table("ops_info_points").
		Where("workstation_id = ?", workstationID).
		Count(&total).Error; err != nil {
		return diag, fmt.Errorf("诊断：统计信息点总数失败: %w", err)
	}
	diag.InfoPointTotal = int(total)

	// 4.3 active 信息点明细（status=0 且未软删，与 CTE 的 WHERE 一致）
	type infoPointRow struct {
		DeviceID *string
		PortID   *string
	}
	var points []infoPointRow
	if err := s.db.WithContext(ctx).
		Table("ops_info_points").
		Select("device_id, port_id").
		Where("workstation_id = ? AND deleted_at IS NULL AND status = ?",
			workstationID, int(operations.InfoPointStatusNormal)).
		Scan(&points).Error; err != nil {
		return diag, fmt.Errorf("诊断：查询信息点明细失败: %w", err)
	}
	diag.InfoPointActive = len(points)

	// 4.4 已绑定端口的信息点
	portIDs := make([]string, 0, len(points))
	for _, p := range points {
		if p.PortID != nil && *p.PortID != "" {
			diag.InfoPointWithPort++
			portIDs = append(portIDs, *p.PortID)
		}
	}

	// 4.5 端口是否真实存在于 sys_device_port_status（CTE 的 JOIN 条件）
	existingPorts := make(map[string]struct{}, len(portIDs))
	if len(portIDs) > 0 {
		var resolved []string
		if err := s.db.WithContext(ctx).
			Table("sys_device_port_status").
			Where("id IN ?", portIDs).
			Pluck("id", &resolved).Error; err != nil {
			return diag, fmt.Errorf("诊断：查询端口存在性失败: %w", err)
		}
		for _, id := range resolved {
			existingPorts[id] = struct{}{}
		}
	}

	// 4.6 端口在 + 设备在（CTE 的 EXISTS(sys_network_device) 防御）
	// 注意：CTE 的 EXISTS 不额外过滤 sys_network_device.deleted_at，此处保持一致。
	deviceIDs := make([]string, 0, len(points))
	for _, p := range points {
		if p.PortID == nil || *p.PortID == "" {
			continue
		}
		if _, ok := existingPorts[*p.PortID]; !ok {
			continue
		}
		diag.PortResolved++
		if p.DeviceID != nil && *p.DeviceID != "" {
			deviceIDs = append(deviceIDs, *p.DeviceID)
		}
	}

	if len(deviceIDs) > 0 {
		var resolved []string
		if err := s.db.WithContext(ctx).
			Table("sys_network_device").
			Where("id IN ?", deviceIDs).
			Pluck("id", &resolved).Error; err != nil {
			return diag, fmt.Errorf("诊断：查询设备存在性失败: %w", err)
		}
		existingDevices := make(map[string]struct{}, len(resolved))
		for _, id := range resolved {
			existingDevices[id] = struct{}{}
		}
		for _, p := range points {
			if p.PortID == nil || *p.PortID == "" || p.DeviceID == nil || *p.DeviceID == "" {
				continue
			}
			if _, ok := existingPorts[*p.PortID]; !ok {
				continue
			}
			if _, ok := existingDevices[*p.DeviceID]; ok {
				diag.DeviceResolved++
			}
		}
	}

	// 4.7 MAC / 资产 / 域控命中数直接取自 devices（与返回给前端的数据同源，不会口径分叉）
	for _, d := range devices {
		if d == nil {
			continue
		}
		diag.MACMatched++
		if d.AssetID != nil && *d.AssetID != "" {
			diag.AssetMatched++
		}
		if d.ADComputerID != nil && *d.ADComputerID != "" {
			diag.ADMatched++
		}
	}

	return diag, nil
}

// classifyPhysicalEmptyReason 把链路计数映射为「空因」结论。
//
// 判定顺序（先命中先返回，任务是约定的四类）：
//
//	未绑定用户       → no_user_bound        ← **最高优先级，压过「有数据」**
//	有数据且已绑用户  → ""（正常）
//	信息点绑定错位    → info_point_misbound
//	设备/MAC 数据缺失 → device_or_mac_missing
//
// 关键决策：**未绑用户压过「有数据」**。3f130 实测 mac_match_count=1（链路是通的）
// 但前端面板仍空白——因为面板按「用户→物理链路」维度呈现，未绑用户时召回结果无从归属。
// 因此「未绑用户」就是该现象的真实空因，必须报 no_user_bound；设备本身的数据不丢，
// 仍在 Devices 里，同时 Warnings 也会提示该告警，两个通道互补。
func classifyPhysicalEmptyReason(diag PhysicalChainDiagnosis, deviceCount int) (string, string) {
	// ① 未绑定用户（3f130 的实际命中项）——**最高优先级，压过「有数据」**
	//
	// 决策依据（重要，勿轻易改回「数据优先」）：
	//   3f130 实测 mac_match_count=1（链路确实是通的），但前端"物理链路"面板仍为空白——
	//   因为该面板按「用户→物理链路」维度呈现，工位未绑用户时召回结果无法归属到任何人。
	//   所以「未绑用户」不是"有数据就不用管"的旁注，而是**这一现象的真实空因**。
	//   若此处对 deviceCount>0 返回 ""，需求「可区分空因状态」对 3f130 就没有解决。
	//   同时 Warnings 仍会输出该告警，两个通道语义互补。
	if !diag.UserBound {
		return PhysicalEmptyReasonNoUserBound,
			"该工位未绑定用户（sys_workstation.user_id 为空）。物理链路面板按" +
				"「用户→信息点→端口→MAC」维度呈现，未绑定用户时链路无法归属，故为空。" +
				"请先在工位管理中为该工位绑定使用人。"
	}

	// 有数据且已绑定用户 → 正常（此时 emptyReason 必须为空，不得误报空因）
	if deviceCount > 0 {
		return PhysicalEmptyReasonNone, ""
	}

	// ② 信息点绑定错位：无 active 信息点，或信息点未绑定端口
	if diag.InfoPointActive == 0 {
		if diag.InfoPointTotal == 0 {
			return PhysicalEmptyReasonInfoPointMisbound,
				"该工位没有任何关联信息点（ops_info_points.workstation_id 未指向本工位）。" +
					"请检查信息点的工位绑定是否正确。"
		}
		return PhysicalEmptyReasonInfoPointMisbound,
			"该工位关联的信息点均不可用（已软删除或 status≠0 正常）。" +
				"请在信息点管理中恢复或启用，并确认 status=0。"
	}
	if diag.InfoPointWithPort == 0 {
		return PhysicalEmptyReasonInfoPointMisbound,
			"该工位的信息点未绑定网络端口（port_id 为空），物理链路在第一步即断开。" +
				"请在信息点管理中关联对应的设备端口。"
	}

	// ③ 设备 / MAC 数据缺失
	if diag.PortResolved == 0 {
		return PhysicalEmptyReasonDeviceOrMACMissing,
			"信息点绑定的端口在端口表（sys_device_port_status）中不存在，链路断开。" +
				"端口可能已被重新采集或删除，请重新采集该设备端口或重新绑定信息点。"
	}
	if diag.DeviceResolved == 0 {
		return PhysicalEmptyReasonDeviceOrMACMissing,
			"信息点指向的网络设备在设备表（sys_network_device）中不存在（orphan 绑定）。" +
				"请修正信息点的 device_id，指向真实存在的网络设备。"
	}

	// 端口与设备都在，但没有任何 MAC 命中（实测与历史都没有）
	return PhysicalEmptyReasonDeviceOrMACMissing,
		"端口与网络设备均正常，但该端口没有任何 MAC 采集记录（实测与历史均未命中）。" +
			"请确认该端口下接入终端在线，并检查 MAC 地址采集任务是否正常执行。"
}

// collectPhysicalWarnings 收集「非致命但必须可见」的告警（有数据时也会产生）。
//
// 为什么独立于 EmptyReason：3f130 的实况是「MAC 链路完全通（mac_match_count=1）
// 但工位未绑定 user_id」。此时 devices 非空 → emptyReason 必须为空串（否则就是误报空因），
// 但「未绑用户」恰恰是前端看到空白的根因，不能就此消失。故用 Warnings 承载，
// 让前端可以在「有 N 台」的同时提示「该工位未绑定用户」。
func collectPhysicalWarnings(diag PhysicalChainDiagnosis, deviceCount int) []string {
	warnings := make([]string, 0, 3)

	// ① 未绑定用户：无论有没有数据都要提示（这是 3f130 的根因）
	if !diag.UserBound {
		warnings = append(warnings,
			"该工位未绑定用户（sys_workstation.user_id 为空）。物理链路面板按用户维度展示，"+
				"即使后端能召回设备，前端仍可能显示为空；请为工位绑定使用人。")
	}

	// ② 有数据但资产详情缺失：设备能显示，只是没有型号/责任人之类的资产字段
	if deviceCount > 0 && diag.AssetMatched == 0 {
		warnings = append(warnings,
			"物理链路设备未匹配到资产记录（ops_asset 未登记对应 MAC），设备详情将是空壳。"+
				"建议补录资产或核对 MAC。")
	}

	// ③ 有数据但纯靠历史 MAC：设备可能已离线
	if deviceCount > 0 && diag.MACMatched > 0 && diag.AssetMatched == 0 && diag.ADMatched == 0 {
		warnings = append(warnings,
			"物理链路既无资产也有无域控记录，设备来源仅靠 MAC 采集，信息可信度较低。")
	}

	return warnings
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
	// Phase 133 / DS-05 裁定 2：目标工位越界 → 403（单条写不豁免）。
	if err := checkWorkstationDeviceWriteScope(ctx, s.db, req.WorkstationID); err != nil {
		return nil, err
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
		WorkstationID:   req.WorkstationID,
		DeviceSource:    models.DeviceSourceManual,
		DeviceSerial:    &req.DeviceSerial,
		AssetID:         assetID,
		DeviceName:      req.DeviceName,
		DeviceModel:     deviceModel,
		DeviceType:      deviceType,
		MACAddress:      req.MACAddress,
		IPAddress:       req.IPAddress,
		ResponsibleUser: req.ResponsibleUser,
		Status:          models.WorkstationDeviceStatusNormal,
		IsPrimary:       false,
		Priority:        0,
		Description:     req.Description,
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
	// Phase 133 / DS-05 裁定 2：目标工位越界 → 403。
	if err := checkWorkstationDeviceWriteScope(ctx, s.db, workstationID); err != nil {
		return err
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
			WorkstationID: workstationID,
			DeviceSource:  models.DeviceSourceAD,
			ADComputerID:  &adDevice.ADComputerID,
			DeviceSerial:  &adDevice.DeviceSerial,
			DeviceName:    &adDevice.DeviceName,
			MACAddress:    &adDevice.MACAddress,
			Status:        models.WorkstationDeviceStatusNormal,
			IsPrimary:     false,
			Priority:      0,
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
	// Phase 133 / DS-05 裁定 2：目标工位越界 → 403。
	if err := checkWorkstationDeviceWriteScope(ctx, s.db, workstationID); err != nil {
		return err
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
			WorkstationID:   workstationID,
			DeviceSource:    models.DeviceSourceAsset,
			AssetID:         &assetDevice.AssetID,
			DeviceSerial:    &assetDevice.DeviceSN,
			DeviceModel:     assetDevice.DeviceModel,
			DeviceType:      assetDevice.DeviceType,
			MACAddress:      assetDevice.MACAddress,
			ResponsibleUser: assetDevice.ResponsibleUser,
			Status:          models.WorkstationDeviceStatusNormal,
			IsPrimary:       false,
			Priority:        0,
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
	// Phase 133 / DS-05 裁定 2：设备关联工位越界 → 403（改前侧）。
	if err := checkWorkstationDeviceWriteScope(ctx, s.db, device.WorkstationID); err != nil {
		return err
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
	// Phase 133 / DS-05 裁定 2：设备关联工位越界 → 403（改前侧）。
	wsID, err := deviceWorkstationID(ctx, s.db, id)
	if err != nil {
		return err
	}
	if err := checkWorkstationDeviceWriteScope(ctx, s.db, wsID); err != nil {
		return err
	}

	err = s.db.WithContext(ctx).
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
	// Phase 133 / DS-05 裁定 2：设备关联工位越界 → 403（改前侧）。
	if err := checkWorkstationDeviceWriteScope(ctx, s.db, device.WorkstationID); err != nil {
		return err
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
	// Phase 133 / DS-05 裁定 2：目标工位越界 → 403（提交侧）。
	if err := checkWorkstationDeviceWriteScope(ctx, s.db, req.WorkstationID); err != nil {
		return err
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
			Status:          models.WorkstationDeviceStatusNormal,
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
	// Phase 133 / DS-05 裁定 2：目标工位越界 → 403（提交侧）。
	if err := checkWorkstationDeviceWriteScope(ctx, s.db, workstationID); err != nil {
		return err
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
			Status:          models.WorkstationDeviceStatusNormal,
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
//  1. sys_workstation 批量查 wsID → userID
//  2. sys_user 批量查 userID → username
//  3. sys_ad_user 批量查 username → user_dn
//  4. sys_ad_computer 批量查 (managed_by IN user_dns OR description LIKE 任意 username)
//  5. 代码层映射: computer → user_dn/username → userID → wsID
//
// 实现要点: 所有 Scan 用 map[string]interface{} 而非 struct, 规避 GORM struct 扫描丢失中文字段的问题
func (s *workstationDeviceService) GetADDevicesByWorkstations(
	ctx context.Context,
	workstationIDs []string,
) (map[string][]*models.WorkstationDevice, error) {
	// Phase 133 / DS-05 裁定 2：按工位可见集裁剪，越界工位不进入结果 map。
	workstationIDs = visibleWorkstationIDs(ctx, s.db, workstationIDs)
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
				Status:        models.WorkstationDeviceStatusNormal,
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
	// Phase 133 / DS-05 裁定 2：按工位可见集裁剪，越界工位不进入结果 map。
	workstationIDs = visibleWorkstationIDs(ctx, s.db, workstationIDs)
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
				BaseModel:       models.BaseModel{ID: fmt.Sprintf("asset-%s-%s", safePrefix(wsID, 8), safePrefix(id, 8))},
				WorkstationID:   wsID,
				DeviceSource:    models.DeviceSourceAsset,
				AssetID:         &id,
				DeviceSerial:    &sn,
				DeviceModel:     strPtr(modelName),
				DeviceType:      strPtr(typeName),
				MACAddress:      strPtr(mac),
				ResponsibleUser: strPtr(nowUser),
				Status:          models.WorkstationDeviceStatusNormal,
				IsPrimary:       false,
				Priority:        0,
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
	// Phase 133 / DS-05 裁定 2：按工位可见集裁剪，越界工位不进入结果 map。
	workstationIDs = visibleWorkstationIDs(ctx, s.db, workstationIDs)
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

	// Phase 148 / DIALECT-02：与单工位共用**唯一一份**共享召回 recallPhysicalRows，
	// 批量按 wsID 分组后套用同一套归一化/去重/Top-1 逻辑 —— 不存在第二套 recall 逻辑。
	// 历史 PG 专属 raw CTE（去重算子 / 正则替换 / 空值末位排序 / id 强转）已整体删除。
	recall, err := recallPhysicalRows(ctx, s.db, workstationIDs, int(operations.InfoPointStatusNormal))
	if err != nil {
		return result, fmt.Errorf("批量查询物理链路设备失败: %w", err)
	}

	// 按 wsID 分组并构造 WorkstationDevice（字段组装与单工位共用 buildPhysicalDevice）
	for wsID, rows := range recall {
		for _, row := range rows {
			result[wsID] = append(result[wsID], buildPhysicalDevice(
				row,
				wsID,
				wsUserMap[wsID],
				fmt.Sprintf("physical-%s-%s", safePrefix(wsID, 8), macIDFragment(row.MACAddress)),
			))
		}
	}

	total := 0
	for _, devs := range result {
		total += len(devs)
	}
	logger.Infof("[GetPhysicalDevicesByWorkstations] 批量查询命中 %d 行, 覆盖 %d 工位",
		total, len(workstationIDs))
	return result, nil
}
// ===========================================================================
// Phase 148 / DIALECT-02：物理链路「共享召回 + Go 层合并」（方案 ②）
// ===========================================================================
//
// 历史：单工位 GetPhysicalDevices 与批量 GetPhysicalDevicesByWorkstations 各写一段
// db.Raw(rawSQL, ...) 的 PG 专属 CTE，含 4 处去重算子、正则替换、空值末位排序、
// 6+ 处 id 强转 —— SQLite 无法执行，测试只能「把报错当预期」。
//
// 根治（本 phase）：两段 CTE 整体删除，改「多次 GORM 链式查询 + Go 层归一化/去重/
// 排序/合并」，并把召回抽成**唯一一份** recallPhysicalRows；单工位与批量都调用它，
// 批量按 wsID 分组后套用同一套逻辑 —— 不存在第二套 recall 逻辑、口径不分叉。
// 范式刻意对齐 diagnosePhysicalChain（GORM chainable + Go 侧集合求交），
// physicalDevicesFetcher 注入语义保留、生产路径恒为 nil。
//
// 与 PG 版 CTE 的等价性论证（逐条对照 sqlite_reconciliation_views.go:43-54 模板）：
//
//  1. MAC 归一化
//     PG:   LOWER(正则去分隔符(COALESCE(mac,'')))
//     Go:   strings.ToLower + 去 '.' ':' '-'（normalizePhysicalMAC）
//     逐字符核对：PG 字符类恰好 = Go NewReplacer 的 3 个分隔符，无遗漏无新增。
//
//  2. 接口名归一化
//     PG:   前缀折叠(去空白(LOWER(iface))),
//                         '^(gigabitethernet|gigabitether|ge|gi)','ge')
//     Go:   normalizePhysicalIface —— ToLower + 去全部空白 + 前缀按长度降序折叠。
//     等价性：PG POSIX 正则同位置取最长匹配；Go 显式按最长前缀优先判断，语义一致
//     （对齐 sqliteIfaceFoldCaseSQL:48-53「组内最长前缀优先」论证）。本 CTE 仅
//     gigabit* 一组，4 前缀按长度降序、组内互不包含，逐组核对无交叉。
//     Go 层用 rune 判定覆盖全部空白（比 SQL 嵌套 REPLACE 只去空格更精确）。
//
//  3. Top-1 去重（PG 去重算子 → Go 分组取首）
//     PG latest_mac / latest_mac_history: 按 (mac, device_id, interface_name) 去重
//       ORDER BY ..., collected_at / last_seen 降序（空值末位）
//     Go: 按 (原始 mac, device_id, 原始 interface_name) 分组，组内取时间最大者
//       （NULL 视为最小 → 排最后，对齐空值末位排序），见 physicalTimeGreater。
//     PG ad_devices: 按 norm_mac 去重，updated_at 降序（空值末位）
//     Go: 按 norm_mac 分组取 updated_at 最大者，同上。
//     PG 最终 SELECT: 按 COALESCE(a.devicesn, COALESCE(mac.mac, hist.mac, '')) 去重
//       ORDER BY ..., confidence 降序（空值末位）, wp.interface_name 空值末位
//     Go: 按 dedupKey 分组，组内 confidence 降序（NULL 最后）+ interface_name 升序
//       （NULL 最后）取首条。
//
//  4. HistoryLastSeen / 置信度 / 字段合并「资产优先」
//     hist.last_seen / CASE confidence / COALESCE(a.mac1, mac, hist) / AD 只补不覆盖：
//     全部在 buildPhysicalRecallRow 与 buildPhysicalDevice 里按 CTE 同序计算，方向一致。
//
//  5. 类型对齐（uuid vs varchar）
//     CTE 用 PG 专有强转把 uuid 转 text 与 varchar 侧比较；Go 层查询用双方言
//     CAST(id AS TEXT) IN ?（PG 接受、SQLite 恒等），不在配置/结构体里出现方言片段。
//
// 不加 isPostgres 方言分支、不下沉双方言视图（CTE 带 ? 参数视图无法参数化）、
// 不新增外部依赖。
// ===========================================================================

// physicalRecallRow 是反推链的统一结果行：单工位 / 批量共用，字段与 PG 版 CTE 的
// 最终 SELECT 逐列对齐（资产优先合并已在 Go 层完成）。
type physicalRecallRow struct {
	WSID              string
	AssetID           *string
	DeviceSerial      *string
	DeviceModel       *string
	DeviceType        *string
	MACAddress        *string
	IPAddress         *string
	ResponsibleUser   *string
	PortName          *string
	InfoPointName     *string
	HistoryLastSeen   *time.Time
	Confidence        *float64
	ADComputerID      *string
	ADDeviceName      *string
	ADOperatingSystem *string
}

// normalizePhysicalMAC 对齐 PG 的「小写 + 正则去分隔符」归一化。
func normalizePhysicalMAC(s string) string {
	return strings.ToLower(strings.NewReplacer(".", "", ":", "", "-", "").Replace(s))
}

// normalizePhysicalIface 对齐 PG 的两段正则替换接口名归一化。
func normalizePhysicalIface(s string) string {
	// LOWER + 去全部 \\s+ 空白（覆盖空格/制表/换行）。
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch r {
		case ' ', '\t', '\n', '\v', '\f', '\r', 0x85, 0xA0:
		default:
			b.WriteRune(r)
		}
	}
	lower := b.String()
	// 前缀折叠：最长前缀优先（对齐 POSIX 最长匹配 / sqliteIfaceFoldCaseSQL:48-53）。
	for _, p := range []string{"gigabitethernet", "gigabitether", "ge", "gi"} {
		if strings.HasPrefix(lower, p) {
			return "ge" + lower[len(p):]
		}
	}
	return lower
}

// physicalNormColSQL 生成「列值 MAC 归一化」的双方言 SQL 表达式（仅用于 IN ? 预取匹配，
// 不参与 SQL 侧去重/排序）。PG 与 SQLite 均支持 LOWER/REPLACE/COALESCE，
// 对齐 sqlite_reconciliation_views.go:21 的正则替换 → 嵌套 REPLACE 翻译规则。
func physicalNormColSQL(col string) string {
	return "LOWER(REPLACE(REPLACE(REPLACE(COALESCE(" + col + ", ''), '.', ''), ':', ''), '-', ''))"
}

// physicalTimeGreater 实现「降序 + 空值末位取首条」的比较：非 NULL > NULL，
// 两个非 NULL 比大小，两个 NULL 视为相等（PG 组内任取，结果等价）。
func physicalTimeGreater(a, b *time.Time) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	return a.After(*b)
}

// 各中间查询的扫描结构（列名显式 gorm tag，避免依赖 GORM 命名猜测）。
type physicalPointScan struct {
	WSID          string `gorm:"column:ws_id"`
	PortID        string `gorm:"column:port_id"`
	DeviceID      string `gorm:"column:device_id"`
	InfoPointName string `gorm:"column:info_point_name"`
}

type physicalMACScan struct {
	MacAddress    string     `gorm:"column:mac_address"`
	DeviceID      string     `gorm:"column:device_id"`
	InterfaceName string     `gorm:"column:interface_name"`
	CollectedAt   *time.Time `gorm:"column:collected_at"`
}

type physicalHistScan struct {
	MacAddress    string     `gorm:"column:mac_address"`
	DeviceID      string     `gorm:"column:device_id"`
	InterfaceName string     `gorm:"column:interface_name"`
	LastSeen      *time.Time `gorm:"column:last_seen"`
}

type physicalAssetScan struct {
	ID              string  `gorm:"column:id"`
	DeviceSN        *string `gorm:"column:devicesn"`
	DeviceModel     *string `gorm:"column:device_model_name"`
	DeviceType      *string `gorm:"column:device_type_name"`
	Mac1            *string `gorm:"column:mac1"`
	Mac2            *string `gorm:"column:mac2"`
	IPAddress       *string `gorm:"column:machine_ip"`
	ResponsibleUser *string `gorm:"column:nowuser_name"`
}

type physicalADScan struct {
	ID              string     `gorm:"column:id"`
	MacAddress      *string    `gorm:"column:mac_address"`
	SerialNumber    *string    `gorm:"column:serial_number"`
	ComputerName    *string    `gorm:"column:computer_name"`
	IPAddress       *string    `gorm:"column:ip_address"`
	OperatingSystem *string    `gorm:"column:operating_system"`
	UpdatedAt       *time.Time `gorm:"column:updated_at"`
}

// physicalMACEntry 是 latest_mac / latest_mac_history 去重后的一条（Go 侧 Top-1 结果）。
type physicalMACEntry struct {
	RawMAC    string
	NormMAC   string
	DeviceID  string
	NormIface string
	LastSeen  *time.Time // 仅 history 用
}

// physicalWPRow 是 workstation_ports CTE 等价的有效端口行。
type physicalWPRow struct {
	WSID            string
	InterfaceName   string
	NormIface       string
	EffectiveDevice string
	InfoPointName   string
}

// physicalStableSortRows 是稳定插入排序（自实现，避免为标准库 sort 引入 import 中间态，
// 同时保证相同键的行相对顺序稳定，对齐 PG 组内任取的确定性）。
func physicalStableSortRows(rows []physicalRecallRow, less func(a, b physicalRecallRow) bool) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && less(rows[j], rows[j-1]); j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
}

func physicalStableSortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// recallPhysicalRows 是物理链路的**唯一**召回入口（单工位 / 批量共用）。
// 用多次 GORM 链式查询定向预取（IN ? 而非全表），Go 层完成归一化、分组 Top-1、
// LEFT JOIN 等价合并、最终按主键去重与排序。
func recallPhysicalRows(
	ctx context.Context,
	db *gorm.DB,
	wsIDs []string,
	statusNormal int,
) (map[string][]physicalRecallRow, error) {
	out := make(map[string][]physicalRecallRow, len(wsIDs))
	for _, id := range wsIDs {
		out[id] = []physicalRecallRow{}
	}
	if len(wsIDs) == 0 {
		return out, nil
	}

	// --- A. workstation_ports 等价：活动信息点 ---
	var points []physicalPointScan
	if err := db.WithContext(ctx).
		Table("ops_info_points").
		Select("workstation_id AS ws_id, port_id, device_id, name AS info_point_name").
		Where("workstation_id IN ? AND deleted_at IS NULL AND status = ?", wsIDs, statusNormal).
		Scan(&points).Error; err != nil {
		return nil, fmt.Errorf("召回：查询活动信息点失败: %w", err)
	}

	portIDSet := make(map[string]struct{})
	deviceIDSet := make(map[string]struct{})
	for _, p := range points {
		if p.PortID != "" {
			portIDSet[p.PortID] = struct{}{}
		}
		if p.DeviceID != "" {
			deviceIDSet[p.DeviceID] = struct{}{}
		}
	}
	portIDs := physicalSortedKeys(portIDSet)
	deviceIDs := physicalSortedKeys(deviceIDSet)

	// --- B. 端口 JOIN：CAST(id AS TEXT) = ip.port_id（对齐 CTE 的 id 强转） ---
	portIface := make(map[string]string, len(portIDs))
	if len(portIDs) > 0 {
		var rows []struct {
			ID            string `gorm:"column:id"`
			InterfaceName string `gorm:"column:interface_name"`
		}
		if err := db.WithContext(ctx).
			Table("sys_device_port_status").
			Select("CAST(id AS TEXT) AS id, interface_name").
			Where("CAST(id AS TEXT) IN ?", portIDs).
			Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("召回：查询端口失败: %w", err)
		}
		for _, r := range rows {
			portIface[r.ID] = r.InterfaceName
		}
	}

	// --- C. EXISTS(sys_network_device) 防御 ---
	existingDevices := make(map[string]struct{}, len(deviceIDs))
	if len(deviceIDs) > 0 {
		var rows []struct {
			ID string `gorm:"column:id"`
		}
		if err := db.WithContext(ctx).
			Table("sys_network_device").
			Select("CAST(id AS TEXT) AS id").
			Where("CAST(id AS TEXT) IN ?", deviceIDs).
			Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("召回：查询网络设备存在性失败: %w", err)
		}
		for _, r := range rows {
			existingDevices[r.ID] = struct{}{}
		}
	}

	wpRows := make([]physicalWPRow, 0, len(points))
	for _, p := range points {
		iface, ok := portIface[p.PortID]
		if !ok {
			continue
		}
		if _, ok := existingDevices[p.DeviceID]; !ok {
			continue
		}
		wpRows = append(wpRows, physicalWPRow{
			WSID:            p.WSID,
			InterfaceName:   iface,
			NormIface:       normalizePhysicalIface(iface),
			EffectiveDevice: p.DeviceID,
			InfoPointName:   p.InfoPointName,
		})
	}
	if len(wpRows) == 0 {
		return out, nil
	}

	// --- D. latest_mac：按 (mac, device_id, interface_name) 分组，collected_at 取 Top-1 ---
	macByKey := make(map[string][]physicalMACEntry)
	if len(deviceIDs) > 0 {
		var scans []physicalMACScan
		if err := db.WithContext(ctx).
			Table("sys_device_mac_address").
			Select("mac_address, CAST(device_id AS TEXT) AS device_id, interface_name, collected_at").
			Where("CAST(device_id AS TEXT) IN ?", deviceIDs).
			Scan(&scans).Error; err != nil {
			return nil, fmt.Errorf("召回：查询实测 MAC 失败: %w", err)
		}
		type groupKey struct{ mac, dev, iface string }
		best := make(map[groupKey]physicalMACEntry)
		bestSeen := make(map[groupKey]*time.Time)
		has := make(map[groupKey]bool)
		for _, s := range scans {
			k := groupKey{s.MacAddress, s.DeviceID, s.InterfaceName}
			if !has[k] || physicalTimeGreater(s.CollectedAt, bestSeen[k]) {
				best[k] = physicalMACEntry{
					RawMAC:    s.MacAddress,
					NormMAC:   normalizePhysicalMAC(s.MacAddress),
					DeviceID:  s.DeviceID,
					NormIface: normalizePhysicalIface(s.InterfaceName),
				}
				bestSeen[k] = s.CollectedAt
				has[k] = true
			}
		}
		for _, e := range best {
			key := physicalJoinKey(e.NormIface, e.DeviceID)
			macByKey[key] = append(macByKey[key], e)
		}
	}

	// --- E. latest_mac_history：同上，按 last_seen 取 Top-1，带 HistoryLastSeen ---
	histByKey := make(map[string][]physicalMACEntry)
	if len(deviceIDs) > 0 {
		var scans []physicalHistScan
		if err := db.WithContext(ctx).
			Table("sys_device_mac_history").
			Select("mac_address, CAST(device_id AS TEXT) AS device_id, interface_name, last_seen").
			Where("CAST(device_id AS TEXT) IN ?", deviceIDs).
			Scan(&scans).Error; err != nil {
			return nil, fmt.Errorf("召回：查询历史 MAC 失败: %w", err)
		}
		type groupKey struct{ mac, dev, iface string }
		best := make(map[groupKey]physicalMACEntry)
		bestSeen := make(map[groupKey]*time.Time)
		has := make(map[groupKey]bool)
		for _, s := range scans {
			k := groupKey{s.MacAddress, s.DeviceID, s.InterfaceName}
			if !has[k] || physicalTimeGreater(s.LastSeen, bestSeen[k]) {
				best[k] = physicalMACEntry{
					RawMAC:    s.MacAddress,
					NormMAC:   normalizePhysicalMAC(s.MacAddress),
					DeviceID:  s.DeviceID,
					NormIface: normalizePhysicalIface(s.InterfaceName),
					LastSeen:  s.LastSeen,
				}
				bestSeen[k] = s.LastSeen
				has[k] = true
			}
		}
		for _, e := range best {
			key := physicalJoinKey(e.NormIface, e.DeviceID)
			histByKey[key] = append(histByKey[key], e)
		}
	}

	// --- F. 归一化 MAC 集合（供 asset / ad 定向预取） ---
	normMacSet := make(map[string]struct{})
	for _, list := range macByKey {
		for _, e := range list {
			if e.NormMAC != "" {
				normMacSet[e.NormMAC] = struct{}{}
			}
		}
	}
	for _, list := range histByKey {
		for _, e := range list {
			if e.NormMAC != "" {
				normMacSet[e.NormMAC] = struct{}{}
			}
		}
	}
	normMacs := physicalSortedKeys(normMacSet)

	// --- G. ops_asset（资产优先）：按 norm(mac1)/norm(mac2) IN ? 定向预取 ---
	assetByNormMAC := make(map[string][]physicalAssetScan)
	deviceSNSet := make(map[string]struct{})
	if len(normMacs) > 0 {
		var scans []physicalAssetScan
		where := "deleted_at IS NULL AND (" + physicalNormColSQL("mac1") + " IN ? OR " + physicalNormColSQL("mac2") + " IN ?)"
		if err := db.WithContext(ctx).
			Table("ops_asset").
			Select("id, devicesn, device_model_name, device_type_name, mac1, mac2, machine_ip, nowuser_name").
			Where(where, normMacs, normMacs).
			Scan(&scans).Error; err != nil {
			return nil, fmt.Errorf("召回：查询资产失败: %w", err)
		}
		for _, a := range scans {
			if a.Mac1 != nil {
				n := normalizePhysicalMAC(*a.Mac1)
				if n != "" {
					assetByNormMAC[n] = append(assetByNormMAC[n], a)
				}
			}
			if a.Mac2 != nil {
				n := normalizePhysicalMAC(*a.Mac2)
				if n != "" {
					assetByNormMAC[n] = append(assetByNormMAC[n], a)
				}
			}
			if a.DeviceSN != nil && *a.DeviceSN != "" {
				deviceSNSet[*a.DeviceSN] = struct{}{}
			}
		}
	}
	deviceSNs := physicalSortedKeys(deviceSNSet)

	// --- H. ad_devices：按 norm_mac 分组（updated_at 降序、空值末位，取 Top-1）
	//        + serial_number = a.devicesn 补充命中，一次 IN ? 预取后在 Go 层去重 ---
	adByNormMAC := make(map[string]physicalADScan)
	adBySerial := make(map[string]physicalADScan)
	{
		var scans []physicalADScan
		q := db.WithContext(ctx).
			Table("sys_ad_computer").
			Select("id, mac_address, serial_number, computer_name, ip_address, operating_system, updated_at").
			Where("deleted_at IS NULL")
		if len(normMacs) > 0 || len(deviceSNs) > 0 {
			q = q.Where("("+physicalNormColSQL("mac_address")+" IN ? OR serial_number IN ?)", normMacs, deviceSNs)
		}
		if err := q.Scan(&scans).Error; err != nil {
			return nil, fmt.Errorf("召回：查询域控设备失败: %w", err)
		}
		bestTime := make(map[string]*time.Time)
		for _, a := range scans {
			if a.MacAddress != nil {
				n := normalizePhysicalMAC(*a.MacAddress)
				if n != "" {
					if _, exists := adByNormMAC[n]; !exists || physicalTimeGreater(a.UpdatedAt, bestTime[n]) {
						adByNormMAC[n] = a
						bestTime[n] = a.UpdatedAt
					}
				}
			}
			if a.SerialNumber != nil && *a.SerialNumber != "" {
				sk := "sn:" + *a.SerialNumber
				if _, exists := adBySerial[*a.SerialNumber]; !exists || physicalTimeGreater(a.UpdatedAt, bestTime[sk]) {
					adBySerial[*a.SerialNumber] = a
					bestTime[sk] = a.UpdatedAt
				}
			}
		}
	}

	// --- I. LEFT JOIN 等价合并：wp × latest_mac × latest_mac_history → 候选行 ---
	dedupSeen := make(map[string][]physicalRecallRow)
	for _, wp := range wpRows {
		key := physicalJoinKey(wp.NormIface, wp.EffectiveDevice)
		macs := macByKey[key]
		hists := histByKey[key]
		// SQL LEFT JOIN：0 条侧补一个 nil 占位，保证端口行仍产出（confidence=0.0）。
		macList := make([]*physicalMACEntry, 0, len(macs)+1)
		for i := range macs {
			macList = append(macList, &macs[i])
		}
		if len(macList) == 0 {
			macList = append(macList, nil)
		}
		histList := make([]*physicalMACEntry, 0, len(hists)+1)
		for i := range hists {
			histList = append(histList, &hists[i])
		}
		if len(histList) == 0 {
			histList = append(histList, nil)
		}

		for _, mac := range macList {
			for _, hist := range histList {
				// COALESCE(mac.norm_mac, hist.norm_mac)
				normMAC := ""
				if mac != nil {
					normMAC = mac.NormMAC
				} else if hist != nil {
					normMAC = hist.NormMAC
				}

				// 资产优先：按 normMAC 找 asset（对齐 SQL LIMIT 语义取第一条）
				var asset *physicalAssetScan
				if normMAC != "" {
					if list := assetByNormMAC[normMAC]; len(list) > 0 {
						asset = &list[0]
					}
				}

				// AD：norm_mac 命中 或 serial_number = a.devicesn
				var ad *physicalADScan
				if normMAC != "" {
					if a, ok := adByNormMAC[normMAC]; ok {
						ad = &a
					}
				}
				if ad == nil && asset != nil && asset.DeviceSN != nil && *asset.DeviceSN != "" {
					if a, ok := adBySerial[*asset.DeviceSN]; ok {
						ad = &a
					}
				}

				row := buildPhysicalRecallRow(wp, mac, hist, asset, ad)
				// dedupKey = COALESCE(a.devicesn, COALESCE(mac.mac, hist.mac, ''))
				dk := wp.WSID + "\x00" + physicalDedupKey(row)
				dedupSeen[dk] = append(dedupSeen[dk], row)
			}
		}
	}

	// --- J. 最终按主键去重等价：组内排序取首条（confidence 降序、空值末位,
	//        interface_name 空值末位），输出按 (wsID, dedupKey) 稳定排序 ---
	candidates := make(map[string][]physicalRecallRow)
	for _, rows := range dedupSeen {
		physicalStableSortRows(rows, func(a, b physicalRecallRow) bool {
			ci, cj := physicalConfidence(a.Confidence), physicalConfidence(b.Confidence)
			if ci != cj {
				return ci > cj // DESC
			}
			// interface_name 升序，NULL/空 排最后（对齐空值末位排序）
			return physicalStringValue(a.PortName) < physicalStringValue(b.PortName)
		})
		wsID := rows[0].WSID
		candidates[wsID] = append(candidates[wsID], rows[0])
	}
	for wsID := range candidates {
		rows := candidates[wsID]
		physicalStableSortRows(rows, func(a, b physicalRecallRow) bool {
			return physicalDedupKey(a) < physicalDedupKey(b)
		})
		out[wsID] = rows
	}
	return out, nil
}

// physicalJoinKey 构造 latest_mac / history 的 LEFT JOIN 键（norm_iface + device_id）。
func physicalJoinKey(normIface, deviceID string) string {
	return normIface + "\x00" + deviceID
}

// physicalDedupKey 对齐 PG 的 COALESCE(a.devicesn, COALESCE(mac.mac, hist.mac, ''))。
func physicalDedupKey(r physicalRecallRow) string {
	if r.DeviceSerial != nil && *r.DeviceSerial != "" {
		return *r.DeviceSerial
	}
	if r.MACAddress != nil {
		return *r.MACAddress
	}
	return ""
}

// physicalConfidence 取置信度数值（NULL 视为最小，对齐降序空值末位）。
func physicalConfidence(c *float64) float64 {
	if c == nil {
		return -1
	}
	return *c
}

// physicalStringValue 取指针字符串（NULL → 空串，用于空值末位排序）。
func physicalStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// physicalSortedKeys 把 set 稳定排序成切片（保证 IN ? 顺序稳定、测试可复现）。
func physicalSortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	physicalStableSortStrings(out)
	return out
}

// buildPhysicalRecallRow 按 PG 版最终 SELECT 的列顺序，用 Go 侧候选组装一行。
func buildPhysicalRecallRow(
	wp physicalWPRow,
	mac, hist *physicalMACEntry,
	asset *physicalAssetScan,
	ad *physicalADScan,
) physicalRecallRow {
	row := physicalRecallRow{WSID: wp.WSID}

	// 资产字段（优先）
	if asset != nil {
		row.AssetID = physicalStrPtr(asset.ID)
		row.DeviceSerial = asset.DeviceSN
		row.DeviceModel = asset.DeviceModel
		row.DeviceType = asset.DeviceType
		row.IPAddress = asset.IPAddress
		row.ResponsibleUser = asset.ResponsibleUser
	}
	// MAC：COALESCE(a.mac1, mac.mac_address, hist.mac_address)
	if asset != nil && asset.Mac1 != nil && *asset.Mac1 != "" {
		row.MACAddress = asset.Mac1
	} else if mac != nil {
		row.MACAddress = physicalStrPtr(mac.RawMAC)
	} else if hist != nil {
		row.MACAddress = physicalStrPtr(hist.RawMAC)
	}
	// 端口 / 信息点
	row.PortName = physicalStrPtr(wp.InterfaceName)
	row.InfoPointName = physicalStrPtr(wp.InfoPointName)
	// HistoryLastSeen / 置信度分级：实测=1.0，仅历史=0.5，无MAC=0.0（对齐 SQL CASE）
	if hist != nil {
		row.HistoryLastSeen = hist.LastSeen
	}
	confidence := 0.0
	if mac != nil {
		confidence = 1.0
	} else if hist != nil {
		confidence = 0.5
	}
	row.Confidence = &confidence
	// AD 侧只补字段，不覆盖资产字段
	if ad != nil {
		row.ADComputerID = physicalStrPtr(ad.ID)
		row.ADDeviceName = ad.ComputerName
		row.ADOperatingSystem = ad.OperatingSystem
	}
	return row
}

func physicalStrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// buildPhysicalDevice 把统一召回行转换成 WorkstationDevice（单工位 / 批量共用）。
// 字段冲突策略、历史关联降级提示与既有两处实现逐字一致；ID / ResponsibleUserID
// 由调用方注入（单工位用序号 ID + workstation.UserID，批量用 wsID 前缀 ID + 预取 user_id）。
func buildPhysicalDevice(
	row physicalRecallRow,
	wsID string,
	responsibleUserID *string,
	id string,
) *models.WorkstationDevice {
	// 字段冲突策略：资产优先
	var deviceName *string
	if row.ADDeviceName != nil && *row.ADDeviceName != "" {
		deviceName = row.ADDeviceName
	} else if row.DeviceModel != nil && *row.DeviceModel != "" {
		dm := *row.DeviceModel // 资产侧没有 device_name 字段,用 model 兜底
		deviceName = &dm
	}
	// Phase 3 v3 兜底: 当 AD + 资产都没有名字时, 用 MAC + 端口名兜底
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
	// 历史关联降级提示: 设备在实测 MAC 未命中、但历史命中时追加（置信度由召回行计算）。
	if row.Confidence != nil && *row.Confidence < 1.0 && row.HistoryLastSeen != nil && !row.HistoryLastSeen.IsZero() {
		hint := fmt.Sprintf("\\n历史关联 (最后上线时间: %s)", row.HistoryLastSeen.Format("2006-01-02 15:04:05"))
		if portDesc == nil {
			pd := hint
			portDesc = &pd
		} else {
			pdv := *portDesc + hint
			portDesc = &pdv
		}
	}

	return &models.WorkstationDevice{
		BaseModel:         models.BaseModel{ID: id},
		WorkstationID:     wsID,
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
		ResponsibleUserID: responsibleUserID,
		Status:            models.WorkstationDeviceStatusNormal,
		IsPrimary:         false,
		Priority:          0,
		Description:       portDesc,
		Confidence:        row.Confidence,
		HistoryLastSeen:   row.HistoryLastSeen,
	}
}
