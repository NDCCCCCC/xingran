package portcollection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/device"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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

			deviceCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
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
			applogger.Infof("[端口采集] 设备 %s 发生 panic: %v", device.DeviceName, r)
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
		applogger.Infof("[端口采集] %s (%s): 获取连接失败: %v", device.DeviceName, device.IPAddress, err)
		result.ErrorMessage = err.Error()
		return result
	}

	if !conn.Acquire() {
		result.ErrorMessage = "无法获取连接锁"
		return result
	}
	defer conn.Release()

	wrapper := conn.GetWrapper()
	if wrapper == nil {
		result.ErrorMessage = "wrapper 为空"
		return result
	}

	// 获取接口列表
	interfaces, err := parseInterfaceList(wrapper, device.Vendor)
	if err != nil {
		applogger.Infof("[端口采集] %s (%s): 解析接口列表失败: %v", device.DeviceName, device.IPAddress, err)
		result.ErrorMessage = err.Error()
		return result
	}

	applogger.Infof("[端口采集] %s (%s): 获取到 %d 个接口", device.DeviceName, device.IPAddress, len(interfaces))

	// 获取接口状态和描述信息
	descriptionMap := make(map[string]InterfaceDescription)
	if device.Vendor == models.VendorRuijie || device.Vendor == models.VendorMaipu {
		descs, err := parseInterfaceDescriptions(wrapper, device.Vendor)
		if err == nil {
			descriptionMap = descs
		}
	}

	// 批量获取802.1X状态
	dot1xMap, err := getAllDot1xStatus(wrapper, device.Vendor, s.templateCache)
	if err != nil {
		applogger.Infof("[警告] 获取802.1X状态失败: %v", err)
		dot1xMap = make(map[string]Dot1xInfo)
	}

	// 批量获取端口安全配置
	securityMap, err := getAllPortSecurity(wrapper, device.Vendor, s.templateCache)
	if err != nil {
		applogger.Infof("[警告] 获取端口安全配置失败: %v", err)
		securityMap = make(map[string]PortSecurityInfo)
	}

	collectionTime := time.Now()
	var portStatuses []*models.DevicePortStatus

	if len(interfaces) == 0 && len(descriptionMap) > 0 {
		// 降级方案
		for ifaceName, desc := range descriptionMap {
			portStatus := buildPortStatusFromDesc(device.ID, ifaceName, desc, dot1xMap, securityMap, collectionTime)
			portStatuses = append(portStatuses, portStatus)
		}
	} else {
		// 正常流程
		for _, iface := range interfaces {
			normalizedName := normalizeInterfaceName(iface.Name)
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
	err = s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}, {Name: "interface_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"admin_status", "oper_status", "description", "vlan", "duplex", "speed", "port_type", "dot1x_enabled", "dot1x_port_status", "port_security_enabled", "port_security_mode", "max_mac_count", "current_mac_count", "collected_at"}),
	}).Create(&portStatuses).Error
	if err != nil {
		applogger.Infof("[端口采集] %s (%s): 批量保存失败: %v", device.DeviceName, device.IPAddress, err)
		result.SuccessCount = 0
		result.FailedCount = len(portStatuses)
	} else {
		applogger.Infof("[端口采集] %s (%s): 成功=%d", device.DeviceName, device.IPAddress, len(portStatuses))
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
