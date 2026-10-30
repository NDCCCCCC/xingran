package portcollection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// portCollectionDeviceTimeout 单设备端口采集超时
const portCollectionDeviceTimeout = 10 * time.Minute

// CollectionService 端口采集服务
type CollectionService struct {
	db            *gorm.DB
	executor      *device.DeviceExecutor
	maxConcurrent int
	templateCache *TemplateCache
}

// NewCollectionService 创建端口采集服务
func NewCollectionService(db *gorm.DB, executor *device.DeviceExecutor) *CollectionService {
	svc := &CollectionService{
		db:            db,
		executor:      executor,
		maxConcurrent: 10,
		templateCache: NewTemplateCache(),
	}
	svc.loadConfigFromDB()
	return svc
}

// CollectionResult 采集结果
type CollectionResult struct {
	DeviceID       string    `json:"deviceId"`
	DeviceName     string    `json:"deviceName"`
	SuccessCount   int       `json:"successCount"`
	FailedCount    int       `json:"failedCount"`
	ErrorMessage   string    `json:"errorMessage"`
	CollectionTime time.Time `json:"collectionTime"`
}

// CollectAllDevices 采集所有设备的端口状态
func (s *CollectionService) CollectAllDevices(ctx context.Context) ([]*CollectionResult, error) {
	var devices []models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("status = ?", models.DeviceStatusOnline).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询设备列表失败: %w", err)
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("没有在线设备")
	}

	var results []*CollectionResult
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, s.maxConcurrent)

	for _, dev := range devices {
		wg.Add(1)
		go func(device models.NetworkDevice) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			deviceCtx, cancel := context.WithTimeout(ctx, portCollectionDeviceTimeout)
			defer cancel()

			result := s.collectDevicePort(deviceCtx, &device)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(dev)
	}

	wg.Wait()
	return results, nil
}

// CollectDevice 采集单个设备的端口状态
func (s *CollectionService) CollectDevice(ctx context.Context, deviceID string) (*CollectionResult, error) {
	var device models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", deviceID).First(&device).Error; err != nil {
		return nil, fmt.Errorf("设备不存在: %w", err)
	}

	return s.collectDevicePort(ctx, &device), nil
}

// collectDevicePort 采集设备端口状态（内部方法）
func (s *CollectionService) collectDevicePort(ctx context.Context, device *models.NetworkDevice) *CollectionResult {
	defer func() {
		if r := recover(); r != nil {
			applogger.Warnf("[端口采集] 设备 %s 发生 panic: %v", device.DeviceName, r)
		}
	}()

	result := &CollectionResult{
		DeviceID:       device.ID,
		DeviceName:     device.DeviceName,
		CollectionTime: time.Now(),
	}

	pool := s.executor.GetScheduler().GetConnectionPool()
	conn, err := pool.GetConnection(ctx, device.ID)
	if err != nil {
		applogger.Warnf("[端口采集] %s (%s): 获取连接失败: %v", device.DeviceName, device.IPAddress, err)
		result.ErrorMessage = err.Error()
		return result
	}

	// F-14 Phase 31: GetConnection 内部已 refCount +1,
	// 直接 defer ReleaseRef() (仅 -1 refCount,不操作 mu)。
	defer conn.ReleaseRef()

	wrapper := conn.GetWrapper()
	if wrapper == nil {
		result.ErrorMessage = "wrapper 为空"
		return result
	}

	// 华为/H3C设备：只使用 display interface description
	// 锐捷/迈普设备：使用 display interface brief + description
	var interfaces []InterfaceInfo
	descriptionMap := make(map[string]InterfaceDescription)

	if device.Vendor == models.VendorHuawei || device.Vendor == models.VendorH3C {
		// 华为/H3C：只解析接口描述
		descs, err := parseInterfaceDescriptions(wrapper, device.Vendor)
		if err != nil {
			applogger.Debugf("[端口采集] %s (%s): 解析接口描述失败: %v", device.DeviceName, device.IPAddress, err)
		} else {
			descriptionMap = descs
			applogger.Debugf("[端口采集] %s (%s): 获取到 %d 个接口描述", device.DeviceName, device.IPAddress, len(descs))
		}
	} else {
		// 锐捷/迈普：先获取接口列表
		interfaces, err = parseInterfaceList(wrapper, device.Vendor)
		if err != nil {
			applogger.Warnf("[端口采集] %s (%s): 解析接口列表失败: %v", device.DeviceName, device.IPAddress, err)
			result.ErrorMessage = err.Error()
			return result
		}
		applogger.Debugf("[端口采集] %s (%s): 获取到 %d 个接口", device.DeviceName, device.IPAddress, len(interfaces))

		// 再获取接口描述
		descs, err := parseInterfaceDescriptions(wrapper, device.Vendor)
		if err != nil {
			applogger.Debugf("[端口采集] %s (%s): 解析接口描述失败: %v", device.DeviceName, device.IPAddress, err)
		} else {
			descriptionMap = descs
			applogger.Debugf("[端口采集] %s (%s): 获取到 %d 个接口描述", device.DeviceName, device.IPAddress, len(descs))
		}
	}

	// 获取华为/H3C设备的VLAN信息
	vlanMap := make(map[string]*int)
	if device.Vendor == models.VendorHuawei || device.Vendor == models.VendorH3C {
		vlanMap, err = parseInterfaceVLANInfo(wrapper, device.Vendor)
		if err != nil {
			applogger.Warnf("[端口采集] %s (%s): 获取VLAN信息失败: %v", device.DeviceName, device.IPAddress, err)
			vlanMap = make(map[string]*int)
		}
	}

	// 批量获取802.1X状态
	dot1xMap, err := getAllDot1xStatus(wrapper, device.Vendor, s.templateCache)
	if err != nil {
		applogger.Warnf("[端口采集] 获取802.1X状态失败: %v", err)
		dot1xMap = make(map[string]Dot1xInfo)
	} else {
		// 调试：输出dot1xMap的键
		applogger.Debugf("[端口采集] dot1xMap包含 %d 个接口:", len(dot1xMap))
		for ifaceName := range dot1xMap {
			applogger.Debugf("[端口采集]   - %s", ifaceName)
		}
	}

	// 批量获取端口安全配置（仅锐捷/迈普）
	var securityMap map[string]PortSecurityInfo
	if device.Vendor == models.VendorRuijie || device.Vendor == models.VendorMaipu {
		securityMap, err = getAllPortSecurity(wrapper, device.Vendor, s.templateCache)
		if err != nil {
			applogger.Warnf("[端口采集] 获取端口安全配置失败: %v", err)
			securityMap = make(map[string]PortSecurityInfo)
		}
	} else {
		securityMap = make(map[string]PortSecurityInfo)
	}

	collectionTime := time.Now()
	var portStatuses []*models.DevicePortStatus

	if device.Vendor == models.VendorHuawei || device.Vendor == models.VendorH3C {
		// 华为/H3C：使用接口列表 + descriptionMap（确保所有接口都被处理）
		interfaces, err = parseInterfaceList(wrapper, device.Vendor)
		if err != nil {
			// 接口列表是端口记录的必要数据来源，解析失败（如连接已 Closed）必须终止该设备采集，
			// 避免空端口列表进入后续批量 upsert 触发 GORM "empty slice found" 错误。
			// 与锐捷/迈普分支（上方 line 142-147）保持一致的早返回语义。
			applogger.Warnf("[端口采集] %s (%s): 解析接口列表失败: %v", device.DeviceName, device.IPAddress, err)
			result.ErrorMessage = fmt.Sprintf("解析接口列表失败: %v", err)
			return result
		}
		applogger.Debugf("[端口采集] %s (%s): 获取到 %d 个接口", device.DeviceName, device.IPAddress, len(interfaces))
		emptySecurityMap := make(map[string]PortSecurityInfo)
		for _, iface := range interfaces {
			normalizedName := NormalizeInterfaceName(iface.Name)
			var adminStatus, operStatus, description string
			if desc, ok := descriptionMap[normalizedName]; ok {
				adminStatus = desc.AdminStatus
				operStatus = desc.OperStatus
				description = desc.Description
			}
			portStatus := buildPortStatus(device.ID, normalizedName, iface, adminStatus, operStatus, description, dot1xMap, emptySecurityMap, collectionTime)
			// 添加VLAN信息（优先使用vlanMap中的值）
			if vlan, ok := vlanMap[normalizedName]; ok {
				portStatus.VLAN = vlan
			}
			portStatuses = append(portStatuses, portStatus)
		}
	} else {
		// 锐捷/迈普：使用 interfaces + descriptionMap
		for _, iface := range interfaces {
			normalizedName := NormalizeInterfaceName(iface.Name)
			var adminStatus, operStatus, description string
			if desc, ok := descriptionMap[normalizedName]; ok {
				adminStatus = desc.AdminStatus
				operStatus = desc.OperStatus
				description = desc.Description
			}

			portStatus := buildPortStatus(device.ID, normalizedName, iface, adminStatus, operStatus, description, dot1xMap, securityMap, collectionTime)
			portStatuses = append(portStatuses, portStatus)
		}
	}

	// 批量保存（使用 OnConflict 处理重复键）
	// 使用 device_id 和 interface_name 作为唯一键，如果存在则更新
	// 判空保护：GORM 对空/nil 切片会生成空 VALUES 子句并返回 "empty slice found" 错误。
	// 当采集流程因连接 Closed、接口列表解析返回空等场景导致 portStatuses 为空时，
	// 跳过 upsert 并记录日志，而非让 GORM 抛错。
	if len(portStatuses) == 0 {
		applogger.Warnf("[端口采集] %s (%s): 未采集到任何端口数据，跳过批量保存", device.DeviceName, device.IPAddress)
		result.SuccessCount = 0
		result.FailedCount = 0
		if result.ErrorMessage == "" {
			result.ErrorMessage = "未采集到任何端口数据"
		}
		return result
	}

	// 多设备同名接口 (如每台交换机都有 GE4/0/1、Vlanif26、NULL0) 是合法场景:
	// 唯一约束为复合键 (device_id, interface_name) (见 migration_177 + model uniqueIndex tag),
	// 下方 OnConflict{device_id, interface_name} 会按设备各自 UPSERT,互不覆盖。
	// 历史的 [C-fix] ownership clash 检查基于"interface_name 跨设备唯一"的错误前提,
	// 误杀所有非首采设备的通用接口名端口,已移除。勿重新引入跨设备同名拦截。

	err = s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}, {Name: "interface_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"admin_status", "oper_status", "description", "vlan", "duplex", "speed", "port_type", "dot1x_enabled", "dot1x_port_status", "dot1x_user_limit", "collected_at"}),
	}).Create(&portStatuses).Error
	if err != nil {
		applogger.Warnf("[端口采集] %s (%s): 批量保存失败: %v", device.DeviceName, device.IPAddress, err)
		result.SuccessCount = 0
		result.FailedCount = len(portStatuses)
		if result.ErrorMessage == "" {
			result.ErrorMessage = err.Error()
		}
	} else {
		applogger.Debugf("[端口采集] %s (%s): 成功=%d", device.DeviceName, device.IPAddress, len(portStatuses))
		result.SuccessCount = len(portStatuses)
	}

	return result
}

// loadConfigFromDB 从数据库加载并发数配置
func (s *CollectionService) loadConfigFromDB() {
	// 配置加载逻辑
}

// ReloadConfig 重新加载配置
func (s *CollectionService) ReloadConfig() {
	s.loadConfigFromDB()
}
